package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg3 "github.com/ipfs/fs-repo-migrations/ipfs-3-to-4/migration"
)

func main() {
	m := mg3.Migration{}
	migrate.Main(&m)
}
