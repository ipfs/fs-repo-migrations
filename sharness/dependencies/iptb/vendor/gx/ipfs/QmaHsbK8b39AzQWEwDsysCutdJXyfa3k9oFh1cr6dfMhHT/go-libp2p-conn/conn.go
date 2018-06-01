package conn

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	lgbl "gx/ipfs/QmPDZJxtWGfcwLPazJxD4h3v3aDs43V7UNAVs3Jz1Wo7o4/go-libp2p-loggables"
	tpt "gx/ipfs/QmPUHzTLPZFYqv8WqcBTuMFYTgeom4uHHEaxzk7bd5GYZB/go-libp2p-transport"
	logging "gx/ipfs/QmTG23dvpBCBjqQwyDxV8CQT6jmS4PSftNr1VqHhE3MLy7/go-log"
	mpool "gx/ipfs/QmWBug6eBS7AxRdCDVuSY5CnSit7cS2XnPFYJWqWDumhCG/go-msgio/mpool"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	iconn "gx/ipfs/QmYDNqBAMWVMHKndYR35Sd8PfEVWBiDmpHYkuRJTunJDeJ/go-libp2p-interface-conn"
	peer "gx/ipfs/QmcJukH2sAFjY3HdBKq35WDzWoL3UUu2gt9wdfqZTUyM74/go-libp2p-peer"
	ic "gx/ipfs/Qme1knMqwt1hKZbc1BmQFmnm9f36nyQGwXxPGVpVJ9rMK5/go-libp2p-crypto"
)

var log = logging.Logger("conn")

// ReleaseBuffer puts the given byte array back into the appropriate
// global buffer pool based on its capacity.
func ReleaseBuffer(b []byte) {
	log.Debugf("Releasing buffer! (cap,size = %d, %d)", cap(b), len(b))
	mpool.ByteSlicePool.Put(uint32(cap(b)), b)
}

// singleConn represents a single connection to another Peer (IPFS Node).
type singleConn struct {
	local  peer.ID
	remote peer.ID
	maconn tpt.Conn

	eventMu sync.Mutex
	event   io.Closer
}

// newConn constructs a new connection
func newSingleConn(ctx context.Context, local, remote peer.ID, maconn tpt.Conn) iconn.Conn {
	ml := lgbl.Dial("conn", local, remote, maconn.LocalMultiaddr(), maconn.RemoteMultiaddr())

	conn := &singleConn{
		local:  local,
		remote: remote,
		maconn: maconn,
		event:  log.EventBegin(ctx, "connLifetime", ml),
	}

	log.Debugf("newSingleConn %p: %v to %v", conn, local, remote)
	return conn
}

// close is the internal close function, called by ContextCloser.Close
func (c *singleConn) Close() error {
	c.eventMu.Lock()
	if c.event != nil {
		evt := c.event
		c.event = nil
		defer evt.Close()
	}
	c.eventMu.Unlock()

	// close underlying connection
	return c.maconn.Close()
}

// ID is an identifier unique to this connection.
func (c *singleConn) ID() string {
	return iconn.ID(c)
}

func (c *singleConn) String() string {
	return iconn.String(c, "singleConn")
}

func (c *singleConn) LocalAddr() net.Addr {
	return c.maconn.LocalAddr()
}

func (c *singleConn) RemoteAddr() net.Addr {
	return c.maconn.RemoteAddr()
}

func (c *singleConn) LocalPrivateKey() ic.PrivKey {
	return nil
}

func (c *singleConn) RemotePublicKey() ic.PubKey {
	return nil
}

func (c *singleConn) SetDeadline(t time.Time) error {
	return c.maconn.SetDeadline(t)
}
func (c *singleConn) SetReadDeadline(t time.Time) error {
	return c.maconn.SetReadDeadline(t)
}

func (c *singleConn) SetWriteDeadline(t time.Time) error {
	return c.maconn.SetWriteDeadline(t)
}

// LocalMultiaddr is the Multiaddr on this side
func (c *singleConn) LocalMultiaddr() ma.Multiaddr {
	return c.maconn.LocalMultiaddr()
}

// RemoteMultiaddr is the Multiaddr on the remote side
func (c *singleConn) RemoteMultiaddr() ma.Multiaddr {
	return c.maconn.RemoteMultiaddr()
}

func (c *singleConn) Transport() tpt.Transport {
	return c.maconn.Transport()
}

// LocalPeer is the Peer on this side
func (c *singleConn) LocalPeer() peer.ID {
	return c.local
}

// RemotePeer is the Peer on the remote side
func (c *singleConn) RemotePeer() peer.ID {
	return c.remote
}

// Read reads data, net.Conn style
func (c *singleConn) Read(buf []byte) (int, error) {
	return c.maconn.Read(buf)
}

// Write writes data, net.Conn style
func (c *singleConn) Write(buf []byte) (int, error) {
	return c.maconn.Write(buf)
}
