GO111MODULE = on

MIG_DIRS = $(shell ls -d ipfs-*-to-*)

.PHONY: all build clean cmd sharness test test_go

all: build

show: $(MIG_DIRS)
	@echo "$(MIG_DIRS)"

build: $(shell ls -d ipfs-*-to-* | sed -e 's/ipfs/build.ipfs/') cmd
	@echo OK

build.%: MIGRATION=$*
build.%:
	make -C $(MIGRATION)

cmd: cmd/fs-repo-migrations/fs-repo-migrations

cmd/fs-repo-migrations/fs-repo-migrations:
	cd cmd/fs-repo-migrations && go build

sharness:
	make -C sharness

test: test_go sharness

clean: $(shell ls -d ipfs-*-to-* | sed -e 's/ipfs/clean.ipfs/')
	@make -C sharness clean
	@cd cmd/fs-repo-migrations && go clean
	@echo OK

clean.%: MIGRATION=$*
clean.%:
	make -C $(MIGRATION) clean

test_go: $(shell ls -d ipfs-*-to-* | sed -e 's/ipfs/test_go.ipfs/')
	@echo OK

test_go.%: MIGRATION=$*
test_go.%:
	@cd $(MIGRATION) && go test -mod=vendor
