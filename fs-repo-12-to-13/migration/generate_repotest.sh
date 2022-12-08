#!/bin/bash

# This script can be used to generate the repotest folder that is used
# to test the migration, using go-ipfs v0.8.0.

export IPFS_PATH=repotest

# use server profile to have some no announces and filters in place
ipfs init -p server -e

# add a forced announces as they must also be updated
ipfs config --json Addresses.Announce "[\"/ip4/1.2.3.4/tcp/1337\",\"/ip4/1.2.3.4/udp/1337/quic\"]"
ipfs config --json Addresses.NoAnnounce "[\"/ip4/1.2.3.4/tcp/1337\",\"/ip4/1.2.3.4/udp/1337/quic\"]"
ipfs config --json Addresses.AppendAnnounce "[\"/ip4/1.2.3.4/tcp/1337\",\"/ip4/1.2.3.4/udp/1337/quic\"]"

# we only update the config, remove anything not config related to make the tree clearer
rm -rf repotest/{blocks,datastore,keystore}
