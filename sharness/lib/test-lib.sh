# Test framework for fs-repo-migrations
#
# Copyright (c) 2015 Christian Couder
# MIT Licensed; see the LICENSE file in this repository.
#
# We are using Sharness (https://github.com/mlafeldt/sharness)
# which was extracted from the Git test framework.

SHARNESS_LIB="lib/sharness/sharness.sh"

# Set sharness verbosity. we set the env var directly as
# it's too late to pass in --verbose, and --verbose is harder
# to pass through in some cases.
test "$TEST_VERBOSE" = 1 && verbose=t && echo '# TEST_VERBOSE='"$TEST_VERBOSE"

. "$SHARNESS_LIB" || {
	echo >&2 "Cannot source: $SHARNESS_LIB"
	echo >&2 "Please check Sharness installation."
	exit 1
}

# Please put fs-repo-migrations specific shell functions and variables below

test "$TEST_EXPENSIVE" = 1 && test_set_prereq EXPENSIVE

DEFAULT_DOCKER_IMG="debian"
DOCKER_IMG="$DEFAULT_DOCKER_IMG"

TEST_TRASH_DIR=$(pwd)
TEST_SCRIPTS_DIR=$(dirname "$TEST_TRASH_DIR")
APP_ROOT_DIR=$(dirname "$TEST_SCRIPTS_DIR")

TEST_DIR_BASENAME=$(basename "$TEST_TRASH_DIR")
GUEST_TEST_DIR="sharness/$TEST_DIR_BASENAME"

CERTIFS='/etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt'

# This writes a docker ID on stdout
start_docker() {
	docker run --rm -it -d -v "$CERTIFS" -v "$APP_ROOT_DIR:/mnt" -w "/mnt" "$DOCKER_IMG" /bin/bash
}

# This takes a docker ID and a command as arguments
exec_docker() {
	docker exec -i "$1" /bin/bash -c "$2"
}

# This takes a docker ID as argument
stop_docker() {
	docker stop "$1"
}

# Echo the args, run the cmd, and then also fail,
# making sure a test case fails.
test_fsh() {
	echo "> $@"
	eval "$@"
	echo
	false
}

# Same as sharness' test_cmp but using test_fsh (to see the output).
# We have to do it twice, so the first diff output doesn't show unless it's
# broken.
test_cmp() {
	diff -q "$@" >/dev/null || test_fsh diff -u "$@"
}

# Same as test_cmp above, but we sort files before comparing them.
test_sort_cmp() {
	sort "$1" >"$1_sorted" &&
	sort "$2" >"$2_sorted" &&
	test_cmp "$1_sorted" "$2_sorted"
}

LOCAL_IPFS_UPDATE="../bin/ipfs-update"
GUEST_IPFS_UPDATE="sharness/bin/ipfs-update"

LOCAL_FS_REPO_MIG="../bin/fs-repo-migrations"
GUEST_FS_REPO_MIG="sharness/bin/fs-repo-migrations"

GUEST_RANDOM_FILES="sharness/bin/random-files"

# Install an IPFS version on a docker container
test_install_version() {
	VERSION="$1"

	# Change the PATH so that migration binaries built for this test are found
	# first, and are run by ipfs-update
	test_expect_success "'ipfs-update install' works for $VERSION" '
		DOCPWD=$(exec_docker "$DOCID" "pwd") &&
		DOCPATH=$(exec_docker "$DOCID" "echo \$PATH") &&
		NEWPATH="$DOCPWD/sharness/bin:$DOCPATH" &&
		exec_docker "$DOCID" "export PATH=\"$NEWPATH\" && $GUEST_IPFS_UPDATE --verbose install --allow-downgrade $VERSION" >actual 2>&1 ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update install' output looks good" '
		grep "fetching go-ipfs version $VERSION" actual &&
		grep "Installation complete!" actual ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update version' works for $VERSION" '
		exec_docker "$DOCID" "$GUEST_IPFS_UPDATE version" >actual
	'

	test_expect_success "'ipfs-update version' output looks good" '
		echo "$VERSION" >expected &&
		test_cmp expected actual
	'
}

# Revert to previous IPFS version on a docker container
test_revert_to_version() {
	VERSION="$1"

	# Change the PATH so that migration binaries built for this test are found
	# first, and are run by ipfs-update
	test_expect_success "'ipfs-update install' works for $VERSION" '
		DOCPWD=$(exec_docker "$DOCID" "pwd") &&
		DOCPATH=$(exec_docker "$DOCID" "echo \$PATH") &&
		NEWPATH="$DOCPWD/sharness/bin:$DOCPATH" &&
		exec_docker "$DOCID" "export PATH=\"$NEWPATH\" && $GUEST_IPFS_UPDATE --verbose revert" >actual 2>&1 ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update revert' output looks good" '
		grep "Revert complete" actual ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update version' works for $VERSION" '
		exec_docker "$DOCID" "$GUEST_IPFS_UPDATE version" >actual
	'

	test_expect_success "'ipfs-update version' output looks good" '
		echo "$VERSION" >expected &&
		test_cmp expected actual
	'
}

