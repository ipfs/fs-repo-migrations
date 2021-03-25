package mg1

import (
	"encoding/json"
	"fmt"
	"path"

	bst "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blocks/blockstore"
	bs "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blockservice"
	offline "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/exchange/offline"
	dag "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/merkledag"
	newpin "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/pin"
	u "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"
	dstore "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	flatfs "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/flatfs"
	leveldb "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/leveldb"
	mount "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/mount"
	dsq "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/query"
	sync "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/sync"
	context "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/golang.org/x/net/context"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"
)

var recursePinDatastoreKey = dstore.NewKey("/local/pins/recursive/keys")
var directPinDatastoreKey = dstore.NewKey("/local/pins/direct/keys")
var indirectPinDatastoreKey = dstore.NewKey("/local/pins/indirect/keys")

type Migration struct{}

func (m Migration) Versions() string {
	return "2-to-3"
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

	repo := mfsr.RepoPath(opts.Path)

	log.VLog("  - verifying version is '2'")
	if err := repo.CheckVersion("2"); err != nil {
		return err
	}

	if err := transferPins(opts.Path, opts.Verbose); err != nil {
		return err
	}

	log.Log("pin transfer completed successfuly")

	err = repo.WriteVersion("3")
	if err != nil {
		return err
	}
	log.Log("updated version file")

	log.Log("Migration 2 to 3 succeeded")
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
	if err := repo.CheckVersion("3"); err != nil {
		return err
	}

	if err := revertPins(opts.Path, opts.Verbose); err != nil {
		return err
	}

	// 3) change version number back down
	err = repo.WriteVersion("2")
	if err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Println("lowered version number to 2")
	}

	return nil
}

func openDatastore(repopath string) (dstore.ThreadSafeDatastore, error) {
	log.VLog("  - opening datastore at %q", repopath)
	ldbpath := path.Join(repopath, "datastore")
	ldb, err := leveldb.NewDatastore(ldbpath, nil)
	if err != nil {
		return nil, err
	}

	blockspath := path.Join(repopath, "blocks")
	fds, err := flatfs.New(blockspath, 4)
	if err != nil {
		return nil, err
	}

	return sync.MutexWrap(mount.New([]mount.Mount{
		{
			Prefix:    dstore.NewKey("/blocks"),
			Datastore: fds,
		},
		{
			Prefix:    dstore.NewKey("/"),
			Datastore: ldb,
		},
	})), nil
}

func constructDagServ(ds dstore.ThreadSafeDatastore) (dag.DAGService, error) {
	bstore := bst.NewBlockstore(ds)
	bserve, err := bs.New(bstore, offline.Exchange(bstore))
	if err != nil {
		return nil, err
	}
	return dag.NewDAGService(bserve), nil
}

func transferPins(repopath string, verbose bool) error {
	log.Log("beginning pin transfer")
	ds, err := openDatastore(repopath)
	if err != nil {
		return err
	}

	dserv, err := constructDagServ(ds)
	if err != nil {
		return err
	}

	pinner := newpin.NewPinner(ds, dserv)
	log.VLog("  - created version 3 pinner")

	log.VLog("  - loading recursive pins")
	recKeys, err := loadOldKeys(ds, recursePinDatastoreKey)
	if err != nil {
		return err
	}

	for _, k := range recKeys {
		pinner.PinWithMode(k, newpin.Recursive)
	}
	log.VLog("  - transfered recursive pins")

	log.VLog("  - loading direct pins")
	dirKeys, err := loadOldKeys(ds, directPinDatastoreKey)
	if err != nil {
		return err
	}

	for _, k := range dirKeys {
		pinner.PinWithMode(k, newpin.Direct)
	}
	log.VLog("  - transfered direct pins")

	err = pinner.Flush()
	if err != nil {
		return err
	}
	log.Log("pinner synced to disk")

	// ensure that the 'empty object' exists
	_, err = dserv.Add(new(dag.Node))
	if err != nil {
		return err
	}

	return cleanupOldPins(ds, verbose)
}

