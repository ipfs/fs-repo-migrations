package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
)

var CurrentVersion = 11

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

	vnum, err := migrations.RepoVersion("")
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

	ipfsDir, _ := migrations.IpfsDir()
	fmt.Printf("Found fs-repo version %d at %s\n", vnum, ipfsDir)
	prompt := fmt.Sprintf("Do you want to upgrade this to version %d? [y/n]", *target)
	if !(*yes || YesNoPrompt(prompt)) {
		os.Exit(1)
	}

	err = migrations.RunMigration(context.Background(), *target, "")
	if err != nil {
		fmt.Println("ipfs migration: ", err)
		os.Exit(1)
	}
}
