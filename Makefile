GO111MODULE = on

MIG_DIRS = $(shell ls -d fs-repo-*-to-*)

.PHONY: all build clean cmd sharness test test_go

all: build

show: $(MIG_DIRS)
	@echo "$(MIG_DIRS)"

build: $(shell ls -d fs-repo-*-to-* | sed -e 's/fs-repo/build.fs-repo/') cmd
	@echo OK

build.%: MIGRATION=$*
build.%:
	make -C $(MIGRATION)

cmd: fs-repo-migrations/fs-repo-migrations

fs-repo-migrations/fs-repo-migrations:
	cd fs-repo-migrations && go build

sharness:
	make -C sharness

test: test_go sharness

clean: $(shell ls -d fs-repo-*-to-* | sed -e 's/fs-repo/clean.fs-repo/')
	@make -C sharness clean
	@cd fs-repo-migrations && go clean
	@echo OK

clean.%: MIGRATION=$*
clean.%:
	make -C $(MIGRATION) clean

test_go: $(shell ls -d fs-repo-*-to-* | sed -e 's/fs-repo/test_go.fs-repo/')
	@echo OK

test_go.%: MIGRATION=$*
test_go.%:
	@cd $(MIGRATION)/migration && go test -mod=vendor
