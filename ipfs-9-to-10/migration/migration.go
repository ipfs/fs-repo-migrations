package mg9

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/repolock"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
	log "github.com/ipfs/fs-repo-migrations/stump"
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

	basepath := filepath.Join(opts.Path, "config")
	v9path := filepath.Join(opts.Path, "config-v9")
	if err := os.Rename(basepath, v9path); err != nil {
		if os.IsNotExist(err) {
			_, err2 := os.Stat(v9path)
			if err2 == nil {
				log.Log("... config already renamed to config-v9, continuing")
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}

	log.Log("> Upgrading config to new format")

	if err := convertFile(v9path, basepath, true, ver9to10Bootstrap, ver9to10Swarm); err != nil {
		if opts.NoRevert {
			return err
		}
		err := os.Rename(v9path, basepath)
		if err != nil {
			log.Error(err)
		}
		return err
	}

	if err := repo.WriteVersion("10"); err != nil {
		log.Error("failed to update version file to 10")
		return err
	}

	log.Log("updated version file")

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

	phasefile := filepath.Join(opts.Path, "revert-phase")
	basepath := filepath.Join(opts.Path, "config")
	v10path := filepath.Join(opts.Path, "config-v10")

	phase, err := readPhase(phasefile)
	if err != nil {
		return fmt.Errorf("reading revert phase: %s", err)
	}

	for ; phase < 3; phase++ {
		switch phase {
		case 0:
			if err := os.Rename(basepath, v10path); err != nil {
				return err
			}
		case 1:
			if err := convertFile(v10path, basepath, false, ver10to9Bootstrap, ver10to9Swarm); err != nil {
				return err
			}
		case 2:
			if err := repo.WriteVersion("9"); err != nil {
				return err
			}
			if opts.Verbose {
				fmt.Println("lowered version number to 9")
			}
		}
		if err := writePhase(phasefile, phase+1); err != nil {
			return err
		}
	}
	os.Remove(phasefile)

	return nil
}
