package cbornode

import (
	"bufio"
	"bytes"

	cbor "github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/polydawn/refmt/cbor"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/polydawn/refmt/pretty"
	"github.com/ipfs/fs-repo-migrations/ipfs-10-to-11/_vendor/github.com/polydawn/refmt/shared"
)

//HumanReadable returns a string representation of a CBOR blob
func HumanReadable(blob []byte) (string, error) {
	reader := bytes.NewReader(blob)

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	err := shared.TokenPump{
		TokenSource: cbor.NewDecoder(cbor.DecodeOptions{}, reader),
		TokenSink:   pretty.NewEncoder(writer),
	}.Run()

	if err != nil {
		return "", err
	}

	if err = writer.Flush(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
