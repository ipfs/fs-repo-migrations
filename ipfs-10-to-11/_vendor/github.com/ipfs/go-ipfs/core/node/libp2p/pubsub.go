package libp2p

import (
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core/discovery"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-pubsub"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/go.uber.org/fx"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipfs/core/node/helpers"
)

func FloodSub(pubsubOptions ...pubsub.Option) interface{} {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle, host host.Host, disc discovery.Discovery) (service *pubsub.PubSub, err error) {
		return pubsub.NewFloodSub(helpers.LifecycleCtx(mctx, lc), host, append(pubsubOptions, pubsub.WithDiscovery(disc))...)
	}
}

func GossipSub(pubsubOptions ...pubsub.Option) interface{} {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle, host host.Host, disc discovery.Discovery) (service *pubsub.PubSub, err error) {
		return pubsub.NewGossipSub(helpers.LifecycleCtx(mctx, lc), host, append(
			pubsubOptions,
			pubsub.WithDiscovery(disc),
			pubsub.WithFloodPublish(true))...,
		)
	}
}
