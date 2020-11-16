package plugin

import (
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipfs/core/coredag"

	ipld "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipld-format"
)

// PluginIPLD is an interface that can be implemented to add handlers for
// for different IPLD formats
type PluginIPLD interface {
	Plugin

	RegisterBlockDecoders(dec ipld.BlockDecoder) error
	RegisterInputEncParsers(iec coredag.InputEncParsers) error
}
