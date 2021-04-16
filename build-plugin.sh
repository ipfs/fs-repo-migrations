#!/bin/bash
#
# Build a migration with datastore plugins
#
# This script builds a migration with datastore plugins.  Specify the
# migration, such as 10-to-11, and one or more plugin repos to build with.  A
# specific version of a plugin is specified by following it with
# @<version_or_hash>.  The migration binary is built in its module
# subdirectory.  Run the migration binary directly, or copy it into a directory
# in PATH to be run by ipfs-update or fs-repo-migrations.
#
# Example:
# ./build-plugin.sh 10-to-11 github.com/ipfs/go-ds-s3 github.com/ipfs/go-ds-swift@v0.1.0
#
set -eou pipefail

function usage() {
    echo "usage: $0 x-to-y plugin_repo[@<version_or_hash>] ...">&2
    echo "example: $0 10-to-11 github.com/ipfs/go-ds-s3" >&2
}

if [ $# -lt 2 ]; then
    echo "too few arguments" >&2
    echo >&2
    usage
    exit 1
fi

MIGRATION="$1"
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
    local plugin_repo="$1"
    echo "===> Bundling plugin $plugin_repo info go-ipfs for migration"
    local plugin_version=latest
    if [[ "$plugin_repo" == *"@"* ]]; then
        plugin_version="$(echo $plugin_repo | cut -d '@' -f 2)"
        plugin_repo="$(echo $plugin_repo | cut -d '@' -f 1)"
    fi
    echo "plugin version: $plugin_version"
    local plugin_name="$(echo $plugin_repo | rev | cut -d '-' -f 1 | rev)"
    local ds_name="${plugin_name}ds"
    
    pushd "$BUILD_GOIPFS"
    go get "${plugin_repo}@${plugin_version}"
    popd
    echo "$ds_name ${plugin_repo}/plugin 0" >> "${BUILD_GOIPFS}/plugin/loader/preload_list"
}

function build_migration() {
    echo "===> Building go-ipfs with datastore plugins"
    sed -i '/^\tgo fmt .*/a \\tgo mod tidy' "${BUILD_GOIPFS}/plugin/loader/Rules.mk"
    make -C "$BUILD_GOIPFS" build

    local mig="$1"
    echo
    echo "===> Building migration $mig with plugins"
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
    echo "===> Done building migration $mig with plugins"
}

migration="$(get_migration ${MIGRATION})"
if [ -z "$migration" ]; then
    echo "migration $migration does not exist" >&2
    exit 1
fi

clone_ipfs "$migration"
if [ $? -ne 0 ]; then
    continue
fi

shift 1
for repo in "$@"; do
    bundle_ipfs_plugin "$repo"
done

build_migration "$migration"
