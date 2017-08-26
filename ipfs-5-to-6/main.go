package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg5 "github.com/ipfs/fs-repo-migrations/ipfs-5-to-6/migration"
)

func main() {
	m := mg5.Migration{}
	migrate.Main(&m)
}
