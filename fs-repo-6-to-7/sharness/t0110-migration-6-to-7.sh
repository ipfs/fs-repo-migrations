#!/bin/sh

test_description="Test migration 6 to 7"

. lib/test-lib.sh

# setup vars for tests

export IPTB_ROOT="$(pwd)/.iptb"
ipfsi() {
    dir="$1"
    shift
    IPFS_PATH="$IPTB_ROOT/testbeds/default/$dir" ipfs "$@"
}

export DEBUG=true

export GOPATH="$(pwd)/gopath"
mkdir -p gopath/bin
export PATH="$(pwd)/../bin:$GOPATH/bin:$PATH"

test_install_ipfs_nd_6_7() {
    # Avoid setting this globally. That way, we *test* the dist path set in go-ipfs
    export IPFS_DIST_PATH="/ipfs/QmVuTFnsc1T7ubG2Xh4c9d7qNURPgjPPCdPvuEUj9UEVgr"
    test_install_ipfs_nd "$@"
    unset IPFS_DIST_PATH
}

test_install_ipfs_nd_6_7 "v0.4.15"

test_expect_success "iptb init" '
	iptb testbed create -type localipfs -count 2 -init
'

for i in 0 1
do
	test_expect_success "set configs up for node $i" '
		ipfsi "$i" config Ipns.RepublishPeriod 2s &&
		ipfsi "$i" config Ipns.RecordLifetime 5s &&
		ipfsi "$i" config --json Ipns.ResolveCacheSize 0
	'
done

test_expect_success "generate keys" '
	ID1=$(ipfsi 0 id -f"<id>") &&
	ID2=$(ipfsi 0 key gen --type=rsa --size=2048 second) &&
	ID3=$(ipfsi 0 key gen --type=rsa --size=2048 third)
'

i=0
counter() {
		i=$(( i + 1 ))
		printf "file%d" $i
}

test_publish() {
	test_expect_success "publish names" '
    F1=/ipfs/$(counter | ipfsi 0 add -q) &&
    F2=/ipfs/$(counter | ipfsi 0 add -q) &&
    F3=/ipfs/$(counter | ipfsi 0 add -q) &&
    printf "$F1\n$F2\n$F3\n" > resolve_expected &&
    ipfsi 0 name publish -t 5s $F1 &&
		ipfsi 0 name publish -t 5s --key=second $F2 &&
		ipfsi 0 name publish -t 5s --key=third $F3
	'
}

test_resolve_succeeds() {
		rm -f resolve_actual
		test_expect_success "test resolve succeeds" '
			ipfsi 1 name resolve $ID1 >> resolve_actual &&
			ipfsi 1 name resolve $ID2 >> resolve_actual &&
			ipfsi 1 name resolve $ID3 >> resolve_actual &&
			test_cmp resolve_expected resolve_actual
		'
}

test_resolve_fails() {
		test_expect_success "test resolve fails" '
			! ipfsi 1 name resolve $ID1 &&
			! ipfsi 1 name resolve $ID2 &&
			! ipfsi 1 name resolve $ID3
		'
}

test_start_0() {
  test_expect_success "start cluster" '
		iptb start -wait 0 -- --migrate=true && iptb connect 0 1 && go-sleep 3s
	'
}

test_stop_0() {
  test_expect_success "stop node 0" '
		iptb stop 0 && go-sleep 6s
	'
}

test_start() {
	test_expect_success "start cluster" '
    iptb start -wait -- --migrate=true && iptb connect 0 1 && go-sleep 3s
	'
}

test_stop() {
	test_expect_success "shutdown cluster" '
		iptb stop && go-sleep 6s
	'
}

test_resolution() {
	test_resolve_succeeds
  test_stop_0
	test_resolve_fails
  test_start_0
	test_resolve_succeeds
	test_publish
	test_resolve_succeeds
  test_stop_0
	test_resolve_fails
  test_start_0
	test_resolve_succeeds
}

test_start
test_publish
test_resolution
test_stop

test_install_ipfs_nd_6_7 "v0.4.16-dev-dspre"

test_start
test_resolution
test_stop

test_expect_success "'fs-repo-6-to-7 -revert' fails without -path" '
  IPFS_PATH="$IPTB_ROOT/testbeds/default/0" test_must_fail fs-repo-6-to-7 -revert
'

test_expect_success "'fs-repo-6-to-7 -revert' succeeds" '
	fs-repo-6-to-7 -revert -path="$IPTB_ROOT/testbeds/default/0" &&
	fs-repo-6-to-7 -revert -path="$IPTB_ROOT/testbeds/default/1"
'

test_install_ipfs_nd "v0.4.15"

test_start
test_resolution
test_stop

test_done
