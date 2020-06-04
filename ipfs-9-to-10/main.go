package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg9 "github.com/ipfs/fs-repo-migrations/ipfs-9-to-10/migration"
)

func main() {
	m := mg9.Migration{}
	migrate.Main(&m)
}
