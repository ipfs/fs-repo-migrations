package mg3

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

	flatfs "github.com/ipfs/fs-repo-migrations/fs-repo-4-to-5/go-ds-flatfs"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "4-to-5"
}

func (m Migration) Reversible() bool {
	return true
}

func revertStep2(ffspath string) error {
	if err := flatfs.DowngradeV1toV0(ffspath); err != nil {
		return fmt.Errorf("reverting flatfsv1 upgrade: %s", err)
	}
	return nil
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

	log.VLog("  - verifying version is '4'")
	if err := repo.CheckVersion("4"); err != nil {
		return err
	}

	basepath := filepath.Join(opts.Path, "blocks")
	ffspath := filepath.Join(opts.Path, "blocks-v4")
	if err := os.Rename(basepath, ffspath); err != nil {
		// the error returned is unreliable so instead check that the
		// old path doesn't exist and the new one does and is a directory
		if _, err2 := os.Stat(basepath); os.IsNotExist(err2) {
			if fi, err2 := os.Stat(ffspath); err2 == nil && fi.IsDir() {
				log.Log("... blocks already renamed to blocks-v4, continuing")
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
		err := os.Rename(ffspath, basepath)
		if err != nil {
			log.Error(err)
		}
		return e
	}

	log.Log("> Upgrading datastore format to have sharding specification file")
	if err := flatfs.UpgradeV0toV1(ffspath, 5); err != nil {
		if os.IsExist(err) {
			id, err2 := flatfs.ReadShardFunc(ffspath)
			if err2 == nil && id.String() == flatfs.Prefix(5).String() {
				log.Log("... datastore already has sharding specification file, continuing")
				err = nil
			}
		}
		if err != nil {
			return revert1(err)
		}
	}

	tempffs := filepath.Join(opts.Path, "blocks-v5")
	log.Log("> creating a new flatfs datastore with new format")
	if err := flatfs.Create(tempffs, flatfs.NextToLast(2)); err != nil {
		if err == flatfs.ErrDatastoreExists {
			log.Log("... new flatfs datastore already exists continuing")
			err = nil
		}
		if err != nil {
			if err2 := revertStep2(ffspath); err2 != nil {
				log.Error(err2)
			}
			return revert1(err)
		}
	}

	revert3 := func(mainerr error) error {
		log.Error("failed to convert flatfs datastore: %s", mainerr)
		if opts.NoRevert {
			return mainerr
		}
		log.Log("attempting to revert...")

		if _, err := os.Stat(filepath.Join(ffspath, "SHARDING")); os.IsNotExist(err) {
			flatfs.UpgradeV0toV1(ffspath, 5)
		}

		if err := flatfs.Move(tempffs, ffspath, os.Stdout); err != nil {
			log.Error("reverting flatfs conversion failed: %s", err)
			log.Error("Please file a bug report at https://github.com/ipfs/fs-repo-migrations")
			return err
		}

		if err := os.Remove(tempffs); err != nil {
			log.Error("cleaning up temp flatfs directory: %s", err)
		}

		if err := revertStep2(ffspath); err != nil {
			log.Error(err)
		}

		return revert1(mainerr)
	}

	log.Log("> converting current flatfs datastore to new format")
	if err := flatfs.Move(ffspath, tempffs, os.Stdout); err != nil {
		return revert3(err)
	}

	log.Log("> moving new datastore into place")
	if err := os.Remove(ffspath); err != nil {
		return revert3(fmt.Errorf("removing supposedly empty old flatfs dir: %s", err))
	}

	revert4 := func(mainerr error) error {
		if opts.NoRevert {
			return mainerr
		}
		if err := os.Mkdir(ffspath, 0755); err != nil {
			log.Error("recreating flatfs directory: %s", err)
			return err
		}

		return revert3(mainerr)
	}

	log.Log("> moving transferred datastore back into place")
	if err := os.Rename(tempffs, basepath); err != nil {
		return revert4(fmt.Errorf("moving new datastore into place of the old one: %s", err))
	}

	err = repo.WriteVersion("5")
	if err != nil {
		log.Error("failed to update version file to 5")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 4 to 5 succeeded")
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
	if err := repo.CheckVersion("5"); err != nil {
		return err
	}

	phasefile := filepath.Join(opts.Path, "revert-phase")
	basepath := filepath.Join(opts.Path, "blocks")
	v5path := filepath.Join(opts.Path, "blocks-v5")
	v4path := filepath.Join(opts.Path, "blocks-v4")

	phase, err := readPhase(phasefile)
	if err != nil {
		return fmt.Errorf("reading revert phase: %s", err)
	}

	for ; phase < 6; phase++ {
		switch phase {
		case 0:
			if err := os.Rename(basepath, v5path); err != nil {
				return err
			}
		case 1:
			if err := flatfs.Create(v4path, flatfs.Prefix(5)); err != nil {
				return err
			}

		case 2:
			if err := flatfs.Move(v5path, v4path, os.Stdout); err != nil {
				return err
			}

		case 3:
			if err := flatfs.DowngradeV1toV0(v4path); err != nil {
				return err
			}

		case 4:
			if err := os.Rename(v4path, basepath); err != nil {
				return err
			}

		case 5:
			err = repo.WriteVersion("4")
			if err != nil {
				return err
			}
			if opts.Verbose {
				fmt.Println("lowered version number to 4")
			}
		}
		if err := writePhase(phasefile, phase+1); err != nil {
			return err
		}
	}
	os.Remove(v5path)
	os.Remove(phasefile)

	return nil
}
