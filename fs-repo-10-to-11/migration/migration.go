package mg10

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-filestore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	"github.com/ipfs/go-ipfs-pinner/pinconv"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
)

type Migration struct{}

var verbose bool

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
}

func (m Migration) Versions() string {
	return "10-to-11"
}

func (m Migration) Reversible() bool {
	return true
}

func (m Migration) Apply(opts migrate.Options) error {
	const (
		fromVer = 10
		toVer   = 11
	)
	verbose = opts.Verbose
	log.Printf("applying %s repo migration", m.Versions())

	ver, err := migrations.RepoVersion(opts.Path)
	if err != nil {
		return err
	}

	if ver != fromVer {
		return fmt.Errorf("versions differ (expected: %d, actual: %d)", fromVer, ver)
	}

	if err = setupPlugins(opts.Path); err != nil {
		return fmt.Errorf("failed to setup plugins: %v", err)
	}

	// Set to previous version to avoid "needs migration" error.  This is safe
	// for this migration since repo has not changed.
	fsrepo.RepoVersion = 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !fsrepo.IsInitialized(opts.Path) {
		return fmt.Errorf("ipfs repo %q not initialized", opts.Path)
	}

	if verbose {
		log.Printf("opening datastore at %q", opts.Path)
	}
	r, err := fsrepo.Open(opts.Path)
	if err != nil {
		return fmt.Errorf("cannot open datastore: %v", err)
	}
	defer r.Close()

	if err = transferPins(ctx, r); err != nil {
		return fmt.Errorf("failed to transfer pins: %v", err)
	}

	err = migrations.WriteRepoVersion(opts.Path, toVer)
	if err != nil {
		return fmt.Errorf("failed to update version file to %d: %v", toVer, err)
	}

	log.Print("updated version file")
	log.Printf("Migration %d to %d succeeded", fromVer, toVer)
	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	verbose = opts.Verbose
	log.Print("reverting migration")

	err := setupPlugins(opts.Path)
	if err != nil {
		return fmt.Errorf("failed to setup plugins: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !fsrepo.IsInitialized(opts.Path) {
		return fmt.Errorf("ipfs repo %q not initialized", opts.Path)
	}

	if verbose {
		log.Printf("opening datastore at %q", opts.Path)
	}
	r, err := fsrepo.Open(opts.Path)
	if err != nil {
		return fmt.Errorf("cannot open datastore: %v", err)
	}
	defer r.Close()

	if err = revertPins(ctx, r); err != nil {
		return err
	}

	err = migrations.WriteRepoVersion(opts.Path, 10)
	if err != nil {
		return fmt.Errorf("failed to update version file to 10: %v", err)
	}

	log.Print("updated version file")
	return nil
}

type syncDagService struct {
	format.DAGService
	syncFn func() error
}

func (s *syncDagService) Sync() error {
	return s.syncFn()
}

type batchWrap struct {
	datastore.Datastore
}

func (d *batchWrap) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(d), nil
}

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(path.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func makeStore(r repo.Repo) (datastore.Datastore, format.DAGService, format.DAGService, error) {
	dstr := r.Datastore()
	dstore := &batchWrap{dstr}

	bstore := blockstore.NewBlockstore(dstr)
	bserv := blockservice.New(bstore, offline.Exchange(bstore))
	dserv := merkledag.NewDAGService(bserv)
	internalDag := merkledag.NewDAGService(bserv)

	syncFn := func() error {
		err := dstore.Sync(blockstore.BlockPrefix)
		if err != nil {
			return fmt.Errorf("cannot sync blockstore: %v", err)
		}
		err = dstore.Sync(filestore.FilestorePrefix)
		if err != nil {
			return fmt.Errorf("cannot sync filestore: %v", err)
		}
		return nil
	}
	syncDs := &syncDagService{dserv, syncFn}
	syncInternalDag := &syncDagService{internalDag, syncFn}

	return dstore, syncDs, syncInternalDag, nil
}

func transferPins(ctx context.Context, r repo.Repo) error {
	log.Print("upgrading pinning to use datastore")

	dstore, dserv, internalDag, err := makeStore(r)
	if err != nil {
		return err
	}

	_, toDSCount, err := pinconv.ConvertPinsFromIPLDToDS(ctx, dstore, dserv, internalDag)
	if err != nil {
		return errors.New("failed to convert ipld pin data into datastore")
	}
	log.Printf("converted %d pins from ipld storage into datastore", toDSCount)
	return nil
}

func revertPins(ctx context.Context, r repo.Repo) error {
	log.Print("reverting pinning to use ipld storage")

	dstore, dserv, internalDag, err := makeStore(r)
	if err != nil {
		return err
	}

	_, toIPLDCount, err := pinconv.ConvertPinsFromDSToIPLD(ctx, dstore, dserv, internalDag)
	if err != nil {
		return errors.New("failed to convert pin data from datastore to ipld pinner")
	}
	log.Printf("converted %d pins from datastore to ipld storage", toIPLDCount)
	return nil
}
