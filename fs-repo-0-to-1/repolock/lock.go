package lock

import (
	"fmt"
	"io"
	"path"

	"github.com/ipfs/fs-repo-migrations/tools/lock"
)

var errRepoLock = `failed to acquire repo lock at %s/%s
Is a daemon running? please stop it before running migration`

// LockFile is the filename of the daemon lock, relative to config dir
// TODO rename repo lock and hide name
const LockFile = "daemon.lock"

func Lock(confdir string) (io.Closer, error) {
	c, err := lock.Lock(path.Join(confdir, LockFile))
	if err != nil {
		return nil, fmt.Errorf(errRepoLock, confdir, LockFile)
	}
	return c, nil
}
