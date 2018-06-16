package mg7

import (
	"bufio"
	"context"
	"fmt"
	"os"
	path "path/filepath"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/repolock"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
	log "github.com/ipfs/fs-repo-migrations/stump"

	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	bstore "gx/ipfs/QmaG4DZ4JaqEfvPWt5nPPgoTzhc1tr1T3f4Nu9Jpdm8ymY/go-ipfs-blockstore"
	fsrepo "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo/fsrepo"
	verifcid "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/thirdparty/verifcid"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
	blk "gx/ipfs/Qmej7nf81hi2x2tvjRBF3mcp74sQyuDH4VMYDGd1YtXjb2/go-block-format"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "7-to-8"
}

func (m Migration) Reversible() bool {
	return true
}

func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	log.VLog("locking repo at %q", opts.Path)
	lk, err := lock.Lock2(opts.Path)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)

	log.VLog("  - verifying version is '7'")
	if err := repo.CheckVersion("7"); err != nil {
		return err
	}

	// open the repo
	fsrepo.RepoVersion = 7
	frepo, err := fsrepo.Open(opts.Path)
	if err != nil {
		return err
	}

	// create a mock ipfs node
	n, err := NewMockNode(frepo)
	if err != nil {
		return err
	}

	// First Scan all keys and collect identity and insecure hashes
	ch, err := n.Blockstore.AllKeysChan(context.TODO())
	if err != nil {
		return err
	}
	var idHashes, insecureCids []*cid.Cid
	for c := range ch {
		err = verifcid.ValidateCid(c)
		if err != nil {
			insecureCids = append(insecureCids, c)
			continue
		}
		prefix := c.Prefix()
		if prefix.MhType == mh.ID {
			idHashes = append(idHashes, c)
		}
	}

	// used to collect errors so we can provide the user with as much
	// information as possible before aborting
	var errors []string
	// helper to report errors in a sane manor
	count := 0
	errlog, err := os.Create(path.Join(opts.Path, "7-to-8-errlog"))
	reportErr := func(fmtStr string, a ...interface{}) {
		if count < 10 {
			fmt.Fprintf(os.Stderr, fmtStr, a...)
		} else if count == 10 {
			fmt.Fprintf(os.Stderr, "too many errors see '%s' for full log", "7-to-8-errlog")
		}
		fmt.Fprintf(errlog, fmtStr, a...)
		count++
	}

	// Next check if any of the insecure hashes are pinned
	pinned, err := n.Pinning.CheckIfPinned(insecureCids...)
	if err != nil {
		return err
	}
	if len(pinned) > 0 {
		count = 0
		for _, p := range pinned {
			reportErr("insecure hash %s %s\n", p.Key.String(), p.String())
		}
		errors = append(errors, "pinned insecure hashes")
	}

	// Next check if there are any insure hashes as children of the
	// files root, note a insure child will cause a problem even if
	// the block is not the the local store, so we report all insecure
	// children found
	type pc struct {
		parent *cid.Cid
		child  *cid.Cid
	}
	var insecureMfs []pc
	var checkChildren func(*cid.Cid) error
	checkChildren = func(parent *cid.Cid) error {
		links, err := ipld.GetLinks(context.TODO(), n.DAG, parent)
		if err != nil {
			return err
		}
		for _, lnk := range links {
			c := lnk.Cid
			err = verifcid.ValidateCid(c)
			if err != nil {
				insecureMfs = append(insecureMfs, pc{parent: parent, child: c})
				continue
			}

			err := checkChildren(c)
			if err != nil {
				return err
			}
		}
		return nil
	}
	rootCid := n.FilesRoot.Cid()
	err = verifcid.ValidateCid(rootCid)
	if err != nil {
		errors = append(errors, "files root uses an insure hash")
	} else {
		err = checkChildren(n.FilesRoot.Cid())
		if err != nil {
			return err
		}
	}
	if len(insecureMfs) > 0 {
		for _, l := range insecureMfs {
			reportErr("insecure hash %s found as child of %s, which is a child of the files root\n",
				l.child.String(), l.parent.String())
		}
		errors = append(errors, "files root has insecure children")
	}

	errlog.Close()
	if len(errors) > 0 {
		errors = append(errors, "can not continue")
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	opt := os.Getenv("IPFS_7_TO_8_OPTS")
	if opt != "GC_INSECURE" {
		return fmt.Errorf("will not garbage collect insecure hashes, to continue set IPFS_7_TO_8_OPTS=GC_INSECURE")
	}

	fh, err := os.Create(path.Join(opts.Path, "7-to-8-deleted"))
	if err != nil {
		return err
	}
	deleteBlock := func(c *cid.Cid) {
		fmt.Fprintf(fh, "%s\n", c.String())
		err = n.Blockstore.DeleteBlock(c)
		_ = err // FIXME: Deal with errors here, ignore not found, report others
		//if err != nil {
		//	return err
		//}
	}
	defer fh.Close()

	for _, c := range idHashes {
		deleteBlock(c)
	}

	for _, c := range insecureCids {
		deleteBlock(c)
	}

	frepo.Close()

	err = repo.WriteVersion("8")
	if err != nil {
		log.Error("failed to update version file to 8")
		return err
	}

	log.Log("updated version file")

	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")

	// open the repo
	fsrepo.RepoVersion = 8
	frepo, err := fsrepo.Open(opts.Path)
	if err != nil {
		return err
	}

	// attach the blockstore
	bs := bstore.NewBlockstore(frepo.Datastore())

	fh, err := os.Open(path.Join(opts.Path, "7-to-8-deleted"))
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(fh)

	var idBlocks []blk.Block
	for scanner.Scan() {
		cidStr := scanner.Text()
		c, err := cid.Decode(cidStr)
		if err != nil {
			return err
		}
		dmh, err := mh.Decode(c.Hash())
		if err != nil {
			return err
		}
		if dmh.Code == mh.ID {
			b, err := blk.NewBlockWithCid(dmh.Digest, c)
			if err != nil {
				return err
			}
			idBlocks = append(idBlocks, b)
		} else {
			// FIXME: log the fact that insecure block could not be restored
		}
	}

	err = bs.PutMany(idBlocks)
	if err != nil {
		return err
	}

	frepo.Close()

	err = mfsr.RepoPath(opts.Path).WriteVersion("7")
	if err != nil {
		log.Error("failed to update version file to 7")
		return err
	}

	log.Log("updated version file")

	return nil
}
