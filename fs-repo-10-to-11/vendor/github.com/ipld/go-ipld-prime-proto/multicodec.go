package dagpb

import (
	"errors"
	"io"

	ipld "github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

var (
	// ErrNoAutomaticDecoding means the NodeBuilder must provide a fast path decoding method on its own
	ErrNoAutomaticDecoding = errors.New("No automatic decoding for this type, node builder must provide fast path")
	// ErrNoAutomaticEncoding means the Node must provide a fast path encoding method on its own
	ErrNoAutomaticEncoding                           = errors.New("No automatic encoding for this type, node must provide fast path")
	_                      cidlink.MulticodecDecoder = PBDecoder
	_                      cidlink.MulticodecEncoder = PBEncoder
	_                      cidlink.MulticodecDecoder = RawDecoder
	_                      cidlink.MulticodecEncoder = RawEncoder
)

func init() {
	cidlink.RegisterMulticodecDecoder(0x70, PBDecoder)
	cidlink.RegisterMulticodecEncoder(0x70, PBEncoder)
	cidlink.RegisterMulticodecDecoder(0x55, RawDecoder)
	cidlink.RegisterMulticodecEncoder(0x55, RawEncoder)
}

// PBDecoder is a decoder function for Dag Protobuf nodes
func PBDecoder(nb ipld.NodeAssembler, r io.Reader) error {
	// Probe for a builtin fast path.  Shortcut to that if possible.
	//  (ipldcbor.NodeBuilder supports this, for example.)
	type detectFastPath interface {
		DecodeDagProto(io.Reader) error
	}
	if nb2, ok := nb.(detectFastPath); ok {
		return nb2.DecodeDagProto(r)
	}
	// Okay, generic builder path.
	return ErrNoAutomaticDecoding
}

// PBEncoder is a encoder function that encodes to Dag Protobuf
func PBEncoder(n ipld.Node, w io.Writer) error {
	// Probe for a builtin fast path.  Shortcut to that if possible.
	//  (ipldcbor.Node supports this, for example.)
	type detectFastPath interface {
		EncodeDagProto(io.Writer) error
	}
	if n2, ok := n.(detectFastPath); ok {
		return n2.EncodeDagProto(w)
	}
	// Okay, generic inspection path.
	return ErrNoAutomaticEncoding
}

// RawDecoder is a decoder function for raw coded nodes
func RawDecoder(nb ipld.NodeAssembler, r io.Reader) error {
	// Probe for a builtin fast path.  Shortcut to that if possible.
	//  (ipldcbor.NodeBuilder supports this, for example.)
	type detectFastPath interface {
		DecodeDagRaw(io.Reader) error
	}
	if nb2, ok := nb.(detectFastPath); ok {
		return nb2.DecodeDagRaw(r)
	}
	// Okay, generic builder path.
	return ErrNoAutomaticDecoding
}

// RawEncoder encodes a node to a raw block structure
func RawEncoder(n ipld.Node, w io.Writer) error {
	// Probe for a builtin fast path.  Shortcut to that if possible.
	//  (ipldcbor.Node supports this, for example.)
	type detectFastPath interface {
		EncodeDagRaw(io.Writer) error
	}
	if n2, ok := n.(detectFastPath); ok {
		return n2.EncodeDagRaw(w)
	}
	// Okay, generic inspection path.
	return ErrNoAutomaticEncoding
}
