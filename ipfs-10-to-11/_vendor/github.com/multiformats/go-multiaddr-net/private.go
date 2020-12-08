package manet

import (
	ma "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/multiformats/go-multiaddr"
	upstream "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/multiformats/go-multiaddr/net"
)

// Deprecated: use github.com/multiformats/go-multiaddr/net
var Private4 = upstream.Private4

// Deprecated: use github.com/multiformats/go-multiaddr/net
var Private6 = upstream.Private6

// Deprecated: use github.com/multiformats/go-multiaddr/net
var Unroutable4 = upstream.Unroutable4

// Deprecated: use github.com/multiformats/go-multiaddr/net
var Unroutable6 = upstream.Unroutable6

// Deprecated: use github.com/multiformats/go-multiaddr/net
func IsPublicAddr(a ma.Multiaddr) bool {
	return upstream.IsPublicAddr(a)
}

// Deprecated: use github.com/multiformats/go-multiaddr/net
func IsPrivateAddr(a ma.Multiaddr) bool {
	return upstream.IsPrivateAddr(a)
}
