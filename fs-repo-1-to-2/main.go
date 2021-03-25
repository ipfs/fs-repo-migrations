package main

import (
	mg1 "github.com/ipfs/fs-repo-migrations/fs-repo-1-to-2/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg1.Migration{}
	migrate.Main(&m)
}
