#!/bin/bash
#
# Building migrations with datastore plugin
#
# This script builds migrations with a datastore plugin.  Specify the datastore
# plugin repo and one or more migrations to build.  Each migration binary is
# built in its module subdirectory.  Run a migration binary directly, or copy
# it into a directory in PATH to be run by ipfs-update or fs-repo-migrations.
#
# Example:
# ./build-plugin.sh github.com/ipfs/go-ds-s3 10-to-11 11-to-12
#
set -eou pipefail

function usage() {
    echo "usage: $0 plugin_repo x-to-y ...">&2
    echo "example: $0 github.com/ipfs/go-ds-s3 10-to-11" >&2
}

if [ $# -lt 2 ]; then
    echo "too few arguments" >&2
    echo >&2
    usage
    exit 1
fi

plugin_repo="$1"
plugin_name="$(echo $plugin_repo | rev | cut -d '-' -f 1 | rev)"
ds_name="${plugin_name}ds"

BUILD_DIR="$(mktemp -d --suffix=migration_build)"
BUILD_GOIPFS="${BUILD_DIR}/go-ipfs"
IPFS_REPO="github.com/ipfs/go-ipfs"
IPFS_REPO_URL="https://${IPFS_REPO}"

function cleanup {
    rm -rf "${BUILD_DIR}"
}
trap cleanup EXIT

function get_migration() {
    for i in *${1}; do
        if [ -d "$i" ]; then
            echo "$i"
            return 0
        fi
    done
}

function clone_ipfs() {
    local mig="$1"
    if [ ! -d "${mig}/vendor/github.com/ipfs/go-ipfs" ]; then
        echo "migration $mig does not support datastore plugins" >&2
        return 1
    fi

    pushd "$mig"
    local ver="$(go list -f '{{.Version}}' -m github.com/ipfs/go-ipfs)"
    popd
    if echo "$ver" | grep -E 'v.*.[0-9]{14}-[0-9a-f]{12}$'; then
        local commit="$(echo $ver | rev | cut -d '-' -f 1 | rev)"
        echo "===> Getting go-ipfs commit $commit"
        git clone "$IPFS_REPO_URL" "$BUILD_GOIPFS"
        pushd "$BUILD_GOIPFS"
        git checkout "$commit"
        popd
    else
        echo "===> Getting go-ipfs branch $ver"
        git clone -b "$ver" "$IPFS_REPO_URL" "$BUILD_GOIPFS"
    fi
}
    
function bundle_ipfs_plugin() {
    echo "===> Building go-ipfs with datastore plugin $plugin_name"
    pushd "$BUILD_GOIPFS"
    go get "${plugin_repo}@latest"
    popd

    echo "$ds_name ${plugin_repo}/plugin 0" >> "${BUILD_GOIPFS}/plugin/loader/preload_list"
    sed -i '/^\tgo fmt .*/a \\tgo mod tidy' "${BUILD_GOIPFS}/plugin/loader/Rules.mk"
    make -C "$BUILD_GOIPFS" build
}

function build_migration() {
    local mig="$1"
    echo
    echo "===> Building migration $mig with $ds_name plugin"
    pushd "$mig"
    go mod edit -replace "${IPFS_REPO}=${BUILD_GOIPFS}"
    go mod vendor
    go build -mod=vendor
    # Cleanup temporary modifications
    rm -rf vendor
    git checkout vendor go.mod
    if [ -e go.sum ]; then
        git checkout go.sum
    fi
    popd
    echo "===> Done building migration $mig with $ds_name plugin"
}

shift 1
for i in "$@"; do
    migration="$(get_migration ${i})"
    if [ -z "$migration" ]; then
        echo "migration $i does not exist" >&2
        exit 1
    fi

    clone_ipfs "$migration"
    if [ $? -ne 0 ]; then
        continue
    fi

    bundle_ipfs_plugin

    build_migration "$migration"
done
