#!/bin/sh

test_description="Migration 2 to 3 and 3 to 2"

GUEST_IPFS_2_TO_3="sharness/bin/fs-repo-2-to-3"

. lib/test-lib.sh

test_expect_success "start a docker container" '
	DOCID=$(start_docker)
'

test_install_version "v0.3.10"

test_expect_success "'ipfs init' succeeds" '
	exec_docker "$DOCID" "IPFS_PATH=/root/.ipfs BITS=2048 ipfs init" >actual 2>&1 ||
	test_fsh cat actual
'

test_expect_success ".ipfs/ has been created" '
	exec_docker "$DOCID" "test -d  /root/.ipfs && test -f /root/.ipfs/config"
	exec_docker "$DOCID" "test -d  /root/.ipfs/datastore && test -d /root/.ipfs/blocks"
'

test_expect_success "'ipfs add -r' succeeds" '
	mkdir mountdir &&
	mkdir mountdir/planets &&
	echo "Hello Mars!" >mountdir/planets/mars.txt &&
	echo "Hello Venus!" >mountdir/planets/venus.txt &&
	exec_docker "$DOCID" "cd \"$GUEST_TEST_DIR\" && ipfs add -r mountdir/planets" >actual
'

test_expect_success "'ipfs add -r' output looks good" '
	PLANETS="QmWSgS32xQEcXMeqd3YPJLrNBLSdsfYCep2U7CFkyrjXwY" &&
	MARS="QmPrrHqJzto9m7SyiRzarwkqPcCSsKR2EB1AyqJfe8L8tN" &&
	VENUS="QmU5kp3BH3B8tnWUU2Pikdb2maksBNkb92FHRr56hyghh4" &&
	echo "added $MARS planets/mars.txt" >expected &&
	echo "added $VENUS planets/venus.txt" >>expected &&
	echo "added $PLANETS planets" >>expected &&
	test_cmp expected actual
'

test_expect_success "ipfs cat succeeds" '
	exec_docker "$DOCID" "ipfs cat \"$MARS\" \"$VENUS\"" >actual
'

test_expect_success "ipfs cat output looks good" '
	cat mountdir/planets/mars.txt mountdir/planets/venus.txt >expected &&
	test_cmp expected actual
'

test_expect_success "'fs-repo-2-to-3' works" '
	exec_docker "$DOCID" "$GUEST_IPFS_2_TO_3 -path=/root/.ipfs" >actual
'

test_expect_success "'fs-repo-2-to-3' output looks good" '
	grep "Migration 2 to 3 succeeded" actual ||
	test_fsh cat actual
'

test_install_version "v0.4.0"

test_expect_success "ipfs cat succeeds with hashes from previous version" '
	exec_docker "$DOCID" "ipfs cat \"$MARS\" \"$VENUS\"" >actual
'

test_expect_success "ipfs cat output looks good" '
	cat mountdir/planets/mars.txt mountdir/planets/venus.txt >expected &&
	test_cmp expected actual
'

test_expect_success "'fs-repo-2-to-3 -revert' succeeds" '
	exec_docker "$DOCID" "$GUEST_IPFS_2_TO_3 -revert -path=/root/.ipfs" >actual
'

test_expect_success "'fs-repo-2-to-3 -revert' output looks good" '
	grep "writing keys:" actual ||
	test_fsh cat actual
'

test_install_version "v0.3.10"

test_expect_success "ipfs cat succeeds with hashes from previous version" '
	exec_docker "$DOCID" "ipfs cat \"$MARS\" \"$VENUS\"" >actual
'

test_expect_success "ipfs cat output looks good" '
	cat mountdir/planets/mars.txt mountdir/planets/venus.txt >expected &&
	test_cmp expected actual
'

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
