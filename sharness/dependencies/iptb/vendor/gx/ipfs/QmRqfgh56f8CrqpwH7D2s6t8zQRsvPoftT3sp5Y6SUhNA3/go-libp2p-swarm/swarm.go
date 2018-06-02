// Package swarm implements a connection muxer with a pair of channels
// to synchronize all network communication.
package swarm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	transport "gx/ipfs/QmPUHzTLPZFYqv8WqcBTuMFYTgeom4uHHEaxzk7bd5GYZB/go-libp2p-transport"
	spdy "gx/ipfs/QmQFhPsJCp82az4SXbziP9QcVSqggEELnV9wGZqMR1EfMB/go-smux-spdystream"
	"gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	goprocessctx "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/context"
	mafilter "gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	logging "gx/ipfs/QmTG23dvpBCBjqQwyDxV8CQT6jmS4PSftNr1VqHhE3MLy7/go-log"
	addrutil "gx/ipfs/QmTGSre9j1otFgsr1opCUQDXTPSM6BTZnMWwPeA5nYJM7w/go-addr-util"
	ws "gx/ipfs/QmU6Yf4jad9eAv9FmCqcYU2S7iN7aQwY2erG376ZDs1j7a/go-ws-transport"
	ps "gx/ipfs/QmVXXxPsnDY16szK4gPy1oz4qKd8HHshemX1miZR2frtJo/go-peerstream"
	psmss "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	metrics "gx/ipfs/QmVvu4bS5QLfS19ePkp5Wgzn2ZUma5oXTT9BgDFyQLxUZF/go-libp2p-metrics"
	mconn "gx/ipfs/QmVvu4bS5QLfS19ePkp5Wgzn2ZUma5oXTT9BgDFyQLxUZF/go-libp2p-metrics/conn"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	inet "gx/ipfs/QmXoz9o2PT3tEzf7hicegwex5UgVP54n3k82K7jrWFyN86/go-libp2p-net"
	pst "gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	conn "gx/ipfs/QmaHsbK8b39AzQWEwDsysCutdJXyfa3k9oFh1cr6dfMhHT/go-libp2p-conn"
	peer "gx/ipfs/QmcJukH2sAFjY3HdBKq35WDzWoL3UUu2gt9wdfqZTUyM74/go-libp2p-peer"
	ipnet "gx/ipfs/Qmd3oYWVLCVWryDV6Pobv6whZcvDXAHqS3chemZ658y4a8/go-libp2p-interface-pnet"
	pstore "gx/ipfs/QmdeiKhUy1TVGBaKxt7y1QmBDLBdisSrLJ1x58Eoj4PXUh/go-libp2p-peerstore"
	tcpt "gx/ipfs/QmdxKHpkZCTV3C7xdE1iJdPfFm5LVvMPvirdFmKu1TimzY/go-tcp-transport"
	ci "gx/ipfs/Qme1knMqwt1hKZbc1BmQFmnm9f36nyQGwXxPGVpVJ9rMK5/go-libp2p-crypto"
	filter "gx/ipfs/Qmf2UAmRwDG4TvnkQpHZWPAzw7rpCYVhxmRXmYxXr5LD1g/go-maddr-filter"
)

var log = logging.Logger("swarm2")

// PSTransport is the default peerstream transport that will be used by
// any libp2p swarms.
var PSTransport pst.Transport

func init() {
	msstpt := psmss.NewBlankTransport()

	ymxtpt := &yamux.Transport{
		AcceptBacklog:          8192,
		ConnectionWriteTimeout: time.Second * 10,
		KeepAliveInterval:      time.Second * 30,
		EnableKeepAlive:        true,
		MaxStreamWindowSize:    uint32(1024 * 512),
		LogOutput:              ioutil.Discard,
	}

	msstpt.AddTransport("/yamux/1.0.0", ymxtpt)
	msstpt.AddTransport("/spdy/3.1.0", spdy.Transport)

	// allow overriding of muxer preferences
	if prefs := os.Getenv("LIBP2P_MUX_PREFS"); prefs != "" {
		msstpt.OrderPreference = strings.Fields(prefs)
	}

	PSTransport = msstpt
}

// Swarm is a connection muxer, allowing connections to other peers to
// be opened and closed, while still using the same Chan for all
// communication. The Chan sends/receives Messages, which note the
// destination or source Peer.
//
// Uses peerstream.Swarm
type Swarm struct {
	swarm *ps.Swarm
	local peer.ID
	peers pstore.Peerstore
	connh ConnHandler

	dsync *DialSync
	backf dialbackoff
	dialT time.Duration // mainly for tests

	dialer *conn.Dialer

	notifmu sync.RWMutex
	notifs  map[inet.Notifiee]ps.Notifiee

	transports []transport.Transport

	// filters for addresses that shouldnt be dialed
	Filters *filter.Filters

	// file descriptor rate limited
	fdRateLimit chan struct{}

	proc goprocess.Process
	ctx  context.Context
	bwc  metrics.Reporter

	limiter *dialLimiter

	protec ipnet.Protector
}

