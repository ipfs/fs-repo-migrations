package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
)

func yesNoPrompt(prompt string) bool {
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

func createFetcher(distPath string) migrations.Fetcher {
	const userAgent = "fs-repo-migrations"

	if distPath == "" {
		distPath = migrations.GetDistPathEnv(migrations.LatestIpfsDist)
	}

	return migrations.NewMultiFetcher(
		newIpfsFetcher(distPath, 0),
		migrations.NewHttpFetcher(distPath, "", userAgent, 0))
}

func main() {
	distPath := flag.String("distpath", "", "specify the distributions build to use")
	revertOk := flag.Bool("revert-ok", false, "allow running migrations backward")
	targetStr := flag.String("to", "latest", "repo version to upgrade to, or \"latest\" for latest repo version")
	version := flag.Bool("v", false, "print latest migration available and exit")
	yes := flag.Bool("y", false, "answer yes to all prompts")
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unrecognized arguments")
		flag.Usage()
		os.Exit(1)
	}

	fetcher := createFetcher(*distPath)

	var (
		err    error
		target int
	)

	if *version {
		target, err = latestRepoMigration(fetcher)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(target)
		return
	}

	switch *targetStr {
	case "":
		fmt.Fprintln(os.Stderr, "Please specify a target version to migrate the repo to")
		os.Exit(1)
	case "latest":
		target, err = latestRepoMigration(fetcher)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		target, err = strconv.Atoi(*targetStr)
		if err != nil || target < 1 {
			fmt.Fprintln(os.Stderr, "invalid target version, \"to\" must be a positive integer")
			os.Exit(1)
		}
		// Check that target is not greater than latest version.  Ignore error
		// here since may be offline and all migrations already in PATH.
		latest, _ := latestRepoMigration(fetcher)
		if latest != 0 && target > latest {
			fmt.Fprintln(os.Stderr, "migration version", target, "does not exist, latest is", latest)
			os.Exit(1)
		}

	}

	vnum, err := migrations.RepoVersion("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ipfs migration: ", err)
		os.Exit(1)
	}

	if vnum > target && !*revertOk {
		fmt.Fprintln(os.Stderr, "ipfs migration: attempt to run backward migration\nTo allow, run this command again with --revert-ok")
		os.Exit(1)
	}

	if vnum == target {
		fmt.Fprintln(os.Stderr, "ipfs migration: already at version", target)
		return
	}

	ipfsDir, _ := migrations.IpfsDir("")
	fmt.Printf("Found fs-repo version %d at %s\n", vnum, ipfsDir)
	prompt := fmt.Sprintf("Do you want to upgrade this to version %d? [y/n]", target)
	if !(*yes || yesNoPrompt(prompt)) {
		os.Exit(1)
	}

	err = migrations.RunMigration(context.Background(), fetcher, target, "", *revertOk)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ipfs migration: ", err)
		os.Exit(1)
	}
}

func latestRepoMigration(fetcher migrations.Fetcher) (int, error) {
	// TODO: Remove currentVersion and get list of all fs-repo-*-to-* in one
	// request and calculate latest
	//
	// When searching for latest migration, start looking using this repo version
	const currentVersion = 11

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var latestMigration int
	var err error
	for i := currentVersion; err == nil; i++ {
		dist := fmt.Sprintf("fs-repo-%d-to-%d", i-1, i)
		_, err = migrations.LatestDistVersion(ctx, fetcher, dist, false)
		if err == nil {
			latestMigration = i
		}
	}
	if latestMigration == 0 {
		return 0, fmt.Errorf("Could not get available repo migrations: %s", err)
	}
	return latestMigration, nil
}
