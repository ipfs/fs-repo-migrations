package main

import (
	mg0 "github.com/ipfs/fs-repo-migrations/ipfs-0-to-1/migration"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

func main() {
	m := &mg0.Migration{}
	migrate.Main(m)
}
