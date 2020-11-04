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
	mg10 "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/migration"
	homedir "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/mitchellh/go-homedir"
	mg2 "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/migration"
	mg3 "github.com/ipfs/fs-repo-migrations/ipfs-3-to-4/migration"
	mg4 "github.com/ipfs/fs-repo-migrations/ipfs-4-to-5/migration"
	mg5 "github.com/ipfs/fs-repo-migrations/ipfs-5-to-6/migration"
	mg6 "github.com/ipfs/fs-repo-migrations/ipfs-6-to-7/migration"
	mg7 "github.com/ipfs/fs-repo-migrations/ipfs-7-to-8/migration"
	mg8 "github.com/ipfs/fs-repo-migrations/ipfs-8-to-9/migration"
	mg9 "github.com/ipfs/fs-repo-migrations/ipfs-9-to-10/migration"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
)

var CurrentVersion = 10

var migrations = []gomigrate.Migration{
	&mg0.Migration{},
	&mg1.Migration{},
	&mg2.Migration{},
	&mg3.Migration{},
	&mg4.Migration{},
	&mg5.Migration{},
	&mg6.Migration{},
	&mg7.Migration{},
	&mg8.Migration{},
	&mg9.Migration{},
	&mg10.Migration{},
}

func GetIpfsDir() (string, error) {
	ipfspath := os.Getenv("IPFS_PATH")
	if ipfspath != "" {
		return ipfspath, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	if home == "" {
		return "", fmt.Errorf("could not determine IPFS_PATH, home dir not set")
	}

	v0defaultDir := path.Join(home, ".go-ipfs")
	_, err = os.Stat(v0defaultDir)
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

func runMigration(from int, to int) error {
	fmt.Printf("===> Running migration %d to %d...\n", from, to)
	path, err := GetIpfsDir()
	if err != nil {
		return err
	}

	opts := gomigrate.Options{}
	opts.Path = path
	opts.Verbose = true

	if to > from {
		err = migrations[from].Apply(opts)
	} else if to < from {
		err = migrations[to].Revert(opts)
	} else {
		// catch this earlier. expected invariant violated.
		err = fmt.Errorf("attempt to run migration to same version")
	}
	if err != nil {
		return fmt.Errorf("migration %d to %d failed: %s", from, to, err)
	}
	fmt.Printf("===> Migration %d to %d succeeded!\n", from, to)
	return nil
}

func doMigrate(from, to int) error {
	step := 1
	if from > to {
		step = -1
	}

	for cur := from; cur != to; cur += step {
		err := runMigration(cur, cur+step)
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
	target := flag.Int("to", CurrentVersion, "specify version to upgrade to")
	yes := flag.Bool("y", false, "answer yes to all prompts")
	version := flag.Bool("v", false, "print highest repo version handled and exit")
	revertOk := flag.Bool("revert-ok", false, "allow running migrations backward")

	flag.Parse()

	if *version {
		fmt.Println(CurrentVersion)
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

	if vnum > *target && !*revertOk {
		fmt.Println("ipfs migration: attempt to run backward migration\nTo allow, run this command again with --revert-ok")
		os.Exit(1)
	}

	if vnum == *target {
		fmt.Println("ipfs migration: already at target version number")
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
