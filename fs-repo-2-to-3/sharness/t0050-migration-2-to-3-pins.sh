#!/bin/sh

test_description="Migration 2 to 3 and 3 to 2 with pins"

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

HASH_FILE6="QmRsBC3Y2G6VRPYGAVpZczx1W7Xw54MtM1NcLKTkn6rx3U"
HASH_FILE5="QmaN3PtyP8DcVGHi3Q2Fcp7CfAFVcVXKddWbHoNvaA41zf"
HASH_FILE4="QmV1aiVgpDknKQugrK59uBUbMrPnsQM1F9FXbFcfgEvUvH"
HASH_FILE3="QmZrr4Pzqp3NnMzMfbMhNe7LghfoUFHVx7c9Po9GZrhKZ7"
HASH_FILE2="QmSkjTornLY72QhmK9NvAz26815pTaoAL42rF8Qi3w2WBP"
HASH_FILE1="QmbgX4aXhSSY88GHmPQ4roizD8wFwPX8jzTLjc8VAp89x4"
HASH_DIR4="QmW98gV71Ns4bX7QbgWAqLiGF3SDC1JpveZSgBh4ExaSAd"
HASH_DIR3="QmRsCaNBMkweZ9vHT5PJRd2TT9rtNKEKyuognCEVxZxF1H"
HASH_DIR2="QmTUTQAgeVfughDSFukMZLbfGvetDJY7Ef5cDXkKK4abKC"
HASH_DIR1="QmNyZVFbgvmzguS2jVMRb8PQMNcCMJrn9E3doDhBbcPNTY"

test_expect_success "'ipfs add dir' succeeds" '
	mkdir dir1 &&
	mkdir dir1/dir2 &&
	mkdir dir1/dir2/dir4 &&
	mkdir dir1/dir3 &&
	echo "some text 1" >dir1/file1 &&
	echo "some text 2" >dir1/file2 &&
	echo "some text 3" >dir1/file3 &&
	echo "some text 1" >dir1/dir2/file1 &&
	echo "some text 4" >dir1/dir2/file4 &&
	echo "some text 1" >dir1/dir2/dir4/file1 &&
	echo "some text 2" >dir1/dir2/dir4/file2 &&
	echo "some text 6" >dir1/dir2/dir4/file6 &&
	echo "some text 2" >dir1/dir3/file2 &&
	echo "some text 5" >dir1/dir3/file5 &&
	exec_docker "$DOCID" "cd \"$GUEST_TEST_DIR\" && ipfs add -q -r dir1" >actualall &&
	tail -n1 actualall >actual1 &&
	echo "$HASH_DIR1" >expected1 &&
	exec_docker "$DOCID" "ipfs repo gc" && # remove the patch chaff
	test_cmp expected1 actual1
'

test_expect_success "objects are there" '
	exec_docker "$DOCID" "ipfs cat $HASH_FILE6" >FILE6_a &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE5" >FILE5_a &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE4" >FILE4_a &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE3" >FILE3_a &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE2" >FILE2_a &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE1" >FILE1_a &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR3"   >DIR3_a &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR4"   >DIR4_a &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR2"   >DIR2_a &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR1"   >DIR1_a
'

test_expect_success "ipfs object links $HASH_DIR1 works" '
	exec_docker "$DOCID" "ipfs object links $HASH_DIR1" > DIR1_objlink
'

test_expect_success "added dir was pinned recursively" '
	exec_docker "$DOCID" "ipfs pin ls --type=recursive" > recurse_actual1 &&
	grep "$HASH_DIR1" recurse_actual1 ||
	test_fsh cat recurse_actual1
'

test_expect_success "'ipfs pin ls --type=all' works" '
	exec_docker "$DOCID" "ipfs pin ls --type=all" > all_actual1
'

test_expect_success "'fs-repo-2-to-3' works" '
	exec_docker "$DOCID" "$GUEST_IPFS_2_TO_3 -path=/root/.ipfs" >actual
