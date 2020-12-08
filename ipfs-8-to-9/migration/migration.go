package mg8

import (
	base32 "encoding/base32"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/tools/go-migrate"
	mfsr "github.com/ipfs/fs-repo-migrations/tools/mfsr"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "8-to-9"
}

func (m Migration) Reversible() bool {
	return true
}

const keyFilenamePrefix = "key_"

const keystoreRoot = "keystore"

func isEncoded(name string) bool {
	_, err := decode(name)
	return err == nil
}

func encode(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("key name must be at least one character")
	}

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encodedName := encoder.EncodeToString([]byte(name))
	return keyFilenamePrefix + strings.ToLower(encodedName), nil
}

func decode(name string) (string, error) {
	if !strings.HasPrefix(name, keyFilenamePrefix) {
		return "", fmt.Errorf("key's filename has unexpected format")
	}

	nameWithoutPrefix := strings.ToUpper(name[len(keyFilenamePrefix):])
	decoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	data, err := decoder.DecodeString(nameWithoutPrefix)

	return string(data), err
}

func (m Migration) Apply(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("applying %s repo migration", m.Versions())

	err := m.encodeDecode(
		opts,
		isEncoded, // skip if already encoded
		encode,
	)

	if err != nil {
		return err
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion("9")
	if err != nil {
		log.Error("failed to update version file to 9")
		return err
	}

	log.Log("updated version file")

	log.Log("Migration 8 to 9 succeeded")
	return nil
}

func (m Migration) encodeDecode(opts migrate.Options, shouldApplyCodec func(string) bool, codec func(string) (string, error)) error {
	keystoreRoot := filepath.Join(opts.Path, keystoreRoot)
	fileInfos, err := ioutil.ReadDir(keystoreRoot)

	if err != nil {
		return err
	}

	for _, info := range fileInfos {
		if info.IsDir() {
			log.Log("skipping ", info.Name(), " as it is directory!")
			continue
		}

		if shouldApplyCodec(info.Name()) {
			log.Log("skipping ", info.Name(), ". Already in expected format!")
			continue
		}

		log.VLog("Renaming key's filename: ", info.Name())
		encodedName, err := codec(info.Name())
		if err != nil {
			return err
		}

		src := filepath.Join(keystoreRoot, info.Name())
		dest := filepath.Join(keystoreRoot, encodedName)

		if err := os.Rename(src, dest); err != nil {
			return err
		}
	}
	return nil
}

func (m Migration) Revert(opts migrate.Options) error {
	log.Verbose = opts.Verbose
	log.Log("reverting migration")

	err := m.encodeDecode(
		opts,
		func(name string) bool {
			return !isEncoded(name) // skip if not encoded
		},
		decode,
	)

	if err != nil {
		return err
	}

	err = mfsr.RepoPath(opts.Path).WriteVersion("8")
	if err != nil {
		log.Error("failed to update version file to 8")
		return err
	}

	log.Log("updated version file")

	return nil
}
