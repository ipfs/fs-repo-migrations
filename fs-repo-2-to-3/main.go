package main

import (
	mg2 "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg2.Migration{}
	migrate.Main(&m)
}
