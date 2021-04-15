#!/bin/sh

test_description="Test migration 7 to 9"

. lib/test-lib.sh

export IPFS_DIST_PATH="/ipfs/QmVxxcTSuryJYdQJGcS8SyhzN7NBNLTqVPAxpu6gp2ZcrR"
export GOPATH="$(pwd)/gopath"
mkdir -p gopath/bin
export PATH="$(pwd)/../bin:$GOPATH/bin:$PATH"
echo $IPFS_PATH

test_install_ipfs_nd "v0.4.23"

test_init_ipfs_nd

# We need bootstrap addresses to migrate. These are defaults from v0.4.23.
test_expect_success "add bootstrap addresses" '
  test_config_set --json Bootstrap "[
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN\",
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa\",
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb\",
  \"/dnsaddr/bootstrap.libp2p.io/ipfs/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt\",
  \"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ\",
  \"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM\",
  \"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu\",
  \"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64\",
  \"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd\",
  \"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM\",
  \"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu\",
  \"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64\",
  \"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd\"
]"
'

test_expect_success "remember old bootstrap" '
  ipfs config Bootstrap |  grep -E -o "Qm[a-zA-Z0-9]+" | sort > bootstrap_old
'

# no need to launch daemon as this works offline

test_expect_success "add some keys to the keystore" '
  ipfs key gen -t rsa thisISKey1 && ipfs key gen -t ed25519 key2
'

test_expect_success "ipfs key list" '
  ipfs key list > key_list
'

test_expect_success "run migration 7 to 8" '
  fs-repo-7-to-8 -verbose -path="$IPFS_PATH" > migration_output8
'

test_expect_success "run migration 8 to 9" '
  fs-repo-8-to-9 -verbose -path="$IPFS_PATH" > migration_output9
'

test_expect_success "migration processed keys" '
  grep "thisISKey1" migration_output9 &&
  grep "key2" migration_output9
'

test_expect_success "migrated files exist" '
  [ -f "${IPFS_PATH}/keystore/key_orugs42jknfwk6jr" ] &&
  [ -f "${IPFS_PATH}/keystore/key_nnsxsmq" ]
'

test_install_ipfs_nd "v0.5.0-rc2"

test_expect_success "ipfs key list is the same" '
   ipfs key list > new_key_list
   test_cmp key_list new_key_list
'

test_expect_success "migration revert to 8 succeeds" '
  fs-repo-8-to-9 -revert -verbose -path="$IPFS_PATH" > revert_output8
'

test_expect_success "migration revert to 7 succeeds" '
  fs-repo-7-to-8 -revert -verbose -path="$IPFS_PATH" > revert_output7
'

test_install_ipfs_nd "v0.4.23"

test_expect_success "bootstrap addresses were reverted" '
  ipfs config Bootstrap |  grep -E -o "Qm[a-zA-Z0-9]+" | sort > bootstrap_revert
  test_cmp bootstrap_old bootstrap_revert
'

test_expect_success "ipfs key list is the same after revert" '
  ipfs key list > revert_key_list
  test_cmp key_list revert_key_list
'

test_done
