#!/bin/sh

test_description="Test migration 5 to 6"

. lib/test-lib.sh

# setup vars for tests

export IPFS_DIST_PATH="/ipfs/QmXkB6ByNJ1UXLN76MFJ4aeCPQD44MNtJ8Bg8YH2umFML2"

DEPTH=3
NBDIR=3
NBFILE=6
PINTOTAL=20

PINEACH=$(expr $PINTOTAL / 2)

echo "DEPTH: $DEPTH"
echo "NBDIR: $NBDIR"
echo "NBFILE: $NBFILE"
echo "PINTOTAL: $PINTOTAL"
echo "PINEACH: $PINEACH"

export GOPATH="$(pwd)/gopath"
mkdir -p gopath/bin
export PATH="$(pwd)/../bin:$GOPATH/bin:$PATH"


test_install_ipfs_nd "v0.4.10"

test_init_ipfs_nd

test_launch_ipfs_daemon

test_expect_success "make a couple files" '
	rm -rf manyfiles &&
	random-files -depth=$DEPTH -dirs=$NBDIR -files=$NBFILE manyfiles > filenames
'

test_expect_success "add a few files" '
	ipfs add -r -q manyfiles | tee hashes
'

test_expect_success "get full ref list" '
	ipfs refs local | sort > start_refs
'

test_kill_ipfs_daemon

test_install_ipfs_nd "v0.4.11-dev-dspre"

test_launch_ipfs_daemon

test_expect_success "list all refs after migration" '
	ipfs refs local | sort > after_refs
'

test_expect_success "refs look right" '
	comm -23 start_refs after_refs > missing_refs &&
	touch empty_refs_file &&
	test_cmp missing_refs empty_refs_file
'

test_kill_ipfs_daemon

test_expect_success "'fs-repo-5-to-6 -revert' fails without -path" '
	test_must_fail fs-repo-5-to-6 -revert
'

test_expect_success "'fs-repo-5-to-6 -revert' succeeds" '
	fs-repo-5-to-6 -revert -path="$IPFS_PATH" >actual
'

test_install_ipfs_nd "v0.4.10"

test_launch_ipfs_daemon

test_expect_success "list all refs after migration" '
	ipfs refs local | sort > after_refs
'

test_expect_success "refs look right" '
	comm -23 start_refs after_refs > missing_refs &&
	touch empty_refs_file &&
	test_cmp missing_refs empty_refs_file
'

test_kill_ipfs_daemon

test_done
