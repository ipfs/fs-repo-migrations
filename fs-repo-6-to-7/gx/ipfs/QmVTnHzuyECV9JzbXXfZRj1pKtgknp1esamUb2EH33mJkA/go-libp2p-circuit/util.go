package relay

import (
	"encoding/binary"
	"errors"
	"io"

	pb "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmVTnHzuyECV9JzbXXfZRj1pKtgknp1esamUb2EH33mJkA/go-libp2p-circuit/pb"

	ma "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	ggio "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/io"
	proto "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	peer "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	mh "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
)

func peerToPeerInfo(p *pb.CircuitRelay_Peer) (pstore.PeerInfo, error) {
	if p == nil {
		return pstore.PeerInfo{}, errors.New("nil peer")
	}

	h, err := mh.Cast(p.Id)
	if err != nil {
		return pstore.PeerInfo{}, err
	}

	addrs := make([]ma.Multiaddr, len(p.Addrs))
	for i := 0; i < len(addrs); i++ {
		a, err := ma.NewMultiaddrBytes(p.Addrs[i])
		if err != nil {
			return pstore.PeerInfo{}, err
		}
		addrs[i] = a
	}

	return pstore.PeerInfo{ID: peer.ID(h), Addrs: addrs}, nil
}

func peerInfoToPeer(pi pstore.PeerInfo) *pb.CircuitRelay_Peer {
	addrs := make([][]byte, len(pi.Addrs))
	for i := 0; i < len(addrs); i++ {
		addrs[i] = pi.Addrs[i].Bytes()
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
