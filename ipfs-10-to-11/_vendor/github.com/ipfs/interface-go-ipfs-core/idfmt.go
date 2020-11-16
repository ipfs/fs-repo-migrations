package iface

import (
	peer "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core/peer"
	mbase "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/multiformats/go-multibase"
)

func FormatKeyID(id peer.ID) string {
	if s, err := peer.ToCid(id).StringOfBase(mbase.Base36); err != nil {
		panic(err)
	} else {
		return s
	}
}

// FormatKey formats the given IPNS key in a canonical way.
func FormatKey(key Key) string {
	return FormatKeyID(key.ID())
}
