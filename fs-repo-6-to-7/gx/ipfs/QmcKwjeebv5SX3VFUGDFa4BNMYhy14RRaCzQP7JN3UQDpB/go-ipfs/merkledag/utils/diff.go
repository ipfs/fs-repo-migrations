package dagutils

import (
	"context"
	"fmt"
	"path"

	dag "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/merkledag"

	cid "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// These constants define the changes that can be applied to a DAG.
const (
	Add = iota
	Remove
	Mod
)

// Change represents a change to a DAG and contains a reference to the old and
// new CIDs.
type Change struct {
	Type   int
	Path   string
	Before *cid.Cid
	After  *cid.Cid
}

// String prints a human-friendly line about a change.
func (c *Change) String() string {
	switch c.Type {
	case Add:
		return fmt.Sprintf("Added %s at %s", c.After.String(), c.Path)
	case Remove:
		return fmt.Sprintf("Removed %s from %s", c.Before.String(), c.Path)
	case Mod:
		return fmt.Sprintf("Changed %s to %s at %s", c.Before.String(), c.After.String(), c.Path)
	default:
		panic("nope")
	}
}

// ApplyChange applies the requested changes to the given node in the given dag.
func ApplyChange(ctx context.Context, ds ipld.DAGService, nd *dag.ProtoNode, cs []*Change) (*dag.ProtoNode, error) {
	e := NewDagEditor(nd, ds)
	for _, c := range cs {
		switch c.Type {
		case Add:
			child, err := ds.Get(ctx, c.After)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*dag.ProtoNode)
			if !ok {
				return nil, dag.ErrNotProtobuf
			}

			err = e.InsertNodeAtPath(ctx, c.Path, childpb, nil)
			if err != nil {
				return nil, err
			}

		case Remove:
			err := e.RmLink(ctx, c.Path)
			if err != nil {
				return nil, err
			}

		case Mod:
			err := e.RmLink(ctx, c.Path)
			if err != nil {
				return nil, err
			}
			child, err := ds.Get(ctx, c.After)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*dag.ProtoNode)
			if !ok {
				return nil, dag.ErrNotProtobuf
			}

			err = e.InsertNodeAtPath(ctx, c.Path, childpb, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	return e.Finalize(ctx, ds)
}

// Diff returns a set of changes that transform node 'a' into node 'b'
func Diff(ctx context.Context, ds ipld.DAGService, a, b ipld.Node) ([]*Change, error) {
	// Base case where both nodes are leaves, just compare
	// their CIDs.
	if len(a.Links()) == 0 && len(b.Links()) == 0 {
		if a.Cid().Equals(b.Cid()) {
			return []*Change{}, nil
		}
		return []*Change{
			&Change{
				Type:   Mod,
				Before: a.Cid(),
				After:  b.Cid(),
			},
		}, nil
	}

	var out []*Change
	cleanA := a.Copy().(*dag.ProtoNode)
	cleanB := b.Copy().(*dag.ProtoNode)

	// strip out unchanged stuff
	for _, lnk := range a.Links() {
		l, _, err := b.ResolveLink([]string{lnk.Name})
		if err == nil {
			if l.Cid.Equals(lnk.Cid) {
				// no change... ignore it
			} else {
				anode, err := lnk.GetNode(ctx, ds)
				if err != nil {
					return nil, err
				}

				bnode, err := l.GetNode(ctx, ds)
				if err != nil {
					return nil, err
				}

				anodepb, ok := anode.(*dag.ProtoNode)
				if !ok {
					return nil, dag.ErrNotProtobuf
				}

				bnodepb, ok := bnode.(*dag.ProtoNode)
				if !ok {
					return nil, dag.ErrNotProtobuf
				}

				sub, err := Diff(ctx, ds, anodepb, bnodepb)
				if err != nil {
					return nil, err
				}

				for _, subc := range sub {
					subc.Path = path.Join(lnk.Name, subc.Path)
					out = append(out, subc)
				}
			}
			cleanA.RemoveNodeLink(l.Name)
			cleanB.RemoveNodeLink(l.Name)
		}
	}

	for _, lnk := range cleanA.Links() {
		out = append(out, &Change{
			Type:   Remove,
			Path:   lnk.Name,
			Before: lnk.Cid,
		})
	}
	for _, lnk := range cleanB.Links() {
		out = append(out, &Change{
			Type:  Add,
			Path:  lnk.Name,
			After: lnk.Cid,
		})
	}

	return out, nil
}

// Conflict represents two incompatible changes and is returned by MergeDiffs().
type Conflict struct {
	A *Change
	B *Change
}

// MergeDiffs takes two slice of changes and adds them to a single slice.
// When a Change from b happens to the same path of an existing change in a,
// a conflict is created and b is not added to the merged slice.
// A slice of Conflicts is returned and contains pointers to the
// Changes involved (which share the same path).
func MergeDiffs(a, b []*Change) ([]*Change, []Conflict) {
	var out []*Change
	var conflicts []Conflict
	paths := make(map[string]*Change)
	for _, c := range a {
		paths[c.Path] = c
	}

	for _, c := range b {
		if ca, ok := paths[c.Path]; ok {
			conflicts = append(conflicts, Conflict{
				A: ca,
				B: c,
			})
		} else {
			out = append(out, c)
		}
	}
	for _, c := range paths {
		out = append(out, c)
	}
	return out, conflicts
}
