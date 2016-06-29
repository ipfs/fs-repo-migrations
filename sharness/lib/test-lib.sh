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
	docker run -it -d -v "$CERTIFS" -v "$APP_ROOT_DIR:/mnt" -w "/mnt" "$DOCKER_IMG" /bin/bash
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

GUEST_IPFS_0_TO_1="sharness/bin/ipfs-0-to-1"
GUEST_IPFS_1_TO_2="sharness/bin/ipfs-1-to-2"
GUEST_IPFS_2_TO_3="sharness/bin/ipfs-2-to-3"
GUEST_IPFS_3_TO_4="sharness/bin/ipfs-3-to-4"

GUEST_RANDOM_FILES="sharness/bin/random-files"

# Install an IPFS version on a docker container
test_install_version() {
	VERSION="$1"

	# We have to change the PATH as ipfs-update might call fs-repo-migrations
	test_expect_success "'ipfs-update install' works for $VERSION" '
		DOCPWD=$(exec_docker "$DOCID" "pwd") &&
		DOCPATH=$(exec_docker "$DOCID" "echo \$PATH") &&
		NEWPATH="$DOCPWD/sharness/bin:$DOCPATH" &&
		exec_docker "$DOCID" "export PATH=\"$NEWPATH\" && $GUEST_IPFS_UPDATE --verbose install $VERSION" >actual 2>&1 ||
		test_fsh cat actual
	'

	test_expect_success "'ipfs-update install' output looks good" '
		grep "fetching ipfs version $VERSION" actual &&
		grep "installation complete." actual ||
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
		exec_docker "$docid" "pkill ipfs"
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
