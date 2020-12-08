package swarm

import (
	"fmt"
	"strings"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-libp2p-core/transport"

	ma "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/multiformats/go-multiaddr"
)

// TransportForDialing retrieves the appropriate transport for dialing the given
// multiaddr.
func (s *Swarm) TransportForDialing(a ma.Multiaddr) transport.Transport {
	protocols := a.Protocols()
	if len(protocols) == 0 {
		return nil
	}

	s.transports.RLock()
	defer s.transports.RUnlock()
	if len(s.transports.m) == 0 {
		log.Error("you have no transports configured")
		return nil
	}

	for _, p := range protocols {
		transport, ok := s.transports.m[p.Code]
		if !ok {
			continue
		}
		if transport.Proxy() {
			return transport
		}
	}

	return s.transports.m[protocols[len(protocols)-1].Code]
}

// TransportForListening retrieves the appropriate transport for listening on
// the given multiaddr.
func (s *Swarm) TransportForListening(a ma.Multiaddr) transport.Transport {
	protocols := a.Protocols()
	if len(protocols) == 0 {
		return nil
	}

	s.transports.RLock()
	defer s.transports.RUnlock()
	if len(s.transports.m) == 0 {
		log.Error("you have no transports configured")
		return nil
	}

	selected := s.transports.m[protocols[len(protocols)-1].Code]
	for _, p := range protocols {
		transport, ok := s.transports.m[p.Code]
		if !ok {
			continue
		}
		if transport.Proxy() {
			selected = transport
		}
	}
	return selected
}

// AddTransport adds a transport to this swarm.
//
// Satisfies the Network interface from go-libp2p-transport.
func (s *Swarm) AddTransport(t transport.Transport) error {
	protocols := t.Protocols()

	if len(protocols) == 0 {
		return fmt.Errorf("useless transport handles no protocols: %T", t)
	}

	s.transports.Lock()
	defer s.transports.Unlock()
	var registered []string
	for _, p := range protocols {
		if _, ok := s.transports.m[p]; ok {
			proto := ma.ProtocolWithCode(p)
			name := proto.Name
			if name == "" {
				name = fmt.Sprintf("unknown (%d)", p)
			}
			registered = append(registered, name)
		}
	}
	if len(registered) > 0 {
		return fmt.Errorf(
			"transports already registered for protocol(s): %s",
			strings.Join(registered, ", "),
		)
	}

	for _, p := range protocols {
		s.transports.m[p] = t
	}
	return nil
}