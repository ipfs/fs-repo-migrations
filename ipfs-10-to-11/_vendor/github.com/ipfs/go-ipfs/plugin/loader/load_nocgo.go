// +build !cgo,!noplugin
// +build linux darwin freebsd

package loader

import (
	"errors"

	iplugin "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipfs/plugin"
)

func init() {
	loadPluginFunc = nocgoLoadPlugin
}

func nocgoLoadPlugin(fi string) ([]iplugin.Plugin, error) {
	return nil, errors.New("not built with cgo support")
}
