package sync

import (
	"fmt"
	"os"

	ds "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	dsq "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/query"
)

type datastore struct {
	child ds.Datastore
}

// Wrap shims a datastore such than _any_ operation failing triggers a panic
// This is useful for debugging invariants.
func Wrap(d ds.Datastore) ds.Shim {
	return &datastore{child: d}
}

func (d *datastore) Children() []ds.Datastore {
	return []ds.Datastore{d.child}
}

func (d *datastore) Put(key ds.Key, value interface{}) error {
	err := d.child.Put(key, value)
	if err != nil {
		fmt.Fprintf(os.Stdout, "panic datastore: %s", err)
		panic("panic datastore: Put failed")
	}
	return nil
}

func (d *datastore) Get(key ds.Key) (interface{}, error) {
	val, err := d.child.Get(key)
	if err != nil {
		fmt.Fprintf(os.Stdout, "panic datastore: %s", err)
		panic("panic datastore: Get failed")
	}
	return val, nil
}

func (d *datastore) Has(key ds.Key) (bool, error) {
	e, err := d.child.Has(key)
	if err != nil {
		fmt.Fprintf(os.Stdout, "panic datastore: %s", err)
		panic("panic datastore: Has failed")
	}
	return e, nil
}

func (d *datastore) Delete(key ds.Key) error {
	err := d.child.Delete(key)
	if err != nil {
		fmt.Fprintf(os.Stdout, "panic datastore: %s", err)
		panic("panic datastore: Delete failed")
	}
	return nil
}

func (d *datastore) Query(q dsq.Query) (dsq.Results, error) {
	r, err := d.child.Query(q)
	if err != nil {
		fmt.Fprintf(os.Stdout, "panic datastore: %s", err)
		panic("panic datastore: Query failed")
	}
	return r, nil
}
