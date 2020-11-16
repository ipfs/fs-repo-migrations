package nat

import (
	"net"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/libp2p/go-netroute"
)

func getDefaultGateway() (net.IP, error) {
	router, err := netroute.New()
	if err != nil {
		return nil, err
	}

	_, ip, _, err := router.Route(net.IPv4zero)
	return ip, err
}
