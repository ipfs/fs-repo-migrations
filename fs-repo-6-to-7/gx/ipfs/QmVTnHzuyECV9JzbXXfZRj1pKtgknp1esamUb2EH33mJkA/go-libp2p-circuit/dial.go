package relay

import (
	"context"
	"fmt"
	"math/rand"

	tpt "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmVxtCwKFMmwcjhQXsGj6m4JAW7nGb9hRoErH9jpgqcLxA/go-libp2p-transport"
	ma "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	peer "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

var _ tpt.Dialer = (*RelayDialer)(nil)

type RelayDialer Relay

func (d *RelayDialer) Relay() *Relay {
	return (*Relay)(d)
}

func (r *Relay) Dialer() *RelayDialer {
	return (*RelayDialer)(r)
}

func (d *RelayDialer) Dial(a ma.Multiaddr) (tpt.Conn, error) {
	return d.DialContext(d.ctx, a)
}

func (d *RelayDialer) DialContext(ctx context.Context, a ma.Multiaddr) (tpt.Conn, error) {
	if !d.Matches(a) {
		return nil, fmt.Errorf("%s is not a relay address", a)
	}
	parts := ma.Split(a)

	spl, _ := ma.NewMultiaddr("/p2p-circuit")

	var relayaddr, destaddr ma.Multiaddr
	for i, p := range parts {
		if p.Equal(spl) {
			relayaddr = ma.Join(parts[:i]...)
			destaddr = ma.Join(parts[i+1:]...)
			break
		}
	}

	dinfo, err := pstore.InfoFromP2pAddr(destaddr)
	if err != nil {
		return nil, err
	}

	if len(relayaddr.Bytes()) == 0 {
		// unspecific relay address, try dialing using known hop relays
		return d.tryDialRelays(ctx, *dinfo)
	}

	rinfo, err := pstore.InfoFromP2pAddr(relayaddr)
	if err != nil {
		return nil, err
	}

	return d.Relay().DialPeer(ctx, *rinfo, *dinfo)
}

func (d *RelayDialer) tryDialRelays(ctx context.Context, dinfo pstore.PeerInfo) (tpt.Conn, error) {
	var relays []peer.ID
	d.mx.Lock()
	for p := range d.relays {
		relays = append(relays, p)
	}
	d.mx.Unlock()

	// shuffle list of relays, avoid overloading a specific relay
	for i := range relays {
		j := rand.Intn(i + 1)
		relays[i], relays[j] = relays[j], relays[i]
	}

	for _, relay := range relays {
		if len(d.host.Network().ConnsToPeer(relay)) == 0 {
			continue
		}

		rctx, cancel := context.WithTimeout(ctx, HopConnectTimeout)
		c, err := d.Relay().DialPeer(rctx, pstore.PeerInfo{ID: relay}, dinfo)
		cancel()

		if err == nil {
			return c, nil
		}

		log.Debugf("error opening relay connection through %s: %s", dinfo.ID, err.Error())
	}

	return nil, fmt.Errorf("Failed to dial through %d known relay hosts", len(relays))
}

func (d *RelayDialer) Matches(a ma.Multiaddr) bool {
	_, err := a.ValueForProtocol(P_CIRCUIT)
	return err == nil
}
