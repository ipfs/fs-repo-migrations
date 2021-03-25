package mg6

import (
	"fmt"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"

	dshelp "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmTmqJGRQfuH8eKWD1FjThwPRipt1QhqJQNZ8MpzmfAAxo/go-ipfs-ds-help"
	record "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmUpttFinNDmNPgFwKN8sZK6BUtBmA68Y4KdSBDXa8t9sJ/go-libp2p-record"
	dhtpb "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmUpttFinNDmNPgFwKN8sZK6BUtBmA68Y4KdSBDXa8t9sJ/go-libp2p-record/pb"
	ds "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	proto "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	peer "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	ci "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	namesys "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/namesys"
	repo "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo"
	fsrepo "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo/fsrepo"
	mfsr "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmcKwjeebv5SX3VFUGDFa4BNMYhy14RRaCzQP7JN3UQDpB/go-ipfs/repo/fsrepo/migrations"
	base32 "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmfVj3x4D6Jkq9SEoi5n2NmoUomLwoeiwnYz2KQa15wRw6/base32"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "6-to-7"
}

func (m Migration) Reversible() bool {
	return true
}

func myKey(r repo.Repo) (ci.PrivKey, error) {
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}

	sk, err := cfg.Identity.DecodePrivateKey("passphrase todo!")
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		return nil, err
	}
	idCfg, err := peer.IDB58Decode(cfg.Identity.PeerID)
	if err != nil {
		return nil, err
	}

	if pid != idCfg {
		return nil, fmt.Errorf(
			"private key in config does not match id: %s != %s",
			pid,
			idCfg,
		)
	}
	return sk, nil
}

func applyForKey(dstore ds.Datastore, k ci.PrivKey) error {
	id, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %s", err)
	}
	_, ipns := namesys.IpnsKeysForID(id)
	record, err := dstore.Get(dshelp.NewKeyFromBinary([]byte(ipns)))
	if err == ds.ErrNotFound {
		log.VLog("no IPNS record for key found")
		return nil
	}
	if err != nil {
		return fmt.Errorf("datastore error: %s", err)
	}

	recordbytes, ok := record.([]byte)
	if !ok {
		return fmt.Errorf("unexpected type returned from datastore: %#v", record)
	}
	dhtrec := new(dhtpb.Record)
	err = proto.Unmarshal(recordbytes, dhtrec)
	if err != nil {
		return fmt.Errorf("failed to decode DHT record: %s", err)
	}

	val := dhtrec.GetValue()
	newkey := ds.NewKey("/ipns/" + base32.RawStdEncoding.EncodeToString([]byte(id)))
	err = dstore.Put(newkey, val)
	if err != nil {
		return fmt.Errorf("failed to write new IPNS record: %s", err)
	}
	return nil
}

func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	r, err := fsrepo.Open(opts.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	ks := r.Keystore()
	keys, err := ks.List()
	if err != nil {
		return err
	}

	dstore := r.Datastore()

	sk, err := myKey(r)
	if err != nil {
		return err
	}

	log.VLog("migrating IPNS record for key: self")
	err = applyForKey(dstore, sk)
	if err != nil {
		return err
	}

	for _, keyName := range keys {
		log.VLog("migrating IPNS record for key:", keyName)
		k, err := ks.Get(keyName)
		if err != nil {
			return err
		}
		err = applyForKey(dstore, k)
		if err != nil {
			return err
		}
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion(7)
	if err != nil {
		log.Error("failed to update version file to 7")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 6 to 7 succeeded")
	return nil
}

func revertForKey(dstore ds.Datastore, sk ci.PrivKey, k ci.PrivKey) error {
	id, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %s", err)
	}

	_, ipns := namesys.IpnsKeysForID(id)

	newkey := ds.NewKey("/ipns/" + base32.RawStdEncoding.EncodeToString([]byte(id)))
	val, err := dstore.Get(newkey)
	if err == ds.ErrNotFound {
		log.VLog("no IPNS record for key found")
		return nil
	}
	if err != nil {
		return fmt.Errorf("datastore error: %s", err)
	}
	value, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("unexpected type returned from datastore: %#v", val)
	}

	dhtrec, err := record.MakePutRecord(sk, ipns, value, true)
	if err != nil {
		return fmt.Errorf("failed to create DHT record: %s", err)
	}

	data, err := proto.Marshal(dhtrec)
	if err != nil {
		return fmt.Errorf("failed to marshal DHT record: %s", err)
	}

	err = dstore.Put(dshelp.NewKeyFromBinary([]byte(ipns)), data)
	if err != nil {
		return fmt.Errorf("failed to write DHT record: %s", err)
	}
	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")

	// We're downgrading from version 7.
	fsrepo.RepoVersion = 7
	r, err := fsrepo.Open(opts.Path)
	fsrepo.RepoVersion = 6
	if err != nil {
		return err
	}
	defer r.Close()

	log.VLog("decoding private key")

	sk, err := myKey(r)
	if err != nil {
		return err
	}

	ks := r.Keystore()
	keys, err := ks.List()
	if err != nil {
		return err
	}

	dstore := r.Datastore()

	log.VLog("migrating IPNS record for key: self")
	revertForKey(dstore, sk, sk)

	for _, keyName := range keys {
		log.VLog("migrating IPNS record for key:", keyName)
		k, err := ks.Get(keyName)
		if err != nil {
			return err
		}
		revertForKey(dstore, sk, k)
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion(6)
	if err != nil {
		log.Error("failed to downgrade version file to 6")
		return err
	}

	log.Log("updated version file")

	return nil
}
