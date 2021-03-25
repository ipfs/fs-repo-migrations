package pin

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"sort"
	"unsafe"

	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/gogo/protobuf/proto"
	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/merkledag"
	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/pin/internal/pb"
	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"
	"github.com/ipfs/fs-repo-migrations/fs-repo-2-to-3/Godeps/_workspace/src/golang.org/x/net/context"
)

const (
	defaultFanout = 256
	maxItems      = 8192
)

func randomSeed() (uint32, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

func hash(seed uint32, k util.Key) uint32 {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], seed)
	h := fnv.New32a()
	_, _ = h.Write(buf[:])
	_, _ = io.WriteString(h, string(k))
	return h.Sum32()
}

type itemIterator func() (k util.Key, data []byte, ok bool)

type keyObserver func(util.Key)

// refcount is the marshaled format of refcounts. It may change
// between versions; this is valid for version 1. Changing it may
// become desirable if there are many links with refcount > 255.
//
// There are two guarantees that need to be preserved, if this is
// changed:
//
//     - the marshaled format is of fixed size, matching
//       unsafe.Sizeof(refcount(0))
//     - methods of refcount handle endianness, and may
//       in later versions need encoding/binary.
type refcount uint8

func (r refcount) Bytes() []byte {
	return []byte{byte(r)}
}

// readRefcount returns the idx'th refcount in []byte, which is
// assumed to be a sequence of refcount.Bytes results.
func (r *refcount) ReadFromIdx(buf []byte, idx int) {
	*r = refcount(buf[idx])
}

type sortByHash struct {
	links []*merkledag.Link
	data  []byte
}

func (s sortByHash) Len() int {
	return len(s.links)
}

func (s sortByHash) Less(a, b int) bool {
	return bytes.Compare(s.links[a].Hash, s.links[b].Hash) == -1
}

func (s sortByHash) Swap(a, b int) {
	s.links[a], s.links[b] = s.links[b], s.links[a]
	if len(s.data) != 0 {
		const n = int(unsafe.Sizeof(refcount(0)))
		tmp := make([]byte, n)
		copy(tmp, s.data[a*n:a*n+n])
		copy(s.data[a*n:a*n+n], s.data[b*n:b*n+n])
		copy(s.data[b*n:b*n+n], tmp)
	}
}

func storeItems(ctx context.Context, dag merkledag.DAGService, estimatedLen uint64, iter itemIterator, internalKeys keyObserver) (*merkledag.Node, error) {
	seed, err := randomSeed()
	if err != nil {
		return nil, err
	}
	n := &merkledag.Node{
		Links: make([]*merkledag.Link, 0, defaultFanout+maxItems),
	}
	for i := 0; i < defaultFanout; i++ {
		n.Links = append(n.Links, &merkledag.Link{Hash: emptyKey.ToMultihash()})
	}
	internalKeys(emptyKey)
	hdr := &pb.Set{
		Version: proto.Uint32(1),
		Fanout:  proto.Uint32(defaultFanout),
		Seed:    proto.Uint32(seed),
	}
	if err := writeHdr(n, hdr); err != nil {
		return nil, err
	}
	hdrLen := len(n.Data)

	if estimatedLen < maxItems {
		// it'll probably fit
		for i := 0; i < maxItems; i++ {
			k, data, ok := iter()
			if !ok {
				// all done
				break
			}
			n.Links = append(n.Links, &merkledag.Link{Hash: k.ToMultihash()})
			n.Data = append(n.Data, data...)
		}
		// sort by hash, also swap item Data
		s := sortByHash{
			links: n.Links[defaultFanout:],
			data:  n.Data[hdrLen:],
		}
		sort.Stable(s)
	}

	// wasteful but simple
	type item struct {
		k    util.Key
		data []byte
	}
	hashed := make(map[uint32][]item)
	for {
		k, data, ok := iter()
		if !ok {
			break
		}
		h := hash(seed, k)
		hashed[h] = append(hashed[h], item{k, data})
	}
	for h, items := range hashed {
		childIter := func() (k util.Key, data []byte, ok bool) {
			if len(items) == 0 {
				return "", nil, false
			}
			first := items[0]
			items = items[1:]
			return first.k, first.data, true
		}
		child, err := storeItems(ctx, dag, uint64(len(items)), childIter, internalKeys)
		if err != nil {
			return nil, err
		}
		size, err := child.Size()
		if err != nil {
			return nil, err
		}
		childKey, err := dag.Add(child)
		if err != nil {
			return nil, err
		}
		internalKeys(childKey)
		l := &merkledag.Link{
			Name: "",
			Hash: childKey.ToMultihash(),
			Size: size,
			Node: child,
		}
		n.Links[int(h%defaultFanout)] = l
	}
	return n, nil
}

