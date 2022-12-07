// package mg12 contains the code to perform 12-13 repository migration in Kubo.
// This just change some config fields to add webtransport listens on ones that quic uses,
// and removes some hardcoded defaults that are no longer present on fresh 'ipfs init'.
package mg12

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"

	"github.com/ipfs/fs-repo-migrations/fs-repo-12-to-13/atomicfile"
)

const backupSuffix = ".12-to-13.bak"

// Migration implements the migration described above.
type Migration struct{}

// Versions returns the current version string for this migration.
func (m Migration) Versions() string {
	return "12-to-13"
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

	log.VLog("  - verifying version is '12'")
	if err := repo.CheckVersion("12"); err != nil {
		return err
	}

	log.Log("> Upgrading config to new format")

	path := filepath.Join(opts.Path, "config")
	if err := convertFile(path); err != nil {
		return err
	}

	if err := repo.WriteVersion("13"); err != nil {
		log.Error("failed to update version file to 13")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 12 to 13 succeeded")
	return nil
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
	if err := repo.CheckVersion("13"); err != nil {
		return err
	}

	cfg := filepath.Join(opts.Path, "config")
	if err := os.Rename(cfg+backupSuffix, cfg); err != nil {
		return err
	}

	if err := repo.WriteVersion("12"); err != nil {
		return err
	}
	if opts.Verbose {
		log.Log("lowered version number to 12")
	}

	return nil
}

// convertFile converts a config file from 12 version to 13
func convertFile(path string) error {
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
		backup.Abort()
		return err
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		backup.Abort()
		return err
	}

	// Create a temp file to write the output to on success
	out, err := atomicfile.New(path, 0600)
	if err != nil {
		in.Close()
		backup.Abort()
		return err
	}

	err = convert(in, out)

	in.Close()

	if err != nil {
		// There was an error so abort writing the output and clean up temp file
		out.Abort()
		backup.Abort()
	} else {
		// Write the output and clean up temp file
		out.Close()
		backup.Close()
	}

	return err
}

// convert converts the config from one version to another
func convert(in io.Reader, out io.Writer) error {
	confMap := make(map[string]any)
	if err := json.NewDecoder(in).Decode(&confMap); err != nil {
		return err
	}

	// run this first to avoid having both quic and quic-v1 webtransport addresses
	runOnAllAddressFields(confMap, multiaddrPatternReplace(false, "/quic/webtransport", "/quic-v1/webtransport", "/p2p-circuit"))
	runOnAllAddressFields(confMap, multiaddrPatternReplace(true, "/quic", "/quic-v1", "/p2p-circuit"))
	runOnAllAddressFields(confMap, multiaddrPatternReplace(true, "/quic-v1", "/quic-v1/webtransport", "/p2p-circuit", "/webtransport"))

	convertRouting(confMap)
	convertReprovider(confMap)
	convertConnMgr(confMap)

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

func multiaddrPatternReplace(add bool, old, new string, notBefore ...string) func(in []any) (out []any) {
	return func(in []any) (out []any) {
		for _, w := range in {
			if add {
				out = append(out, w)
			}

			v, ok := w.(string)
			if !ok {
				continue
			}

			var r string
			last := len(v)
		ScanLoop:
			for i := len(v); i != 0; {
				i--
				if hasPrefixAndEndsOrSlash(v[i:], old) {
					r = new + v[i+len(old):last] + r
					last = i
				}
				for _, not := range notBefore {
					if hasPrefixAndEndsOrSlash(v[i:], not) {
						break ScanLoop
					}
				}
			}
			r = v[:last] + r

			// always append if we didn't appended previously
			if !add || r != v {
				out = append(out, r)
			}
		}
		return
	}
}

func hasPrefixAndEndsOrSlash(s, prefix string) bool {
	return strings.HasPrefix(s, prefix) && (len(prefix) == len(s) || s[len(prefix)] == '/')
}

func runOnAllAddressFields[T any, O any](m map[string]any, transformer func(T) O) {
	applyChangeOnLevelPlusOnes(m, transformer, "Addresses", "Announce", "AppendAnnounce", "NoAnnounce", "Swarm")
	applyChangeOnLevelPlusOnes(m, transformer, "Swarm", "AddrFilters")
}

// this walk one step in m, then walk all of vs, then try to cast to T, if all of this succeeded for thoses elements, pass it through transform
func applyChangeOnLevelPlusOnes[T any, O any, K comparable, V comparable](m map[K]any, transform func(T) O, l0 K, vs ...V) {
	if addresses, ok := m[l0].(map[V]any); ok {
		for _, v := range vs {
			if addrs, ok := addresses[v].(T); ok {
				addresses[v] = transform(addrs)
			}
		}
	}
}
