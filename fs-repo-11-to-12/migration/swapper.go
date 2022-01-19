package mg11

import (
	"errors"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	log "github.com/ipfs/fs-repo-migrations/tools/stump"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
)

// SyncSize specifies how much we batch data before committing and syncing.
var SyncSize uint64 = 20 * 1024 * 1024 // 20MiB

// NWorkers sets the number of swapping threads to run when applying a
// migration.
var NWorkers int = 4

func init() {
	workerEnvVar := "IPFS_FS_MIGRATION_11_TO_12_NWORKERS"
	syncSizeEnvVar := "IPFS_FS_MIGRATION_11_TO_12_SYNC_SIZE_BYTES"
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

// Run lists all the keys in the datastore and triggers a swap operation for
// those corresponding to CIDv1s (replacing them by their raw multihash).
// When dryRun is true, it will not perform any changes, but notify SwapCh
// as if it would.
//
// Run returns the total number of keys swapped.
func (cswap *CidSwapper) Run(dryRun bool) (uint64, error) {
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
		return cswap.swapWorker(dryRun, resultsCh)
	}
	return cswap.runWorkers(NWorkers, swapWorkerFunc)
}

// Revert allows to undo any operations made by Run(). The given channel should
// receive Swap objects as they were sent by Run. It returns the number of
// swap operations performed.
func (cswap *CidSwapper) Revert(unswapCh <-chan Swap) (uint64, error) {
	swapWorkerFunc := func() (uint64, uint64) {
		return cswap.unswapWorker(unswapCh)
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

// swapWorkers reads query results from a channel and renames CIDv1 keys to
// raw multihashes by reading the blocks and storing them with the new
// key. Returns the number of keys swapped and the number of errors.
func (cswap *CidSwapper) swapWorker(dryRun bool, resultsCh <-chan query.Result) (uint64, uint64) {
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
			// complain if we find anything that is not a CID but
			// leave it as it is.  This can potentially be raw
			// multihashes that are not CIDv0s (i.e. using
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

		if dryRun {
			sw.swapped++ // otherwise this is done in sw.swap
		} else {
			// delete the old keys
			err = sw.swap(oldKey, newKey, false)
			if err != nil {
				log.Error("swapping %s for %s: %s", oldKey, newKey, err)
				errored++
				continue
			}
		}

		if cswap.SwapCh != nil {
			cswap.SwapCh <- Swap{Old: oldKey, New: newKey}
		}
	}

	if !dryRun {
		// final sync
		err := sw.syncAndDelete()
		if err != nil {
			log.Error("error performing last sync: %s", err)
			errored++
		}
		err = sw.sync() // sync deleted items
		if err != nil {
			log.Error("error performing last sync for deletions: %s", err)
			errored++
		}
	}

	return sw.swapped, errored
}

// unswap worker takes notifications from unswapCh (as they would be sent by
// the swapWorker) and undoes them. It ignores NotFound errors so that reverts
// can succeed even if they failed half-way.
func (cswap *CidSwapper) unswapWorker(unswapCh <-chan Swap) (uint64, uint64) {
	var errored uint64

	swker := &swapWorker{
		store:      cswap.Store,
		syncPrefix: cswap.Prefix,
	}

	// Process keys from the results channel
	for sw := range unswapCh {
		// During revert, we always keep the old keys.  This addresses
		// corner cases with keys that were referenced with both CIDv1
		// and CIDv0 and with pins added post migration.
		err := swker.swap(sw.New, sw.Old, true)

		// Did the user GC a migrated block?
		if err == ds.ErrNotFound {
			log.Error("could not revert %s->%s. Could not find %s. Was it GC'ed? This error is not fatal", sw.Old, sw.New, sw.New)
			continue
		} else if err != nil {
			log.Error("swapping %s for %s: %s", sw.New, sw.Old, err)
			errored++
			continue
		}
		if cswap.SwapCh != nil {
			cswap.SwapCh <- Swap{Old: sw.New, New: sw.Old}
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
	log.Log("Migration worker syncing after %d objects migrated", sw.swapped)
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
