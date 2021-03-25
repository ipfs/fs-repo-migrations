package main

import (
	mg10 "github.com/ipfs/fs-repo-migrations/fs-repo-10-to-11/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg10.Migration{}
	migrate.Main(&m)
}
