package main

import (
	mg3 "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg3.Migration{}
	migrate.Main(&m)
}
