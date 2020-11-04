package mg10

import (
	"fmt"
	"path"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/repolock"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
	log "github.com/ipfs/fs-repo-migrations/stump"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/mount"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/ipfs/go-ds-flatfs"
	"github.com/ipfs/go-ds-leveldb"
	"github.com/ipfs/go-filestore"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-ipfs-exchange-offline"
	"github.com/ipfs/go-ipfs-pinner/dspinner"
	"github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "10-to-11"
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

	if err = transferPins(opts.Path); err != nil {
		log.Error("failed to transfer pins:", err.Error())
		return err
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion("11")
	if err != nil {
		log.Error("failed to update version file to 11")
		return err
	}

	log.Log("updated version file")
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

	if err = revertPins(opts.Path); err != nil {
		return err
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion("10")
	if err != nil {
		log.Error("failed to update version file to 10")
		return err
	}

	log.Log("updated version file")
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

func makeStore(repopath string) (datastore.Datastore, format.DAGService, format.DAGService, error) {
	log.VLog("  - opening datastore at %q", repopath)
	ldbpath := path.Join(repopath, "datastore")
	ldb, err := leveldb.NewDatastore(ldbpath, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	blockspath := path.Join(repopath, "blocks")
	fds, err := flatfs.Open(blockspath, true)
	if err != nil {
		return nil, nil, nil, err
	}

	mdb := dssync.MutexWrap(mount.New([]mount.Mount{
		{
			Prefix:    datastore.NewKey("/blocks"),
			Datastore: fds,
		},
		{
			Prefix:    datastore.NewKey("/"),
			Datastore: ldb,
		},
	}))
	var dstore datastore.Batching
	dstore = &batchWrap{mdb}

	bstore := blockstore.NewBlockstore(dstore)
	bserv := blockservice.New(bstore, offline.Exchange(bstore))
	dserv := merkledag.NewDAGService(bserv)
	internalDag := merkledag.NewDAGService(bserv)

	syncFn := func() error {
		if err := dstore.Sync(blockstore.BlockPrefix); err != nil {
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

func transferPins(repopath string) error {
	log.Log("> Upgrading pinning to use datastore")

	dstore, dserv, internalDag, err := makeStore(repopath)
	if err != nil {
		return err
	}

	log.Log("importing from ipld pinner")

	_, impCount, err := dspinner.ImportFromIPLDPinner(dstore, dserv, internalDag)
	if err != nil {
		log.Error("failed to import pin data into datastore")
		return err
	}
	log.Log("imported %d pins from dag storage into datastore", impCount)
	return nil
}

func revertPins(repopath string) error {
	log.Log("> Reverting pinning to use DAG storage")

	dstore, dserv, internalDag, err := makeStore(repopath)
	if err != nil {
		return err
	}

	_, expCount, err := dspinner.ExportToIPLDPinner(dstore, dserv, internalDag)
	if err != nil {
		log.Error("failed to export pin data from datastore to ipld pinner")
		return err
	}
	log.Log("exported %d pins from datastore to dag storage", expCount)
	return nil
}
