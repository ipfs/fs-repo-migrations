package mg11

import (
	"errors"
	flatfs "github.com/ipfs/go-ds-flatfs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/ipfs/fs-repo-migrations/tools/stump"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	query "github.com/ipfs/go-datastore/query"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
)

// SyncSize specifies how much we batch data before committing and syncing.
var SyncSize uint64 = 100 * 1024 * 1024 // 100MiB

// NWorkers sets the number of swapping threads to run when applying a
// migration.
var NWorkers int = 1

var EnableFlatFSFastPath bool = true

func init() {
	workerEnvVar := "IPFS_FS_MIGRATION_11_TO_12_NWORKERS"
	syncSizeEnvVar := "IPFS_FS_MIGRATION_11_TO_12_SYNC_SIZE_BYTES"
	flatfsFastPathEnvVar := "IPFS_FS_MIGRATION_11_TO_12_ENABLE_FLATFS_FASTPATH"
	if nworkersStr, nworkerInEnv := os.LookupEnv(workerEnvVar); nworkerInEnv {
		nworkers, err := strconv.Atoi(nworkersStr)
		if err != nil {
			panic(err)
		}
		if nworkers < 1 {
			panic("number of workers must be at least 1")
		}
		NWorkers = nworkers
	}

	if syncSizeStr, syncSizeInEnv := os.LookupEnv(syncSizeEnvVar); syncSizeInEnv {
		syncSize, err := strconv.ParseUint(syncSizeStr, 10, 64)
		if err != nil {
			panic(err)
		}
		if syncSize < 1 {
			panic("sync size bytes must be at least 1")
		}
		SyncSize = syncSize
	}

	if flatfsFastPathStr, flatfsFastPathInEnv := os.LookupEnv(flatfsFastPathEnvVar); flatfsFastPathInEnv {
		enableFlatfsFastPath, err := strconv.ParseBool(flatfsFastPathStr)
		if err != nil {
			panic(err)
		}
		EnableFlatFSFastPath = enableFlatfsFastPath
	}
}

// Swap holds the datastore keys for the original CID and for the
// destination Multihash.
type Swap struct {
	Old ds.Key
	New ds.Key
}

// CidSwapper reads all the keys in a datastore and replaces
// them with their raw multihash.
type CidSwapper struct {
	Prefix ds.Key      // A prefix/namespace to limit the query.
	Store  ds.Batching // the datastore to migrate.
	SwapCh chan Swap   // a channel that gets notified for every swap
}

// Prepare performs a dry run without copying anything but notifying SwapCh
// as it runs.
//
// Retruns the total number of keys swapped.
func (cswap *CidSwapper) Prepare() (uint64, error) {
	// Query all keys. We will loop all keys
	// and swap those that can be parsed as CIDv1.
	queryAll := query.Query{
		Prefix:   cswap.Prefix.String(),
		KeysOnly: true,
	}

	results, err := cswap.Store.Query(queryAll)
	if err != nil {
		return 0, err
	}
	defer results.Close()
	resultsCh := results.Next()
	swapWorkerFunc := func() (uint64, uint64) {
		return cswap.prepareWorker(resultsCh) // dry-run=true
	}
	return cswap.runWorkers(NWorkers, swapWorkerFunc)
}

// Run performs a migration reading the Swaps that need to be performed
// from the given Swap channel. The swaps can be obtained with Prepare().
//
// Run returns the total number of keys swapped.
func (cswap *CidSwapper) Run(swapCh <-chan Swap) (uint64, error) {
	swapWorkerFunc := func() (uint64, uint64) {
		return cswap.swapWorker(swapCh, false) // reverting=false
	}
	return cswap.runWorkers(NWorkers, swapWorkerFunc)
}

// Revert allows to undo any operations made by Run(). The given channel should
// receive Swap objects as they were sent by Run. It returns the number of
// swap operations performed.
func (cswap *CidSwapper) Revert(unswapCh <-chan Swap) (uint64, error) {
	swapWorkerFunc := func() (uint64, uint64) {
		return cswap.swapWorker(unswapCh, true) // reverting=true
	}
	return cswap.runWorkers(NWorkers, swapWorkerFunc)
}

// Run workers launches several workers to run the given function which returns
// number of swapped items and number of errors.
func (cswap *CidSwapper) runWorkers(nWorkers int, f func() (uint64, uint64)) (uint64, error) {
	var total uint64
	var nErrors uint64
	var wg sync.WaitGroup
	wg.Add(nWorkers)
	for i := 0; i < nWorkers; i++ {
		go func() {
			defer wg.Done()
			n, e := f()
			atomic.AddUint64(&total, n)
			atomic.AddUint64(&nErrors, e)
		}()
	}
	wg.Wait()
	if nErrors > 0 {
		return total, errors.New("errors happened during the migration. Consider running it again")
	}
	return total, nil
}

