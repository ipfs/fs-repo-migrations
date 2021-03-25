package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"time"

	api "github.com/ipfs/go-ipfs-api"
	"github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
)

const (
	shellUpTimeout    = 2 * time.Second
	defaultFetchLimit = 1024 * 1024 * 512

	// Local IPFS API
	apiFile = "api"
)

type ipfsFetcher struct {
	distPath string
	limit    int64
}

// newIpfsFetcher creates a new IpfsFetcher
//
// Specifying "" for distPath sets the default IPNS path.
// Specifying 0 for fetchLimit sets the default, -1 means no limit.
func newIpfsFetcher(distPath string, fetchLimit int64) *ipfsFetcher {
	f := &ipfsFetcher{
		limit:    defaultFetchLimit,
		distPath: migrations.LatestIpfsDist,
	}

	if distPath != "" {
		if !strings.HasPrefix(distPath, "/") {
			distPath = "/" + distPath
		}
		f.distPath = distPath
	}

	if fetchLimit != 0 {
		if fetchLimit == -1 {
			fetchLimit = 0
		}
		f.limit = fetchLimit
	}

	return f
}

// Fetch attempts to fetch the file at the given path, from the distribution
// site configured for this HttpFetcher.  Returns io.ReadCloser on success,
// which caller must close.
func (f *ipfsFetcher) Fetch(ctx context.Context, filePath string) (io.ReadCloser, error) {
	sh, _, err := apiShell("")
	if err != nil {
		return nil, err
	}

	resp, err := sh.Request("cat", path.Join(f.distPath, filePath)).Send(ctx)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	if f.limit != 0 {
		return migrations.NewLimitReadCloser(resp.Output, f.limit), nil
	}
	return resp.Output, nil
}

// apiShell creates a new ipfs api shell and checks that it is up.  If the shell
// is available, then the shell and ipfs version are returned.
func apiShell(ipfsDir string) (*api.Shell, string, error) {
	apiEp, err := apiEndpoint("")
	if err != nil {
		return nil, "", err
	}
	sh := api.NewShell(apiEp)
	sh.SetTimeout(shellUpTimeout)
	ver, _, err := sh.Version()
	if err != nil {
		return nil, "", errors.New("ipfs api shell not up")
	}
	sh.SetTimeout(0)
	return sh, ver, nil
}

// apiEndpoint reads the api file from the local ipfs install directory and
// returns the address:port read from the file.  If the ipfs directory is not
// specified then the default location is used.
func apiEndpoint(ipfsDir string) (string, error) {
	ipfsDir, err := migrations.CheckIpfsDir(ipfsDir)
	if err != nil {
		return "", err
	}

	apiData, err := ioutil.ReadFile(path.Join(ipfsDir, apiFile))
	if err != nil {
		return "", err
	}

	val := strings.TrimSpace(string(apiData))
	parts := strings.Split(val, "/")
	if len(parts) != 5 {
		return "", fmt.Errorf("incorrectly formatted api string: %q", val)
	}

	return parts[2] + ":" + parts[4], nil
}
