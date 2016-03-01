package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	util "github.com/ipfs/ipfs-update/util"
	stump "github.com/whyrusleeping/stump"
)

func CheckMigration() error {
	stump.Log("checking if repo migration is needed...")
	p := util.IpfsDir()

	vfilePath := filepath.Join(p, "version")
	_, err := os.Stat(vfilePath)
	if os.IsNotExist(err) {
		stump.VLog("  - no prexisting repo to migrate")
		return nil
	}

	oldverB, err := ioutil.ReadFile(vfilePath)
	if err != nil {
		return err
	}

	oldver := strings.Trim(string(oldverB), "\n \t")
	stump.VLog("  - old repo version is", oldver)

	nbinver, err := util.RunCmd("", "ipfs", "version", "--repo")
	if err != nil {
		stump.Log("Failed to check new binary repo version.")
		stump.VLog("Reason: ", err)
		stump.Log("This is not an error.")
		stump.Log("This just means that you may have to manually run the migration")
		stump.Log("You will be prompted to do so upon starting the ipfs daemon if necessary")
		return nil
	}

	stump.VLog("  - repo version of new binary is ", nbinver)

	if oldver != nbinver {
		stump.Log("  - Migration required")
		return RunMigration(oldver, nbinver)
	}

	stump.VLog("  - no migration required")

	return nil
}

func RunMigration(oldv, newv string) error {
	migrateBin := "fs-repo-migrations"
	stump.VLog("  - checking for migrations binary...")
	_, err := exec.LookPath(migrateBin)
	if err != nil {
		stump.VLog("  - migrations not found on system, attempting to install")
		err := GetMigrations()
		if err != nil {
			return err
		}
	}

	// check to make sure migrations binary supports our target version
	err = verifyMigrationSupportsVersion(newv)
	if err != nil {
		return err
	}

	cmd := exec.Command(migrateBin, "-to", newv, "-y")

	cmd.Stdout = stump.LogOut
	cmd.Stderr = stump.ErrOut

	stump.Log("running migration: '%s -to %s -y'", migrateBin, newv)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("migration failed: %s", err)
	}

	stump.Log("migration succeeded!")
	return nil
}

func GetMigrations() error {
	// first, check if go is installed
	_, err := exec.LookPath("go")
	if err == nil {
		return getMigrationsGoGet()
	}

	// TODO: try and fetch from gobuilder
	stump.Log("could not find or install fs-repo-migrations, please manually install it")
	stump.Log("before running ipfs-update again.")
	return fmt.Errorf("failed to find migrations binary")
}

func getMigrationsGoGet() error {
	stump.VLog("  - fetching migrations using 'go get'")
	cmd := exec.Command("go", "get", "-u", "github.com/ipfs/fs-repo-migrations")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s", string(out), err)
	}
	stump.VLog("  - success. verifying...")

	// verify we can see the binary now
	p, err := exec.LookPath("fs-repo-migrations")
	if err != nil {
		return fmt.Errorf("install succeeded, but failed to find binary afterwards. (%s)", err)
	}
	stump.VLog("  - fs-repo-migrations now installed at %s", p)

	return nil
}

func verifyMigrationSupportsVersion(v string) error {
	stump.VLog("  - verifying migration supports version %s", v)
	vn, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("given migration version was not a number: %q", v)
	}

	sn, err := migrationsVersion()
	if err != nil {
		return err
	}

	if sn >= vn {
		return nil
	}

	stump.VLog("  - migrations doesnt support version %s, attempting to update")
	err = GetMigrations()
	if err != nil {
		return err
	}

	stump.VLog("  - migrations updated")

	sn, err = migrationsVersion()
	if err != nil {
		return err
	}

	if sn >= vn {
		return nil
	}

	return fmt.Errorf("no known migration supports version %s", v)
}

func migrationsVersion() (int, error) {
	out, err := exec.Command("fs-repo-migrations", "-v").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to check migrations version")
	}

	vs := strings.Trim(string(out), " \n\t")
	vn, err := strconv.Atoi(vs)
	if err != nil {
		return 0, fmt.Errorf("migrations binary version check did not return a number")
	}

	return vn, nil
}
