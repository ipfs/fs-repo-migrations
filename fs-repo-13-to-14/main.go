package main

import (
	mg13 "github.com/ipfs/fs-repo-migrations/fs-repo-13-to-14/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg13.Migration{}
	migrate.Main(m)
}
