#!/bin/bash

set -eo pipefail

if [ -d _vendor ]; then
    rm -rf _vendor.prev
    mv _vendor _vendor.prev
fi

# Create a version of go-ipfs with s3 plugin built in
tmpdir=/tmp/tmp.fs-repo-migration_build
rm -rf "$tmpdir"
git clone -b release-v0.8.0 https://github.com/ipfs/go-ipfs "${tmpdir}/go-ipfs"
pushd "${tmpdir}/go-ipfs"
go get github.com/ipfs/go-ds-s3@latest
echo "s3ds github.com/ipfs/go-ds-s3/plugin 0" >> plugin/loader/preload_list
make build
popd

sed -i "s,\"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/,\"," *.go **/*.go

# Generate the vendor directory using a go.mod file
echo "===> generating contents of _vendor/"
cp go_mod_gen_vendor go.mod
go clean -modcache
go mod vendor
mv vendor _vendor
rm -rf _vendor/github.com/ipfs/fs-repo-migrations
find _vendor/ -name go.mod -delete
find _vendor/ -name go.sum -delete

# Get deps that start with url (string with . before first /) and remove the
# golang.org/x/net dep because that is vendored that at the top level
DEPS_FILE=deps_file.txt
echo "===> creating $DEPS_FILE"
go list -deps | sed -E -n '/^[^/]+[.].+$/p' | sed '/golang.org\/x\/net/d' | sed '/github.com\/ipfs\/fs-repo-migrations/d' > "$DEPS_FILE"

echo "golang.org/x/sys/windows" >> "$DEPS_FILE"
echo "github.com/alexbrainman/goissue34681" >> "$DEPS_FILE"
echo "github.com/libp2p/go-openssl" >> "$DEPS_FILE"
echo "github.com/libp2p/go-openssl/utils" >> "$DEPS_FILE"
echo "github.com/libp2p/go-sockaddr/net"  >> "$DEPS_FILE"
echo "github.com/libp2p/go-sockaddr" >> "$DEPS_FILE"
echo "github.com/libp2p/go-sockaddr/net"  >> "$DEPS_FILE"
echo "golang.org/x/crypto/ed25519/internal/edwards25519" >> "$DEPS_FILE"
echo "github.com/spacemonkeygo/spacelog" >> "$DEPS_FILE"
echo "github.com/marten-seemann/qtls" >> "$DEPS_FILE"

# Edit the import path if a .go file imports anything in the _vendor directory
echo "===> modifying import paths in _vendor"
cat "$DEPS_FILE" | while read line; do find _vendor/ -name '*.go' | xargs sed -i "s,\"$line\",\"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/$line\","; done

# Edit go files outside _vendor only if not already edited
echo "===> modifying non-vendor import paths in .go files not in _vendor"
cat "$DEPS_FILE" | while read line; do sed -i "s,\"$line\",\"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/$line\"," *.go **/*.go; done

# Remove import restrictions
echo "===> removing import restrictions"
find ./ -name '*.go' | xargs sed -i 's,\(package \w\+\) // import .*,\1,'

go vet
echo "===> all done"

rm -rf "$tmpdir"
rm -f go.mod go.sum "$DEPS_FILE"
rm -rf _vendor.prev
