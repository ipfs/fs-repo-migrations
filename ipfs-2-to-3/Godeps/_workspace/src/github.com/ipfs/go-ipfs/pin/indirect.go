package pin

import "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"

type indirectPin struct {
	refCounts map[util.Key]uint64
}

func newIndirectPin() *indirectPin {
	return &indirectPin{
		refCounts: make(map[util.Key]uint64),
	}
}

func (i *indirectPin) Increment(k util.Key) {
	i.refCounts[k]++
}

func (i *indirectPin) Decrement(k util.Key) {
	i.refCounts[k]--
	if i.refCounts[k] == 0 {
		delete(i.refCounts, k)
	}
}

func (i *indirectPin) HasKey(k util.Key) bool {
	_, found := i.refCounts[k]
	return found
}

func (i *indirectPin) GetRefs() map[util.Key]uint64 {
	return i.refCounts
}
