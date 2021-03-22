package mg3

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	blocks "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blocks"
	util "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/ipfs/go-ipfs/util"
	dstore "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	flatfs "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore/flatfs"
	leveldb "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore/leveldb"
	mount "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore/mount"
	dsq "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore/query"
	sync "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-datastore/sync"
	rename "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/Godeps/_workspace/src/github.com/jbenet/go-os-rename"
	base32 "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/base32"
	nuflatfs "github.com/ipfs/fs-repo-migrations/fs-repo-3-to-4/flatfs"
	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	lock "github.com/ipfs/fs-repo-migrations/tools/repolock"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"
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
type txFunc func(dstore.Datastore, dstore.Key, []byte, mkKeyFunc) error

func validateNewKey(s string) bool {
	parts := strings.Split(s, "/")

	kpart := parts[len(parts)-1]
	v, err := base32.RawStdEncoding.DecodeString(kpart)
	if err == nil && len(v) == 34 {
		return true
	}

	return false
}

func oldKeyFunc(prefix string) func(util.Key) dstore.Key {
	return func(k util.Key) dstore.Key {
		return dstore.NewKey(prefix + string(k))
	}
}

func validateOldKey(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return false
	}

	kpart := s[2+len(parts[1]):]
	v, err := base32.RawStdEncoding.DecodeString(kpart)
	if err == nil && len(v) == 34 {
		// already transfered to new format
		return false
	}

	return true
}

