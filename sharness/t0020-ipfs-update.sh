#!/bin/sh

test_description="Basic ipfs-update tests"

. lib/test-lib.sh

test_expect_success "ipfs-update binary is here" '
	test -f "$LOCAL_IPFS_UPDATE"
'

test_expect_success "'ipfs-update versions' works" '
	"$LOCAL_IPFS_UPDATE" versions >actual ||
	test_fsh cat actual
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

test_install_version "v0.3.9"

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
