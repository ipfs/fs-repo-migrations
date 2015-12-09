// package exchange defines the IPFS Exchange interface
package exchange

import (
	"io"

	blocks "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blocks"
	u "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"
	context "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/golang.org/x/net/context"
)

// Any type that implements exchange.Interface may be used as an IPFS block
// exchange protocol.
type Interface interface {
	// GetBlock returns the block associated with a given key.
	GetBlock(context.Context, u.Key) (*blocks.Block, error)

	GetBlocks(context.Context, []u.Key) (<-chan *blocks.Block, error)

	// TODO Should callers be concerned with whether the block was made
	// available on the network?
	HasBlock(context.Context, *blocks.Block) error

	io.Closer
}
