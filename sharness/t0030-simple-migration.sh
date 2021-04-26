#!/bin/sh

test_description="Simple fs-repo-migrations tests"

. lib/test-lib.sh

test_expect_success "fs-repo-migrations binary is here" '
	test -f "$LOCAL_FS_REPO_MIG"
'

test_expect_success "'fs-repo-migrations -v' works" '
	"$LOCAL_FS_REPO_MIG" -v >actual
'

test_expect_success "'fs-repo-migrations -v' output looks good" '
	echo "11" >expected &&
	test_cmp expected actual
'

test_expect_success "start a docker container" '
	DOCID=$(start_docker)
'

test_expect_success "fs-repo-migrations binary is on the container" '
	exec_docker "$DOCID" "test -f $GUEST_FS_REPO_MIG"
'

test_expect_success "'fs-repo-migrations -v' works" '
	exec_docker "$DOCID" "$GUEST_FS_REPO_MIG -v" >actual
'

test_expect_success "'fs-repo-migrations -v' output looks good" '
	echo "11" >expected &&
	test_cmp expected actual
'

test_install_version "v0.3.9"

test_expect_success "'ipfs init' succeeds" '
	exec_docker "$DOCID" "IPFS_PATH=/root/.ipfs BITS=2048 ipfs init" >actual 2>&1 ||
	test_fsh cat actual
'

test_expect_success ".ipfs/ has been created" '
	exec_docker "$DOCID" "test -d  /root/.ipfs && test -f /root/.ipfs/config"
	exec_docker "$DOCID" "test -d  /root/.ipfs/datastore && test -d /root/.ipfs/blocks"
'

test_expect_success "'fs-repo-migrations -y' works" '
	exec_docker "$DOCID" "$GUEST_FS_REPO_MIG -y -to=3" >actual 2>&1
'

test_expect_success "'fs-repo-migrations -y' output looks good" '
	grep "fs-repo migrated to version 3" actual || 
	test_fsh cat actual
'

test_install_version "v0.4.0"

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
