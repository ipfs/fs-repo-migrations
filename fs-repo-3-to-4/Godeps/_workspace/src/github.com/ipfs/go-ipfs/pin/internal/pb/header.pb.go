// Code generated by protoc-gen-gogo.
// source: header.proto
// DO NOT EDIT!

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	header.proto

It has these top-level messages:
	Set
*/
package pb

import proto "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/gogo/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Set struct {
	// 1 for now, library will refuse to handle entries with an unrecognized version.
	Version *uint32 `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	// how many of the links are subtrees
	Fanout *uint32 `protobuf:"varint,2,opt,name=fanout" json:"fanout,omitempty"`
	// hash seed for subtree selection, a random number
	Seed             *uint32 `protobuf:"fixed32,3,opt,name=seed" json:"seed,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Set) Reset()         { *m = Set{} }
func (m *Set) String() string { return proto.CompactTextString(m) }
func (*Set) ProtoMessage()    {}

func (m *Set) GetVersion() uint32 {
	if m != nil && m.Version != nil {
		return *m.Version
	}
	return 0
}

func (m *Set) GetFanout() uint32 {
	if m != nil && m.Fanout != nil {
		return *m.Fanout
	}
	return 0
}

func (m *Set) GetSeed() uint32 {
	if m != nil && m.Seed != nil {
		return *m.Seed
	}
	return 0
}

func init() {
}