func readHdr(n *merkledag.Node) (*pb.Set, []byte, error) {
	hdrLenRaw, consumed := binary.Uvarint(n.Data)
	if consumed <= 0 {
		return nil, nil, errors.New("invalid Set header length")
	}
	buf := n.Data[consumed:]
	if hdrLenRaw > uint64(len(buf)) {
		return nil, nil, errors.New("impossibly large Set header length")
	}
	// as hdrLenRaw was <= an int, we now know it fits in an int
	hdrLen := int(hdrLenRaw)
	var hdr pb.Set
	if err := proto.Unmarshal(buf[:hdrLen], &hdr); err != nil {
		return nil, nil, err
	}
	buf = buf[hdrLen:]

	if v := hdr.GetVersion(); v != 1 {
		return nil, nil, fmt.Errorf("unsupported Set version: %d", v)
	}
	if uint64(hdr.GetFanout()) > uint64(len(n.Links)) {
		return nil, nil, errors.New("impossibly large Fanout")
	}
	return &hdr, buf, nil
}

func writeHdr(n *merkledag.Node, hdr *pb.Set) error {
	hdrData, err := proto.Marshal(hdr)
	if err != nil {
		return err
	}
	n.Data = make([]byte, binary.MaxVarintLen64, binary.MaxVarintLen64+len(hdrData))
	written := binary.PutUvarint(n.Data, uint64(len(hdrData)))
	n.Data = n.Data[:written]
	n.Data = append(n.Data, hdrData...)
	return nil
}

type walkerFunc func(buf []byte, idx int, link *merkledag.Link) error

func walkItems(ctx context.Context, dag merkledag.DAGService, n *merkledag.Node, fn walkerFunc, children keyObserver) error {
	hdr, buf, err := readHdr(n)
	if err != nil {
		return err
	}
	// readHdr guarantees fanout is a safe value
	fanout := hdr.GetFanout()
	for i, l := range n.Links[fanout:] {
		if err := fn(buf, i, l); err != nil {
			return err
		}
	}
	for _, l := range n.Links[:fanout] {
		children(util.Key(l.Hash))
		if util.Key(l.Hash) == emptyKey {
			continue
		}
		subtree, err := l.GetNode(ctx, dag)
		if err != nil {
			return err
		}
		if err := walkItems(ctx, dag, subtree, fn, children); err != nil {
			return err
		}
	}
	return nil
}

func loadSet(ctx context.Context, dag merkledag.DAGService, root *merkledag.Node, name string, internalKeys keyObserver) ([]util.Key, error) {
	l, err := root.GetNodeLink(name)
	if err != nil {
		return nil, err
	}
	internalKeys(util.Key(l.Hash))
	n, err := l.GetNode(ctx, dag)
	if err != nil {
		return nil, err
	}

	var res []util.Key
	walk := func(buf []byte, idx int, link *merkledag.Link) error {
		res = append(res, util.Key(link.Hash))
		return nil
	}
	if err := walkItems(ctx, dag, n, walk, internalKeys); err != nil {
		return nil, err
	}
	return res, nil
}

func loadMultiset(ctx context.Context, dag merkledag.DAGService, root *merkledag.Node, name string, internalKeys keyObserver) (map[util.Key]uint64, error) {
	l, err := root.GetNodeLink(name)
	if err != nil {
		return nil, err
	}
	internalKeys(util.Key(l.Hash))
	n, err := l.GetNode(ctx, dag)
	if err != nil {
		return nil, err
	}

	refcounts := make(map[util.Key]uint64)
	walk := func(buf []byte, idx int, link *merkledag.Link) error {
		var r refcount
		r.ReadFromIdx(buf, idx)
		refcounts[util.Key(link.Hash)] += uint64(r)
		return nil
	}
	if err := walkItems(ctx, dag, n, walk, internalKeys); err != nil {
		return nil, err
	}
	return refcounts, nil
}

func storeSet(ctx context.Context, dag merkledag.DAGService, keys []util.Key, internalKeys keyObserver) (*merkledag.Node, error) {
	iter := func() (k util.Key, data []byte, ok bool) {
		if len(keys) == 0 {
			return "", nil, false
		}
		first := keys[0]
		keys = keys[1:]
		return first, nil, true
	}
	n, err := storeItems(ctx, dag, uint64(len(keys)), iter, internalKeys)
	if err != nil {
		return nil, err
	}
	k, err := dag.Add(n)
	if err != nil {
		return nil, err
	}
	internalKeys(k)
	return n, nil
}

func storeMultiset(ctx context.Context, dag merkledag.DAGService, refcounts map[util.Key]uint64, internalKeys keyObserver) (*merkledag.Node, error) {
	iter := func() (k util.Key, data []byte, ok bool) {
		// Every call of this function returns the next refcount item.
		//
		// This function splits out the uint64 reference counts as
		// smaller increments, as fits in type refcount. Most of the
		// time the refcount will fit inside just one, so this saves
		// space.
		//
		// We use range here to pick an arbitrary item in the map, but
		// not really iterate the map.
		for k, refs := range refcounts {
			num := ^refcount(0)
			if refs <= uint64(num) {
				// Remaining count fits in a single item; remove the
				// key from the map.
				num = refcount(refs)
				delete(refcounts, k)
			} else {
				// Count is too large to fit in one item, the key will
				// repeat in some later call.
				refcounts[k] -= uint64(num)
			}
			return k, num.Bytes(), true
		}
		return "", nil, false
	}
	n, err := storeItems(ctx, dag, uint64(len(refcounts)), iter, internalKeys)
	if err != nil {
		return nil, err
	}
	k, err := dag.Add(n)
	if err != nil {
		return nil, err
	}
	internalKeys(k)
	return n, nil
}
