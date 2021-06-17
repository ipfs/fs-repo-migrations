#!/bin/bash

# This script can be used to generate the repotest folder that is used
# to test the migration, using go-ipfs v0.8.0.

export IPFS_PATH=repotest

mkdir a
echo "file1" > a/file1
mkdir b
echo "file2" > b/file2
mkdir c
echo "file3" > c/file3

ipfs init

# A: add with both v0 and v1
ipfs add -r --pin=false a
ipfs add -r --cid-version=1 --raw-leaves=false --pin=false a

# B: add with v1
ipfs add -r --cid-version=1 --raw-leaves=false --pin=false b

# C: add with v0
ipfs add -r --pin=false c