test_start_daemon() {
	docid="$1"
	test_expect_success "'ipfs daemon' succeeds" '
		exec_docker "$docid" "ipfs daemon >actual_daemon 2>daemon_err &"
	'

	test_expect_success "api file shows up" '
		test_docker_wait_for_file "$docid" 20 100ms "$IPFS_PATH/api"
	'
}

test_stop_daemon() {
	docid="$1"
	test_expect_success "kill ipfs daemon" '
		exec_docker "$docid" "kill \$(pidof ipfs)"
	'

	test_expect_success "daemon is not running" '
		test_must_fail exec_docker "$docid" "pidof ipfs"
	'
}

test_init_daemon() {
	docid="$1"
	test_expect_success "'ipfs init' succeeds" '
		export IPFS_PATH=/root/.ipfs &&
		exec_docker "$docid" "IPFS_PATH=$IPFS_PATH BITS=2048 ipfs init" >actual 2>&1 ||
		test_fsh cat actual
	'

	test_expect_success "clear nodes bootstrapping" '
		exec_docker "$docid" "ipfs config Bootstrap --json null"
	'
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

major_number() {
    vers="$1"

    # Hack around 'expr' exiting with code 1 when it outputs 0
    case "$vers" in
        0) echo "0" ;;
        0.*) echo "0" ;;
        *) expr "$vers" : "\([^.]*\).*" || return 1
    esac
}

check_at_least_version() {
    MIN_VERS="$1"
    CUR_VERS="$2"
    PROG_NAME="$3"

    # Get major, minor and fix numbers for each version
    MIN_MAJ=$(major_number "$MIN_VERS") || die "No major version number in '$MIN_VERS' for '$PROG_NAME'"
    CUR_MAJ=$(major_number "$CUR_VERS") || die "No major version number in '$CUR_VERS' for '$PROG_NAME'"

    if MIN_MIN=$(expr "$MIN_VERS" : "[^.]*\.\([^.]*\).*"); then
        MIN_FIX=$(expr "$MIN_VERS" : "[^.]*\.[^.]*\.\([^.]*\).*") || MIN_FIX="0"
    else
        MIN_MIN="0"
        MIN_FIX="0"
    fi
    if CUR_MIN=$(expr "$CUR_VERS" : "[^.]*\.\([^.]*\).*"); then
        CUR_FIX=$(expr "$CUR_VERS" : "[^.]*\.[^.]*\.\([^.]*\).*") || CUR_FIX="0"
    else
        CUR_MIN="0"
        CUR_FIX="0"
    fi

    # Compare versions
    test "$CUR_MAJ" -lt "$MIN_MAJ" && return 1
    test "$CUR_MAJ" -gt "$MIN_MAJ" && return 0
    test "$CUR_MIN" -lt "$MIN_MIN" && return 1
    test "$CUR_MIN" -gt "$MIN_MIN" && return 0
    test "$CUR_FIX" -ge "$MIN_FIX"
}

# Print the repo version corresponding to an IPFS_VERSION passed as
# argument.
test_get_repo_version() {
	IPFS_VERSION="$1"

	REPO_VERSION=1

	check_at_least_version "0.3.0" "$IPFS_VERSION" "ipfs" && REPO_VERSION=2

	check_at_least_version "0.4.0" "$IPFS_VERSION" "ipfs" && REPO_VERSION=3

	echo "$REPO_VERSION"
}

test_repo_version() {
	IPFS_VERSION="$1"

	test_expect_success "repository version looks good" '
		test_get_repo_version "$IPFS_VERSION" >expected &&
		exec_docker "$DOCID" "cat \"$IPFS_PATH/version\"" >actual &&
		test_cmp expected actual
	'
}

test_install_ipfs_nd() {
	VERSION="$1"

	# Change the PATH so that migration binaries built for this test are found
	# first, and are run by ipfs-update
	test_expect_success "'ipfs-update install' works for $VERSION" '
		ipfs-update --verbose install --allow-downgrade $VERSION > actual 2>&1 ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update install' output looks good" '
		grep "fetching go-ipfs version $VERSION" actual &&
		grep "Installation complete!" actual ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update version' works for $VERSION" '
		ipfs-update version > actual 2>&1
	'

	test_expect_success "'ipfs-update version' output looks good" '
		echo "$VERSION" >expected &&
		test_cmp expected actual
	'
}

