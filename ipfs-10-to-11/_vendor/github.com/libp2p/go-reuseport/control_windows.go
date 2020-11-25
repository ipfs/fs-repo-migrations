package reuseport

import (
	"syscall"

	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/golang.org/x/sys/windows"
)

func Control(network, address string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	})
}
