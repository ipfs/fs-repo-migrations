// +build cgo,!noplugin
// +build linux darwin freebsd

package loader

import (
	"errors"
	"plugin"

	iplugin "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/ipfs/go-ipfs/plugin"
)

func init() {
	loadPluginFunc = unixLoadPlugin
}

func unixLoadPlugin(fi string) ([]iplugin.Plugin, error) {
	pl, err := plugin.Open(fi)
	if err != nil {
		return nil, err
	}
	pls, err := pl.Lookup("Plugins")
	if err != nil {
		return nil, err
	}

	typePls, ok := pls.(*[]iplugin.Plugin)
	if !ok {
		return nil, errors.New("filed 'Plugins' didn't contain correct type")
	}

	return *typePls, nil
}
