GO111MODULE = on

check:
	@echo "verifying that we have no external dependencies"
	@test "$$(go list -m all)" = "github.com/ipfs/fs-repo-migrations"

install:
	go install
	@echo "fs-repo-migrations now installed, type 'fs-repo-migrations' to run"

test: test_go sharness

test_go:
	go test ./ipfs-5-to-6/... ./ipfs-7-to-8/... # go test ./... fails see #66

sharness:
	make -C sharness

.PHONY: test test_go sharness
