// package mg15 contains the code to perform 15-16 repository migration in Kubo.
// This handles the following:
// - Add /webrtc-direct listener if preexisting /udp/ /quic-v1 exists
package mg15

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"

	"github.com/ipfs/fs-repo-migrations/fs-repo-15-to-16/atomicfile"
)

const backupSuffix = ".15-to-16.bak"

// Migration implements the migration described above.
type Migration struct{}

// Versions returns the current version string for this migration.
func (m Migration) Versions() string {
	return "15-to-16"
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

	log.VLog("  - verifying version is '15'")
	if err := repo.CheckVersion("15"); err != nil {
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

	if err := repo.WriteVersion("16"); err != nil {
		log.Error("failed to update version file to 16")
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

	log.Log("Migration 15 to 16 succeeded")
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
	if err := repo.CheckVersion("16"); err != nil {
		return err
	}

	cfg := filepath.Join(opts.Path, "config")
	if err := os.Rename(cfg+backupSuffix, cfg); err != nil {
		return err
	}

	if err := repo.WriteVersion("15"); err != nil {
		return err
	}
	if opts.Verbose {
		log.Log("lowered version number to 15")
	}

	return nil
}

var quicRegex = regexp.MustCompilePOSIX("/quic-v1$")

// convert converts the config from one version to another
func convert(in io.Reader, out io.Writer) error {
	confMap := make(map[string]any)
	if err := json.NewDecoder(in).Decode(&confMap); err != nil {
		return err
	}

	// Append /webrtc-direct listener if /udp/../quic-v1 is present in any of .Addresses fields
	if err := func() error {
		a, ok := confMap["Addresses"]
		if !ok {
			return nil
		}
		addresses, ok := a.(map[string]any)
		if !ok {
			fmt.Printf("invalid type for .Addresses got %T expected json map; skipping .Addresses\n", a)
			return nil
		}

		for _, addressToRemove := range [...]string{"Swarm", "Announce", "AppendAnnounce", "NoAnnounce"} {
			s, ok := addresses[addressToRemove]
			if !ok {
				continue
			}

			swarm, ok := s.([]interface{})
			if !ok {
				fmt.Printf("invalid type for .Addresses.%s got %T expected json array; skipping .Addresses.%s\n", addressToRemove, s, addressToRemove)
				continue
			}

			var newSwarm []interface{}
			uniq := map[string]struct{}{}
			for _, v := range swarm {
				if addr, ok := v.(string); ok {

					// if /quic-v1, add /webrtc-direct under the same port
					if quicRegex.MatchString(addr) {
						newAddr := quicRegex.ReplaceAllString(addr, "/webrtc-direct")

						if _, ok := uniq[newAddr]; ok {
							continue
						}
						uniq[newAddr] = struct{}{}

						newSwarm = append(newSwarm, newAddr)
					}

					// keep original addr
					if _, ok := uniq[addr]; ok {
						continue
					}
					uniq[addr] = struct{}{}

					newSwarm = append(newSwarm, addr)
					continue
				}
				newSwarm = append(newSwarm, v)
			}
			addresses[addressToRemove] = newSwarm
		}
		return nil
	}(); err != nil {
		return err
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
