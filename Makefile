GO111MODULE = on

MIG_DIRS = $(shell ls -d fs-repo-*-to-*)
IGNORED_DIRS := $(shell cat ignored-migrations)
ACTIVE_DIRS := $(filter-out $(IGNORED_DIRS),$(MIG_DIRS))

.PHONY: all build clean cmd sharness test test_go test_14_to_15 test_15_to_16

all: build

show: $(MIG_DIRS)
	@echo "$(MIG_DIRS)"

build: $(subst fs-repo,build.fs-repo,$(ACTIVE_DIRS)) cmd
	@echo OK

build.%: MIGRATION=$*
build.%:
	make -C $(MIGRATION)

cmd: fs-repo-migrations/fs-repo-migrations

fs-repo-migrations/fs-repo-migrations:
	cd fs-repo-migrations && go build

sharness:
	make -C sharness

test: test_go test_14_to_15 test_15_to_16 sharness

clean: $(subst fs-repo,clean.fs-repo,$(ACTIVE_DIRS))
	@make -C sharness clean
	@cd fs-repo-migrations && go clean
	@echo OK

clean.%: MIGRATION=$*
clean.%:
	make -C $(MIGRATION) clean

test_go: $(subst fs-repo,test_go.fs-repo,$(ACTIVE_DIRS))
	@echo OK

test_go.%: MIGRATION=$*
test_go.%:
	@cd $(MIGRATION)/migration && go test -mod=vendor

test_12_to_13:
	@cd fs-repo-12-to-13/not-sharness && ./test.sh

test_13_to_14:
	@cd fs-repo-13-to-14/not-sharness && ./test.sh

test_14_to_15:
	@cd fs-repo-14-to-15/not-sharness && ./test.sh

test_15_to_16:
	@cd fs-repo-15-to-16/test-e2e && ./test.sh
