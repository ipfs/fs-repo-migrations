// package mg13 contains the code to perform 13-14 repository migration in Kubo.
// This just move the AcceleratedDHTClient from the Experimental section to the Routing one.
package mg13

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"

	"github.com/ipfs/fs-repo-migrations/fs-repo-13-to-14/atomicfile"
)

const backupSuffix = ".13-to-14.bak"

// Migration implements the migration described above.
type Migration struct{}

// Versions returns the current version string for this migration.
func (m Migration) Versions() string {
	return "13-to-14"
}

// Reversible returns true, as we keep old config around
func (m Migration) Reversible() bool {
	return true
}

// Apply update the config.
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

	log.VLog("  - verifying version is '13'")
	if err := repo.CheckVersion("13"); err != nil {
		return err
	}

	log.Log("> Upgrading config to new format")

	path := filepath.Join(opts.Path, "config")
	in, err := os.Open(path)
	if err != nil {
		return err
	}

	// make backup
	backup, err := atomicfile.New(path+backupSuffix, 0600)
	if err != nil {
		return err
	}
	if _, err := backup.ReadFrom(in); err != nil {
		panicOnError(backup.Abort())
		return err
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		panicOnError(backup.Abort())
		return err
	}

	// Create a temp file to write the output to on success
	out, err := atomicfile.New(path, 0600)
	if err != nil {
		panicOnError(backup.Abort())
		panicOnError(in.Close())
		return err
	}

	if err := convert(in, out); err != nil {
		panicOnError(out.Abort())
		panicOnError(backup.Abort())
		panicOnError(in.Close())
		return err
	}

	if err := in.Close(); err != nil {
		panicOnError(out.Abort())
		panicOnError(backup.Abort())
	}

	if err := repo.WriteVersion("14"); err != nil {
		log.Error("failed to update version file to 14")
		// There was an error so abort writing the output and clean up temp file
		panicOnError(out.Abort())
		panicOnError(backup.Abort())
		return err
	} else {
		// Write the output and clean up temp file
		panicOnError(out.Close())
		panicOnError(backup.Close())
	}

	log.Log("updated version file")

	log.Log("Migration 13 to 14 succeeded")
	return nil
}

// panicOnError is reserved for checks we can't solve transactionally if an error occurs
func panicOnError(e error) {
	if e != nil {
		panic(fmt.Errorf("error can't be dealt with transactionally: %w", e))
	}
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
	if err := repo.CheckVersion("14"); err != nil {
		return err
	}

	cfg := filepath.Join(opts.Path, "config")
	if err := os.Rename(cfg+backupSuffix, cfg); err != nil {
		return err
	}

	if err := repo.WriteVersion("13"); err != nil {
		return err
	}
	if opts.Verbose {
		log.Log("lowered version number to 13")
	}

	return nil
}

// convert converts the config from one version to another
func convert(in io.Reader, out io.Writer) error {
	confMap := make(map[string]any)
	if err := json.NewDecoder(in).Decode(&confMap); err != nil {
		return err
	}

	// Move AcceleratedDHTClient key.
	var acceleratedDHTClient bool
	if e, ok := confMap["Experimental"]; ok {
		exp, ok := e.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid type for .Experimental got %T expected json map", e)
		}
		if a, ok := exp["AcceleratedDHTClient"]; ok {
			acc, ok := a.(bool)
			if !ok {
				return fmt.Errorf("invalid type for .Experimental.AcceleratedDHTClient got %T expected bool", e)
			}
			acceleratedDHTClient = acc
			delete(exp, "AcceleratedDHTClient")

			if len(exp) == 0 {
				delete(confMap, "Experimental")
			}
		}
	}

	// If the key missing insert new into routing
	var rr map[string]any
	if r, ok := confMap["Routing"]; ok {
		rr, ok = r.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid type for .Routing, got %T expected json map", r)
		}
	} else {
		rr = make(map[string]any)
		confMap["Routing"] = rr
	}
	if _, ok := rr["AcceleratedDHTClient"]; !ok {
		// Only add the key if it's not already present in the destination
		rr["AcceleratedDHTClient"] = acceleratedDHTClient
	}

	fixed, err := json.MarshalIndent(confMap, "", "  ")
	if err != nil {
		return err
	}

	if _, err := out.Write(fixed); err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}
