package main

import (
	mg15 "github.com/ipfs/fs-repo-migrations/fs-repo-15-to-16/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg15.Migration{}
	migrate.Main(m)
}
