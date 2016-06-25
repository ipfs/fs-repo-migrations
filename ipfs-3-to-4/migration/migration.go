package mg3

import (
	"encoding/base32"
	"fmt"
	"path"
	"strings"
	"time"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/repolock"
	blocks "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blocks"
	util "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"
	dstore "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	flatfs "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/flatfs"
	leveldb "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/leveldb"
	mount "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/mount"
	dsq "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/query"
	sync "github.com/ipfs/fs-repo-migrations/ipfs-2-to-3/Godeps/_workspace/src/github.com/jbenet/go-datastore/sync"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
	log "github.com/ipfs/fs-repo-migrations/stump"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "3-to-4"
}

func (m Migration) Reversible() bool {
	return true
}

type validFunc func(string) bool
type mkKeyFunc func(util.Key) dstore.Key
type txFunc func(dstore.Datastore, []byte, mkKeyFunc) error

func validateNewKey(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return false
	}

	kpart := s[2+len(parts[1]):]
	v, err := base32.StdEncoding.DecodeString(kpart)
	if err == nil && len(v) == 34 {
		return true
	}

	return false
}

func makeOldKey(k util.Key) dstore.Key {
	return dstore.NewKey("/blocks/" + string(k))
}

func makeOldPKKey(k util.Key) dstore.Key {
	return dstore.NewKey("/pk/" + string(k))
}

func validateOldKey(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return false
	}

	kpart := s[2+len(parts[1]):]
	v, err := base32.StdEncoding.DecodeString(kpart)
	if err == nil && len(v) == 34 {
		// already transfered to new format
		return false
	}

	return true
}

func makeNewKey(k util.Key) dstore.Key {
	return dstore.NewKey("/blocks/" + base32.StdEncoding.EncodeToString([]byte(k)))
}

func makeNewPKKey(k util.Key) dstore.Key {
	return dstore.NewKey("/pk/" + base32.StdEncoding.EncodeToString([]byte(k)))
}

func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	log.VLog("locking repo at %q", opts.Path)
	lk, err := lock.Lock2(opts.Path)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)

	log.VLog("  - verifying version is '3'")
	if err := repo.CheckVersion("3"); err != nil {
		return err
	}

	ds, err := openDatastore(opts.Path)
	if err != nil {
		return err
	}

	log.Log("transfering blocks to new key format")
	if err := rewriteKeys(ds, "blocks", makeNewKey, validateOldKey, transferBlock); err != nil {
		return err
	}

	log.Log("transferring stored public key records")
	if err := rewriteKeys(ds, "pk", makeNewPKKey, validateOldKey, transferPubKey); err != nil {
		return err
	}

	err = repo.WriteVersion("4")
	if err != nil {
		return err
	}

	log.Log("updated version file")

	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")
	lk, err := lock.Lock2(opts.Path)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)
	if err := repo.CheckVersion("4"); err != nil {
		return err
	}

	ds, err := openDatastore(opts.Path)
	if err != nil {
		return err
	}

	log.Log("reverting blocks to old key format")
	if err := rewriteKeys(ds, "blocks", makeOldKey, validateNewKey, transferBlock); err != nil {
		return err
	}

	log.Log("reverting stored public key records")
	if err := rewriteKeys(ds, "pk", makeOldPKKey, validateNewKey, transferPubKey); err != nil {
		return err
	}

	// 3) change version number back down
	err = repo.WriteVersion("3")
	if err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Println("lowered version number to 3")
	}

	return nil
}

func openDatastore(repopath string) (dstore.ThreadSafeDatastore, error) {
	log.VLog("  - opening datastore at %q", repopath)
	ldbpath := path.Join(repopath, "datastore")
	ldb, err := leveldb.NewDatastore(ldbpath, nil)
	if err != nil {
		return nil, err
	}

	blockspath := path.Join(repopath, "blocks")
	fds, err := flatfs.New(blockspath, 4)
	if err != nil {
		return nil, err
	}

	return sync.MutexWrap(mount.New([]mount.Mount{
		{
			Prefix:    dstore.NewKey("/blocks"),
			Datastore: fds,
		},
		{
			Prefix:    dstore.NewKey("/"),
			Datastore: ldb,
		},
	})), nil
}

func rewriteKeys(ds dstore.Datastore, pref string, mkKey mkKeyFunc, valid validFunc, transfer txFunc) error {

	log.Log("gathering keys...")
	res, err := ds.Query(dsq.Query{
		Prefix:   pref,
		KeysOnly: true,
	})
	if err != nil {
		return err
	}

	entries, err := res.Rest()
	if err != nil {
		return err
	}

	log.Log("got %d keys, beginning transfer. This will take some time.", len(entries))

	before := time.Now()
	var skipped int
	for i, e := range entries {
		fmt.Printf("\r[%d / %d]", i, len(entries))
		if skipped > 0 {
			fmt.Printf(" (skipped: %d)", skipped)
		}
		if i%10 == 9 {
			took := time.Now().Sub(before)
			av := took / time.Duration(i)
			estim := av * time.Duration(len(entries)-i)
			est := strings.Split(estim.String(), ".")[0]

			fmt.Printf("  Approx time remaining: %ss  ", est)
		}

		if !valid(e.Key) {
			skipped++
			continue
		}

		curk := dstore.NewKey(e.Key)
		blk, err := ds.Get(curk)
		if err != nil {
			return err
		}

		blkd, ok := blk.([]byte)
		if !ok {
			log.Error("data %q was not a []byte", e.Key)
			continue
		}

		err = transfer(ds, blkd, mkKey)
		if err != nil {
			return err
		}

		err = ds.Delete(curk)
		if err != nil {
			return err
		}
	}

	return nil
}

func transferBlock(ds dstore.Datastore, data []byte, mkKey mkKeyFunc) error {
	b := blocks.NewBlock(data)
	dsk := mkKey(b.Key())
	err := ds.Put(dsk, b.Data)
	if err != nil {
		return err
	}

	return nil
}

func transferPubKey(ds dstore.Datastore, data []byte, mkKey mkKeyFunc) error {
	k := util.Key(util.Hash(data))
	dsk := mkKey(k)
	return ds.Put(dsk, data)
}
