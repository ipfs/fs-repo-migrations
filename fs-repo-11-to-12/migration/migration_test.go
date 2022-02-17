package mg11

import (
	"context"
	"os"
	"testing"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	"github.com/otiai10/copy"
)

const (
	repoPath    = "repotest"
	workingRepo = "repotest_copy"
)

func TestGenericMigration(t *testing.T) {
	origSetting := EnableFlatFSFastPath
	defer func() {
		EnableFlatFSFastPath = origSetting
	}()
	EnableFlatFSFastPath = false
	testMigrationBase(t)
}

func TestFlatFSMigration(t *testing.T) {
	origSetting := EnableFlatFSFastPath
	defer func() {
		EnableFlatFSFastPath = origSetting
	}()
	EnableFlatFSFastPath = true
	testMigrationBase(t)
}

// TestMigration works on an IPFS repository as created by running steps.sh
// with ipfs v0.8.0 on using the $repotest folder:
//
// added Qmae3RedM7SNkWGsdzYzsr6svmsFdsva4WoTvYYsWhUSVz a/file1
// added QmbyWz5YD4qpPjhETVBKTE3QJ5ikNBtx3GKq2iCYj8Jj7E a
// added bafybeifwydgv5er4hw2uqggfbzdiozfmuf6n7ylmtcg7ivxcrrtyn5xv5u a/file1
// added bafybeidpdazyjl3tqbuo5ln4pzvqafvwnvn2qwvgenvalipy442otc4ipq a
// added bafybeidbl36uxj43k6xyd2fsrcbfhstgvzq7cflggxlexe5gkokthufj2a b/file2
// added bafybeie4pduk2uwvr5dq36wnbhxspgox7dtqo3fprri4r2wpa7vrej5jqq b
// added Qmesmmf1EEG1orJb6XdK6DabxexsseJnCfw8pqWgonbkoj c/file3
// added QmT3zhz9ZZjEpbzWib95EQ5ESUQs4YasrMQwPScpNGLEXZ c
func testMigrationBase(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure we work on a clean copy of the provided repo
	copyRepo(t)

	opts := migrate.Options{
		Verbose: true,
		Flags: migrate.Flags{
			Path:    workingRepo,
			Revert:  false,
			Verbose: true,
		},
	}
	m := Migration{}
	err := m.open(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Print out the blocks before migrating (for debugging)
	origBlocks := blocks(t, m.dstore)
	t.Logf("Blocks pre-migration: %d", len(origBlocks))
	for _, origb := range origBlocks {
		k := ds.NewKey(origb)
		c, err := dsKeyToCid(ds.NewKey(k.BaseNamespace()))
		if err != nil {
			t.Error("block before migration cannot be parsed")
		}
		t.Log(origb, " -> ", c)
	}

	// Apply the migration
	err = m.Apply(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Re-open store, as Apply() closes things.
	err = m.open(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Obtain blocks post-migration
	postBlocks := blocks(t, m.dstore)

	t.Logf("Blocks post-migration: %d", len(postBlocks))

	// All blocks should be parseable.
	// We do not check what there is or not. We will just
	// expect that they are all CIDv0 and that revert
	// will put everything in its place.
	for _, postb := range postBlocks {
		k := ds.NewKey(postb)
		c, err := dsKeyToCid(ds.NewKey(k.BaseNamespace()))
		if err != nil {
			t.Error("key after migration cannot be parsed")
		}
		t.Log(postb, " -> ", c)

		if c.Version() != 0 {
			t.Error("CidV1 keys left after migration")
		}
	}

	// Set MFS root to an CIDv1 corresponding to folder b.  This means
	// this block will need to be created in CIDv1 fashion after
	// revert. Note, it is not the same as the one from the B-folder-tree
	// added with CidV1 as that one points to a CIDv1 link, while this one
	// has a CIDv0 link inside.
	cidV1FromB, err := cid.Decode("bafybeie4pduk2uwvr5dq36wnbhxspgox7dtqo3fprri4r2wpa7vrej5jqq")
	if err != nil {
		t.Fatal(err)
	}

	writeMFSRoot(t, m.dstore, cidV1FromB)

	// Pin a CIDv1 corresponding to folder C which was added with CIDv0
	// only. This means, we will have to create this block in
	// CIDv1-addressed fashion on revert.
	cidV1FromC, err := cid.Decode("bafybeicgaywc7bcz5dlkbvvjbm7d2epi3wm5yilfuuex6llrxyzr2mlhba")
	if err != nil {
		t.Fatal(err)
	}

	pinner, dags, err := getPinner(ctx, m.dstore)
	if err != nil {
		t.Fatal(err)
	}

	// We need a node for pinning
	nd, err := dags.Get(ctx, cidV1FromC)
	if err != nil {
		t.Fatal(err)
	}

	// Pin the node.
	err = pinner.Pin(ctx, nd, true)
	if err != nil {
		t.Fatal(err)
	}

	err = pinner.Flush(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Now revert the migration.
	opts.Revert = true
	err = m.Revert(opts)
	if err != nil {
		t.Fatal(err)
	}

	revertedBlocks := blocks(t, m.dstore)
	t.Logf("Blocks post-revert: %d", len(revertedBlocks))

	// Check all the original blocks are in the reverted version
	for _, origB := range origBlocks {
		found := false
		for _, revB := range revertedBlocks {
			if origB == revB {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("block %s not found after revert", origB)
		}
	}

	// Print the list of reverted blocks for debugging.
	for _, revB := range revertedBlocks {
		k := ds.NewKey(revB)
		c, err := dsKeyToCid(ds.NewKey(k.BaseNamespace()))
		if err != nil {
			t.Error("key after revert cannot be parsed")
		}
		t.Log(revB, " -> ", c)
	}

	// Re-open, as Apply() closes things.
	err = m.open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer m.dstore.Close()

	// Check that the CIDv1s that we explicitally pinned or
	// added to MFS are now retrievable as CIDv1-addressed nodes.
	postMigrationCids := []cid.Cid{
		cidV1FromB,
		cidV1FromC,
	}

	for _, c := range postMigrationCids {
		k := blocksPrefix.Child(cidToDsKey(c))
		_, err := m.dstore.Get(k)
		if err != nil {
			t.Errorf("expected key %s not found", k)
		}
	}
}

func copyRepo(t *testing.T) {
	t.Log("setting up working IPFS folder")
	os.RemoveAll(workingRepo)
	err := copy.Copy(repoPath, workingRepo)
	if err != nil {
		t.Fatal(err)
	}
}

func blocks(t *testing.T, dstore ds.Batching) []string {
	t.Helper()
	queryAll := query.Query{
		KeysOnly: true,
		Prefix:   "/blocks",
	}

	results, err := dstore.Query(queryAll)
	if err != nil {
		t.Fatal(err)
	}
	defer results.Close()
	entries, err := results.Rest()
	if err != nil {
		t.Fatal(err)
	}
	var blocks []string
	for _, e := range entries {
		blocks = append(blocks, e.Key)
	}
	return blocks
}

func writeMFSRoot(t *testing.T, dstore ds.Batching, c cid.Cid) {
	t.Helper()
	err := dstore.Put(mfsRootKey, c.Bytes())
	if err != nil {
		t.Fatal(err)
	}
}