'

test_expect_success "'fs-repo-2-to-3' output looks good" '
	grep "Migration 2 to 3 succeeded" actual ||
	test_fsh cat actual
'

test_install_version "v0.4.0"

test_expect_success "added dir is still pinned recursively" '
	exec_docker "$DOCID" "ipfs pin ls --type=recursive" > recurse_actual2 &&
	grep "$HASH_DIR1" recurse_actual2 ||
	test_fsh cat recurse_actual2
'

test_expect_success "list of pinned object hasn't changed" '
	exec_docker "$DOCID" "ipfs pin ls --type=all" > all_actual2 &&
	test_sort_cmp all_actual1 all_actual2
'

test_expect_success "objects are still there" '
	exec_docker "$DOCID" "ipfs cat $HASH_FILE6" >FILE6_b &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE5" >FILE5_b &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE4" >FILE4_b &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE3" >FILE3_b &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE2" >FILE2_b &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE1" >FILE1_b &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR3"   >DIR3_b &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR4"   >DIR4_b &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR2"   >DIR2_b &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR1"   >DIR1_b
'

test_expect_success "objects haven't changed" '
	test_cmp FILE6_a FILE6_b &&
	test_cmp FILE5_a FILE5_b &&
	test_cmp FILE4_a FILE4_b &&
	test_cmp FILE3_a FILE3_b &&
	test_cmp FILE2_a FILE2_b &&
	test_cmp FILE1_a FILE1_b &&
	test_cmp DIR3_a DIR3_b &&
	test_cmp DIR4_a DIR4_b &&
	test_cmp DIR2_a DIR2_b &&
	test_cmp DIR1_a DIR1_b
'

test_expect_success "'ipfs repo gc' succeeds" '
	exec_docker "$DOCID" "ipfs repo gc"
'

test_expect_success "'fs-repo-2-to-3 -revert' succeeds" '
	exec_docker "$DOCID" "$GUEST_IPFS_2_TO_3 -verbose -revert -path=/root/.ipfs" >actual
'

test_expect_success "'fs-repo-2-to-3 -revert' output looks good" '
	grep "writing keys:" actual ||
	test_fsh cat actual
'

test_install_version "v0.3.10"

test_expect_success "added dir is still pinned recursively" '
	exec_docker "$DOCID" "ipfs pin ls --type=recursive" > recurse_actual3 &&
	grep "$HASH_DIR1" recurse_actual3 ||
	test_fsh cat recurse_actual3
'

test_expect_success "list of pinned object hasn't changed" '
	exec_docker "$DOCID" "ipfs pin ls --type=all" > all_actual3 &&
	test_sort_cmp all_actual1 all_actual3
'

test_expect_success "objects are still there" '
	exec_docker "$DOCID" "ipfs cat $HASH_FILE6" >FILE6_c &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE5" >FILE5_c &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE4" >FILE4_c &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE3" >FILE3_c &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE2" >FILE2_c &&
	exec_docker "$DOCID" "ipfs cat $HASH_FILE1" >FILE1_c &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR3"   >DIR3_c &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR4"   >DIR4_c &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR2"   >DIR2_c &&
	exec_docker "$DOCID" "ipfs ls $HASH_DIR1"   >DIR1_c
'

test_expect_success "objects haven't changed" '
	test_cmp FILE6_a FILE6_c &&
	test_cmp FILE5_a FILE5_c &&
	test_cmp FILE4_a FILE4_c &&
	test_cmp FILE3_a FILE3_c &&
	test_cmp FILE2_a FILE2_c &&
	test_cmp FILE1_a FILE1_c &&
	test_cmp DIR3_a DIR3_c &&
	test_cmp DIR4_a DIR4_c &&
	test_cmp DIR2_a DIR2_c &&
	test_cmp DIR1_a DIR1_c
'

test_expect_success "'ipfs repo gc' succeeds" '
	exec_docker "$DOCID" "ipfs repo gc"
'

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
