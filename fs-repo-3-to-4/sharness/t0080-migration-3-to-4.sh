#!/bin/sh

test_description="Test migration 3 to 4 with lots of objects"

. lib/test-lib.sh

# setup vars for tests

DEPTH=3
NBDIR=3
NBFILE=6
PINTOTAL=20

if test_have_prereq EXPENSIVE
then
	DEPTH=6
	NBDIR=7
	NBFILE=10
	PINTOTAL=2000
fi

PINEACH=$(expr $PINTOTAL / 2)

echo "DEPTH: $DEPTH"
echo "NBDIR: $NBDIR"
echo "NBFILE: $NBFILE"
echo "PINTOTAL: $PINTOTAL"
echo "PINEACH: $PINEACH"

export GOPATH="$(pwd)/gopath"
mkdir -p gopath/bin
export PATH="$(pwd)/../bin:$GOPATH/bin:$PATH"


test_install_ipfs_nd "v0.4.2"

test_init_ipfs_nd

test_launch_ipfs_daemon

test_expect_success "make a couple files" '
	rm -rf manyfiles &&
	random-files -depth=$DEPTH -dirs=$NBDIR -files=$NBFILE manyfiles > filenames
'

test_expect_success "add a few files" '
	ipfs add -r -q manyfiles | tee hashes
'

test_expect_success "unpin root so we can do things ourselves" '
	ipfs pin rm $(tail -n1 hashes)
'

test_expect_success "select random subset to pin recursively and directly" '
	sort -R hashes | head -n$PINTOTAL > topin &&
	head -n$PINEACH topin > recurpins &&
	tail -n$PINEACH topin > directpins
'

pin_hashes() {
	hashes_file="$1"
	opts="$2"
	for h in `cat $hashes_file`; do
		if ! ipfs pin add $opts $h; then
			return 1
		fi
	done
}

test_expect_success "pin some objects recursively" '
	pin_hashes recurpins
'

test_expect_success "pin some objects directly" '
	pin_hashes directpins "-r=false"
'

test_expect_success "add some files with the path clean bug" '
	printf ba | ipfs add -q > buggy_hashes &&
	printf bbd | ipfs add -q >> buggy_hashes &&
	printf cdbd | ipfs add -q >> buggy_hashes &&
	printf aabdb | ipfs add -q >> buggy_hashes &&
	printf bccac | ipfs add -q >> buggy_hashes &&
	echo 0243397916 | ipfs add -q >> buggy_hashes && # produces /../ in binary key
	sort buggy_hashes -o buggy_hashes
'

test_expect_success "get full ref list" '
	ipfs refs local | sort > start_refs
'

test_expect_success "ensure buggy hashes dont show up in ref list" '
	comm -12 start_refs buggy_hashes > badrefs &&
	test ! -s badrefs
'

test_expect_success "get pin lists" '
	ipfs pin ls --type=recursive | sort > start_rec_pins &&
	ipfs pin ls --type=direct | sort > start_dir_pins &&
	ipfs pin ls --type=indirect | sort > start_ind_pins
'

test_kill_ipfs_daemon

test_expect_success "'fs-repo-3-to-4 -no-revert' fails" '
	test_must_fail fs-repo-3-to-4 -no-revert -path="$IPFS_PATH"
'

test_install_ipfs_nd "v0.4.3"

test_launch_ipfs_daemon

test_expect_success "list all refs after migration" '
	ipfs refs local | sort > after_refs
'

test_expect_success "list all pins after migration" '
	ipfs pin ls --type=recursive | sort > after_rec_pins &&
	ipfs pin ls --type=direct | sort > after_dir_pins &&
	ipfs pin ls --type=indirect | sort > after_ind_pins
'

test_expect_success "refs look right" '
	comm -23 start_refs after_refs > missing_refs &&
	touch empty_refs_file &&
	test_cmp missing_refs empty_refs_file
'

test_expect_success "pins all look the same" '
	test_cmp start_rec_pins after_rec_pins &&
	test_cmp start_dir_pins after_dir_pins &&
	test_cmp start_ind_pins after_ind_pins
'

test_expect_success "manually compute gc set" '
	cat after_rec_pins after_dir_pins after_ind_pins | sort > all_pinned
'

test_expect_success "run a gc" '
	ipfs repo gc | sort > gc_out	
'

test_expect_success "no pinned objects were gc'ed" '
	comm -12 gc_out all_pinned > gced_pinned_objects &&
	test_cmp empty_refs_file gced_pinned_objects
'

