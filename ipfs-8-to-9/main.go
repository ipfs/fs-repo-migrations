package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg8 "github.com/ipfs/fs-repo-migrations/ipfs-8-to-9/migration"
)

func main() {
	m := mg8.Migration{}
	migrate.Main(&m)
}
