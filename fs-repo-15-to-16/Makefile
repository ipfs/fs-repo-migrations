.PHONY: build clean

build:
	go build -mod=vendor

clean:
	go clean

test:
	@cd test-e2e && ./test.sh
