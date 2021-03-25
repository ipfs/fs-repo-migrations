package main

import (
	mg6 "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg6.Migration{}
	migrate.Main(&m)
}
