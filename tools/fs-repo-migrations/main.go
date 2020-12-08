package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"

	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	homedir "github.com/mitchellh/go-homedir"
)

var CurrentVersion = 11

func GetIpfsDir() (string, error) {
	ipfspath := os.Getenv("IPFS_PATH")
	if ipfspath != "" {
		expandedPath, err := homedir.Expand(ipfspath)
		if err != nil {
			return "", err
		}
		return expandedPath, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	if home == "" {
		return "", errors.New("could not determine IPFS_PATH, home dir not set")
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

func osExeFileName(s string) string {
	if runtime.GOOS == "windows" {
		return s + ".exe"
	}
	return s
}

func runMigration(from, to int) error {
	fmt.Printf("===> Running migration %d to %d...\n", from, to)
	path, err := GetIpfsDir()
	if err != nil {
		return err
	}

	migrateBin := osExeFileName(fmt.Sprintf("ipfs-%d-to-%d", from, to))
	_, err = exec.LookPath(migrateBin)
	if err != nil {
		return err
	}

	pathArg := fmt.Sprintf("-path=%s", path)
	if to > from {
		cmd := exec.Command(migrateBin, pathArg, "-verbose=true")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
	} else if to < from {
		cmd := exec.Command(migrateBin, pathArg, "-verbose=true", "-revert")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	} else {
		// catch this earlier. expected invariant violated.
		err = errors.New("attempt to run migration to same version")
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
