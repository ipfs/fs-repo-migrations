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

sort > expected_swarm_addresses <<EOF
/ip4/0.0.0.0/tcp/4001
/ip4/0.0.0.0/udp/4001/quic
/ip4/1.2.3.4/udp/4004/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ/p2p-circuit
/ip6/::/tcp/4002
/ip6/::/tcp/4003/ws
/ip6/::/udp/4002/quic
EOF

get_swarm_addresses() {
  ipfs config Addresses.Swarm | sed -n -e 's/^ *"\(.*\)",\?$/\1/p' | sort
}

check_results() {
  test_expect_success "get new swarm addresses" '
    get_swarm_addresses > actual_swarm_addresses
  '


  test_expect_success "migration added quic bootstrapper" '
    test "$(ipfs config Bootstrap | grep -c "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")" -eq 1
  '
  
  test_expect_success "get new swarm config" '
    test_cmp expected_swarm_addresses actual_swarm_addresses
  '
}

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

test_expect_success "configure listen addresses" '
  test_config_set --json Addresses.Swarm "[
  \"/ip4/0.0.0.0/tcp/4001\",
  \"/ip6/::/tcp/4002\",
  \"/ip6/::/tcp/4003/ws\",
  \"/ip4/1.2.3.4/udp/4004/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ/p2p-circuit\"
]"
'

# no need to launch daemon as this works offline

test_expect_success "run migration 9 to 10" '
  echo $IPFS_PATH &&
  fs-repo-9-to-10 -verbose -path="$IPFS_PATH"
'

test_install_ipfs_nd "v0.6.0-dev"

check_results

test_expect_success "re-run migration 9 to 10" '
  echo 9 > "$IPFS_PATH/version" &&
  echo $IPFS_PATH &&
  fs-repo-9-to-10 -verbose -path="$IPFS_PATH"
'

# Shouldn't do anything this time, now that we have an address.
check_results

# Should also work with /ipfs/ addresses
test_expect_success "add bootstrap addresses" '
  test_config_set --json Bootstrap "[
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb\",
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt\",
  \"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ\",
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN\",
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa\"
]"
'

test_expect_success "re-run migration 9 to 10" '
  echo 9 > "$IPFS_PATH/version" &&
  echo $IPFS_PATH &&
  fs-repo-9-to-10 -verbose -path="$IPFS_PATH"
'

# Shouldn't do anything this time, now that we have an address.
check_results

test_expect_success "revert migration 10 to 9 succeeds" '
  fs-repo-9-to-10 -revert -verbose -path="$IPFS_PATH"
'

test_install_ipfs_nd "v0.5.1"

# Should leave everything alone.
check_results

test_done
