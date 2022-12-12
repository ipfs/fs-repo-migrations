package mg12

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var beforeDefaultConfig = `{
  "Addresses": {
    "Announce": [
      "/ip6/::3/tcp/4001/quic",
      "/ip4/3.0.0.0/tcp/4001",
      "/ip4/3.0.0.0/udp/4001/quic"
    ],
    "AppendAnnounce": [
      "/ip6/::2/tcp/4001/quic",
      "/ip4/2.0.0.0/tcp/4001",
      "/ip4/2.0.0.0/udp/4001/quic"
    ],
    "NoAnnounce": [
      "/ip6/::1/tcp/4001/quic",
      "/ip4/1.0.0.0/tcp/4001",
      "/ip4/1.0.0.0/udp/4001/quic"
    ],
    "Swarm": [
      "/ip6/::/tcp/4001",
      "/ip6/::/tcp/4001/quic",
      "/ip4/0.0.0.0/tcp/4001",
      "/ip4/0.0.0.0/udp/4001/quic"
    ]
  },
  "AutoNAT": {},
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
  ],
  "Reprovider": {
    "Interval": "12h",
	"Strategy": "all"
  },
  "Routing": {
	"Methods": {},
	"Routers": {},
    "Type": "dht"
  },
  "Swarm": {
    "AddrFilters": [
	  "/ip4/10.0.0.0/ipcidr/8",
	  "/ip4/12.0.0.0/udp/4001/quic"
	],
	"ConnMgr": {
		"GracePeriod": "20s",
		"HighWater": 900,
		"LowWater": 600,
		"Type": "basic"
	}
  }
}`

var afterDefaultConfig = `{
  "Addresses": {
    "Announce": [
      "/ip6/::3/tcp/4001/quic",
      "/ip6/::3/tcp/4001/quic-v1",
      "/ip6/::3/tcp/4001/quic-v1/webtransport",
      "/ip4/3.0.0.0/tcp/4001",
      "/ip4/3.0.0.0/udp/4001/quic",
      "/ip4/3.0.0.0/udp/4001/quic-v1",
      "/ip4/3.0.0.0/udp/4001/quic-v1/webtransport"
    ],
    "AppendAnnounce": [
      "/ip6/::2/tcp/4001/quic",
      "/ip6/::2/tcp/4001/quic-v1",
      "/ip6/::2/tcp/4001/quic-v1/webtransport",
      "/ip4/2.0.0.0/tcp/4001",
      "/ip4/2.0.0.0/udp/4001/quic",
      "/ip4/2.0.0.0/udp/4001/quic-v1",
      "/ip4/2.0.0.0/udp/4001/quic-v1/webtransport"
    ],
    "NoAnnounce": [
      "/ip6/::1/tcp/4001/quic",
      "/ip6/::1/tcp/4001/quic-v1",
      "/ip6/::1/tcp/4001/quic-v1/webtransport",
      "/ip4/1.0.0.0/tcp/4001",
      "/ip4/1.0.0.0/udp/4001/quic",
      "/ip4/1.0.0.0/udp/4001/quic-v1",
      "/ip4/1.0.0.0/udp/4001/quic-v1/webtransport"
    ],
    "Swarm": [
      "/ip6/::/tcp/4001",
      "/ip6/::/tcp/4001/quic",
      "/ip6/::/tcp/4001/quic-v1",
      "/ip6/::/tcp/4001/quic-v1/webtransport",
      "/ip4/0.0.0.0/tcp/4001",
      "/ip4/0.0.0.0/udp/4001/quic",
      "/ip4/0.0.0.0/udp/4001/quic-v1",
      "/ip4/0.0.0.0/udp/4001/quic-v1/webtransport"
    ]
  },
  "AutoNAT": {},
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
  ],
  "Reprovider": {},
  "Routing": {
    "Methods": {},
    "Routers": {}
  },
  "Swarm": {
    "AddrFilters": [
      "/ip4/10.0.0.0/ipcidr/8",
      "/ip4/12.0.0.0/udp/4001/quic",
      "/ip4/12.0.0.0/udp/4001/quic-v1",
      "/ip4/12.0.0.0/udp/4001/quic-v1/webtransport"
    ],
    "ConnMgr": {}
  }
}`

var customConfig = `{
  "Addresses": {
    "Announce": [
      "/ip4/3.0.0.0/tcp/4001"
    ],
    "AppendAnnounce": [
      "/ip4/2.0.0.0/tcp/4001"
    ],
    "NoAnnounce": [
      "/ip4/1.0.0.0/tcp/4001"
    ],
    "Swarm": [
      "/ip6/::/tcp/4001",
      "/ip4/0.0.0.0/tcp/4001"
    ]
  },
  "AutoNAT": {},
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
  ],
  "Reprovider": {
    "Interval": "12h",
	"Strategy": "roots"
  },
  "Routing": {
	"Methods": {},
	"Routers": {},
    "Type": "dhtclient"
  },
  "Swarm": {
    "AddrFilters": [
	  "/ip4/10.0.0.0/ipcidr/8"
	],
	"ConnMgr": {
		"GracePeriod": "20s",
		"HighWater": 10000,
		"LowWater": 5000,
		"Type": "basic"
	}
  }
}`

func TestDefaultConfigMigration(t *testing.T) {
	testConfigMigration(beforeDefaultConfig, afterDefaultConfig, t)
}
func TestCustomConfigMigration(t *testing.T) {
	// user config with custom values for migrated fields is left untouched
	testConfigMigration(customConfig, customConfig, t)
}

func testConfigMigration(beforeConfig string, afterConfig string, t *testing.T) {
	in := strings.NewReader(beforeConfig)
	out := new(bytes.Buffer)

	err := convert(in, out)
	if err != nil {
		t.Fatal(err)
	}

	_, err = out.Write([]byte("\n"))
	if err != nil {
		t.Fatal(err)
	}

	forward := out.String()
	if noSpace(forward) != noSpace(afterConfig) {
		t.Fatalf("Mismatch\nConversion produced:\n%s\nExpected:\n%s\n", forward, afterConfig)
	}
}

var whitespaceRe = regexp.MustCompile(`\s`)

func noSpace(str string) string {
	return whitespaceRe.ReplaceAllString(str, "")
}
