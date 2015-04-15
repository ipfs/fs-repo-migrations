package main

import (
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg0 "github.com/ipfs/fs-repo-migrations/ipfs-0-to-1/migration"
)

func main() {
	m := &mg0.Migration{}
	migrate.Main(m)
}
