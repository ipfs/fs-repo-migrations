package libp2p

import (
	"context"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-peerstore/pstoremem"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/go.uber.org/fx"
)

func Peerstore(lc fx.Lifecycle) peerstore.Peerstore {
	pstore := pstoremem.NewPeerstore()
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return pstore.Close()
		},
	})

	return pstore
}