// prepareWorker reads query results from a channel and renames CIDv1 keys to
// raw multihashes by reading the blocks and storing them with the new
// key. Returns the number of keys swapped and the number of errors.
func (cswap *CidSwapper) prepareWorker(resultsCh <-chan query.Result) (uint64, uint64) {
	var errored uint64

	sw := &swapWorker{
		store:      cswap.Store,
		syncPrefix: cswap.Prefix,
	}

	// Process keys from the results channel
	for res := range resultsCh {
		if res.Error != nil {
			log.Error(res.Error)
			errored++
			continue
		}

		oldKey := ds.NewKey(res.Key)
		c, err := dsKeyToCid(ds.NewKey(oldKey.BaseNamespace())) // remove prefix
		if err != nil {
			// This means we have found something that is not a
			// CID. We leave it as it is. This can potentially be
			// raw multihashes that are not CIDv0s (i.e. using
			// anything other than sha256). They may come from a
			// previous migration.
			log.VLog("could not parse %s as a Cid", oldKey)
			continue
		}
		if c.Version() == 0 { // CidV0 are multihashes, leave them.
			continue
		}

		// Cid Version > 0
		mh := c.Hash()
		// /path/to/old/<cid> -> /path/to/old/<multihash>
		newKey := oldKey.Parent().Child(dshelp.MultihashToDsKey(mh))

		sw.swapped++
		if cswap.SwapCh != nil {
			cswap.SwapCh <- Swap{Old: oldKey, New: newKey}
		}
	}

	return sw.swapped, errored
}

func (cswap *CidSwapper) swapWorkerFlatFS(fsdsPath string, fsdsShard *flatfs.ShardIdV1, swapCh <-chan Swap, reverting bool) (uint64, uint64) {
	var swapped, errored uint64

	const flatfsExtension = ".data"
	prefix := ktds.PrefixTransform{Prefix: blocksPrefix}

	getPath := func(basePath string, key ds.Key) (string, string) {
		child := prefix.InvertKey(key)
		noslash := child.String()[1:]
		dir := filepath.Join(fsdsPath, fsdsShard.Func()(noslash))
		file := filepath.Join(dir, noslash+flatfsExtension)

		return dir, file
	}

	genericSwker := &swapWorker{
		store:      cswap.Store,
		syncPrefix: cswap.Prefix,
	}

	// the frequency with which we log flatfs moves
	const swapLogThreshold = 10000

	// Process keys from the results channel
	for sw := range swapCh {
		if reverting {
			old := sw.Old
			sw.Old = sw.New
			sw.New = old
		}

		if !sw.Old.Parent().Equal(sw.New.Parent()) {
			log.Error("could not swap %s->%s. The namespaces changed. Skipping.", sw.Old, sw.New)
			errored++
			continue
		}

		if !sw.Old.Parent().Equal(blocksPrefix) {
			err := genericSwker.swap(sw.Old, sw.New, reverting)

			// The datastore does not have the block we are planning to
			// migrate.
			if err == ds.ErrNotFound {
				log.Error("could not swap %s->%s. Could not find %s even though it was in the backup file. Skipping.", sw.Old, sw.New, sw.Old)
				continue
			} else if err != nil {
				log.Error("swapping %s->%s: %s", sw.Old, sw.New, err)
				errored++
				continue
			}

			if cswap.SwapCh != nil {
				cswap.SwapCh <- Swap{Old: sw.Old, New: sw.New}
			}
			continue
		}

		_, oldPath := getPath(fsdsPath, sw.Old)
		newDir, newPath := getPath(fsdsPath, sw.New)

		_, err := os.Stat(oldPath)
		if err != nil {
			log.Error("could not swap %s->%s. Could not find %s even though it was in the backup file %s. Skipping.", sw.Old, sw.New, sw.Old, err.Error())
			continue
		}

		if err := os.Mkdir(newDir, 0755); err != nil && !os.IsExist(err) {
			log.Error("could not swap %s->%s. Skipping.", sw.Old, sw.New, err.Error())
			continue
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			log.Error("could not swap %s->%s. Skipping.", sw.Old, sw.New, err.Error())
			errored++
			continue
		}
		swapped++

		if swapped%swapLogThreshold == 0 {
			log.Log("%v: Migration worker has moved %d flatfs files and %d in total", time.Now(), swapLogThreshold, swapped)
		}

		if cswap.SwapCh != nil {
			cswap.SwapCh <- Swap{Old: sw.Old, New: sw.New}
		}
	}

	// log the leftover flatfs moves that were not already logged
	if rem := swapped % swapLogThreshold; rem != 0 {
		log.Log("%v: Migration worker has moved %d flatfs files and %d in total", time.Now(), rem, swapped)
	}

	// handle generic worker sync
	// final sync to added things
	err := genericSwker.syncAndDelete()
	if err != nil {
		log.Error("error performing last sync: %s", err)
		errored++
	}
	err = genericSwker.sync() // final sync for deletes.
	if err != nil {
		log.Error("error performing last sync for deletions: %s", err)
		errored++
	}

	return genericSwker.swapped + swapped, errored
}

