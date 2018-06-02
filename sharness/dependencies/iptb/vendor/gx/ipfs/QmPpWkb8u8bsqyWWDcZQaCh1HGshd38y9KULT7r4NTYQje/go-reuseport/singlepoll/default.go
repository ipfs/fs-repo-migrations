// +build !linux

package singlepoll

import (
	"context"
	"errors"

	"gx/ipfs/QmPpWkb8u8bsqyWWDcZQaCh1HGshd38y9KULT7r4NTYQje/go-reuseport/poll"
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
