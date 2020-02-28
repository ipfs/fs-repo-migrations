GO111MODULE = on

install:
	go install -mod=vendor
	@echo "fs-repo-migrations now installed, type 'fs-repo-migrations' to run"

test: test_go sharness

test_go:
	go build -mod=vendor
	go test -mod=vendor  ./...

sharness:
	make -C sharness

.PHONY: test test_go sharness