test_expect_success "list all pins after gc" '
	ipfs pin ls --type=recursive | sort > gc_rec_pins &&
	ipfs pin ls --type=direct | sort > gc_dir_pins &&
	ipfs pin ls --type=indirect | sort > gc_ind_pins
'

test_expect_success "pins all look the same" '
	test_cmp start_rec_pins gc_rec_pins &&
	test_cmp start_dir_pins gc_dir_pins &&
	test_cmp start_ind_pins gc_ind_pins
'

test_expect_success "fetch all refs" '
	ipfs refs local | sort | uniq > post_gc_refs
'

first_elems() {
	cat "$1" | awk '{ print $1 }'
}

test_expect_success "get just hashes of pins" '
	first_elems all_pinned | sort | uniq > all_pinned_refs
'

test_kill_ipfs_daemon

test_can_fetch_buggy_hashes() {
	ref_file="$1"
	for ref in `cat $ref_file`; do
		if ! ipfs block get $ref > /dev/null; then
			echo "FAILURE: $ref"
			return 1
		fi
	done
}

# this bug was fixed in 0.4.3
test_expect_success "no pinned objects are missing from local refs" '
	comm -23 all_pinned_refs post_gc_refs > missing_pinned_objects &&
	printf "" > empty_file &&
	test_cmp empty_file missing_pinned_objects
'

test_expect_success "make a couple more files" '
	random-files -depth=$DEPTH -dirs=$NBDIR -files=$NBFILE many_more_files > more_filenames
'

test_expect_success "add the new files" '
	ipfs add -r -q many_more_files | tee more_hashes
'

test_expect_success "unpin root so we can do things ourselves" '
	ipfs pin rm $(tail -n1 more_hashes)
'

test_expect_success "select random subset to pin recursively and directly" '
	sort -R more_hashes | head -n$PINTOTAL > more_topin &&
	head -n$PINEACH more_topin > more_recurpins &&
	tail -n$PINEACH more_topin > more_directpins
'

test_expect_success "pin some objects recursively" '
	pin_hashes more_recurpins
'

test_expect_success "pin some objects directly" '
	pin_hashes more_directpins "-r=false"
'

test_expect_success "get full ref list" '
	ipfs refs local | sort > more_start_refs
'

test_expect_success "get pin lists" '
	ipfs pin ls --type=recursive | sort > more_start_rec_pins &&
	ipfs pin ls --type=direct | sort > more_start_dir_pins &&
	ipfs pin ls --type=indirect | sort > more_start_ind_pins
'

test_expect_success "'fs-repo-3-to-4 -revert' fails without -path" '
	test_must_fail fs-repo-3-to-4 -revert
'

test_expect_success "'fs-repo-3-to-4 -revert' succeeds" '
	fs-repo-3-to-4 -revert -path="$IPFS_PATH" >actual
'

test_expect_success "'fs-repo-3-to-4 -revert' output looks good" '
	grep "reverting blocks" actual ||
	test_fsh cat actual
'

test_install_ipfs_nd "v0.4.2"

test_launch_ipfs_daemon

test_expect_success "list all refs after reverting migration" '
	ipfs refs local | sort > after_revert_refs
'

test_expect_success "list all pins after reverting migration" '
	ipfs pin ls --type=recursive | sort > after_revert_rec_pins &&
	ipfs pin ls --type=direct | sort > after_revert_dir_pins &&
	ipfs pin ls --type=indirect | sort > after_revert_ind_pins
'

test_can_fetch_buggy_hashes() {
	ref_file="$1"
	for ref in `cat $ref_file`; do
		if ! ipfs block get $ref > /dev/null; then
			echo "FAILURE: $ref"
			return 1
		fi
	done
}

test_expect_success "refs look right" '
	comm -23 more_start_refs after_revert_refs > missing_refs &&
	test_can_fetch_buggy_hashes missing_refs
'

test_expect_success "pins all look the same" '
	test_cmp more_start_rec_pins after_revert_rec_pins &&
	test_cmp more_start_dir_pins after_revert_dir_pins &&
	test_cmp more_start_ind_pins after_revert_ind_pins
'

test_expect_success "manually compute gc set" '
	cat after_revert_rec_pins after_revert_dir_pins after_revert_ind_pins | sort > after_revert_all_pinned
'

test_expect_success "run a gc" '
	ipfs repo gc | sort > gc_out
'

test_expect_success "no pinned objects were gc'ed" '
	comm -12 gc_out after_revert_all_pinned > gced_pinned_objects &&
	test_cmp empty_refs_file gced_pinned_objects
'

test_kill_ipfs_daemon 

test_done
