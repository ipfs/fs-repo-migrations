TMPDIR = /tmp/tmp.fs-repo-migration_build
TMP_GOIPFS = $(TMPDIR)/go-ipfs
BIN = ipfs-10-to-11

.PHONY: build clean s3plugin s3goipfs

all: build

build:
	go build -mod=vendor

s3goipfs:
	rm -rf "$(TMPDIR)"
	mkdir -p "$(TMPDIR)"
	#git clone -b release-v0.8.0 https://github.com/ipfs/go-ipfs "$(TMP_GOIPFS)"
	git clone https://github.com/ipfs/go-ipfs "$(TMP_GOIPFS)"
	cd "$(TMP_GOIPFS)" && git checkout 852dfab997f58d9c36765c2fae7e619390f44fdb
	cd "$(TMP_GOIPFS)" && go get github.com/ipfs/go-ds-s3@latest
	echo "s3ds github.com/ipfs/go-ds-s3/plugin 0" >> $(TMP_GOIPFS)/plugin/loader/preload_list
	make -C $(TMP_GOIPFS) build
	go mod edit -replace  github.com/ipfs/go-ipfs=/tmp/tmp.fs-repo-migration_build/go-ipfs
	go mod vendor

s3plugin: s3goipfs build
	@echo "===> Build with S3 plugin"

clean:
	go clean
	rm -rf "$(TMPDIR)" vendor
	git checkout vendor go.mod go.sum
