#!/bin/sh

test_description="Test migration 2 to 3 with lots of objects"

. lib/test-lib.sh


test_expect_success "start a docker container" '
	DOCID=$(start_docker)
'

drun() {
	exec_docker "$DOCID" "$@"
}

test_install_version "v0.3.11"

test_expect_success "'ipfs init' succeeds" '
	exec_docker "$DOCID" "IPFS_PATH=/root/.ipfs BITS=2048 ipfs init" >actual 2>&1 ||
	test_fsh cat actual
'

# START DAEMON HERE

test_expect_success "make a couple files" '
	#drun "random-files -depth=6 -dirs=7 -files=10 manyfiles" > filenames
	drun "$GUEST_RANDOM_FILES -depth=3 -dirs=3 -files=10 manyfiles" > filenames
'

test_expect_success "add a few files" '
	drun "ipfs add -r -q manyfiles" > hashes
'

test_expect_success "unpin root so we can do things ourselves" '
	drun "ipfs pin rm $(tail -n1 hashes)"
'

test_expect_success "select random subset to pin recursively and directly" '
	sort -R hashes | head -n2000 > topin &&
	head -n1000 topin > recurpins &&
	tail -n1000 topin > directpins 
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
	pin_hashes recurpins "-r=false"
'

test_expect_success "get full ref list" '
	drun "ipfs refs local" > start_refs
'

test_expect_success "get pin lists" '
	drun "ipfs pin ls --type=recursive" > start_rec_pins &&
	drun "ipfs pin ls --type=direct" > start_dir_pins &&
	drun "ipfs pin ls --type=indirect" > start_ind_pins
'

# STOP DAEMON

# UPGRADE TO 0.4.0 AND RUN MIGRATION

# VERIFY EVERYTHING

test_expect_success "stop docker container" '
	stop_docker "$DOCID"
'

test_done