func cleanupOldPins(ds dstore.Datastore, verbose bool) error {
	log.Log("cleaning old pins")
	err := cleanupKeyspace(ds, recursePinDatastoreKey)
	if err != nil {
		return err
	}
	log.VLog("  - cleaned up oldstyle recursive pins")

	err = cleanupKeyspace(ds, directPinDatastoreKey)
	if err != nil {
		return err
	}
	log.VLog("  - cleaned up oldstyle direct pins")

	err = cleanupKeyspace(ds, indirectPinDatastoreKey)
	if err != nil {
		return err
	}
	log.VLog("  - cleaned up oldstyle indirect pins")

	return nil
}

func cleanupKeyspace(ds dstore.Datastore, k dstore.Key) error {
	log.VLog("  - deleting recursePin root key: %q", recursePinDatastoreKey)
	err := ds.Delete(recursePinDatastoreKey)
	if err != nil {
		return err
	}

	res, err := ds.Query(dsq.Query{
		Prefix:   recursePinDatastoreKey.String(),
		KeysOnly: true,
	})
	if err != nil {
		return err
	}
	for k := range res.Next() {
		log.VLog("  - deleting pin key: %q", k.Key)
		err := ds.Delete(dstore.NewKey(k.Key))
		if err != nil {
			res.Close()
			return err
		}
	}

	return nil
}

func revertPins(repopath string, verbose bool) error {
	log.VLog("  - reverting pins")
	ds, err := openDatastore(repopath)
	if err != nil {
		return err
	}

	log.VLog("  - construct dagservice")
	dserv, err := constructDagServ(ds)
	if err != nil {
		return err
	}

	log.VLog("  - load pinner")
	pinner, err := newpin.LoadPinner(ds, dserv)
	if err != nil {
		return err
	}

	log.VLog("  - write old recursive keys")
	if err := writeOldKeys(ds, recursePinDatastoreKey, pinner.RecursiveKeys()); err != nil {
		return err
	}

	log.VLog("  - write old direct keys")
	if err := writeOldKeys(ds, directPinDatastoreKey, pinner.DirectKeys()); err != nil {
		return err
	}

	log.VLog("  - write old indirect pins")
	ikeys, err := indirectPins(pinner, dserv)
	if err != nil {
		return err
	}

	if err := writeOldIndirPins(ds, indirectPinDatastoreKey, ikeys); err != nil {
		return err
	}

	return nil
}

func indirectPins(pinner newpin.Pinner, dserv dag.DAGService) (map[u.Key]int, error) {
	out := make(map[u.Key]int)
	var mapkeys func(nd *dag.Node) error
	mapkeys = func(nd *dag.Node) error {
		for _, lnk := range nd.Links {
			k := u.Key(lnk.Hash)
			out[k]++

			child, err := dserv.Get(context.Background(), k)
			if err != nil {
				return err
			}

			if err := mapkeys(child); err != nil {
				return err
			}
		}
		return nil
	}

	for _, k := range pinner.RecursiveKeys() {
		nd, err := dserv.Get(context.Background(), k)
		if err != nil {
			return nil, err
		}

		if err := mapkeys(nd); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func writeOldIndirPins(to dstore.Datastore, k dstore.Key, pins map[u.Key]int) error {
	fmt.Println("INDIRECT PINS: ", pins)
	refs := make(map[string]int)
	for k, v := range pins {
		refs[k.String()] = int(v)
	}

	b, err := json.Marshal(refs)
	if err != nil {
		return err
	}

	return to.Put(k, b)
}

func loadOldKeys(from dstore.Datastore, k dstore.Key) ([]u.Key, error) {
	v, err := from.Get(k)
	if err != nil {
		return nil, err
	}

	vb, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("pinset %s was not stored as []byte", v)
	}

	var keys []u.Key
	err = json.Unmarshal(vb, &keys)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func writeOldKeys(to dstore.Datastore, k dstore.Key, pins []u.Key) error {
	log.Log("writing keys: %q", k, pins)
	b, err := json.Marshal(pins)
	if err != nil {
		return err
	}

	return to.Put(k, b)
}
