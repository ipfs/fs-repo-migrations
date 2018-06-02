package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg6 "github.com/ipfs/fs-repo-migrations/ipfs-6-to-7/migration"
)

func main() {
	m := mg6.Migration{}
	migrate.Main(&m)
}
