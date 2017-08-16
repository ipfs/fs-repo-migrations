package mg5

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
)

// convFunc does an inplace conversion of the "datastore"
// configuration (represted as a map[string]interface{}) from one
// version to another
type convFunc func(ds ciConfig) error

// convertFile converts a config file from one version to another, the
// converted config is stored in
func convertFile(orig string, new string, convFunc convFunc) (ciConfig, error) {
	in, err := os.Open(orig)
	if err != nil {
		return ciConfig{}, err
	}
	out, err := os.Create(new)
	if err != nil {
		return ciConfig{}, err
	}
	return convert(in, out, convFunc)
}

// convert converts the config from one version to another, returns
// the converted config as a map[string]interface{}
func convert(in io.Reader, out io.Writer, convFunc convFunc) (ciConfig, error) {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return ciConfig{}, err
	}
	confMap := make(map[string]interface{})
	err = json.Unmarshal(data, &confMap)
	if err != nil {
		return ciConfig{}, err
	}
	conf := newCiConfig(confMap)
	ds, _ := conf.get("datastore").(map[string]interface{})
	if ds == nil {
		return ciConfig{}, fmt.Errorf("Datastore field missing or of the wrong type")
	}
	err = convFunc(newCiConfig(ds))
	if err != nil {
		return ciConfig{}, err
	}
	fixed, err := json.MarshalIndent(confMap, "", "  ")
	if err != nil {
		return ciConfig{}, err
	}
	out.Write(fixed)
	out.Write([]byte("\n"))
	return conf, nil
}

func ver5to6(ds ciConfig) error {
	noSyncVal := ds.get("nosync")
	if noSyncVal == nil {
		noSyncVal = interface{}(false)
	}
	noSync, ok := noSyncVal.(bool)
	if !ok {
		return fmt.Errorf("unsupported value for Datastore.NoSync fields: %v", noSyncVal)
	}
	ds.del("nosync")

	dsTypeVal := ds.get("type")
	if dsTypeVal == nil {
		dsTypeVal = interface{}("")
	}
	dsType, ok := dsTypeVal.(string)
	if !ok || (dsType != "default" && dsType != "leveldb" && dsType != "") {
		return fmt.Errorf("unsupported value for Datastore.Type fields: %s", dsType)
	}
	ds.del("type")

	// Path and Params never appear to have been used so just delete them
	ds.del("path")
	ds.del("params")

	ds.set("Spec", datastoreSpec(!noSync))
	return nil
}

func ver6to5(ds ciConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("incompatible config detected, downgrade not possible: %v", r.(error))
		}
	}()
	spec := ds.get("Spec").(map[string]interface{})
	if spec == nil {
		return fmt.Errorf("Datastore.Spec field missing or not a json object, can't downgrade")
	}
	mounts, _ := spec["mounts"].([]interface{})
	if mounts == nil {
		return fmt.Errorf("Datastore.Spec.mounts field is missing or not an array")
	}
	var root, blocks interface{}
	sync := true
	for _, mount := range mounts {
		switch mountpoint := mount.(map[string]interface{})["mountpoint"].(string); mountpoint {
		case "/blocks":
			sync = mount.(map[string]interface{})["child"].(map[string]interface{})["sync"].(bool)
			blocks = mount
		case "/":
			root = mount
		default:
			return fmt.Errorf("unknown mountpoint: %s", mountpoint)
		}
	}
	// normalize spec
	spec["mounts"] = []interface{}{blocks, root}
	expected := datastoreSpec(sync)
	if !reflect.DeepEqual(spec, expected) {
		return fmt.Errorf("Datastore.Spec field not of a supported value, can't downgrade")
	}
	ds.del("Spec")
	ds.set("Type", "leveldb")
	ds.set("Params", nil)
	ds.set("NoSync", !sync)
	println("ver6to5 done!")
	return nil
}

func datastoreSpec(sync bool) map[string]interface{} {
	return map[string]interface{}{
		"type": "mount",
		"mounts": []interface{}{
			map[string]interface{}{
				"mountpoint": "/blocks",
				"type":       "measure",
				"prefix":     "flatfs.datastore",
				"child": map[string]interface{}{
					"type":      "flatfs",
					"path":      "blocks",
					"sync":      sync,
					"shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
				},
			},
			map[string]interface{}{
				"mountpoint": "/",
				"type":       "measure",
				"prefix":     "leveldb.datastore",
				"child": map[string]interface{}{
					"type":        "levelds",
					"path":        "datastore",
					"compression": "none",
				},
			},
		},
	}
}
