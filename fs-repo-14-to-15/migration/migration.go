// package mg14 contains the code to perform 14-15 repository migration in Kubo.
// This handles the following:
// - Replaces bootstrap QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ from /quic to /quic-v1
// - Removes /quic only addresses from .Addresses.Swarm
package mg14

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"

	"github.com/ipfs/fs-repo-migrations/fs-repo-14-to-15/atomicfile"
)

const backupSuffix = ".14-to-15.bak"

// Migration implements the migration described above.
type Migration struct{}

// Versions returns the current version string for this migration.
func (m Migration) Versions() string {
	return "14-to-15"
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

	log.VLog("  - verifying version is '14'")
	if err := repo.CheckVersion("14"); err != nil {
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

	if err := repo.WriteVersion("15"); err != nil {
		log.Error("failed to update version file to 15")
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

	log.Log("Migration 14 to 15 succeeded")
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
	if err := repo.CheckVersion("15"); err != nil {
		return err
	}

	cfg := filepath.Join(opts.Path, "config")
	if err := os.Rename(cfg+backupSuffix, cfg); err != nil {
		return err
	}

	if err := repo.WriteVersion("14"); err != nil {
		return err
	}
	if opts.Verbose {
		log.Log("lowered version number to 14")
	}

	return nil
}

// convert converts the config from one version to another
func convert(in io.Reader, out io.Writer) error {
	confMap := make(map[string]any)
	if err := json.NewDecoder(in).Decode(&confMap); err != nil {
		return err
	}

	// Upgrade bootstrapper QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ from /quic to /quic-v1
	if b, ok := confMap["Bootstrap"]; ok {
		bootstrap, ok := b.([]interface{})
		if !ok {
			return fmt.Errorf("invalid type for .Bootstrap got %T expected json array", b)
		}

		for i, v := range bootstrap {
			if v == "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ" {
				bootstrap[i] = "/ip4/104.131.131.82/udp/4001/quic-v1/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
			}
		}
	}

	// Remove /quic only addresses from .Addresses.Swarm
	if a, ok := confMap["Addresses"]; ok {
		addresses, ok := a.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid type for .Addresses got %T expected json map", a)
		}

		if s, ok := addresses["Swarm"]; ok {
			swarm, ok := s.([]interface{})
			if !ok {
				return fmt.Errorf("invalid type for .Addresses.Swarm got %T expected json array", s)
			}

			var newSwarm []interface{}
			for _, v := range swarm {
				if addr, ok := v.(string); ok {
					if !strings.Contains(addr, "/quic") || strings.Contains(addr, "/quic-v1") {
						newSwarm = append(newSwarm, addr)
					}
				} else {
					newSwarm = append(newSwarm, v)
				}
			}
			addresses["Swarm"] = newSwarm
		}
	}

	// Save new config
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
