package dagpb

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ipfs/go-cid"
	merkledag_pb "github.com/ipfs/go-merkledag/pb"
	"github.com/ipld/go-ipld-prime/fluent"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// byteAccessor is a reader interface that can access underlying bytes
type byteAccesor interface {
	Bytes() []byte
}

// DecodeDagProto is a fast path decoding to protobuf
// from PBNode__Builders
func (nb *_PBNode__Builder) DecodeDagProto(r io.Reader) error {
	var pbn merkledag_pb.PBNode
	var encoded []byte
	var err error
	byteBuf, ok := r.(byteAccesor)
	if ok {
		encoded = byteBuf.Bytes()
	} else {
		encoded, err = ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("io error during unmarshal. %v", err)
		}
	}
	if err := pbn.Unmarshal(encoded); err != nil {
		return fmt.Errorf("unmarshal failed. %v", err)
	}
	return fluent.Recover(func() {
		fb := fluent.WrapAssembler(nb)
		fb.CreateMap(-1, func(fmb fluent.MapAssembler) {
			fmb.AssembleEntry("Links").CreateList(len(pbn.Links), func(flb fluent.ListAssembler) {
				for _, link := range pbn.Links {
					hash, err := cid.Cast(link.GetHash())

					if err != nil {
						panic(fluent.Error{Err: fmt.Errorf("unmarshal failed. %v", err)})
					}
					flb.AssembleValue().CreateMap(-1, func(fmb fluent.MapAssembler) {
						fmb.AssembleEntry("Hash").AssignLink(cidlink.Link{Cid: hash})
						fmb.AssembleEntry("Name").AssignString(link.GetName())
						fmb.AssembleEntry("Tsize").AssignInt(int(link.GetTsize()))
					})
				}
			})
			fmb.AssembleEntry("Data").AssignBytes(pbn.GetData())
		})
	})
}

// EncodeDagProto is a fast path encoding to protobuf
// for PBNode types
func (nd PBNode) EncodeDagProto(w io.Writer) error {
	pbn := new(merkledag_pb.PBNode)
	pbn.Links = make([]*merkledag_pb.PBLink, 0, nd.FieldLinks().Length())
	linksIter := nd.FieldLinks().ListIterator()
	for !linksIter.Done() {
		_, nlink, err := linksIter.Next()
		if err != nil {
			return err
		}
		link := nlink.(PBLink)
		var hash []byte
		if link.FieldHash().Exists() {
			cid := link.FieldHash().Must().Link().(cidlink.Link).Cid
			if cid.Defined() {
				hash = cid.Bytes()
			}
		}
		var name *string
		if link.FieldName().Exists() {
			tmp := link.FieldName().Must().String()
			name = &tmp
		}
		var tsize *uint64
		if link.FieldTsize().Exists() {
			tmp := uint64(link.FieldTsize().Must().Int())
			tsize = &tmp
		}
		pbn.Links = append(pbn.Links, &merkledag_pb.PBLink{
			Hash:  hash,
			Name:  name,
			Tsize: tsize})
	}
	pbn.Data = nd.FieldData().Bytes()
	data, err := pbn.Marshal()
	if err != nil {
		return fmt.Errorf("marshal failed. %v", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf(" error during marshal. %v", err)
	}
	return nil
}

// DecodeDagRaw is a fast path decoding to protobuf
// from RawNode__Builders
func (nb *_RawNode__Builder) DecodeDagRaw(r io.Reader) error {
	return fluent.Recover(func() {
		fnb := fluent.WrapAssembler(nb)
		byteBuf, ok := r.(byteAccesor)
		if ok {
			fnb.AssignBytes(byteBuf.Bytes())
			return
		}
		data, err := ioutil.ReadAll(r)
		if err != nil {
			panic(fluent.Error{Err: fmt.Errorf("io error during unmarshal. %v", err)})
		}
		fnb.AssignBytes(data)
	})
}

// EncodeDagRaw is a fast path encoding to protobuf
// for RawNode types
func (nd RawNode) EncodeDagRaw(w io.Writer) error {
	_, err := w.Write(nd.Bytes())
	if err != nil {
		return fmt.Errorf(" error during marshal. %v", err)
	}
	return nil
}
