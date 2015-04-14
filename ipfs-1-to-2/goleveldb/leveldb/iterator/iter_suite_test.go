package iterator_test

import (
	"testing"

	"github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/goleveldb/leveldb/testutil"
)

func TestIterator(t *testing.T) {
	testutil.RunSuite(t, "Iterator Suite")
}
