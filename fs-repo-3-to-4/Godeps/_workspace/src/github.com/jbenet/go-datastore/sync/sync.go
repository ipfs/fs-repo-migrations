package sync

import (
	"sync"

	ds "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	dsq "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore/query"
)

// MutexDatastore contains a child datastire and a mutex.
// used for coarse sync
type MutexDatastore struct {
	sync.RWMutex

	child ds.Datastore
}

// MutexWrap constructs a datastore with a coarse lock around
// the entire datastore, for every single operation
func MutexWrap(d ds.Datastore) ds.ThreadSafeDatastore {
	return &MutexDatastore{child: d}
}

// Children implements Shim
func (d *MutexDatastore) Children() []ds.Datastore {
	return []ds.Datastore{d.child}
}

// IsThreadSafe implements ThreadSafeDatastore
func (d *MutexDatastore) IsThreadSafe() {}

// Put implements Datastore.Put
func (d *MutexDatastore) Put(key ds.Key, value interface{}) (err error) {
	d.Lock()
	defer d.Unlock()
	return d.child.Put(key, value)
}

// Get implements Datastore.Get
func (d *MutexDatastore) Get(key ds.Key) (value interface{}, err error) {
	d.RLock()
	defer d.RUnlock()
	return d.child.Get(key)
}

// Has implements Datastore.Has
func (d *MutexDatastore) Has(key ds.Key) (exists bool, err error) {
	d.RLock()
	defer d.RUnlock()
	return d.child.Has(key)
}

// Delete implements Datastore.Delete
func (d *MutexDatastore) Delete(key ds.Key) (err error) {
	d.Lock()
	defer d.Unlock()
	return d.child.Delete(key)
}

// KeyList implements Datastore.KeyList
func (d *MutexDatastore) Query(q dsq.Query) (dsq.Results, error) {
	d.RLock()
	defer d.RUnlock()
	return d.child.Query(q)
}
