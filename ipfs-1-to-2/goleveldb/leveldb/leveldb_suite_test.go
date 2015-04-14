package leveldb

import (
	"testing"

	"github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/goleveldb/leveldb/testutil"
)

func TestLevelDB(t *testing.T) {
	testutil.RunSuite(t, "LevelDB Suite")
}
