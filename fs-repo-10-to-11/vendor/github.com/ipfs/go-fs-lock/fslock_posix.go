// +build !plan9

package fslock

import (
	"strings"
	"syscall"
)

func lockedByOthers(err error) bool {
	return err == syscall.EAGAIN || strings.Contains(err.Error(), "resource temporarily unavailable")
}
