#!/bin/sh

test_description="Test migration 4 to 5 woth -no-revert"

. lib/test-lib.sh

# setup vars for tests

export IPFS_DIST_PATH="/ipfs/QmRgeXaMTu426NcVvTNywAR7HyFBFGhvRc9FuTtKx3Hfno"

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

test_install_ipfs_nd "v0.4.4"

test_init_ipfs_nd

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
test_expect_success "create a permission problem" '
	RANDOM_DIR="`(cd "$IPFS_PATH"/blocks/ && ls -d CI??? | sort | head -3 | tail -1)`" &&
	chmod 000 "$IPFS_PATH/blocks/$RANDOM_DIR"
'

test_expect_success "fs-repo-4-to-5 -no-revert should fail" '
	test_must_fail fs-repo-4-to-5 -no-revert -path "$IPFS_PATH"
'

test_expect_success "fs-repo-4-to-5 -no-revert leaves repo in inconsistent state" '
	test -d "$IPFS_PATH"/blocks-v4 &&
	test -d "$IPFS_PATH"/blocks-v5
'

test_expect_success "fix permission problem" '
	chmod 700 "$IPFS_PATH/blocks-v4/$RANDOM_DIR"
'

test_expect_success "fs-repo-4-to-5 -no-revert should be okay now" '
	fs-repo-4-to-5 -no-revert -path "$IPFS_PATH"
'

test_install_ipfs_nd "v0.4.5-pre2"

test_expect_success "list all refs after migration" '
	ipfs refs local | sort > after_refs
'

test_expect_success "refs look right" '
	comm -23 start_refs after_refs > missing_refs &&
	touch empty_refs_file &&
	test_cmp missing_refs empty_refs_file
'

test_done
