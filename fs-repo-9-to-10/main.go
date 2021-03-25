package main

import (
	mg9 "github.com/ipfs/fs-repo-migrations/fs-repo-9-to-10/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg9.Migration{}
	migrate.Main(&m)
}
