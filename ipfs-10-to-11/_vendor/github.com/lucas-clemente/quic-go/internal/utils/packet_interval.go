package utils

import "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/lucas-clemente/quic-go/internal/protocol"

// PacketInterval is an interval from one PacketNumber to the other
type PacketInterval struct {
	Start protocol.PacketNumber
	End   protocol.PacketNumber
}