func newKeyFunc(prefix string) func(util.Key) dstore.Key {
	return func(k util.Key) dstore.Key {
		return dstore.NewKey(prefix + base32.RawStdEncoding.EncodeToString([]byte(k)))
	}
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

	dsold, dsnew, err := openDatastores(opts.Path)
	if err != nil {
		return err
	}

	log.Log("transfering blocks to new key format")
	if err := transferBlocks(filepath.Join(opts.Path, "blocks")); err != nil {
		return err
	}

	/*
		if err := rewriteKeys(dsold, dsnew, "blocks", newKeyFunc("/blocks/"), validateOldKey, transferBlock); err != nil {
			return err
		}
	*/

	log.Log("transferring stored public key records")
	if err := rewriteKeys(dsold, dsnew, "pk", newKeyFunc("/pk/"), validateOldKey, transferPubKey); err != nil {
		return err
	}

	log.Log("transferring stored ipns records")
	if err := rewriteKeys(dsold, dsnew, "ipns", newKeyFunc("/ipns/"), validateOldKey, transferIpnsEntries); err != nil {
		return err
	}

	err = repo.WriteVersion("4")
	if err != nil {
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 3 to 4 succeeded")
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

	oldds, newds, err := openDatastores(opts.Path)
	if err != nil {
		return err
	}

	log.Log("reverting blocks to old key format")
	if err := rewriteKeys(newds, oldds, "blocks", oldKeyFunc("/blocks/"), validateNewKey, transferBlock); err != nil {
		return err
	}

	if err := cleanEmptyDirs(filepath.Join(opts.Path, "blocks")); err != nil {
		return err
	}

	log.Log("reverting stored public key records")
	if err := rewriteKeys(newds, oldds, "pk", oldKeyFunc("/pk/"), validateNewKey, transferPubKey); err != nil {
		return err
	}

	log.Log("reverting stored ipns records")
	if err := rewriteKeys(newds, oldds, "ipns", oldKeyFunc("/ipns/"), validateNewKey, revertIpnsEntries); err != nil {
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

func openDatastores(repopath string) (a, b dstore.ThreadSafeDatastore, e error) {
	log.VLog("  - opening datastore at %q", repopath)
	ldbpath := path.Join(repopath, "datastore")
	ldb, err := leveldb.NewDatastore(ldbpath, nil)
	if err != nil {
		return nil, nil, err
	}

	blockspath := path.Join(repopath, "blocks")
	nfds, err := nuflatfs.New(blockspath, 5, true)
	if err != nil {
		return nil, nil, err
	}

	ofds, err := flatfs.New(blockspath, 4)
	if err != nil {
		return nil, nil, err
	}

	oldds := sync.MutexWrap(mount.New([]mount.Mount{
		{
			Prefix:    dstore.NewKey("/blocks"),
			Datastore: ofds,
		},
		{
			Prefix:    dstore.NewKey("/"),
			Datastore: ldb,
		},
	}))

	newds := sync.MutexWrap(mount.New([]mount.Mount{
		{
			Prefix:    dstore.NewKey("/blocks"),
			Datastore: nfds,
		},
		{
			Prefix:    dstore.NewKey("/"),
			Datastore: ldb,
		},
	}))
	return oldds, newds, nil
}

func rewriteKeys(oldds, newds dstore.Datastore, pref string, mkKey mkKeyFunc, valid validFunc, transfer txFunc) error {

	log.Log("gathering keys...")
	res, err := oldds.Query(dsq.Query{
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

	prog := NewProgress(len(entries))
	for _, e := range entries {
		prog.Next()

		if !valid(e.Key) {
			prog.Skip()
			continue
		}

		curk := dstore.NewKey(e.Key)
		blk, err := oldds.Get(curk)
		if err != nil {
			return err
		}

		blkd, ok := blk.([]byte)
		if !ok {
			log.Error("data %q was not a []byte", e.Key)
			continue
		}

		err = transfer(newds, curk, blkd, mkKey)
		if err != nil {
			return err
		}

		err = oldds.Delete(curk)
		if err != nil {
			return err
		}
	}
	fmt.Println()

	return nil
}

func transferBlock(ds dstore.Datastore, oldk dstore.Key, data []byte, mkKey mkKeyFunc) error {
	b := blocks.NewBlock(data)
	dsk := mkKey(b.Key())
	err := ds.Put(dsk, b.Data)
	if err != nil {
		return err
	}

	return nil
}

func transferPubKey(ds dstore.Datastore, oldk dstore.Key, data []byte, mkKey mkKeyFunc) error {
	k := util.Key(util.Hash(data))
	dsk := mkKey(k)
	return ds.Put(dsk, data)
}

func transferIpnsEntries(ds dstore.Datastore, oldk dstore.Key, data []byte, mkkey mkKeyFunc) error {
	if len(oldk.String()) != 40 {
		log.Log(" - skipping malformed ipns record: %q", oldk)
		return nil
	}
	dsk := dstore.NewKey("/ipns/" + base32.RawStdEncoding.EncodeToString([]byte(oldk.String()[6:])))
	return ds.Put(dsk, data)
}

func revertIpnsEntries(ds dstore.Datastore, oldk dstore.Key, data []byte, mkkey mkKeyFunc) error {
	if len(oldk.String()) != 61 {
		log.Log(" - skipping malformed ipns record: %q", oldk)
		return nil
	}
	dec, err := base32.RawStdEncoding.DecodeString(oldk.String()[6:])
	if err != nil {
		return err
	}

	dsk := dstore.NewKey("/ipns/" + string(dec))
	return ds.Put(dsk, data)
}

func transferBlocks(flatfsdir string) error {
	var keys []string
	dots := 0
	filepath.Walk(flatfsdir, func(p string, i os.FileInfo, err error) error {
		if dots%200 == 0 {
			fmt.Printf("\r                         \renumerating keys")
		}
		if dots%40 == 39 {
			fmt.Printf(".")
		}
		dots++

		if i.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		if len(p) <= len(flatfsdir)+1 {
			return nil
		}

		rel := p[len(flatfsdir)+1:]
		if !strings.HasSuffix(rel, ".data") {
			fmt.Println("skipping (no .data): ", rel)
			return nil
		}

		keys = append(keys, p)
		return nil
	})

	prog := NewProgress(len(keys))
	for _, p := range keys {
		prog.Next()
		rel := p[len(flatfsdir)+1:]

		justkey := rel[:len(rel)-5]
		if validateNewKey(justkey) {
			prog.Skip()
			fmt.Printf("skipping %s, already in new format\n", justkey)
			continue
		}

		_, fi := filepath.Split(rel[:len(rel)-5])
		k, err := hex.DecodeString(fi)
		if err != nil {
			fmt.Printf("failed to decode: %s\n", p)
			return err
		}

		if len(k) != 34 {
			data, err := ioutil.ReadFile(p)
			if err != nil {
				return err
			}

			key := blocks.NewBlock(data).Key()
			k = []byte(key)
		}

		nname := base32.RawStdEncoding.EncodeToString(k) + ".data"
		dirname := nname[:5]
		nfiname := filepath.Join(flatfsdir, dirname, nname)

		err = os.MkdirAll(filepath.Join(flatfsdir, dirname), 0755)
		if err != nil {
			return err
		}

		err = rename.Rename(p, nfiname)
		if err != nil {
			return err
		}
	}

	fmt.Println()

	err := cleanEmptyDirs(flatfsdir)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func cleanEmptyDirs(dir string) error {
	children, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, c := range children {
		if !c.IsDir() {
			continue
		}

		cdir := filepath.Join(dir, c.Name())
		blocks, err := ioutil.ReadDir(cdir)
		if err != nil {
			return err
		}

		if len(blocks) == 0 {
			err := os.Remove(cdir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type progress struct {
	total   int
	current int
	skipped int

	start time.Time
}

func NewProgress(total int) *progress {
	return &progress{
		total: total,
		start: time.Now(),
	}
}

func (p *progress) Skip() {
	p.skipped++
}

func (p *progress) Next() {
	p.current++
	fmt.Printf("\r[%d / %d]", p.current, p.total)
	if p.skipped > 0 {
		fmt.Printf(" (skipped: %d)", p.skipped)
	}

	if p.current%10 == 9 {
		took := time.Now().Sub(p.start)
		av := took / time.Duration(p.current)
		estim := av * time.Duration(p.total-p.current)
		est := strings.Split(estim.String(), ".")[0]

		fmt.Printf("  Approx time remaining: %ss  ", est)
	}
}
