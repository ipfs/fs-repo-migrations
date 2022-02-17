// package mg11 contains the code to perform 11-12 repository migration in
// go-ipfs. This performs a switch to raw multihashes for all keys in the
// go-ipfs datastore (https://github.com/ipfs/go-ipfs/issues/6815).
package mg11

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/ipfs/fs-repo-migrations/tools/stump"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	filestore "github.com/ipfs/go-filestore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	gc "github.com/ipfs/go-ipfs/gc"
)

const backupFile = "11-to-12-cids.txt"

// mfs root key path in the datastore
var mfsRootKey = ds.NewKey("/local/filesroot")

// blocks prefix in the datastore
var blocksPrefix = ds.NewKey("/blocks")

// filestore prefix in the datastore
var filestorePrefix = filestore.FilestorePrefix

// keyspace that we will be migrating and that contains paths in the form
// <prefix>/<cid>.
var migrationPrefixes = []ds.Key{
	blocksPrefix,
	filestorePrefix,
}

// Migration implements the migration described above.
type Migration struct {
	dstore ds.Batching
}

// Versions returns the current version string for this migration.
func (m *Migration) Versions() string {
	return "11-to-12"
}

// Reversible returns true.
func (m *Migration) Reversible() bool {
	return true
}

// Apply runs the migration and writes a log file that can be used by Revert.
// Steps:
// - Open a raw datastore using go-ipfs settings
// - Simulate the migration and save a backup log
// - Run the migration by storing all CIDv1 addressed logs as raw-multihash
//   addressed.
func (m *Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	lk, err := m.lock(opts)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)

	log.VLog("  - verifying version is '11'")
	if err := repo.CheckVersion("11"); err != nil {
		return err
	}

	err = m.open(opts)
	if err != nil {
		return err
	}
	defer m.dstore.Close()

	log.VLog("  - starting CIDv1 to raw multihash block migration")

	f, err := createBackupFile(opts.Path, backupFile)
	if err != nil {
		log.Error(err)
		return err
	}
	buf := bufio.NewWriter(f)

	swapCh := make(chan Swap, 1000)
	writingDone := make(chan struct{})

	// Write the old CIDv1-key of every swapped item to the buffer.
	go func() {
		for sw := range swapCh {
			// Only write the Old string (a CID). We can derive
			// the multihash from it.
			fmt.Fprint(buf, sw.Old.String(), "\n")
		}
		close(writingDone)
	}()

	// DRY RUN: A swapper for each migration prefix and write everything
	// to the file.
	//
	// Reasoning: We do not want a broken or unwritten file if the
	// migration blows up. If it does blow up half-way through, and we
	// need to run it again, we will only append to this file. Having
	// potential duplicate entries in the backup file will not break
	// reverts.
	for _, prefix := range migrationPrefixes {
		log.VLog("  - Adding keys in prefix %s to backup file", prefix)
		cidSwapper := CidSwapper{Prefix: prefix, Store: m.dstore, SwapCh: swapCh}
		total, err := cidSwapper.Prepare() // DRY RUN
		if err != nil {
			close(swapCh)
			log.Error(err)
			return err
		}
		log.Log("%d CIDv1 keys added to backup file for %s", total, prefix)
	}
	close(swapCh)
	// Wait for our writing to finish before doing the flushing.
	<-writingDone
	buf.Flush()
	f.Close()

	err = m.scanAndSwap(filepath.Join(opts.Path, backupFile), false) // revert=false
	if err != nil {
		log.Error(err)
		return err
	}

	// Wrap up, we are now in repo-version 12.
	if err := repo.WriteVersion("12"); err != nil {
		log.Error("failed to write version file")
		return err
	}
	log.Log("updated version file")

	return nil
}

// Revert attempts to undo the migration using the log file written by Apply.
// Steps:
// - Read the backup log and write all entries as a CIDv1-addressed block
// - Do the same with the MFS root
// - Do the same with all the CidV1 blocks recursively referenced in the pinset
//
// Revert does not delete blocks that are reverted so cover some corner cases.
func (m *Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting %s repo migration", m.Versions())

	lk, err := m.lock(opts)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)

	log.VLog("  - verifying version is '12'")
	if err := repo.CheckVersion("12"); err != nil {
		return err
	}

	log.VLog("  - starting raw multihash to CIDv1 block migration")
	err = m.open(opts)
	if err != nil {
		return err
	}
	defer m.dstore.Close()

	// Open revert path for reading
	backupPath := filepath.Join(opts.Path, backupFile)
	err = m.scanAndSwap(backupPath, true) // revert = true
	if err != nil {
		log.Error(err)
		return err
	}

	// Wrap up the Revert. We are back at version 11.
	if err := repo.WriteVersion("11"); err != nil {
		log.Error("failed to write version file")
		return err
	}

	log.Log("reverted version file to version 11")

	// Move the backup file out of the way.
	err = os.Rename(backupPath, backupPath+".reverted")
	if err != nil {
		log.Error("could not rename the backup file, but migration worked: %s", err)
		return err
	}
	return nil
}