// unswap worker takes notifications from unswapCh (as they would be sent by
// the swapWorker) and undoes them. It ignores NotFound errors so that reverts
// can succeed even if they failed half-way.
func (cswap *CidSwapper) swapWorker(swapCh <-chan Swap, reverting bool) (uint64, uint64) {
	// Use the more generic datastore swapper if the FlatFS fast path has been explicitly disabled
	// Also use it for reversion since the FlatFS specific code doesn't specifically
	// handle some reversion edge cases.
	if !EnableFlatFSFastPath || reverting {
		return cswap.swapWorkerDS(swapCh, reverting)
	}

	// Use the more generic datastore swapper if not using a simple FlatFS setup.
	fsdsPath, fsDsShard, err := IsBasicFlatFSBlockstore(cswap.Store)
	if err != nil {
		return cswap.swapWorkerDS(swapCh, reverting)
	}

	return cswap.swapWorkerFlatFS(fsdsPath, fsDsShard, swapCh, reverting)
}

func (cswap *CidSwapper) swapWorkerDS(swapCh <-chan Swap, reverting bool) (uint64, uint64) {
	var errored uint64

	swker := &swapWorker{
		store:      cswap.Store,
		syncPrefix: cswap.Prefix,
	}

	// Process keys from the results channel
	for sw := range swapCh {
		if reverting {
			old := sw.Old
			sw.Old = sw.New
			sw.New = old
		}
		err := swker.swap(sw.Old, sw.New, reverting)

		// The datastore does not have the block we are planning to
		// migrate.
		if err == ds.ErrNotFound {
			log.Error("could not swap %s->%s. Could not find %s even though it was in the backup file. Skipping.", sw.Old, sw.New, sw.Old)
			continue
		} else if err != nil {
			log.Error("swapping %s->%s: %s", sw.Old, sw.New, err)
			errored++
			continue
		}
		if cswap.SwapCh != nil {
			cswap.SwapCh <- Swap{Old: sw.Old, New: sw.New}
		}
	}

	// final sync to added things
	err := swker.syncAndDelete()
	if err != nil {
		log.Error("error performing last sync: %s", err)
		errored++
	}
	err = swker.sync() // final sync for deletes.
	if err != nil {
		log.Error("error performing last sync for deletions: %s", err)
		errored++
	}

	return swker.swapped, errored
}

// swapWorker swaps old keys for new keys, syncing to disk regularly
// and notifying swapCh of the changes.
type swapWorker struct {
	swapped     uint64
	curSyncSize uint64

	store      ds.Batching
	syncPrefix ds.Key

	toDelete []ds.Key
}

// swap replaces old keys with new ones. It Syncs() when the
// number of items written reaches SyncSize. Upon that it proceeds
// to delete the old items.
func (sw *swapWorker) swap(old, new ds.Key, keepOld bool) error {
	v, err := sw.store.Get(old)
	vLen := uint64(len(v))
	if err != nil {
		return err
	}
	if err := sw.store.Put(new, v); err != nil {
		return err
	}

	// Sometimes we want to keep multihashes (CidV0s) on revert.
	if !keepOld {
		sw.toDelete = append(sw.toDelete, old)
	}

	sw.swapped++
	sw.curSyncSize += vLen

	// We have copied about 10MB
	if sw.curSyncSize >= SyncSize {
		sw.curSyncSize = 0
		err = sw.syncAndDelete()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sw *swapWorker) syncAndDelete() error {
	err := sw.sync()
	if err != nil {
		return err
	}

	// Delete all the old keys
	for _, o := range sw.toDelete {
		if err := sw.store.Delete(o); err != nil {
			return err
		}
	}
	sw.toDelete = nil
	return nil
}

func (sw *swapWorker) sync() error {
	log.Log("%v: Generic migration worker syncing after %d objects migrated", time.Now(), sw.swapped)
	err := sw.store.Sync(sw.syncPrefix)
	if err != nil {
		return err
	}
	return nil
}

// Copied from go-ipfs-ds-help as that one is gone.
func dsKeyToCid(dsKey ds.Key) (cid.Cid, error) {
	kb, err := dshelp.BinaryFromDsKey(dsKey)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.Cast(kb)
}

// Copied from go-ipfs-ds-help as that one is gone.
func cidToDsKey(k cid.Cid) ds.Key {
	return dshelp.NewKeyFromBinary(k.Bytes())
}
