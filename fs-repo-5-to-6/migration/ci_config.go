package mg5

import (
	"strings"
)

// ciConfig is a case insensitive map to represent a IPFS
// configuration when unmarshaled as a map[string]interface{} instead
// of specialized structures.  It is needed becuase JSON library is
// case insensitive when matching json values to struct fields
type ciConfig struct {
	conf   map[string]interface{}
	fromLC map[string]string
}

func newCiConfig(conf map[string]interface{}) ciConfig {
	return ciConfig{
		conf:   conf,
		fromLC: lcMap(conf),
	}
}

func lcMap(conf map[string]interface{}) map[string]string {
	fromLC := make(map[string]string)
	for key, _ := range conf {
		fromLC[strings.ToLower(key)] = key
	}
	return fromLC
}

// get gets a key, returns nil if the key doesn't exist
func (c ciConfig) get(origkey string) interface{} {
	lckey := strings.ToLower(origkey)
	key, ok := c.fromLC[lckey]
	if !ok {
		return nil
	}
	return c.conf[key] // should not panic, key should exist
}

func (c ciConfig) set(origkey string, val interface{}) {
	lckey := strings.ToLower(origkey)
	key, ok := c.fromLC[lckey]
	if ok {
		c.conf[key] = val
	}
	c.fromLC[lckey] = origkey
	c.conf[origkey] = val
}

func (c ciConfig) del(origkey string) {
	lckey := strings.ToLower(origkey)
	key, ok := c.fromLC[lckey]
	if ok {
		delete(c.conf, key)
	}
}
