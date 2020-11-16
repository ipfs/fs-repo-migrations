package node

import (
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-graphsync"
	gsimpl "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-graphsync/impl"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-graphsync/storeutil"
	blockstore "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipfs-blockstore"
	libp2p "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/go.uber.org/fx"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipfs/core/node/helpers"
)

// Graphsync constructs a graphsync
func Graphsync(lc fx.Lifecycle, mctx helpers.MetricsCtx, host libp2p.Host, bs blockstore.GCBlockstore) graphsync.GraphExchange {
	ctx := helpers.LifecycleCtx(mctx, lc)

	network := network.NewFromLibp2pHost(host)
	return gsimpl.New(ctx, network,
		storeutil.LoaderForBlockstore(bs),
		storeutil.StorerForBlockstore(bs),
	)
}
