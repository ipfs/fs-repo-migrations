#!/bin/sh

test_description="Test migration 9 to 10"

. lib/test-lib.sh

# Dist specially built with a v0.6.0-dev release
# TODO: Replace with real dist path (this is just a tmp one for testing)
export IPFS_DIST_PATH="/ipfs/Qmc44rcSEktAc7qeEriLaDJ5Up9qq5bEnKyX8iVCjzx7BL"

export GOPATH="$(pwd)/gopath"
mkdir -p gopath/bin
export PATH="$(pwd)/../bin:$GOPATH/bin:$PATH"
echo $IPFS_PATH

test_install_ipfs_nd "v0.5.1"

test_init_ipfs_nd

# We need bootstrap addresses to migrate. These are defaults from v0.5.1
test_expect_success "add bootstrap addresses" '
  test_config_set --json Bootstrap "[
  \"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb\",
  \"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt\",
  \"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ\",
  \"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN\",
  \"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa\"
]"
'

test_expect_success "remember old swarm config" '
  ipfs config Addresses.Swarm | sort > swarm_old
'

# no need to launch daemon as this works offline

test_expect_success "run migration 9 to 10" '
  echo $IPFS_PATH &&
  ipfs-9-to-10 -verbose -path="$IPFS_PATH"
'

test_install_ipfs_nd "v0.6.0-dev"

test_expect_success "migration added quic bootstrapper" '
  ipfs config Bootstrap | grep "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
'

test_expect_success "migration added quic listener" '
  ipfs config Addresses.Swarm | grep "quic"
'

test_expect_success "revert migration 10 to 9 succeeds" '
  ipfs-9-to-10 -revert -verbose -path="$IPFS_PATH"
'

test_install_ipfs_nd "v0.5.1"

test_expect_success "does not revert bootstrap address" '
  ipfs config Bootstrap | grep "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
'

test_done
