// Package dsindex provides secondary indexing functionality for a datastore.
package dsindex

import (
	"path"

	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
)

// Indexer maintains a secondary index.  Each value of the secondary index maps
// to one more primary keys.
type Indexer interface {
	// Add adds the a to the an index
	Add(index, id string) error

	// Delete deletes the specified key from the index.  If the key is not in
	// the datastore, this method returns no error.
	Delete(index, id string) error

	// DeleteAll deletes all keys in the given index.  If a key is not in the
	// datastore, this method returns no error.
	DeleteAll(index string) (count int, err error)

	// ForEach calls the function for each key in the specified index, until
	// there are no more keys, or until the function returns false.  If index
	// is empty string, then all index names are iterated.
	ForEach(index string, fn func(index, id string) bool) error

	// HasKey determines the specified index contains the specified primary key
	HasKey(index, id string) (bool, error)

	// HasAny determines if any key is in the specified index.  If index is
	// empty string, then all indexes are searched.
	HasAny(index string) (bool, error)

	// Search returns all keys for the given index
	Search(index string) (ids []string, err error)

	// Synchronize the indexes in this Indexer to match those of the given
	// Indexer. The indexPath prefix is not synchronized, only the index/key
	// portion of the indexes.
	SyncTo(reference Indexer) (changed bool, err error)
}

// indexer is a simple implementation of Indexer.  This implementation relies
// on the underlying data store supporting efficent querying by prefix.
//
// TODO: Consider adding caching
type indexer struct {
	dstore    ds.Datastore
	indexPath string
}

// New creates a new datastore index.  All indexes are stored prefixed with the
// specified index path.
func New(dstore ds.Datastore, indexPath string) Indexer {
	return &indexer{
		dstore:    dstore,
		indexPath: indexPath,
	}
}

func (x *indexer) Add(index, id string) error {
	key := ds.NewKey(path.Join(x.indexPath, index, id))
	return x.dstore.Put(key, []byte{})
}

func (x *indexer) Delete(index, id string) error {
	return x.dstore.Delete(ds.NewKey(path.Join(x.indexPath, index, id)))
}

func (x *indexer) DeleteAll(index string) (int, error) {
	ents, err := x.queryPrefix(path.Join(x.indexPath, index))
	if err != nil {
		return 0, err
	}

	for i := range ents {
		err = x.dstore.Delete(ds.NewKey(ents[i].Key))
		if err != nil {
			return 0, err
		}
	}

	return len(ents), nil
}

func (x *indexer) ForEach(index string, fn func(idx, id string) bool) error {
	q := query.Query{
		Prefix:   path.Join(x.indexPath, index),
		KeysOnly: true,
	}
	results, err := x.dstore.Query(q)
	if err != nil {
		return err
	}

	for {
		r, ok := results.NextSync()
		if !ok {
			break
		}
		if r.Error != nil {
			err = r.Error
			break
		}

		ent := r.Entry
		if !fn(path.Base(path.Dir(ent.Key)), path.Base(ent.Key)) {
			break
		}
	}
	results.Close()

	return err
}

func (x *indexer) HasKey(index, id string) (bool, error) {
	return x.dstore.Has(ds.NewKey(path.Join(x.indexPath, index, id)))
}

func (x *indexer) HasAny(index string) (bool, error) {
	var any bool
	err := x.ForEach(index, func(idx, id string) bool {
		any = true
		return false
	})
	return any, err
}

func (x *indexer) Search(index string) ([]string, error) {
	ents, err := x.queryPrefix(path.Join(x.indexPath, index))
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, nil
	}

	ids := make([]string, len(ents))
	for i := range ents {
		ids[i] = path.Base(ents[i].Key)
	}
	return ids, nil
}

func (x *indexer) SyncTo(ref Indexer) (bool, error) {
	// Build reference index map
	refs := map[string]string{}
	err := ref.ForEach("", func(idx, id string) bool {
		refs[id] = idx
		return true
	})
	if err != nil {
		return false, err
	}
	if len(refs) == 0 {
		return false, nil
	}

	// Compare current indexes
	var delKeys []string
	err = x.ForEach("", func(idx, id string) bool {
		refIdx, ok := refs[id]
		if ok && refIdx == idx {
			// same in both; delete from refs, do not add to delKeys
			delete(refs, id)
		} else {
			delKeys = append(delKeys, path.Join(x.indexPath, idx, id))
		}
		return true
	})
	if err != nil {
		return false, err
	}

	// Items in delKeys are indexes that no longer exist
	for i := range delKeys {
		err = x.dstore.Delete(ds.NewKey(delKeys[i]))
		if err != nil {
			return false, err
		}
	}

	// What remains in refs are indexes that need to be added
	for k, v := range refs {
		err = x.dstore.Put(ds.NewKey(path.Join(x.indexPath, v, k)), nil)
		if err != nil {
			return false, err
		}
	}

	return len(refs) != 0 || len(delKeys) != 0, nil
}

func (x *indexer) queryPrefix(prefix string) ([]query.Entry, error) {
	q := query.Query{
		Prefix:   prefix,
		KeysOnly: true,
	}
	results, err := x.dstore.Query(q)
	if err != nil {
		return nil, err
	}
	return results.Rest()
}
