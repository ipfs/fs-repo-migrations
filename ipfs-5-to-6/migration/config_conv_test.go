package mg5

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func TestOldToNew(t *testing.T) {
	in := strings.NewReader(oldConfig)
	out := new(bytes.Buffer)
	_, err := convert(in, out, ver5to6)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewToOld(t *testing.T) {
	in := strings.NewReader(newConfig)
	out := new(bytes.Buffer)
	_, err := convert(in, out, ver6to5)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOldToNewRoundtrip(t *testing.T) {
	out := new(bytes.Buffer)
	_, err := convert(strings.NewReader(oldConfigNoPath), out, ver5to6)
	if err != nil {
		t.Fatal(err)
	}
	convConfig := out.String()
	out = new(bytes.Buffer)
	_, err = convert(strings.NewReader(convConfig), out, ver6to5)
	if err != nil {
		t.Fatal(err)
	}
	revertedConfig := out.String()
	if oldConfigNoPath != revertedConfig {
		ioutil.WriteFile("test_config.orig", []byte(oldConfigNoPath), 0644)
		ioutil.WriteFile("test_config.new", []byte(revertedConfig), 0644)
		t.Fatal("roundtrip config's differ see file test_config.orig and test_config.new")
	}
}

func TestNewToOldRoundtrip(t *testing.T) {
	out := new(bytes.Buffer)
	_, err := convert(strings.NewReader(newConfig), out, ver6to5)
	if err != nil {
		t.Fatal(err)
	}
	convConfig := out.String()
	out = new(bytes.Buffer)
	_, err = convert(strings.NewReader(convConfig), out, ver5to6)
	if err != nil {
		t.Fatal(err)
	}
	revertedConfig := out.String()
	if newConfig != revertedConfig {
		ioutil.WriteFile("test_config.orig", []byte(oldConfigNoPath), 0644)
		ioutil.WriteFile("test_config.new", []byte(revertedConfig), 0644)
		t.Fatal("roundtrip config's differ see file test_config.orig and test_config.new")
	}
}

// oldConfig is an old configuration file with Datastore.NoSync set to the non-default value of true
var oldConfig = `{
  "API": {
    "HTTPHeaders": null
  },
  "Addresses": {
    "API": "/ip4/127.0.0.1/tcp/5001",
    "Gateway": "/ip4/127.0.0.1/tcp/8080",
    "Swarm": [
      "/ip4/0.0.0.0/tcp/4001",
      "/ip6/::/tcp/4001"
    ]
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
  ],
  "Datastore": {
    "BloomFilterSize": 0,
    "GCPeriod": "1h",
    "HashOnRead": false,
    "NoSync": true,
    "Params": null,
    "Path": "/tmp/ipfs99/datastore",
    "StorageGCWatermark": 90,
    "StorageMax": "10GB",
    "Type": "leveldb"
  },
  "Discovery": {
    "MDNS": {
      "Enabled": true,
      "Interval": 10
    }
  },
  "Experimental": {
    "FilestoreEnabled": false
  },
  "Gateway": {
    "HTTPHeaders": {
      "Access-Control-Allow-Headers": [
        "X-Requested-With"
      ],
      "Access-Control-Allow-Methods": [
        "GET"
      ],
      "Access-Control-Allow-Origin": [
        "*"
      ]
    },
    "PathPrefixes": [],
    "RootRedirect": "",
    "Writable": false
  },
  "Identity": {
    "PeerID": "QmRBh36P5E5iLjLYsAh6dvavazfLYBW6XiFo9yCTrk2A2R"
  },
  "Ipns": {
    "RecordLifetime": "",
    "RepublishPeriod": "",
    "ResolveCacheSize": 128
  },
  "Mounts": {
    "FuseAllowOther": false,
    "IPFS": "/ipfs",
    "IPNS": "/ipns"
  },
  "Reprovider": {
    "Interval": "12h"
  },
  "SupernodeRouting": {
    "Servers": null
  },
  "Swarm": {
    "AddrFilters": null,
    "DisableBandwidthMetrics": false
  },
  "Tour": {
    "Last": ""
  }
}
`

// oldConfig is an old configuration file with Datastore.NoSync set to the non-default value of true
var oldConfigNoPath = `{
  "API": {
    "HTTPHeaders": null
  },
  "Addresses": {
    "API": "/ip4/127.0.0.1/tcp/5001",
    "Gateway": "/ip4/127.0.0.1/tcp/8080",
    "Swarm": [
      "/ip4/0.0.0.0/tcp/4001",
      "/ip6/::/tcp/4001"
    ]
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
  ],
  "Datastore": {
    "BloomFilterSize": 0,
    "GCPeriod": "1h",
    "HashOnRead": false,
    "NoSync": true,
    "Params": null,
    "StorageGCWatermark": 90,
    "StorageMax": "10GB",
    "Type": "leveldb"
  },
  "Discovery": {
    "MDNS": {
      "Enabled": true,
      "Interval": 10
    }
  },
  "Experimental": {
    "FilestoreEnabled": false
  },
  "Gateway": {
    "HTTPHeaders": {
      "Access-Control-Allow-Headers": [
        "X-Requested-With"
      ],
      "Access-Control-Allow-Methods": [
        "GET"
      ],
      "Access-Control-Allow-Origin": [
        "*"
      ]
    },
    "PathPrefixes": [],
    "RootRedirect": "",
    "Writable": false
  },
  "Identity": {
    "PeerID": "QmRBh36P5E5iLjLYsAh6dvavazfLYBW6XiFo9yCTrk2A2R"
  },
  "Ipns": {
    "RecordLifetime": "",
    "RepublishPeriod": "",
    "ResolveCacheSize": 128
  },
  "Mounts": {
    "FuseAllowOther": false,
    "IPFS": "/ipfs",
    "IPNS": "/ipns"
  },
  "Reprovider": {
    "Interval": "12h"
  },
  "SupernodeRouting": {
    "Servers": null
  },
  "Swarm": {
    "AddrFilters": null,
    "DisableBandwidthMetrics": false
  },
  "Tour": {
    "Last": ""
  }
}
`

// newConfig is a new style config with the "sync" value set to the non-default value of false
var newConfig = `{
  "API": {
    "HTTPHeaders": null
  },
  "Addresses": {
    "API": "/ip4/127.0.0.1/tcp/5001",
    "Gateway": "/ip4/127.0.0.1/tcp/8080",
    "Swarm": [
      "/ip4/0.0.0.0/tcp/4001",
      "/ip6/::/tcp/4001"
    ]
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
  ],
  "Datastore": {
    "BloomFilterSize": 0,
    "GCPeriod": "1h",
    "HashOnRead": false,
    "Spec": {
      "mounts": [
        {
          "child": {
            "path": "blocks",
            "shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
            "sync": true,
            "type": "flatfs"
          },
          "mountpoint": "/blocks",
          "prefix": "flatfs.datastore",
          "type": "measure"
        },
        {
          "child": {
            "compression": "none",
            "path": "datastore",
            "type": "levelds"
          },
          "mountpoint": "/",
          "prefix": "leveldb.datastore",
          "type": "measure"
        }
      ],
      "type": "mount"
    },
    "StorageGCWatermark": 90,
    "StorageMax": "10GB"
  },
  "Discovery": {
    "MDNS": {
      "Enabled": true,
      "Interval": 10
    }
  },
  "Experimental": {
    "FilestoreEnabled": false
  },
  "Gateway": {
    "HTTPHeaders": {
      "Access-Control-Allow-Headers": [
        "X-Requested-With"
      ],
      "Access-Control-Allow-Methods": [
        "GET"
      ],
      "Access-Control-Allow-Origin": [
        "*"
      ]
    },
    "PathPrefixes": [],
    "RootRedirect": "",
    "Writable": false
  },
  "Identity": {
    "PeerID": "QmPPSn9u15r423WWrNoB9WcFTxT1tcwS735bp6khNx1qLr"
  },
  "Ipns": {
    "RecordLifetime": "",
    "RepublishPeriod": "",
    "ResolveCacheSize": 128
  },
  "Mounts": {
    "FuseAllowOther": false,
    "IPFS": "/ipfs",
    "IPNS": "/ipns"
  },
  "Reprovider": {
    "Interval": "12h"
  },
  "SupernodeRouting": {
    "Servers": null
  },
  "Swarm": {
    "AddrFilters": null,
    "DisableBandwidthMetrics": false
  },
  "Tour": {
    "Last": ""
  }
}
`
