#!/bin/sh

test_description="Migration to many different versions"

. lib/test-lib.sh

test_expect_success "start a docker container" '
  DOCID=$(start_docker)
'

test_install_version "v0.4.5"

test_expect_success "'ipfs init' succeeds" '
  export IPFS_PATH=/root/.ipfs &&
  exec_docker "$DOCID" "IPFS_PATH=$IPFS_PATH BITS=2048 ipfs init" >actual 2>&1 ||
  test_fsh cat actual
'

test_expect_success ".ipfs/ has been created" '
  exec_docker "$DOCID" "test -d  /root/.ipfs && test -f /root/.ipfs/config"
  exec_docker "$DOCID" "test -d  /root/.ipfs/datastore && test -d /root/.ipfs/blocks"
'

test_install_version "v0.4.0"
test_repo_version "0.4.0"

# ipfs-update should allow migrating back
test_install_version "v0.3.10"
test_repo_version "0.3.10"

test_expect_success "stop docker container" '
  stop_docker "$DOCID"
'

test_done
