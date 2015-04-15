package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	gomigrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/ipfs-0-to-1/mfsr"
	mg0 "github.com/ipfs/fs-repo-migrations/ipfs-0-to-1/migration"
	mg1 "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/migration"
)

var migrations = []gomigrate.Migration{
	&mg0.Migration{},
	&mg1.Migration{},
}

func GetIpfsDir() (string, error) {
	ipfspath := os.Getenv("IPFS_PATH")
	if ipfspath != "" {
		return ipfspath, nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("could not determine IPFS_PATH, home dir not set")
	}

	v0defaultDir := path.Join(home, ".go-ipfs")
	_, err := os.Stat(v0defaultDir)
	if err == nil {
		return v0defaultDir, nil
	}

	if !os.IsNotExist(err) {
		return "", err
	}

	v2defaultDir := path.Join(home, ".ipfs")
	_, err = os.Stat(v2defaultDir)
	if err == nil {
		return v2defaultDir, nil
	}

	return "", err
}

func runMigration(n int) error {
	fmt.Printf("Running migration %d to %s...\n", n, n+1)
	path, err := GetIpfsDir()
	if err != nil {
		return err
	}

	opts := gomigrate.Options{}
	opts.Path = path

	err = migrations[n].Apply(opts)
	if err != nil {
		return fmt.Errorf("migration %d to %d failed: %s", n, n+1, err)
	}
	fmt.Printf("Migration %d to %d succeeded!\n", n, n+1)
	return nil
}

func doMigrate(from, to int) error {
	cur := from
	for ; cur < to; cur++ {
		err := runMigration(cur)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	target := flag.Int("to", 2, "specify version to upgrade to")

	flag.Parse()

	ipfsdir, err := GetIpfsDir()
	if err != nil {
		fmt.Printf("ipfs migration: %s\n", err)
		os.Exit(1)
	}

	ver, err := mfsr.RepoPath(ipfsdir).Version()
	if err != nil {
		fmt.Printf("ipfs migration: %s\n", err)
		os.Exit(1)
	}

	vnum, err := strconv.Atoi(ver)
	if err != nil {
		fmt.Printf("ipfs migration: %s\n", err)
		os.Exit(1)
	}

	if vnum >= *target {
		fmt.Println("ipfs migration: already at or above target version number")
		return
	}

	err = doMigrate(vnum, *target)
	if vnum >= *target {
		fmt.Println("ipfs migration: already at or above target version number")
		return
	}
}
