install:
	go install
	@echo "fs-repo-migrations now installed, type 'fs-repo-migrations' to run"

test: test_go sharness

test_go:
	go test ./ipfs-5-to-6/... # go test ./... fails see #66

sharness:
	make -C sharness

.PHONY: test test_go sharness
