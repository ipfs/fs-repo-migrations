package memdb

import (
	"testing"

	"github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/goleveldb/leveldb/testutil"
)

func TestMemDB(t *testing.T) {
	testutil.RunSuite(t, "MemDB Suite")
}
