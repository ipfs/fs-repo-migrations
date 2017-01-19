package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg4 "github.com/ipfs/fs-repo-migrations/ipfs-4-to-5/migration"
)

func main() {
	m := mg4.Migration{}
	migrate.Main(&m)
}
