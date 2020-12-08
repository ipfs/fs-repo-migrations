package typegen

import (
	"io"

	cid "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-cid"
)

type CborCid cid.Cid

func (c *CborCid) MarshalCBOR(w io.Writer) error {
	return WriteCid(w, cid.Cid(*c))
}

func (c *CborCid) UnmarshalCBOR(r io.Reader) error {
	oc, err := ReadCid(r)
	if err != nil {
		return err
	}
	*c = CborCid(oc)
	return nil
}
