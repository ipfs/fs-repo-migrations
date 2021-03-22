package mg7

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/ipfs/fs-repo-migrations/tools/stump"
)

var (
	dnsAddr           = "bootstrap.libp2p.io"
	dnsBootstrapPeers = []string{
		"QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}
	smallKeyBootstrapPeers = []string{
		"QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z",
		"QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm",
		"QmSoLMeWqB7YGVLJN3pNLQpmmEk35v6wYtsMGLzSr5QBU3",
		"QmSoLju6m7xTh3DuokvT3886QRYqxAzb1kShaanJgW36yx",
		"QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"QmSoLpPVmHKQ4XTPdz8tjDFgdeRFkpV8JgYq8JVJ69RrZm",
		"QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	}
	oldBootstrapAddrs = []string{
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		"/ip6/2604:a880:1:20::203:d001/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip6/2400:6180:0:d0::151:6001/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip6/2604:a880:800:10::4a:5001/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	}
)

// convFunc does an inplace conversion of the "Bootstrap"
// configuration from one version to another
type convFunc func([]string) []string

// convertFile converts a config file from one version to another, the
// converted config is stored in
func convertFile(orig string, new string, convFunc convFunc) error {
	in, err := os.Open(orig)
	if err != nil {
		return err
	}
	out, err := os.Create(new)
	if err != nil {
		return err
	}
	return convert(in, out, convFunc)
}

// convert converts the config from one version to another, returns
// the converted config as a map[string]interface{}
func convert(in io.Reader, out io.Writer, convFunc convFunc) error {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	confMap := make(map[string]interface{})
	if err = json.Unmarshal(data, &confMap); err != nil {
		return err
	}
	bootstrapi, _ := confMap["Bootstrap"].([]interface{})
	if bootstrapi == nil {
		bootstrapi, _ := confMap["bootstrap"].([]interface{})
		if bootstrapi == nil {
			log.Log("Bootstrap field missing or of the wrong type")
			log.Log("Nothing to migrate")
			_, err := out.Write(data)
			return err
		}
	}
	bootstrap := make([]string, len(bootstrapi))
	for i := range bootstrapi {
		bootstrap[i] = bootstrapi[i].(string)
	}
	res := convFunc(bootstrap)
	confMap["Bootstrap"] = res
	fixed, err := json.MarshalIndent(confMap, "", "  ")
	if err != nil {
		return err
	}
	if _, err := out.Write(fixed); err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}

func ver7to8(bootstrap []string) []string {
	hasSmallKey := false
	for _, addr := range bootstrap {
		if ok, _ := isDNSBootstrapPeer(addr); ok {
			// Already has a dnsaddr key so assume the user has custom config
			// that we shouldn't override
			return bootstrap
		}
		if ok, _ := isSmallKeyPeer(addr); ok {
			hasSmallKey = true
		}
	}

	if !hasSmallKey {
		// There are no peers with small keys in the bootstrap list so assume
		// the user has a custom config that we shouldn't override
		return bootstrap
	}

	// Make sure the dnsaddrs bootstrap peers are included
	var res []string
	for _, p := range dnsBootstrapPeers {
		res = append(res, fmt.Sprintf("/dnsaddr/%s/p2p/%s", dnsAddr, p))
	}

	// Filter out dnsaddr peers that we added above, or that have an ID known
	// to belong to a peer with a small key.
	// If we don't recognize the address format (err != nil), assume that the
	// user changed it for a reason and leave it in there.
	for _, addr := range bootstrap {
		if isDNSAddr, err := isDNSBootstrapPeer(addr); !isDNSAddr || err != nil {
			if isSmall, err := isSmallKeyPeer(addr); !isSmall || err != nil {
				// Change the protocol string from "ipfs" to "p2p".
				// Make sure we don't break addresses like
				// /dns4/ipfs-bootstrap.com/...
				// by matching specifically /ipfs/Qm or /ipfs/1...
				addr = strings.Replace(addr, "/ipfs/Qm", "/p2p/Qm", -1)
				addr = strings.Replace(addr, "/ipfs/1", "/p2p/1", -1)
				res = append(res, addr)
			}
		}
	}

	return res
}

func ver8to7(bootstrap []string) []string {
	// If the config doesn't have the new DNS addresses then assume it hasn't
	// been updated to version 8 and bail out
	hasDNSAddrs := false
	for _, addr := range bootstrap {
		if ok, _ := isDNSBootstrapPeer(addr); ok {
			hasDNSAddrs = true
		}
	}
	if !hasDNSAddrs {
		return bootstrap
	}

	// Extract peer IDs from old bootstrap addresses
	var oldBootstrapPeerIDs []string
	for _, oldAddr := range oldBootstrapAddrs {
		if p, err := getAddrPeerID(oldAddr); err == nil {
			oldBootstrapPeerIDs = append(oldBootstrapPeerIDs, p)
		}
	}

	// Make sure the old addresses are included in the result
	res := append([]string{}, oldBootstrapAddrs...)

	// Filter out old addresses added above
	for _, addr := range bootstrap {
		isOldPeer, err := addrPeerIDInList(oldBootstrapPeerIDs, addr)

		// If we don't recognize the address format, just assume the user
		// has changed it on purpose and include it in the results
		if !isOldPeer || err != nil {
			res = append(res, addr)
		}
	}

	return res
}

func parseErr(addr string) error {
	return fmt.Errorf("Could not parse peer ID from addr '%s'", addr)
}

func getAddrPeerID(addr string) (string, error) {
	// eg /ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
	parts := strings.Split(addr, "/")
	if len(parts) < 2 {
		return "", parseErr(addr)
	}

	// Just verify that the address ends with /p2p/something or /ipfs/something
	peerType := parts[len(parts)-2]
	if peerType != "p2p" && peerType != "ipfs" {
		return "", parseErr(addr)
	}

	last := parts[len(parts)-1]
	return last, nil
}

func addrPeerIDInList(peerIDs []string, addr string) (bool, error) {
	addrID, err := getAddrPeerID(addr)
	if err != nil {
		return false, err
	}
	for _, p := range peerIDs {
		if p == addrID {
			return true, nil
		}
	}
	return false, nil
}

func isDNSBootstrapPeer(addr string) (bool, error) {
	return addrPeerIDInList(dnsBootstrapPeers, addr)
}

func isSmallKeyPeer(addr string) (bool, error) {
	return addrPeerIDInList(smallKeyBootstrapPeers, addr)
}
