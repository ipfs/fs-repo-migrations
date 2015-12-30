#!/bin/sh

test_description="Basic ipfs-update tests"

. lib/test-lib.sh

LOCAL_IPFS_UPDATE="../bin/ipfs-update"
GUEST_IPFS_UPDATE="sharness/bin/ipfs-update"

test_expect_success "ipfs-update binary is here" '
	test -f "$LOCAL_IPFS_UPDATE"
'

test_expect_success "'ipfs-update versions' works" '
	"$LOCAL_IPFS_UPDATE" versions >actual
'

test_expect_success "'ipfs-update versions' output looks good" '
	grep v0.3.7 actual &&
	grep v0.3.8 actual &&
	grep v0.3.9 actual
'

test_expect_success "start a docker container" '
	DOCID=$(start_docker)
'

test_expect_success "ipfs-update binary is on the container" '
	exec_docker "$DOCID" "test -f $GUEST_IPFS_UPDATE"
'

test_expect_success "'ipfs-update version' works" '
	exec_docker "$DOCID" "$GUEST_IPFS_UPDATE version" >actual
'

test_expect_success "'ipfs-update version' output looks good" '
	echo "none" >expected &&
	test_cmp expected actual
'

test_expect_success "'ipfs-update versions' works" '
	exec_docker "$DOCID" "$GUEST_IPFS_UPDATE versions" >actual
'

test_expect_success "'ipfs-update versions' output looks good" '
	grep v0.3.7 actual &&
	grep v0.3.8 actual &&
	grep v0.3.9 actual
'

test_expect_success "install an ipfs version in docker container" '
	exec_docker "$DOCID" "$GUEST_IPFS_UPDATE --verbose install v0.3.9" >actual 2>&1 ||
	test_fsh cat actual
'

test_expect_success "'ipfs-update install' output looks good" '
	grep "fetching ipfs version v0.3.9" actual
'

test_expect_success "'ipfs-update version' works" '
	exec_docker "$DOCID" "$GUEST_IPFS_UPDATE version" >actual
'

test_expect_success "'ipfs-update version' output looks good" '
	echo "v0.3.9" >expected &&
	test_cmp expected actual
'

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
