package relay

import (
	"encoding/binary"
	"errors"
	"io"

	pb "gx/ipfs/QmZRbCo2gw7ghw5m7L77a8FvvQTVr62J4hmy8ozpdq7dHF/go-libp2p-circuit/pb"

	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	ggio "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/io"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	peer "gx/ipfs/QmcJukH2sAFjY3HdBKq35WDzWoL3UUu2gt9wdfqZTUyM74/go-libp2p-peer"
	pstore "gx/ipfs/QmdeiKhUy1TVGBaKxt7y1QmBDLBdisSrLJ1x58Eoj4PXUh/go-libp2p-peerstore"
)

func peerToPeerInfo(p *pb.CircuitRelay_Peer) (pstore.PeerInfo, error) {
	if p == nil {
		return pstore.PeerInfo{}, errors.New("nil peer")
	}

	h, err := mh.Cast(p.Id)
	if err != nil {
		return pstore.PeerInfo{}, err
	}

	addrs := make([]ma.Multiaddr, 0, len(p.Addrs))
	for _, addrBytes := range p.Addrs {
		a, err := ma.NewMultiaddrBytes(addrBytes)
		if err == nil {
			addrs = append(addrs, a)
		}
	}

	return pstore.PeerInfo{ID: peer.ID(h), Addrs: addrs}, nil
}

func peerInfoToPeer(pi pstore.PeerInfo) *pb.CircuitRelay_Peer {
	addrs := make([][]byte, len(pi.Addrs))
	for i, addr := range pi.Addrs {
		addrs[i] = addr.Bytes()
	}

	p := new(pb.CircuitRelay_Peer)
	p.Id = []byte(pi.ID)
	p.Addrs = addrs

	return p
}

type delimitedReader struct {
	r   io.Reader
	buf []byte
}

// The gogo protobuf NewDelimitedReader is buffered, which may eat up stream data.
// So we need to implement a compatible delimited reader that reads unbuffered.
// There is a slowdown from unbuffered reading: when reading the message
// it can take multiple single byte Reads to read the length and another Read
// to read the message payload.
// However, this is not critical performance degradation as
// - the reader is utilized to read one (dialer, stop) or two messages (hop) during
//   the handshake, so it's a drop in the water for the connection lifetime.
// - messages are small (max 4k) and the length fits in a couple of bytes,
//   so overall we have at most three reads per message.
func newDelimitedReader(r io.Reader, maxSize int) *delimitedReader {
	return &delimitedReader{r: r, buf: make([]byte, maxSize)}
}

func (d *delimitedReader) ReadByte() (byte, error) {
	buf := d.buf[:1]
	_, err := d.r.Read(buf)
	return buf[0], err
}

func (d *delimitedReader) ReadMsg(msg proto.Message) error {
	mlen, err := binary.ReadUvarint(d)
	if err != nil {
		return err
	}

	if uint64(len(d.buf)) < mlen {
		return errors.New("Message too large")
	}

	buf := d.buf[:mlen]
	_, err = io.ReadFull(d.r, buf)
	if err != nil {
		return err
	}

	return proto.Unmarshal(buf, msg)
}

func newDelimitedWriter(w io.Writer) ggio.WriteCloser {
	return ggio.NewDelimitedWriter(w)
}
