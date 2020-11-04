package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg10 "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/migration"
)

func main() {
	m := mg10.Migration{}
	migrate.Main(&m)
}
