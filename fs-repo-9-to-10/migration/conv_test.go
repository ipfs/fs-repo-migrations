package mg9

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var config = `{
  "Addresses": {
    "Announce": [
      "/ip4/1.2.3.4/tcp/1"
    ],
    "NoAnnounce": [
      "/ip4/5.6.7.8/tcp/2"
    ],
    "Swarm": [
      "/ip4/0.0.0.0/tcp/0",
      "/ip4/1.2.3.4/tcp/1",
      "/ip4/5.6.7.8/tcp/2"
    ]
  },
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa"
  ]
}`

var expForwardConfig = `{
  "Addresses": {
    "Announce": [
      "/ip4/1.2.3.4/tcp/1",
      "/ip4/1.2.3.4/udp/1/quic"
    ],
    "NoAnnounce": [
      "/ip4/5.6.7.8/tcp/2",
      "/ip4/5.6.7.8/udp/2/quic"
    ],
    "Swarm": [
      "/ip4/0.0.0.0/tcp/0",
      "/ip4/1.2.3.4/tcp/1",
      "/ip4/5.6.7.8/tcp/2",
      "/ip4/0.0.0.0/udp/0/quic",
      "/ip4/1.2.3.4/udp/1/quic",
      "/ip4/5.6.7.8/udp/2/quic"
    ]
  },
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
    "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
  ]
}`

func TestConversion(t *testing.T) {
	conf9to10 := new(bytes.Buffer)
	err := convert(strings.NewReader(config), conf9to10, ver9to10Bootstrap, ver9to10Addresses)
	if err != nil {
		t.Fatal(err)
	}

	forward := conf9to10.String()
	if noSpace(forward) != noSpace(expForwardConfig) {
		t.Fatalf("Mismatch\nConversion produced:\n%s\nExpected:\n%s\n", forward, expForwardConfig)
	}
}

var whitespaceRe = regexp.MustCompile(`\s`)

func noSpace(str string) string {
	return whitespaceRe.ReplaceAllString(str, "")
}
