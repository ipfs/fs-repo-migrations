package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
)

const currentVersion = 11

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
	target := flag.Int("to", currentVersion, "specify version to upgrade to")
	yes := flag.Bool("y", false, "answer yes to all prompts")
	version := flag.Bool("v", false, "print latest migrationavailable and exit")
	revertOk := flag.Bool("revert-ok", false, "allow running migrations backward")

	flag.Parse()

	var err error
	if *version {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		var latestMigration int
		for i := currentVersion; err == nil; i++ {
			dist := fmt.Sprintf("ipfs-%d-to-%d", i-1, i)
			_, err = migrations.LatestDistVersion(ctx, dist)
			if err == nil {
				latestMigration = i
			}
		}
		if latestMigration == 0 {
			fmt.Fprintln(os.Stderr, "Could not get available repo migrations:", err)
			os.Exit(1)
		}
		fmt.Println(latestMigration)
		return
	}

	if *target == 0 {
		fmt.Fprintln(os.Stderr, "Please specify a target version to migrate the repo to")
		os.Exit(1)
	}

	vnum, err := migrations.RepoVersion("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ipfs migration: ", err)
		os.Exit(1)
	}

	if vnum > *target && !*revertOk {
		fmt.Fprintln(os.Stderr, "ipfs migration: attempt to run backward migration\nTo allow, run this command again with --revert-ok")
		os.Exit(1)
	}

	if vnum == *target {
		fmt.Fprintln(os.Stderr, "ipfs migration: already at target version number")
		return
	}

	ipfsDir, _ := migrations.IpfsDir("")
	fmt.Printf("Found fs-repo version %d at %s\n", vnum, ipfsDir)
	prompt := fmt.Sprintf("Do you want to upgrade this to version %d? [y/n]", *target)
	if !(*yes || YesNoPrompt(prompt)) {
		os.Exit(1)
	}

	err = migrations.RunMigration(context.Background(), *target, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ipfs migration: ", err)
		os.Exit(1)
	}
}
