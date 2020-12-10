GO111MODULE = on

MIG_DIRS = $(shell ls -d ipfs-*-to-*)

.PHONY: build clean sharness test test_go

show: $(MIG_DIRS)
	@echo "$(MIG_DIRS)"

build: $(shell ls -d ipfs-*-to-* | sed -e 's/ipfs/build.ipfs/')
	@echo OK

build.%: MIGRATION=$*
build.%:
	cd $(MIGRATION) && go build -mod=vendor

sharness:
	make -C sharness

test: test_go sharness

clean: $(shell ls -d ipfs-*-to-* | sed -e 's/ipfs/clean.ipfs/')
	@make -C sharness clean
	@echo OK

clean.%: MIGRATION=$*
clean.%:
	cd $(MIGRATION) && go clean

test_go: $(shell ls -d ipfs-*-to-* | sed -e 's/ipfs/test_go.ipfs/')
	@echo OK

test_go.%: MIGRATION=$*
test_go.%:
	@cd $(MIGRATION) && go test -mod=vendor