func NewSwarm(ctx context.Context, listenAddrs []ma.Multiaddr, local peer.ID,
	peers pstore.Peerstore, bwc metrics.Reporter) (*Swarm, error) {
	return NewSwarmWithProtector(ctx, listenAddrs, local, peers, nil, PSTransport, bwc)
}

// NewSwarm constructs a Swarm, with a Chan.
func NewSwarmWithProtector(ctx context.Context, listenAddrs []ma.Multiaddr, local peer.ID,
	peers pstore.Peerstore, protec ipnet.Protector, tpt pst.Transport, bwc metrics.Reporter) (*Swarm, error) {

	listenAddrs, err := filterAddrs(listenAddrs)
	if err != nil {
		return nil, err
	}

	var wrap func(c transport.Conn) transport.Conn
	if bwc != nil {
		wrap = func(c transport.Conn) transport.Conn {
			return mconn.WrapConn(bwc, c)
		}
	}

	s := &Swarm{
		swarm:  ps.NewSwarm(tpt),
		local:  local,
		peers:  peers,
		ctx:    ctx,
		dialT:  conn.DialTimeout,
		notifs: make(map[inet.Notifiee]ps.Notifiee),
		transports: []transport.Transport{
			tcpt.NewTCPTransport(),
			new(ws.WebsocketTransport),
		},
		bwc:         bwc,
		fdRateLimit: make(chan struct{}, concurrentFdDials),
		Filters:     filter.NewFilters(),
		dialer:      conn.NewDialer(local, peers.PrivKey(local), wrap),
		protec:      protec,
	}
	s.dialer.Protector = protec

	s.dsync = NewDialSync(s.doDial)
	s.limiter = newDialLimiter(s.dialAddr)

	// configure Swarm
	s.proc = goprocessctx.WithContextAndTeardown(ctx, s.teardown)
	s.SetConnHandler(nil) // make sure to setup our own conn handler.

	err = s.setupInterfaces(listenAddrs)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func NewBlankSwarm(ctx context.Context, id peer.ID, privkey ci.PrivKey, pstpt pst.Transport) *Swarm {
	s := &Swarm{
		swarm:       ps.NewSwarm(pstpt),
		local:       id,
		peers:       pstore.NewPeerstore(),
		ctx:         ctx,
		dialT:       conn.DialTimeout,
		notifs:      make(map[inet.Notifiee]ps.Notifiee),
		fdRateLimit: make(chan struct{}, concurrentFdDials),
		Filters:     filter.NewFilters(),
		dialer:      conn.NewDialer(id, privkey, nil),
	}

	// configure Swarm
	s.limiter = newDialLimiter(s.dialAddr)
	s.proc = goprocessctx.WithContextAndTeardown(ctx, s.teardown)
	s.SetConnHandler(nil) // make sure to setup our own conn handler.

	return s
}

func (s *Swarm) AddTransport(t transport.Transport) {
	s.transports = append(s.transports, t)
}

func (s *Swarm) teardown() error {
	return s.swarm.Close()
}

// AddAddrFilter adds a multiaddr filter to the set of filters the swarm will
// use to determine which addresses not to dial to.
func (s *Swarm) AddAddrFilter(f string) error {
	m, err := mafilter.NewMask(f)
	if err != nil {
		return err
	}

	s.Filters.AddDialFilter(m)
	return nil
}

func filterAddrs(listenAddrs []ma.Multiaddr) ([]ma.Multiaddr, error) {
	if len(listenAddrs) > 0 {
		filtered := addrutil.FilterUsableAddrs(listenAddrs)
		if len(filtered) < 1 {
			return nil, fmt.Errorf("swarm cannot use any addr in: %s", listenAddrs)
		}
		listenAddrs = filtered
	}

	return listenAddrs, nil
}

// Listen sets up listeners for all of the given addresses
func (s *Swarm) Listen(addrs ...ma.Multiaddr) error {
	addrs, err := filterAddrs(addrs)
	if err != nil {
		return err
	}

	return s.setupInterfaces(addrs)
}

// Process returns the Process of the swarm
func (s *Swarm) Process() goprocess.Process {
	return s.proc
}

// Context returns the context of the swarm
func (s *Swarm) Context() context.Context {
	return s.ctx
}

// Close stops the Swarm.
func (s *Swarm) Close() error {
	return s.proc.Close()
}

// StreamSwarm returns the underlying peerstream.Swarm
func (s *Swarm) StreamSwarm() *ps.Swarm {
	return s.swarm
}

// SetConnHandler assigns the handler for new connections.
// See peerstream. You will rarely use this. See SetStreamHandler
func (s *Swarm) SetConnHandler(handler ConnHandler) {

	// handler is nil if user wants to clear the old handler.
	if handler == nil {
		s.swarm.SetConnHandler(func(psconn *ps.Conn) {
			s.connHandler(psconn)
		})
		return
	}

	s.swarm.SetConnHandler(func(psconn *ps.Conn) {
		// sc is nil if closed in our handler.
		if sc := s.connHandler(psconn); sc != nil {
			// call the user's handler. in a goroutine for sync safety.
			go handler(sc)
		}
	})
}

// SetStreamHandler assigns the handler for new streams.
// See peerstream.
func (s *Swarm) SetStreamHandler(handler inet.StreamHandler) {
	s.swarm.SetStreamHandler(func(s *ps.Stream) {
		handler((*Stream)(s))
	})
}

// NewStreamWithPeer creates a new stream on any available connection to p
func (s *Swarm) NewStreamWithPeer(ctx context.Context, p peer.ID) (*Stream, error) {
	// if we have no connections, try connecting.
	if !s.HaveConnsToPeer(p) {
		log.Debug("Swarm: NewStreamWithPeer no connections. Attempting to connect...")
		if _, err := s.Dial(ctx, p); err != nil {
			return nil, err
		}
	}
	log.Debug("Swarm: NewStreamWithPeer...")

	// TODO: think about passing a context down to NewStreamWithGroup
	st, err := s.swarm.NewStreamWithGroup(p)
	return (*Stream)(st), err
}

// ConnectionsToPeer returns all the live connections to p
func (s *Swarm) ConnectionsToPeer(p peer.ID) []*Conn {
	return wrapConns(s.swarm.ConnsWithGroup(p))
}

func (s *Swarm) HaveConnsToPeer(p peer.ID) bool {
	return len(s.swarm.ConnsWithGroup(p)) > 0
}

// Connections returns a slice of all connections.
func (s *Swarm) Connections() []*Conn {
	return wrapConns(s.swarm.Conns())
}

// CloseConnection removes a given peer from swarm + closes the connection
func (s *Swarm) CloseConnection(p peer.ID) error {
	conns := s.swarm.ConnsWithGroup(p) // boom.
	for _, c := range conns {
		c.Close()
	}
	return nil
}

// Peers returns a copy of the set of peers swarm is connected to.
func (s *Swarm) Peers() []peer.ID {
	conns := s.Connections()

	seen := make(map[peer.ID]struct{}, len(conns))
	peers := make([]peer.ID, 0, len(conns))
	for _, c := range conns {
		p := c.RemotePeer()
		if _, found := seen[p]; found {
			continue
		}

		seen[p] = struct{}{}
		peers = append(peers, p)
	}
	return peers
}

// LocalPeer returns the local peer swarm is associated to.
func (s *Swarm) LocalPeer() peer.ID {
	return s.local
}

// Backoff returns the dialbackoff object for this swarm.
func (s *Swarm) Backoff() *dialbackoff {
	return &s.backf
}

// notifyAll sends a signal to all Notifiees
func (s *Swarm) notifyAll(notify func(inet.Notifiee)) {
	s.notifmu.RLock()
	for f := range s.notifs {
		go notify(f)
	}
	s.notifmu.RUnlock()
}

// Notify signs up Notifiee to receive signals when events happen
func (s *Swarm) Notify(f inet.Notifiee) {
	// wrap with our notifiee, to translate function calls
	n := &ps2netNotifee{net: (*Network)(s), not: f}

	s.notifmu.Lock()
	s.notifs[f] = n
	s.notifmu.Unlock()

	// register for notifications in the peer swarm.
	s.swarm.Notify(n)
}

// StopNotify unregisters Notifiee fromr receiving signals
func (s *Swarm) StopNotify(f inet.Notifiee) {
	s.notifmu.Lock()
	n, found := s.notifs[f]
	if found {
		delete(s.notifs, f)
	}
	s.notifmu.Unlock()

	if found {
		s.swarm.StopNotify(n)
	}
}

type ps2netNotifee struct {
	net *Network
	not inet.Notifiee
}

func (n *ps2netNotifee) Connected(c *ps.Conn) {
	n.not.Connected(n.net, inet.Conn((*Conn)(c)))
}

func (n *ps2netNotifee) Disconnected(c *ps.Conn) {
	n.not.Disconnected(n.net, inet.Conn((*Conn)(c)))
}

func (n *ps2netNotifee) OpenedStream(s *ps.Stream) {
	n.not.OpenedStream(n.net, (*Stream)(s))
}

func (n *ps2netNotifee) ClosedStream(s *ps.Stream) {
	n.not.ClosedStream(n.net, (*Stream)(s))
}
