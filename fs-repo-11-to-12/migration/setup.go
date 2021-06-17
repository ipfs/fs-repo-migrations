package mg11

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/repolock"
	log "github.com/ipfs/fs-repo-migrations/stump"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	ipfspinner "github.com/ipfs/go-ipfs-pinner"
	"github.com/ipfs/go-ipfs-pinner/dspinner"
	loader "github.com/ipfs/go-ipfs/plugin/loader"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	format "github.com/ipfs/go-ipld-format"
)

// locks the repo
func (m *Migration) lock(opts migrate.Options) (io.Closer, error) {
	log.VLog("locking repo at %q", opts.Path)
	return lock.Lock2(opts.Path)
}

// this is just setup so that we can open the datastore.
// Plugins are loaded once only.
func (m *Migration) setupPlugins(opts migrate.Options) error {
	var err error
	var plugins *loader.PluginLoader
	m.loadPluginsOnce.Do(func() {
		log.VLog("  - loading repo configurations")
		plugins, err = loader.NewPluginLoader(opts.Path)
		if err != nil {
			err = fmt.Errorf("error loading plugins: %s", err)
			return
		}

		if err = plugins.Initialize(); err != nil {
			err = fmt.Errorf("error initializing plugins: %s", err)
			return
		}

		if err = plugins.Inject(); err != nil {
			err = fmt.Errorf("error injecting plugins: %s", err)
			return
		}
	})
	return err
}

// open the datastore and sets m.dstore. This opens it using whatever the
// user's IPFS configuration says that should be used. If we had a datastore,
// we close it and re-open it.
func (m *Migration) open(opts migrate.Options) error {
	if err := m.setupPlugins(opts); err != nil {
		return err
	}

	// Seems we opened it already. Close it and re-open it.
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

// Create a file to store the list of migrated CIDs. If it exists, it is
// opened for appending only.
func createBackupFile(path, name string) (*os.File, error) {
	backupPath := filepath.Join(path, name)
	log.VLog("  - backup file will be written to %s", backupPath)
	_, err := os.Stat(backupPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	// proceed without erroring when the file does not exist but warn
	// if it already exists.
	if err == nil {
		log.Log("WARN: backup file %s already exists. CIDs-Multihash pairs will be appended", backupPath)
	}

	// Open for appending or create it.
	return os.OpenFile(backupPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
}

// Open the backup file for reading.
func getBackupFile(backupPath string) (*os.File, error) {
	log.VLog("  - backup file will be read from %s", backupPath)
	return os.Open(backupPath)
}

// get a pinner so we can pin things upon revert. It uses ipfs-lite to wrap
// the migration datastore as a DAGService.
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

// get the current MFS root.
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
