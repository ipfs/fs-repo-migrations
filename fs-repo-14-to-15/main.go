package main

import (
	mg14 "github.com/ipfs/fs-repo-migrations/fs-repo-14-to-15/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg14.Migration{}
	migrate.Main(m)
}
