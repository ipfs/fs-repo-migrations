package mg9

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "9-to-10"
}

func (m Migration) Reversible() bool {
	return true
}

func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	log.VLog("locking repo at %q", opts.Path)
	lk, err := lock.Lock2(opts.Path)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)

	log.VLog("  - verifying version is '9'")
	if err := repo.CheckVersion("9"); err != nil {
		return err
	}

	log.Log("> Upgrading config to new format")

	path := filepath.Join(opts.Path, "config")
	if err := convertFile(path, ver9to10Bootstrap, ver9to10Addresses); err != nil {
		return err
	}

	if err := repo.WriteVersion("10"); err != nil {
		log.Error("failed to update version file to 10")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 9 to 10 succeeded")
	return nil
}

func writePhase(file string, phase int) error {
	return ioutil.WriteFile(file, []byte(fmt.Sprint(phase)), 0666)
}

func readPhase(file string) (int, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	return strconv.Atoi(string(data))
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")
	lk, err := lock.Lock2(opts.Path)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)
	if err := repo.CheckVersion("10"); err != nil {
		return err
	}

	if err := repo.WriteVersion("9"); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Println("lowered version number to 9")
	}

	return nil
}
