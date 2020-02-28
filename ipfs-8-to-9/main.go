package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg7 "github.com/ipfs/fs-repo-migrations/ipfs-7-to-8/migration"
)

func main() {
	m := mg7.Migration{}
	migrate.Main(&m)
}
