package mg7

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
	return "7-to-8"
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

	log.VLog("  - verifying version is '7'")
	if err := repo.CheckVersion("7"); err != nil {
		return err
	}

	basepath := filepath.Join(opts.Path, "config")
	v7path := filepath.Join(opts.Path, "config-v7")
	if err := os.Rename(basepath, v7path); err != nil {
		if os.IsNotExist(err) {
			_, err2 := os.Stat(v7path)
			if err2 == nil {
				log.Log("... config already renamed to config-v7, continuing")
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}

	log.Log("> Upgrading config to new format")

	if err := convertFile(v7path, basepath, ver7to8); err != nil {
		if opts.NoRevert {
			return err
		}
		err := os.Rename(v7path, basepath)
		if err != nil {
			log.Error(err)
		}
		return err
	}

	if err := repo.WriteVersion("8"); err != nil {
		log.Error("failed to update version file to 8")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 7 to 8 succeeded")
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
	if err := repo.CheckVersion("8"); err != nil {
		return err
	}

	phasefile := filepath.Join(opts.Path, "revert-phase")
	basepath := filepath.Join(opts.Path, "config")
	v8path := filepath.Join(opts.Path, "config-v8")

	phase, err := readPhase(phasefile)
	if err != nil {
		return fmt.Errorf("reading revert phase: %s", err)
	}

	for ; phase < 4; phase++ {
		switch phase {
		case 0:
			if err := os.Rename(basepath, v8path); err != nil {
				return err
			}
		case 1:
			if err := convertFile(v8path, basepath, ver8to7); err != nil {
				return err
			}
		case 2:
			if err := repo.WriteVersion("7"); err != nil {
				return err
			}
			if opts.Verbose {
				fmt.Println("lowered version number to 7")
			}
		}
		if err := writePhase(phasefile, phase+1); err != nil {
			return err
		}
	}
	os.Remove(phasefile)

	return nil
}
