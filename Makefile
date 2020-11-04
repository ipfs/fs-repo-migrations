GO111MODULE = on

install:
	go install
	@echo "fs-repo-migrations now installed, type 'fs-repo-migrations' to run"

test: test_go sharness

test_go:
	go build
	go test $(shell go list ./... | grep -v /gx/)

sharness:
	make -C sharness

.PHONY: test test_go sharness
