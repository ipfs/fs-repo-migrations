package peerset

import (
	peer "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/ipfs/go-ipfs/p2p/peer"
	"sync"
)

// PeerSet is a threadsafe set of peers
type PeerSet struct {
	ps   map[peer.ID]struct{}
	lk   sync.RWMutex
	size int
}

func New() *PeerSet {
	ps := new(PeerSet)
	ps.ps = make(map[peer.ID]struct{})
	ps.size = -1
	return ps
}

func NewLimited(size int) *PeerSet {
	ps := new(PeerSet)
	ps.ps = make(map[peer.ID]struct{})
	ps.size = size
	return ps
}

func (ps *PeerSet) Add(p peer.ID) {
	ps.lk.Lock()
	ps.ps[p] = struct{}{}
	ps.lk.Unlock()
}

func (ps *PeerSet) Contains(p peer.ID) bool {
	ps.lk.RLock()
	_, ok := ps.ps[p]
	ps.lk.RUnlock()
	return ok
}

func (ps *PeerSet) Size() int {
	ps.lk.RLock()
	defer ps.lk.RUnlock()
	return len(ps.ps)
}

// TryAdd Attempts to add the given peer into the set.
// This operation can fail for one of two reasons:
// 1) The given peer is already in the set
// 2) The number of peers in the set is equal to size
func (ps *PeerSet) TryAdd(p peer.ID) bool {
	var success bool
	ps.lk.Lock()
	if _, ok := ps.ps[p]; !ok && (len(ps.ps) < ps.size || ps.size == -1) {
		success = true
		ps.ps[p] = struct{}{}
	}
	ps.lk.Unlock()
	return success
}
