# Run sharness tests here and in all migrations
#
# NOTE: run with TEST_VERBOSE=1 for verbose sharness tests.
#
# To build migrations an run all tests, run make:
#   $ make
#
# To run an individual test, make deps then run the test. For example:
#   $ make deps
#   $ ../fs-repo-2-to-3/sharness/t0040-migration-2-to-3.sh
#

# Find tests in local sharness directory and in sharness directory of every
# migration.
T = $(sort $(wildcard t[0-9][0-9][0-9][0-9]-*.sh))
T += $(sort $(shell find ../fs-repo-*-to-*/sharness -name 't[0-9][0-9][0-9][0-9]-*.sh'))

GO111MODULE = on
LIBDIR = lib
SHARNESSDIR = sharness
AGGREGATE = $(LIBDIR)/$(SHARNESSDIR)/aggregate-results.sh

# Binaries generated
BINS = bin/ipfs-update
BINS += bin/random-files
BINS += bin/go-sleep
BINS += bin/pollEndpoint
BINS += bin/iptb
BINS += bin/fs-repo-migrations

# User might want to override those on the command line
GOFLAGS =

all: aggregate

aggregate: clean-test-results $(T)
	@echo "*** $@ ***"
	ls test-results/t*-*.sh.*.counts | $(AGGREGATE)

$(T): clean-test-results deps
	@echo "*** $@ ***"
	./$@ -v

# Needed dependencies.
deps: sharness $(BINS) build

# Build all migrations
build: $(shell ls -d ../fs-repo-*-to-* | sed -e 's/..\/fs-repo/build.fs-repo/')
	@echo "===> OK built migrations"

build.%: MIGRATION=$*
build.%:
	cd ../$(MIGRATION) && go build $(GOFLAGS) -mod=vendor -o ../sharness/bin/

clean: clean-test-results
	@echo "*** $@ ***"
	-rm -rf $(BINS)
	-rm -f bin/fs-repo-*-to-*

clean-test-results:
	@echo "*** $@ ***"
	-rm -rf test-results
	-rm -f ../actual_daemon ../daemon_err

sharness:
	@echo "*** checking $@ ***"
	lib/install-sharness.sh

find_go_files = $(shell find $(1) -name "*.go")

bin/ipfs-update: BUILD-OPTIONS
	@echo "*** installing $@ ***"
	cd dependencies && go build $(GOFLAGS) -o ../$@ github.com/ipfs/ipfs-update

bin/random-files: BUILD-OPTIONS
	@echo "*** installing $@ ***"
	cd dependencies && go build $(GOFLAGS) -o ../$@ github.com/jbenet/go-random-files/random-files

bin/go-sleep: BUILD-OPTIONS
	@echo "*** installing $@ ***"
	cd dependencies && go build $(GOFLAGS) -o ../$@ github.com/chriscool/go-sleep

bin/iptb: BUILD-OPTIONS
	@echo "*** installing $@ ***"
	cd dependencies && go build $(GOFLAGS) -o ../$@ ./iptb

bin/pollEndpoint: BUILD-OPTIONS
	@echo "*** installing $@ ***"
	cd dependencies && go build $(GOFLAGS) -o ../$@ ./pollEndpoint

bin/fs-repo-migrations: BUILD-OPTIONS
	@echo "*** installing $@ ***"
	cd ../fs-repo-migrations && go build $(GOFLAGS) -o ../sharness/$@

BUILD-OPTIONS: FORCE
	@bin/checkflags '$@' '$(GOFLAGS)' '*** new Go flags ***'

.PHONY: all build clean clean-test-results $(T) aggregate deps sharness FORCE
