package websocket

import (
	"context"

	manet "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmRK2LxanhK2gZq6k6R7vk5ZoYZk8ULSSTB7FzDsMUX6CB/go-multiaddr-net"
	tpt "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmVxtCwKFMmwcjhQXsGj6m4JAW7nGb9hRoErH9jpgqcLxA/go-libp2p-transport"
	ma "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	ws "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZH5VXfAJouGMyCCHTRPGCT3e5MG9Lu78Ln3YAYW1XTts/websocket"
)

type dialer struct{}

func (d *dialer) Dial(raddr ma.Multiaddr) (tpt.Conn, error) {
	return d.DialContext(context.Background(), raddr)
}

func (d *dialer) DialContext(ctx context.Context, raddr ma.Multiaddr) (tpt.Conn, error) {
	wsurl, err := parseMultiaddr(raddr)
	if err != nil {
		return nil, err
	}

	wscon, _, err := ws.DefaultDialer.Dial(wsurl, nil)
	if err != nil {
		return nil, err
	}

	mnc, err := manet.WrapNetConn(NewConn(wscon, nil))
	if err != nil {
		wscon.Close()
		return nil, err
	}

	return &wsConn{
		Conn: mnc,
	}, nil
}

func (d *dialer) Matches(a ma.Multiaddr) bool {
	return WsFmt.Matches(a)
}
