#!/bin/sh

test_description="Test migration 2 to 3 with lots of objects"

. lib/test-lib.sh

# setup vars for tests

DEPTH=3
PINTOTAL=200

if test_have_prereq EXPENSIVE
then
	DEPTH=6
	PINTOTAL=2000
fi

PINEACH=$(expr $PINTOTAL / 2)

test_expect_success "start a docker container" '
	DOCID=$(start_docker)
'

drun() {
	exec_docker "$DOCID" "$@"
}

test_docker_wait_for_file() {
	docid="$1"
	loops="$2"
	delay="$3"
	file="$4"
	fwaitc=0
	while ! exec_docker "$docid" "test -f '$file'"
	do
		if test $fwaitc -ge $loops
		then
			echo "Error: timed out waiting for file: $file"
			return 1
		fi

		go-sleep $delay
		fwaitc=$(expr $fwaitc + 1)
	done
}

test_install_version "v0.3.11"

test_init_daemon "$DOCID"

test_start_daemon "$DOCID"

test_expect_success "make a couple files" '
	drun "$GUEST_RANDOM_FILES -depth=$DEPTH -dirs=7 -files=10 manyfiles" > filenames
'

test_expect_success "add a few files" '
	drun "ipfs add -r -q manyfiles" | tee hashes
'

test_expect_success "unpin root so we can do things ourselves" '
	drun "ipfs pin rm $(tail -n1 hashes)"
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
		if ! drun "ipfs pin add $opts $h"; then
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

test_expect_success "get full ref list" '
	drun "ipfs refs local" | sort > start_refs
'

test_expect_success "get pin lists" '
	drun "ipfs pin ls --type=recursive" | sort > start_rec_pins &&
	drun "ipfs pin ls --type=direct" | sort > start_dir_pins &&
	drun "ipfs pin ls --type=indirect" | sort > start_ind_pins
'

test_stop_daemon $DOCID

test_install_version "v0.4.0-dev"

test_start_daemon $DOCID

test_expect_success "list all refs after migration" '
	drun "ipfs refs local" | sort > after_refs
'

test_expect_success "list all pins after migration" '
	drun "ipfs pin ls --type=recursive" | sort > after_rec_pins &&
	drun "ipfs pin ls --type=direct" | sort > after_dir_pins &&
	drun "ipfs pin ls --type=indirect" | sort > after_ind_pins
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
	drun "ipfs repo gc" | sort > gc_out	
'

test_expect_success "no pinned objects were gc'ed" '
	comm -12 gc_out all_pinned > gced_pinned_objects &&
	test_cmp empty_refs_file gced_pinned_objects
'

test_expect_success "list all pins after gc" '
	drun "ipfs pin ls --type=recursive" | sort > gc_rec_pins &&
	drun "ipfs pin ls --type=direct" | sort > gc_dir_pins &&
	drun "ipfs pin ls --type=indirect" | sort > gc_ind_pins
'

test_expect_success "pins all look the same" '
	test_cmp start_rec_pins gc_rec_pins &&
	test_cmp start_dir_pins gc_dir_pins &&
	test_cmp start_ind_pins gc_ind_pins
'

test_expect_success "fetch all refs" '
	drun "ipfs refs local" | sort | uniq > post_gc_refs
'

first_elems() {
	cat "$1" | awk '{ print $1 }'
}

test_expect_success "get just hashes of pins" '
	first_elems all_pinned | sort | uniq > all_pinned_refs
'

test_stop_daemon $DOCID

test_can_fetch_buggy_hashes() {
	ref_file="$1"
	for ref in `cat $ref_file`; do
		if ! drun "ipfs block get $ref" > /dev/null; then
			echo "FAILURE: $ref"
			return 1
		fi
	done
}

# the refs we're testing here are a result of the go-datastore path bug.
# The byte representation of the hash ends in a '/' character, which gets trimmed
# by go-datastore. This does not imply that there is any data lost, just that
# we cant enumerate those refs properly
test_expect_success "no pinned objects are missing from local refs" '
	comm -23 all_pinned_refs post_gc_refs > missing_pinned_objects &&
	test_can_fetch_buggy_hashes missing_pinned_objects
'

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
