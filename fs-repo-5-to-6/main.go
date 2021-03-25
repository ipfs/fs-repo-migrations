package main

import (
	mg5 "github.com/ipfs/fs-repo-migrations/fs-repo-5-to-6/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg5.Migration{}
	migrate.Main(&m)
}
