package main

import (
	mg4 "github.com/ipfs/fs-repo-migrations/fs-repo-4-to-5/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg4.Migration{}
	migrate.Main(&m)
}
