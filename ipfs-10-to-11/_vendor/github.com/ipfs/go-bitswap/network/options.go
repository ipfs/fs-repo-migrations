package network

import "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core/protocol"

type NetOpt func(*Settings)

type Settings struct {
	ProtocolPrefix     protocol.ID
	SupportedProtocols []protocol.ID
}

func Prefix(prefix protocol.ID) NetOpt {
	return func(settings *Settings) {
		settings.ProtocolPrefix = prefix
	}
}

func SupportedProtocols(protos []protocol.ID) NetOpt {
	return func(settings *Settings) {
		settings.SupportedProtocols = protos
	}
}
