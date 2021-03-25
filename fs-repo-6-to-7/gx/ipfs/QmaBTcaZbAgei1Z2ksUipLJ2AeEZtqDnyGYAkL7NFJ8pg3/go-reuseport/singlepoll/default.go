// +build !linux

package singlepoll

import (
	"context"
	"errors"

	"github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmaBTcaZbAgei1Z2ksUipLJ2AeEZtqDnyGYAkL7NFJ8pg3/go-reuseport/poll"
)

var (
	ErrUnsupportedMode error = errors.New("only 'w' mode is supported on this arch")
)

func PollPark(ctx context.Context, fd int, mode string) error {
	if mode != "w" {
		return ErrUnsupportedMode
	}

	p, err := poll.New(fd)
	if err != nil {
		return err
	}
	defer p.Close()

	return p.WaitWriteCtx(ctx)
}
