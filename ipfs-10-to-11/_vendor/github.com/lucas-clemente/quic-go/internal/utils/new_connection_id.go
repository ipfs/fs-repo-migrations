package utils

import (
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/lucas-clemente/quic-go/internal/protocol"
)

// NewConnectionID is a new connection ID
type NewConnectionID struct {
	SequenceNumber      uint64
	ConnectionID        protocol.ConnectionID
	StatelessResetToken protocol.StatelessResetToken
}
