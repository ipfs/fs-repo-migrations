// package mg11 contains the code to perform 11-12 repository migration in
// go-ipfs. This performs a switch to raw multihashes for all keys in the
// go-ipfs datastore (https://github.com/ipfs/go-ipfs/issues/6815).
package mg11

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/ipfs/fs-repo-migrations/stump"
	format "github.com/ipfs/go-ipld-format"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/repolock"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	filestore "github.com/ipfs/go-filestore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	ipfspinner "github.com/ipfs/go-ipfs-pinner"
	dspinner "github.com/ipfs/go-ipfs-pinner/dspinner"
	gc "github.com/ipfs/go-ipfs/gc"
	loader "github.com/ipfs/go-ipfs/plugin/loader"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
)

const backupFile = "11-to-12-cids.txt"

var mfsRootKey = ds.NewKey("/local/filesroot")

var blocksPrefix = ds.NewKey("/blocks")

var filestorePrefix = filestore.FilestorePrefix

var migrationPrefixes = []ds.Key{
	blocksPrefix,
	filestorePrefix,
}

// Migration implements the migration described above.
type Migration struct {
	plugins *loader.PluginLoader
	dstore  ds.Batching
}

// Versions returns the current version string for this migration.
func (m *Migration) Versions() string {
	return "11-to-12"
}

// Reversible returns true.
func (m *Migration) Reversible() bool {
	return true
}

// lock the repo
func (m *Migration) lock(opts migrate.Options) (io.Closer, error) {
	log.VLog("locking repo at %q", opts.Path)
	return lock.Lock2(opts.Path)
}

func (m *Migration) setupPlugins(opts migrate.Options) error {
	if m.plugins != nil {
		return nil
	}

	log.VLog("  - loading repo configurations")
	plugins, err := loader.NewPluginLoader(opts.Path)
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}
	m.plugins = plugins

	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error injecting plugins: %s", err)
	}

	return nil
}

// open the datastore
func (m *Migration) open(opts migrate.Options) error {
	if err := m.setupPlugins(opts); err != nil {
		return err
	}

	// assume already opened. We cannot initalize plugins twice.
	if m.dstore != nil {
		m.dstore.Close()
	}

	cfg, err := fsrepo.ConfigAt(opts.Path)
	if err != nil {
		return err
	}

	dsc, err := fsrepo.AnyDatastoreConfig(cfg.Datastore.Spec)
	if err != nil {
		return err
	}

	dstore, err := dsc.Create(opts.Path)
	if err != nil {
		return err
	}
	m.dstore = dstore
	return nil
}

// Apply runs the migration and writes a log file that can be used by Revert.
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

	// Prepare backing up of CIDs
	backupPath := filepath.Join(opts.Path, backupFile)
	log.VLog("  - backup file will be written to %s", backupPath)
	_, err = os.Stat(backupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error(err)
			return err
		}
	} else { // backup file exists
		log.Log("WARN: backup file %s already exists. CIDs-Multihash pairs will be appended", backupPath)
	}

	// If it exists, append to it.
	f, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Error(err)
		return err
	}
	defer f.Close()
	buf := bufio.NewWriter(f)

	swapCh := make(chan Swap, 1000)

	writingDone := make(chan struct{})
	go func() {
		for sw := range swapCh {
			// Only write the Old string (a CID). We can derive
			// the multihash from it.
			fmt.Fprint(buf, sw.Old.String(), "\n")
		}
		close(writingDone)
	}()

	// Add all the keys to migrate to the backup file
	for _, prefix := range migrationPrefixes {
		log.VLog("  - Adding keys in prefix %s to backup file", prefix)
		cidSwapper := CidSwapper{Prefix: prefix, Store: m.dstore, SwapCh: swapCh}
		total, err := cidSwapper.Run(true) // DRY RUN
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

	// The backup file is ready. Run the migration.
	for _, prefix := range migrationPrefixes {
		log.VLog("  - Migrating keys in prefix %s", prefix)
		cidSwapper := CidSwapper{Prefix: prefix, Store: m.dstore}
		total, err := cidSwapper.Run(false) // NOT a Dry Run
		if err != nil {
			log.Error(err)
			return err
		}
		log.Log("%d CIDv1 keys in %s have been migrated", total, prefix)
	}

	if err := repo.WriteVersion("12"); err != nil {
		log.Error("failed to write version file")
		return err
	}
	log.Log("updated version file")

	return nil
}

// Revert attempts to undo the migration using the log file written by Apply.
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
	log.VLog("  - backup file will be read from %s", backupPath)
	f, err := os.Open(backupPath)
	if err != nil {
		log.Error(err)
		return err
	}

	unswapCh := make(chan Swap, 1000)
	scanner := bufio.NewScanner(f)
	var scannerErr error

	go func() {
		defer close(unswapCh)

		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			}
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
			// This is the original swap object which is what we
			// wanted to rebuild. Old is the old path and new is
			// the new path and the unswapper will revert this.
			sw := Swap{Old: cidPath, New: mhashPath}
			unswapCh <- sw
		}
		if err := scanner.Err(); err != nil {
			log.Error(err)
			return
		}

		// Ensure all pins/MFS which may have happened post-migration
		// are reverted.
		if err := walkPinsAndMFS(unswapCh, m.dstore); err != nil {
			log.Error(err)
			return
		}

	}()

	// The backup file contains prefixed keys, so we do not need to set
	// them.
	cidSwapper := CidSwapper{Store: m.dstore}
	total, err := cidSwapper.Revert(unswapCh)
	if err != nil {
		log.Error(err)
		return err
	}
	// Revert will only return after unswapCh is closed, so we know
	// scannerErr is safe to read at this point.
	if scannerErr != nil {
		return err
	}

	log.Log("%d multihashes reverted to CidV1s", total)
	if err := repo.WriteVersion("11"); err != nil {
		log.Error("failed to write version file")
		return err
	}

	log.Log("reverted version file to version 11")
	err = f.Close()
	if err != nil {
		log.Error("could not close backup file")
		return err
	}
	err = os.Rename(backupPath, backupPath+".reverted")
	if err != nil {
		log.Error("could not rename the backup file, but migration worked: %s", err)
		return err
	}
	return nil
}

func getPinner(ctx context.Context, dstore ds.Batching) (ipfspinner.Pinner, format.DAGService, error) {
	// Wrapping a datastore all the way up to a DagService.
	// This is the shortest way.
	dags, err := ipfslite.New(
		ctx,
		dstore,
		nil,
		nil,
		&ipfslite.Config{
			Offline: true,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	// Get a pinner.
	pinner, err := dspinner.New(ctx, dstore, dags)
	if err != nil {
		return nil, nil, err
	}
	return pinner, dags, nil
}

func getMFSRoot(dstore ds.Batching) (cid.Cid, error) {
	// Find the MFS root.
	mfsRoot, err := dstore.Get(mfsRootKey)
	if err != nil {
		return cid.Undef, err
	}
	c, err := cid.Cast(mfsRoot)
	if err != nil {
		log.Error(err)
		return cid.Undef, err
	}
	return c, nil
}

func walkPinsAndMFS(unswapCh chan Swap, dstore ds.Batching) error {
	// The easiest way to get a dag service that we can use with the
	// pinner on top of the datastore we opened.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var bestEffortRoots []cid.Cid

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

	// Get a pinner.
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

	// Obtain the total set of CIDs that we need to make sure are
	// not
	gcs, err := gc.ColoredSet(ctx, pinner, dags, bestEffortRoots, output)
	if err != nil {
		return err
	}

	// We have everything. We send unswap requests
	// for all these blocks.
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