test_init_ipfs_nd() {

	test_expect_success "ipfs init succeeds" '
		export IPFS_PATH="$(pwd)/.ipfs" &&
		ipfs init -b=2048 > /dev/null
	'

	test_expect_success "prepare config -- mounting and bootstrap rm" '
		test_config_set Addresses.API "/ip4/127.0.0.1/tcp/0" &&
		test_config_set Addresses.Gateway "/ip4/127.0.0.1/tcp/0" &&
		test_config_set --json Addresses.Swarm "[
  \"/ip4/0.0.0.0/tcp/0\"
]" &&
		ipfs bootstrap rm --all ||
		test_fsh cat "\"$IPFS_PATH/config\""
	'
}

test_config_set() {

	# grab flags (like --bool in "ipfs config --bool")
	test_cfg_flags="" # unset in case.
	test "$#" = 3 && { test_cfg_flags=$1; shift; }

	test_cfg_key=$1
	test_cfg_val=$2

	# when verbose, tell the user what config values are being set
	test_cfg_cmd="ipfs config $test_cfg_flags \"$test_cfg_key\" \"$test_cfg_val\""
	test "$TEST_VERBOSE" = 1 && echo "$test_cfg_cmd"

	# ok try setting the config key/val pair.
	ipfs config $test_cfg_flags "$test_cfg_key" "$test_cfg_val"
	echo "$test_cfg_val" >cfg_set_expected
	ipfs config "$test_cfg_key" >cfg_set_actual
	test_cmp cfg_set_expected cfg_set_actual
}

test_set_address_vars_nd() {
	daemon_output="$1"

	test_expect_success "set up address variables" '
		API_MADDR=$(cat "$IPFS_PATH/api") &&
		API_ADDR=$(convert_tcp_maddr $API_MADDR) &&
		API_PORT=$(port_from_maddr $API_MADDR) &&

		GWAY_MADDR=$(sed -n "s/^Gateway (.*) server listening on //p" "$daemon_output") &&
		GWAY_ADDR=$(convert_tcp_maddr $GWAY_MADDR) &&
		GWAY_PORT=$(port_from_maddr $GWAY_MADDR)
	'

	if ipfs swarm addrs local >/dev/null 2>&1; then
		test_expect_success "set swarm address vars" '
		ipfs swarm addrs local > addrs_out &&
			SWARM_MADDR=$(grep "127.0.0.1" addrs_out) &&
			SWARM_PORT=$(port_from_maddr $SWARM_MADDR)
		'
	fi
}

convert_tcp_maddr() {
	echo $1 | awk -F'/' '{ printf "%s:%s", $3, $5 }'
}

port_from_maddr() {
	echo $1 | awk -F'/' '{ print $NF }'
}

test_launch_ipfs_daemon() {

	args="$@"

	test "$TEST_ULIMIT_PRESET" != 1 && ulimit -n 1024

	test_expect_success "'ipfs daemon' succeeds" '
		pwd &&
		ipfs daemon $args >actual_daemon 2>daemon_err &
	'

	# wait for api file to show up
	test_expect_success "api file shows up" '
		test_wait_for_file 20 100ms "$IPFS_PATH/api"
	'

	test_set_address_vars_nd actual_daemon

	# we say the daemon is ready when the API server is ready.
	test_expect_success "'ipfs daemon' is ready" '
		IPFS_PID=$! &&
		pollEndpoint -ep=/version -host=$API_MADDR -v -tout=1s -tries=60 2>poll_apierr > poll_apiout ||
		test_fsh cat actual_daemon || test_fsh cat daemon_err || test_fsh cat poll_apierr || test_fsh cat poll_apiout
	'
}

test_wait_for_file() {
	loops=$1
	delay=$2
	file=$3
	fwaitc=0
	while ! test -f "$file"
	do
		if test $fwaitc -ge $loops
		then
			echo "Error: timed out waiting for file: $file"
			return 1
		fi

		go-sleep $delay
		fwaitc=`expr $fwaitc + 1`
	done
}


test_kill_repeat_10_sec() {
	# try to shut down once + wait for graceful exit
	kill $1
	for i in $(test_seq 1 100)
	do
		go-sleep 100ms
		! kill -0 $1 2>/dev/null && return
	done

	# if not, try once more, which will skip graceful exit
	kill $1
	go-sleep 1s
	! kill -0 $1 2>/dev/null && return

	# ok, no hope. kill it to prevent it messing with other tests
	kill -9 $1 2>/dev/null
	return 1
}

test_kill_ipfs_daemon() {

	test_expect_success "'ipfs daemon' is still running" '
		kill -0 $IPFS_PID
	'

	test_expect_success "'ipfs daemon' can be killed" '
		test_kill_repeat_10_sec $IPFS_PID
	'
}
