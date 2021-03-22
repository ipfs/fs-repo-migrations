#!/bin/sh

test_description="Migration to many different versions"

GUEST_IPFS_2_TO_3="sharness/bin/fs-repo-2-to-3"

. lib/test-lib.sh

test_expect_success "start a docker container" '
	DOCID=$(start_docker)
'

test_install_version "v0.3.7"

test_expect_success "'ipfs init' succeeds" '
	export IPFS_PATH=/root/.ipfs &&
	exec_docker "$DOCID" "IPFS_PATH=$IPFS_PATH BITS=2048 ipfs init" >actual 2>&1 ||
	test_fsh cat actual
'

test_expect_success ".ipfs/ has been created" '
	exec_docker "$DOCID" "test -d  /root/.ipfs && test -f /root/.ipfs/config"
	exec_docker "$DOCID" "test -d  /root/.ipfs/datastore && test -d /root/.ipfs/blocks"
'

test_repo_version "0.3.7"

test_install_version "v0.3.10"
test_repo_version "0.3.10"

test_install_version "v0.4.0"
test_repo_version "0.4.0"

test_install_version "v0.3.8"

# By design reverting a migration has to be run manually
test_expect_success "'fs-repo-2-to-3 -revert' succeeds" '
	exec_docker "$DOCID" "$GUEST_IPFS_2_TO_3 -revert -path=/root/.ipfs" >actual
'

test_repo_version "0.3.8"

test_install_version "v0.3.10"
test_repo_version "0.3.10"

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
