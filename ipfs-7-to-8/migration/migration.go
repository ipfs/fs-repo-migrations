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

	base32 "github.com/ipfs/fs-repo-migrations/ipfs-7-to-8/base32"
)

type Migration struct{}

func (m Migration) Versions() string {
	return "7-to-8"
}

func (m Migration) Reversible() bool {
	return true
}

const keyFilenamePrefix = "key_"

func isEncoded(name string) bool {
	if !strings.HasPrefix(name, keyFilenamePrefix) {
		return false
	}

	_, err := decode(name)

	return err == nil
}

func encode(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("key name must be at least one character")
	}

	encodedName := base32.RawStdEncoding.EncodeToString([]byte(name))
	return keyFilenamePrefix + strings.ToLower(encodedName), nil
}

func decode(name string) (string, error) {
	if !strings.HasPrefix(name, keyFilenamePrefix) {
		return "", fmt.Errorf("key's filename has unexpected format")
	}

	nameWithoutPrefix := strings.ToUpper(name[len(keyFilenamePrefix):])
	data, err := base32.RawStdEncoding.DecodeString(nameWithoutPrefix)

	if err != nil {
		return "", err
	}

	decodedName := string(data[:])

	return decodedName, nil
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

		if isEncoded(info.Name()) {
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

		if !isEncoded(info.Name()) {
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

	err = mfsr.RepoPath(opts.Path).WriteVersion("7")
	if err != nil {
		log.Error("failed to update version file to 7")
		return err
	}

	log.Log("updated version file")

	return nil
}
