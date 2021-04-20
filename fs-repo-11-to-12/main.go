package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg11 "github.com/ipfs/fs-repo-migrations/ipfs-11-to-12/migration"
)

func main() {
	m := mg11.Migration{}
	migrate.Main(&m)
}
