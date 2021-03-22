package main

import (
	mg8 "github.com/ipfs/fs-repo-migrations/fs-repo-8-to-9/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg8.Migration{}
	migrate.Main(&m)
}
