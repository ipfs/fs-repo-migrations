package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	gomigrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mg0 "github.com/ipfs/fs-repo-migrations/ipfs-0-to-1/migration"
	mg1 "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/migration"
	mg2 "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/migration"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
)

var migrations = []gomigrate.Migration{
	&mg0.Migration{},
	&mg1.Migration{},
	&mg2.Migration{},
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
	fmt.Printf("===> Running migration %d to %d...\n", n, n+1)
	path, err := GetIpfsDir()
	if err != nil {
		return err
	}

	opts := gomigrate.Options{}
	opts.Path = path
	opts.Verbose = true

	err = migrations[n].Apply(opts)
	if err != nil {
		return fmt.Errorf("migration %d to %d failed: %s", n, n+1, err)
	}
	fmt.Printf("===> Migration %d to %d succeeded!\n", n, n+1)
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

func GetVersion(ipfsdir string) (int, error) {
	ver, err := mfsr.RepoPath(ipfsdir).Version()
	if _, ok := err.(mfsr.VersionFileNotFound); ok {
		// No version file in repo == version 0
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	vnum, err := strconv.Atoi(ver)
	if err != nil {
		return 0, err
	}

	return vnum, nil
}

func YesNoPrompt(prompt string) bool {
	var s string
	for {
		fmt.Printf("%s ", prompt)
		fmt.Scanf("%s", &s)
		switch s {
		case "y", "Y":
			return true
		case "n", "N":
			return false
		}
		fmt.Println("Please press either 'y' or 'n'")
	}
}

func main() {
	target := flag.Int("to", 3, "specify version to upgrade to")
	yes := flag.Bool("y", false, "answer yes to all prompts")
	version := flag.Bool("v", false, "print highest repo version and exit")

	flag.Parse()

	if *version {
		fmt.Println(3)
		return
	}

	if *target > len(migrations) {
		fmt.Printf("No known migration to version %d. Try updating this tool.\n", *target)
		os.Exit(1)
	}

	ipfsdir, err := GetIpfsDir()
	if err != nil {
		fmt.Println("ipfs migration: ", err)
		os.Exit(1)
	}

	vnum, err := GetVersion(ipfsdir)
	if err != nil {
		fmt.Println("ipfs migration: ", err)
		os.Exit(1)
	}

	if vnum >= *target {
		fmt.Println("ipfs migration: already at or above target version number")
		return
	}

	fmt.Printf("Found fs-repo version %d at %s\n", vnum, ipfsdir)
	prompt := fmt.Sprintf("Do you want to upgrade this to version %d? [y/n]", *target)
	if !(*yes || YesNoPrompt(prompt)) {
		os.Exit(1)
	}

	err = doMigrate(vnum, *target)
	if err != nil {
		fmt.Println("ipfs migration: ", err)
		os.Exit(1)
	}
}
