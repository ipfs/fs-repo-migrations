package main

import (
	mg7 "github.com/ipfs/fs-repo-migrations/ipfs-7-to-8/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := mg7.Migration{}
	migrate.Main(&m)
}
