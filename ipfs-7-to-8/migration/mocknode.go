package mg7

import (
	"context"
	"fmt"
	"os"
	"syscall"

	bserv "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/blockservice"
	filestore "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/filestore"
	dag "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/merkledag"
	pin "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/pin"
	repo "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo"
	//cfg "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo/config"

	offline "gx/ipfs/QmWM5HhdG5ZQNyHQ5XhMdGmV9CvLpFynQfGpTxN2MEM7Lc/go-ipfs-exchange-offline"
	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	bstore "gx/ipfs/QmaG4DZ4JaqEfvPWt5nPPgoTzhc1tr1T3f4Nu9Jpdm8ymY/go-ipfs-blockstore"
	mdag "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/merkledag"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	exchange "gx/ipfs/QmdcAXgEHUueP4A7b5hjabKn2EooeHgMreMvFC249dGCgc/go-ipfs-exchange-interface"
	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
	//retry "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/retrystore"
)

// IpfsNode is IPFS Core module. It represents an IPFS instance.
type MockNode struct {
	Repo repo.Repo

	// Local node
	Pinning pin.Pinner // the pinning manager

	// Services
	Blockstore bstore.GCBlockstore  // the block store (lower level)
	Filestore  *filestore.Filestore // the filestore blockstore
	BaseBlocks bstore.Blockstore    // the raw blockstore, no filestore wrapping
	GCLocker   bstore.GCLocker      // the locker used to protect the blockstore during gc
	Blocks     bserv.BlockService   // the block service, get/add blocks.
	DAG        ipld.DAGService      // the merkle dag service, get/add objects.
	FilesRoot  *mdag.ProtoNode

	Exchange exchange.Interface // the block exchange + strategy (bitswap)
}

func NewMockNode(repo repo.Repo) (*MockNode, error) {
	n := &MockNode{Repo: repo}
	err := n.setup()
	return n, err
}

func (n *MockNode) setup() error {
	rds := n.Repo.Datastore()
	// rds := &retry.Datastore{
	// 	Batching:    n.Repo.Datastore(),
	// 	Delay:       time.Millisecond * 200,
	// 	Retries:     6,
	// 	TempErrFunc: isTooManyFDError,
	// }

	bs := bstore.NewBlockstore(rds)

	conf, err := n.Repo.Config()
	if err != nil {
		return err
	}

	n.BaseBlocks = bs
	n.GCLocker = bstore.NewGCLocker()
	n.Blockstore = bstore.NewGCBlockstore(bs, n.GCLocker)

	if conf.Experimental.FilestoreEnabled {
		n.Filestore = filestore.NewFilestore(bs, n.Repo.FileManager())
		n.Blockstore = bstore.NewGCBlockstore(n.Filestore, n.GCLocker)
	}

	n.Exchange = offline.Exchange(n.Blockstore)

	n.Blocks = bserv.New(n.Blockstore, n.Exchange)
	n.DAG = dag.NewDAGService(n.Blocks)

	internalDag := dag.NewDAGService(bserv.New(n.Blockstore, offline.Exchange(n.Blockstore)))
	n.Pinning, err = pin.LoadPinner(n.Repo.Datastore(), n.DAG, internalDag)
	if err != nil {
		// TODO: we should move towards only running 'NewPinner' explicitly on
		// node init instead of implicitly here as a result of the pinner keys
		// not being found in the datastore.
		// this is kinda sketchy and could cause data loss
		n.Pinning = pin.NewPinner(n.Repo.Datastore(), n.DAG, internalDag)
	}

	return n.loadFilesRoot()
}

func isTooManyFDError(err error) bool {
	perr, ok := err.(*os.PathError)
	if ok && perr.Err == syscall.EMFILE {
		return true
	}

	return false
}

func (n *MockNode) loadFilesRoot() error {
	dsk := ds.NewKey("/local/filesroot")

	var nd *mdag.ProtoNode
	val, err := n.Repo.Datastore().Get(dsk)

	switch {
	case err == ds.ErrNotFound || val == nil:
		nd = nil
	case err == nil:
		c, err := cid.Cast(val.([]byte))
		if err != nil {
			return err
		}

		rnd, err := n.DAG.Get(context.TODO(), c)
		if err != nil {
			return fmt.Errorf("error loading filesroot from DAG: %s", err)
		}

		pbnd, ok := rnd.(*mdag.ProtoNode)
		if !ok {
			return mdag.ErrNotProtobuf
		}

		nd = pbnd
	default:
		return err
	}

	n.FilesRoot = nd
	return nil
}
