package mg9

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var config = `{
  "Addresses": {
    "Swarm": [
      "/ip4/0.0.0.0/tcp/0"
    ]
  },
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa"
  ],
  "Experimental": {
  	"QUIC": false
  }
}`

var expForwardConfig = `{
  "Addresses": {
    "Swarm": [
      "/ip4/0.0.0.0/tcp/0",
      "/ip4/0.0.0.0/udp/0/quic"
    ]
  },
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
    "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
  ],
  "Experimental": {}
}`

var expRevertedConfig = `{
  "Addresses": {
    "Swarm": [
      "/ip4/0.0.0.0/tcp/0"
    ]
  },
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
    "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
  ],
  "Experimental": {}
}`

func TestConversion(t *testing.T) {
	conf9to10 := new(bytes.Buffer)
	err := convert(strings.NewReader(config), conf9to10, true, ver9to10Bootstrap, ver9to10Swarm)
	if err != nil {
		t.Fatal(err)
	}

	forward := conf9to10.String()
	if noSpace(forward) != noSpace(expForwardConfig) {
		t.Fatalf("Mismatch\nConversion produced:\n%s\nExpected:\n%s\n", forward, expForwardConfig)
	}

	conf10to9 := new(bytes.Buffer)
	err = convert(strings.NewReader(conf9to10.String()), conf10to9, false, ver10to9Bootstrap, ver10to9Swarm)
	if err != nil {
		t.Fatal(err)
	}
	reverted := conf10to9.String()
	if noSpace(reverted) != noSpace(expRevertedConfig) {
		t.Fatalf("Mismatch\nConversion produced:\n%s\nExpected:\n%s\n", reverted, expRevertedConfig)
	}
}

var whitespaceRe = regexp.MustCompile(`\s`)

func noSpace(str string) string {
	return whitespaceRe.ReplaceAllString(str, "")
}
