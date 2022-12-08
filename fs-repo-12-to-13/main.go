package main

import (
	mg12 "github.com/ipfs/fs-repo-migrations/fs-repo-12-to-13/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg12.Migration{}
	migrate.Main(m)
}
