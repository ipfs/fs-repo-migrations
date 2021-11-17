package lock

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ipfs/fs-repo-migrations/ipfs-1-to-2/lock"
)

var errRepoLock = `failed to acquire repo lock at %s/%s
Is a daemon running? please stop it before running migration`

// LockFile is the filename of the daemon lock, relative to config dir
// lock changed names.
const (
	LockFile1 = "daemon.lock"
	LockFile2 = "repo.lock"
)

func Lock1(confdir string) (io.Closer, error) {
	c, err := lock.Lock(path.Join(confdir, LockFile1))
	if err != nil {
		return nil, fmt.Errorf(errRepoLock, confdir, LockFile1)
	}
	return c, nil
}

func Remove1(confdir string) error {
	return os.Remove(path.Join(confdir, LockFile1))
}

func Lock2(confdir string) (io.Closer, error) {
	c, err := lock.Lock(path.Join(confdir, LockFile2))
	if err != nil {
		return nil, fmt.Errorf(errRepoLock, confdir, LockFile2)
	}
	return c, nil
}
