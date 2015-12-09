package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg2 "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/migration"
)

func main() {
	m := mg2.Migration{}
	migrate.Main(&m)
}
