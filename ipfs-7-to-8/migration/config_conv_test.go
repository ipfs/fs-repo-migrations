package mg7

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func arrayMatch(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]struct{})
	for _, i := range a {
		am[i] = struct{}{}
	}
	for _, i := range b {
		if _, ok := am[i]; !ok {
			return false
		}
	}
	return true
}

func matchesExpected(t *testing.T, res []byte, exp []string) bool {
	confMap := make(map[string]interface{})
	if err := json.Unmarshal(res, &confMap); err != nil {
		t.Fatal(err)
	}

	var as []string
	for _, i := range confMap["Bootstrap"].([]interface{}) {
		as = append(as, i.(string))
	}

	return arrayMatch(as, exp)
}

func TestOldToNew(t *testing.T) {
	in := strings.NewReader(v7config)
	out := new(bytes.Buffer)
	if err := convert(in, out, ver7to8); err != nil {
		t.Fatal(err)
	}

	if !matchesExpected(t, out.Bytes(), expectedv7tov8) {
		t.Fatal(fmt.Errorf("Converted does not match expected result\n%s\n%s\n", out.String(), expectedv7tov8))
	}
}

func TestNewToOld(t *testing.T) {
	in := strings.NewReader(v8config)
	out := new(bytes.Buffer)
	if err := convert(in, out, ver8to7); err != nil {
		t.Fatal(err)
	}

	if !matchesExpected(t, out.Bytes(), expectedv8tov7) {
		t.Fatal(fmt.Errorf("Converted does not match expected result\n%s\n%s\n", out.String(), expectedv8tov7))
	}
}

func TestForward(t *testing.T) {
	bootstrap := []string{
		// peer with old key (should be filtered out)
		"/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		// peer with unknown key (should be included, /ipfs/ replaced with /p2p/)
		"/ip4/178.62.158.248/tcp/4001/ipfs/QmSomeNewKeygSp2LA3dPaeykiS1J6DifTC88f5uV562wK",
		// unrecognized format (should be included)
		"/ip4/178.62.158.248/tcp/4001/wut/some-new-format",
		// peer with /dns/ipfs.my-domain.com (should be included, "ipfs" in domain should not be changed)
		"/dns4/ipfs-bootstrap.com/tcp/4001/ipfs/Qm1234567KeygSp2LA3dPaeykiS1J6DifTC88f5u123456",
	}
	exp := []string{
		// new dnsaddr peers
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		// peer with unknown key
		"/ip4/178.62.158.248/tcp/4001/p2p/QmSomeNewKeygSp2LA3dPaeykiS1J6DifTC88f5uV562wK",
		// unrecognized format
		"/ip4/178.62.158.248/tcp/4001/wut/some-new-format",
		// peer with /dns/ipfs.my-domain.com
		"/dns4/ipfs-bootstrap.com/tcp/4001/p2p/Qm1234567KeygSp2LA3dPaeykiS1J6DifTC88f5u123456",
	}
	if res := ver7to8(bootstrap); !arrayMatch(exp, res) {
		fmt.Println(res)
		t.Fatal("Expected forward conversion to succeed")
	}
}

func TestForwardIgnoreConfWithDNSAddrsAlready(t *testing.T) {
	bootstrap := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}
	if res := ver7to8(bootstrap); !arrayMatch(bootstrap, res) {
		t.Fatal("Expected conf with dnsaddr to be ignored")
	}
}

func TestForwardIgnoreConfWithNoSmallKeys(t *testing.T) {
	bootstrap := []string{
		"/ip4/104.131.131.82/tcp/4001/p2p/QmBigKeyvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}
	if res := ver7to8(bootstrap); !arrayMatch(bootstrap, res) {
		t.Fatal("Expected conf with dnsaddr to be ignored")
	}
}

func TestBackward(t *testing.T) {
	bootstrap := []string{
		// new dnsaddr peers (should be included)
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		// peer with old key (should be included)
		"/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		// peer with unknown key (should be included)
		"/ip4/178.62.158.248/tcp/4001/p2p/QmSomeNewKeygSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		// unrecognized format (should be included)
		"/ip4/178.62.158.248/tcp/4001/wut/some-new-format",
	}
	exp := []string{
		// new dnsaddr peers
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		// old addresses
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		"/ip6/2604:a880:1:20::203:d001/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip6/2400:6180:0:d0::151:6001/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip6/2604:a880:800:10::4a:5001/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		// peer with unknown key
		"/ip4/178.62.158.248/tcp/4001/p2p/QmSomeNewKeygSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		// unrecognized format
		"/ip4/178.62.158.248/tcp/4001/wut/some-new-format",
	}
	if res := ver8to7(bootstrap); !arrayMatch(exp, res) {
		fmt.Println(res)
		t.Fatal("Expected backward conversion to succeed")
	}
}

func TestBackwardIgnoreConfWithNoDNSAddr(t *testing.T) {
	bootstrap := []string{
		"/ip4/104.131.131.82/tcp/4001/p2p/QmSomeKeyV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}
	if res := ver8to7(bootstrap); !arrayMatch(bootstrap, res) {
		t.Fatal("Expected conf with no dnsaddr to be ignored")
	}
}

var v7config = `{
	"Some": {
		"Other": "Config"
	},
	"Bootstrap": [
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.176.52/tcp/4001/ipfs/QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z",
		"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/162.243.248.213/tcp/4001/ipfs/QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm",
		"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		"/ip4/178.62.61.185/tcp/4001/ipfs/QmSoLMeWqB7YGVLJN3pNLQpmmEk35v6wYtsMGLzSr5QBU3",
		"/ip4/104.236.151.122/tcp/4001/ipfs/QmSoLju6m7xTh3DuokvT3886QRYqxAzb1kShaanJgW36yx",
		"/ip6/2604:a880:1:20::1f9:9001/tcp/4001/ipfs/QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z",
		"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip6/2604:a880:0:1010::23:d001/tcp/4001/ipfs/QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm",
		"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		"/ip6/2a03:b0c0:1:d0::e7:1/tcp/4001/ipfs/QmSoLMeWqB7YGVLJN3pNLQpmmEk35v6wYtsMGLzSr5QBU3",
		"/ip6/2604:a880:1:20::1d9:6001/tcp/4001/ipfs/QmSoLju6m7xTh3DuokvT3886QRYqxAzb1kShaanJgW36yx"
	]
}
`

var expectedv7tov8 = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
}

var v8config = `{
	"Some": {
		"Other": "Config"
	},
	"Bootstrap": [
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.131.131.83/tcp/4001/p2p/QmcafeMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLaaaa"
	]
}
`

var expectedv8tov7 = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	"/ip4/104.131.131.83/tcp/4001/p2p/QmcafeMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLaaaa",
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
