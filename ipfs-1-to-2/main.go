package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg1 "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/migration"
)

func main() {
	m := mg1.Migration{}
	migrate.Main(&m)
}
