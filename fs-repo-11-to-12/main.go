package main

import (
	mg11 "github.com/ipfs/fs-repo-migrations/fs-repo-11-to-12/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg11.Migration{}
	migrate.Main(&m)
}
