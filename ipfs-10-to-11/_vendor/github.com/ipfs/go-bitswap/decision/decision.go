package decision

import intdec "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-bitswap/internal/decision"

// Expose Receipt externally
type Receipt = intdec.Receipt

// Expose ScoreLedger externally
type ScoreLedger = intdec.ScoreLedger

// Expose ScorePeerFunc externally
type ScorePeerFunc = intdec.ScorePeerFunc