// Receives a backup file which contains all the things that need to be
// migrated and reads every line, performing swaps in the needed direction.
func (m *Migration) scanAndSwap(backupPath string, revert bool) error {
	f, err := getBackupFile(backupPath)
	if err != nil {
		log.Error(err)
		return err
	}
	defer f.Close()

	swapCh := make(chan Swap, 1000)
	scanner := bufio.NewScanner(f)
	var scannerErr error

	// This will send swap objects to the swapping channel as they
	// are read from the backup file on disk. It will also send MFS and
	// pinset pins for reversal when doing a revert.
	go func() {
		defer close(swapCh)

		// Process backup file first.
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			}
			// The backup file only contains the original cidv1
			// path. Do the path massaging magic to figure out the
			// actual Cidv1 and derived Cidv0-path (current key).
			cidPath := ds.NewKey(line)
			cidKey := ds.NewKey(cidPath.BaseNamespace())
			prefix := cidPath.Parent()
			cid, err := dsKeyToCid(cidKey)
			if err != nil {
				log.Error("could not parse cid from backup file: %s", err)
				scannerErr = err
				break
			}
			mhashPath := prefix.Child(dshelp.MultihashToDsKey(cid.Hash()))

			// The swapper will move cidPath to mhashPath, and the unswapper
			// will do the opposite.
			sw := Swap{Old: cidPath, New: mhashPath}
			swapCh <- sw
		}
		if err := scanner.Err(); err != nil {
			log.Error(err)
			return
		}

		if revert {
			// Process MFS/pinset. We have to do this in cases the
			// user has been running with the migration for some
			// time and made changes to the pinset or the MFS
			// root.
			if err := walkPinsAndMFS(swapCh, m.dstore); err != nil {
				log.Error(err)
				return
			}
		}
	}()

	// The backup file contains prefixed keys, so we do not need to set
	// Prefix in the CidSwapper.
	cidSwapper := CidSwapper{Store: m.dstore}
	var total uint64
	if revert {
		total, err = cidSwapper.Revert(swapCh)
	} else {
		total, err = cidSwapper.Run(swapCh)
	}
	if err != nil {
		log.Error(err)
		return err
	}

	// The swapper will only return after swapCh is closed, so we know
	// scannerErr is safe to read at this point.
	if scannerErr != nil {
		return err
	}

	if revert {
		log.Log("%d multihashes swapped to CidV1s", total)
	} else {
		log.Log("%d CidV1s swapped to multihashes", total)
	}
	return nil
}

// sends all pins in the pinset and the MFS root to the unswap channel when
// they are CidV1s. This is used during revert. Pins may have been added or
// MFS root changed at some point between the migration and the revert.  If we
// do not revert those CIDv1s, we might find that go-ipfs does not know
// anymore how to find those blocks (they should be reverted and addressed as
// CIDv1 in the blockstore).
//
// In the best case, most of those blocks will already be stored correctly and
// the revert can swiftly do nothing.
func walkPinsAndMFS(unswapCh chan Swap, dstore ds.Batching) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var bestEffortRoots []cid.Cid

	// There should always be an MFS root. We add it to the list of things
	// to revert.
	mfsRoot, err := getMFSRoot(dstore)
	if err == ds.ErrNotFound {
		log.Error("empty MFS root")
		return err
	} else if err != nil {
		log.Error(err)
		return err
	}

	log.Log("MFS Root: %s\n", mfsRoot)
	bestEffortRoots = append(bestEffortRoots, mfsRoot)

	// Get a pinner so that we can recursively list all pins.
	pinner, dags, err := getPinner(ctx, dstore)
	if err != nil {
		return err
	}

	output := make(chan gc.Result, 10)
	defer close(output)
	// consume any errors sent to this channel
	go func() {
		for r := range output {
			log.Error(r.Error)
		}
	}()

	// This GC method returns adds the things that cannot be GC'ed (so all
	// the things that are pinned).
	gcs, err := gc.ColoredSet(ctx, pinner, dags, bestEffortRoots, output)
	if err != nil {
		return err
	}

	// We have everything. We send unswap requests for all these blocks,
	// when they are CIDv1s.
	err = gcs.ForEach(func(c cid.Cid) error {
		// CidV0s are always fine. We do not need to unswap them.
		if c.Version() == 0 {
			return nil
		}
		mhash := c.Hash()
		mhashKey := dshelp.MultihashToDsKey(mhash)
		cidKey := cidToDsKey(c)
		mhashPath := blocksPrefix.Child(mhashKey)
		cidPath := blocksPrefix.Child(cidKey)
		sw := Swap{Old: cidPath, New: mhashPath}
		unswapCh <- sw
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
