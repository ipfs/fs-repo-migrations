package blockstore

import (
	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/hashicorp/golang-lru"
	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blocks"
	u "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"
	context "github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/golang.org/x/net/context"
)

// WriteCached returns a blockstore that caches up to |size| unique writes (bs.Put).
func WriteCached(bs Blockstore, size int) (Blockstore, error) {
	c, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &writecache{blockstore: bs, cache: c}, nil
}

type writecache struct {
	cache      *lru.Cache // pointer b/c Cache contains a Mutex as value (complicates copying)
	blockstore Blockstore
}

func (w *writecache) DeleteBlock(k u.Key) error {
	w.cache.Remove(k)
	return w.blockstore.DeleteBlock(k)
}

func (w *writecache) Has(k u.Key) (bool, error) {
	if _, ok := w.cache.Get(k); ok {
		return true, nil
	}
	return w.blockstore.Has(k)
}

func (w *writecache) Get(k u.Key) (*blocks.Block, error) {
	return w.blockstore.Get(k)
}

func (w *writecache) Put(b *blocks.Block) error {
	if _, ok := w.cache.Get(b.Key()); ok {
		return nil
	}
	w.cache.Add(b.Key(), struct{}{})
	return w.blockstore.Put(b)
}

func (w *writecache) AllKeysChan(ctx context.Context) (<-chan u.Key, error) {
	return w.blockstore.AllKeysChan(ctx)
}
