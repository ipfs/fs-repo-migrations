package merkledag

import (
	"fmt"
	"github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/Qmej7nf81hi2x2tvjRBF3mcp74sQyuDH4VMYDGd1YtXjb2/go-block-format"

	u "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmNiJuT8Ja3hMVpBHXv3Q6dwmperaQ6JjLtpMQgMCD7xvx/go-ipfs-util"
	cid "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// RawNode represents a node which only contains data.
type RawNode struct {
	blocks.Block
}

// NewRawNode creates a RawNode using the default sha2-256 hash function.
func NewRawNode(data []byte) *RawNode {
	h := u.Hash(data)
	c := cid.NewCidV1(cid.Raw, h)
	blk, _ := blocks.NewBlockWithCid(data, c)

	return &RawNode{blk}
}

// DecodeRawBlock is a block decoder for raw IPLD nodes conforming to `node.DecodeBlockFunc`.
func DecodeRawBlock(block blocks.Block) (ipld.Node, error) {
	if block.Cid().Type() != cid.Raw {
		return nil, fmt.Errorf("raw nodes cannot be decoded from non-raw blocks: %d", block.Cid().Type())
	}
	// Once you "share" a block, it should be immutable. Therefore, we can just use this block as-is.
	return &RawNode{block}, nil
}

var _ ipld.DecodeBlockFunc = DecodeRawBlock

// NewRawNodeWPrefix creates a RawNode with the hash function
// specified in prefix.
func NewRawNodeWPrefix(data []byte, prefix cid.Prefix) (*RawNode, error) {
	prefix.Codec = cid.Raw
	if prefix.Version == 0 {
		prefix.Version = 1
	}
	c, err := prefix.Sum(data)
	if err != nil {
		return nil, err
	}
	blk, err := blocks.NewBlockWithCid(data, c)
	if err != nil {
		return nil, err
	}
	return &RawNode{blk}, nil
}

// Links returns nil.
func (rn *RawNode) Links() []*ipld.Link {
	return nil
}

// ResolveLink returns an error.
func (rn *RawNode) ResolveLink(path []string) (*ipld.Link, []string, error) {
	return nil, nil, ErrLinkNotFound
}

// Resolve returns an error.
func (rn *RawNode) Resolve(path []string) (interface{}, []string, error) {
	return nil, nil, ErrLinkNotFound
}

// Tree returns nil.
func (rn *RawNode) Tree(p string, depth int) []string {
	return nil
}

// Copy performs a deep copy of this node and returns it as an ipld.Node
func (rn *RawNode) Copy() ipld.Node {
	copybuf := make([]byte, len(rn.RawData()))
	copy(copybuf, rn.RawData())
	nblk, err := blocks.NewBlockWithCid(rn.RawData(), rn.Cid())
	if err != nil {
		// programmer error
		panic("failure attempting to clone raw block: " + err.Error())
	}

	return &RawNode{nblk}
}

// Size returns the size of this node
func (rn *RawNode) Size() (uint64, error) {
	return uint64(len(rn.RawData())), nil
}

// Stat returns some Stats about this node.
func (rn *RawNode) Stat() (*ipld.NodeStat, error) {
	return &ipld.NodeStat{
		CumulativeSize: len(rn.RawData()),
		DataSize:       len(rn.RawData()),
	}, nil
}

var _ ipld.Node = (*RawNode)(nil)
