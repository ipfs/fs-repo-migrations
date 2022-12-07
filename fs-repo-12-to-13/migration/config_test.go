package mg12

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
)

var beforeConfig = `{
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
	"ConnMgr": {
		"GracePeriod": "20s",
		"HighWater": 900,
		"LowWater": 600,
		"Type": "basic"
	}
  }
}`

var afterConfig = `{
  "Reprovider": {},
  "Routing": {
	"Methods": {},
	"Routers": {}
  },
  "Swarm": {
    "ConnMgr": {}
  }
}`

func TestKubo18ConfigMigration(t *testing.T) {
	out := new(bytes.Buffer)

	data, err := ioutil.ReadAll(strings.NewReader(beforeConfig))
	if err != nil {
		t.Fatal(err)
	}

	confMap := make(map[string]interface{})
	if err = json.Unmarshal(data, &confMap); err != nil {
		t.Fatal(err)
	}

	// Kubo 0.18
	convertRouting(confMap)
	convertReprovider(confMap)
	convertConnMgr(confMap)

	fixed, err := json.MarshalIndent(confMap, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := out.Write(fixed); err != nil {
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
