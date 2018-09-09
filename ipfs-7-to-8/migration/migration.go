package migration

import (
	"fmt"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	log "github.com/ipfs/fs-repo-migrations/stump"

	dshelp "gx/ipfs/QmTmqJGRQfuH8eKWD1FjThwPRipt1QhqJQNZ8MpzmfAAxo/go-ipfs-ds-help"
	record "gx/ipfs/QmUpttFinNDmNPgFwKN8sZK6BUtBmA68Y4KdSBDXa8t9sJ/go-libp2p-record"
	dhtpb "gx/ipfs/QmUpttFinNDmNPgFwKN8sZK6BUtBmA68Y4KdSBDXa8t9sJ/go-libp2p-record/pb"
	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	namesys "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/namesys"
	repo "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo"
	fsrepo "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo/fsrepo"
	mfsr "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo/fsrepo/migrations"
	pin "gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/pin"
	base32 "gx/ipfs/QmfVj3x4D6Jkq9SEoi5n2NmoUomLwoeiwnYz2KQa15wRw6/base32"
	ds "gx/ipfs/QmVG5gxteQNEMhrS8prJSmU2C9rebtFuTd3SYZ5kE3YZ5k/go-datastore"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "7-to-8"
}

func (m Migration) Reversible() bool {
	return true
}


func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	r, err := fsrepo.Open(opts.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	dstore := r.Datastore()
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))
	dserv := mdag.NewDAGService(bserv)
	
	pinner := pin.NewPinner(dstore, dserv, dserv)

	addPin = func (path string, c *cid.Cid, recursive bool) error {
		if len(path) == 0 || path[len(path)-1] == '/' {
			path += c.String()
		}
		prefix := "direct"
		if recursive {
			prefix = "recursive"
		}
		pinPath := "/local/pins" + "/" + prefix + "/" + path
		return dstore.Put(ds.NewKey(pinPath), c.Bytes())
	}

	
	for directKey := pinner.DirectKeys() {
		if (err = addPin("migrated/", directKey, false)) != nil {
			return err
		}
	}
	
	for recursiveKey := pinner.DirectKeys() {
		if (err = addPin("migrated/", recursiveKey, true)) != nil {
			return err
		}
	}
	
	if (err = dstore.Delete(ds.NewKey("/local/pins")) != nil {
		return err
	}
	
	err = mfsr.RepoPath(opts.Path).WriteVersion(8)
	if err != nil {
		log.Error("failed to update version file to 8")
		return err
	}

	log.Log("updated version file")

	return nil
}

func revertForKey(dstore ds.Datastore, sk ci.PrivKey, k ci.PrivKey) error {
	
	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")

	r, err := fsrepo.Open(opts.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	dstore := r.Datastore()
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))
	dserv := mdag.NewDAGService(bserv)
	
	pinner := pin.NewPinner(dstore, dserv, dserv)
	
	// We're downgrading from version 7.
	fsrepo.RepoVersion = 8
	r, err := fsrepo.Open(opts.Path)
	fsrepo.RepoVersion = 7
	if err != nil {
		return err
	}
	defer r.Close()

	result, err := dstore.Query(dsq.Query{Prefix: "/local/pins/direct/"})
	
	if err != nil {
		return nil, err
	}
	
	for entry := range result.Next() {
		c, err := cid.Cast(entry.Value)
		if err != nil {
			return nil, err
		}
		if !pinner.directPin.Has(c) {
			pinner.directPin.Add(c)
		}
		dstore.Delete(entry.Key)
	}
	
	result, err := dstore.Query(dsq.Query{Prefix: "/local/pins/recursive/"})
	
	if err != nil {
		return nil, err
	}
	
	for entry := range result.Next() {
		c, err := cid.Cast(entry.Value)
		if err != nil {
			return nil, err
		}
		if pinner.directPin.Has(c) {
			pinner.directPin.Remove(c)
		}
		if !pinner.recursivePin.Has(c) {
			pinner.recursivePin.Add(c)
		}
		dstore.Delete(entry.Key)
	}
	
	pinner.Flush()

	err = mfsr.RepoPath(opts.Path).WriteVersion(7)
	if err != nil {
		log.Error("failed to downgrade version file to 7")
		return err
	}

	log.Log("updated version file")

	return nil
}
