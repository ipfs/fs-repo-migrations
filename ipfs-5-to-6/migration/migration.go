package mg5

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
	return "5-to-6"
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

	log.VLog("  - verifying version is '5'")
	if err := repo.CheckVersion("5"); err != nil {
		return err
	}

	basepath := filepath.Join(opts.Path, "config")
	v5path := filepath.Join(opts.Path, "config-v5")
	if err := os.Rename(basepath, v5path); err != nil {
		if os.IsNotExist(err) {
			_, err2 := os.Stat(v5path)
			if err2 == nil {
				log.Log("... config already renamed to config-v5, continuing")
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}

	revert1 := func(e error) error {
		if opts.NoRevert {
			return e
		}
		err := os.Rename(v5path, basepath)
		if err != nil {
			log.Error(err)
		}
		return e
	}

	log.Log("> Upgrading config to new format")

	cfg, err := convertFile(v5path, basepath, ver5to6)
	if err != nil {
		if err != nil {
			return revert1(err)
		}
	}

	// if any part of this is nil it is a programmer error
	dsc, err := AnyDatastoreConfig(
		newCiConfig(cfg.get("datastore").(map[string]interface{})).
			get("spec").(map[string]interface{}))
	if err != nil {
		return revert1(err)
	}

	err = ioutil.WriteFile(
		filepath.Join(opts.Path, "datastore_spec"),
		dsc.DiskSpec().Bytes(),
		0600)
	if err != nil {
		return revert1(err)
	}

	err = repo.WriteVersion("6")
	if err != nil {
		log.Error("failed to update version file to 6")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 5 to 6 succeeded")
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
	if err := repo.CheckVersion("6"); err != nil {
		return err
	}

	phasefile := filepath.Join(opts.Path, "revert-phase")
	basepath := filepath.Join(opts.Path, "config")
	v6path := filepath.Join(opts.Path, "config-v6")

	phase, err := readPhase(phasefile)
	if err != nil {
		return fmt.Errorf("reading revert phase: %s", err)
	}

	for ; phase < 4; phase++ {
		switch phase {
		case 0:
			if err := os.Rename(basepath, v6path); err != nil {
				return err
			}
		case 1:
			if _, err := convertFile(v6path, basepath, ver6to5); err != nil {
				return err
			}
		case 2:
			if err := os.Remove(filepath.Join(opts.Path, "datastore_spec")); err != nil {
				return err
			}
		case 3:
			if err := repo.WriteVersion("5"); err != nil {
				return err
			}
			if opts.Verbose {
				fmt.Println("lowered version number to 5")
			}
		}
		if err := writePhase(phasefile, phase+1); err != nil {
			return err
		}
	}
	os.Remove(phasefile)

	return nil
}
