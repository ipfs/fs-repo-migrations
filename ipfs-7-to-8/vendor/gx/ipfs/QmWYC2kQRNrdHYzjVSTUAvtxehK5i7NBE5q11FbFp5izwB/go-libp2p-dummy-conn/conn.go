package dconn

import (
	"io"
	"net"
	"time"

	ic "github.com/libp2p/go-libp2p-crypto"
	iconn "github.com/libp2p/go-libp2p-interface-conn"
	peer "github.com/libp2p/go-libp2p-peer"
	tpt "github.com/libp2p/go-libp2p-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/whyrusleeping/bufpipe"
)

func NewDummyConnPair() (conn1 iconn.Conn, conn2 iconn.Conn, err error) {
	pipe1 := bufpipe.NewBufferedPipe(1 << 20)
	pipe2 := bufpipe.NewBufferedPipe(1 << 20)

	conn1 = &dummyconn{pipe1, pipe2}
	conn2 = &dummyconn{pipe2, pipe1}
	return
}

type dummyconn struct {
	pipeR *bufpipe.Pipe
	pipeW *bufpipe.Pipe
}

var _ iconn.Conn = (*dummyconn)(nil)

func (d *dummyconn) Close() error {
	d.pipeW.Close(io.ErrClosedPipe)
	return nil
}

func (d *dummyconn) Read(p []byte) (n int, err error) {
	return d.pipeR.Read(p)
}

func (d *dummyconn) Write(p []byte) (n int, err error) {
	return d.pipeW.Write(p)
}

func (*dummyconn) LocalPeer() peer.ID {
	panic("not implemented")
}

func (*dummyconn) Transport() tpt.Transport {
	panic("not implemented")
}

func (*dummyconn) LocalPrivateKey() ic.PrivKey {
	panic("not implemented")
}

func (*dummyconn) LocalMultiaddr() ma.Multiaddr {
	panic("not implemented")
}

func (*dummyconn) RemotePeer() peer.ID {
	panic("not implemented")
}

func (*dummyconn) RemotePublicKey() ic.PubKey {
	panic("not implemented")
}

func (*dummyconn) RemoteMultiaddr() ma.Multiaddr {
	panic("not implemented")
}

func (*dummyconn) ID() string {
	panic("not implemented")
}

func (*dummyconn) LocalAddr() net.Addr {
	panic("not implemented")
}

func (*dummyconn) RemoteAddr() net.Addr {
	panic("not implemented")
}

func (*dummyconn) SetDeadline(t time.Time) error {
	panic("not implemented")
}

func (*dummyconn) SetReadDeadline(t time.Time) error {
	panic("not implemented")
}

func (*dummyconn) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
}
