package mg7

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ipfs/go-cid"
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
			return fmt.Errorf("Bootstrap field missing or of the wrong type")
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
	out.Write(fixed)
	out.Write([]byte("\n"))
	return nil
}

func ver7to8(bootstrap []string) []string {
	// Make sure the dnsaddrs bootstrap peers are included
	var res []string
	for _, p := range dnsBootstrapPeers {
		res = append(res, fmt.Sprintf("/dnsaddr/%s/p2p/%s", dnsAddr, p))
	}

	// Filter out peers that we added above, or that have an ID known to belong
	// to a peer with a small key
	for _, addr := range bootstrap {
		if ok, err := isDNSBootstrapPeer(addr); !ok && err == nil {
			if ok, err = isSmallKeyPeer(addr); !ok && err == nil {
				// Replace /ipfs with /p2p
				addr = strings.Replace(addr, "/ipfs", "/p2p", -1)
				res = append(res, addr)
			}
		}
	}

	return res
}

func ver8to7(bootstrap []string) []string {
	// Make sure the old addresses are included
	res := append([]string{}, oldBootstrapAddrs...)

	oldPeerIDs := make(map[string]struct{})
	for _, addr := range oldBootstrapAddrs {
		pid, err := getAddrPeerID(addr)
		if err != nil {
			panic(err)
		}
		oldPeerIDs[pid] = struct{}{}
	}

	// Filter out old addresses added above, and addresses with the new DNS
	// addresses
	for _, btAddr := range bootstrap {
		pid, err := getAddrPeerID(btAddr)
		if err == nil {
			if _, ok := oldPeerIDs[pid]; !ok && !strings.Contains(btAddr, dnsAddr) {
				res = append(res, btAddr)
			}
		}
	}

	return res
}

func getAddrPeerID(addr string) (string, error) {
	// eg /ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
	parts := strings.Split(addr, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("Could not parse peer ID from addr '%s'", addr)
	}
	last := parts[len(parts)-1]
	if _, err := cid.Decode(last); err != nil {
		return "", fmt.Errorf("Could not parse peer ID from addr '%s'", addr)
	}
	return last, nil
}

func addrPeerIDInList(peers []string, addr string) (bool, error) {
	addrID, err := getAddrPeerID(addr)
	if err != nil {
		return false, err
	}
	for _, p := range peers {
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
