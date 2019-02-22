package mg6

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
	log "github.com/ipfs/fs-repo-migrations/stump"

	mbase "gx/ipfs/QmekxXDhCxCJRNuzmHreuaT3BsuJcsjcXWNrtV9C8DRHtd/go-multibase"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "7-to-8"
}

func (m Migration) Reversible() bool {
	return true
}

const keyFilenamePrefix = "key_"
const encodingBase = mbase.Base32

func is_encoded(name string) bool {
	if !strings.HasPrefix(name, keyFilenamePrefix) {
		return false
	}

	_, err := decode(name)

	return err == nil
}

func decode(name string) (string, error) {
	if !strings.HasPrefix(name, keyFilenamePrefix) {
		return "", fmt.Errorf("Key's filename has unexpexcted format")
	}

	nameWithoutPrefix := name[len(keyFilenamePrefix):]
	encoding, data, err := mbase.Decode(nameWithoutPrefix)

	if err != nil {
		return "", err
	}

	if encoding != encodingBase {
		return "", fmt.Errorf("Key's filename was encoded in unexpexted base")
	}

	return string(data[:]), nil
}

func encode(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("Key name must be at least one character")
	}

	encodedName, err := mbase.Encode(encodingBase, []byte(name))

	if err != nil {
		return "", err
	}

	return keyFilenamePrefix + encodedName, nil
}

func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	keystoreRoot := filepath.Join(opts.Path, "keystore")
	fileInfos, err := ioutil.ReadDir(keystoreRoot)

	if err != nil {
		return err
	}

	for _, info := range fileInfos {
		if info.IsDir() {
			log.Log("skipping ", info.Name(), " as it is directory!")
			continue
		}

		if is_encoded(info.Name()) {
			log.Log("skipping ", info.Name(), " as it is already encoded!")
			continue
		}

		log.VLog("migrating key's filename: ", info.Name())
		encodedName, err := encode(info.Name())
		if err != nil {
			return err
		}

		os.Rename(
			filepath.Join(keystoreRoot, info.Name()),
			filepath.Join(keystoreRoot, encodedName),
		)
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion("8")
	if err != nil {
		log.Error("failed to update version file to 8")
		return err
	}

	log.Log("updated version file")

	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")

	keystoreRoot := filepath.Join(opts.Path, "keystore")
	fileInfos, err := ioutil.ReadDir(keystoreRoot)

	if err != nil {
		return err
	}

	for _, info := range fileInfos {
		if info.IsDir() {
			log.Log("skipping", info.Name(), "as it is directory!")
			continue
		}

		if !is_encoded(info.Name()) {
			log.Log("skipping", info.Name(), "as it is not encoded!")
			continue
		}

		log.VLog("reverting key's filename:", info.Name())
		decodedName, err := decode(info.Name())
		if err != nil {
			return err
		}

		os.Rename(
			filepath.Join(keystoreRoot, info.Name()),
			filepath.Join(keystoreRoot, decodedName),
		)
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion("8")
	if err != nil {
		log.Error("failed to update version file to 8")
		return err
	}

	log.Log("updated version file")

	return nil
}
