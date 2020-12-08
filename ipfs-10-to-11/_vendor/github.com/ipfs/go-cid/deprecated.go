package cid

import (
	mh "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/multiformats/go-multihash"
)

// NewPrefixV0 returns a CIDv0 prefix with the specified multihash type.
// DEPRECATED: Use V0Builder
func NewPrefixV0(mhType uint64) Prefix {
	return Prefix{
		MhType:   mhType,
		MhLength: mh.DefaultLengths[mhType],
		Version:  0,
		Codec:    DagProtobuf,
	}
}

// NewPrefixV1 returns a CIDv1 prefix with the specified codec and multihash
// type.
// DEPRECATED: Use V1Builder
func NewPrefixV1(codecType uint64, mhType uint64) Prefix {
	return Prefix{
		MhType:   mhType,
		MhLength: mh.DefaultLengths[mhType],
		Version:  1,
		Codec:    codecType,
	}
}
