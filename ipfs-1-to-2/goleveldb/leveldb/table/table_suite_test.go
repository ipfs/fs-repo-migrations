package table

import (
	"testing"

	"github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/goleveldb/leveldb/testutil"
)

func TestTable(t *testing.T) {
	testutil.RunSuite(t, "Table Suite")
}
