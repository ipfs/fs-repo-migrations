package mg9

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	log "github.com/ipfs/fs-repo-migrations/stump"
)

var (
	ip4BootstrapAddr  = "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
	quicBootstrapAddr = "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
)

// convArray does an inplace conversion of an array of strings
// configuration from one version to another
type convArray func([]string) []string

// convertFile converts a config file from one version to another
func convertFile(orig string, new string, enableQuic bool, convBootstrap convArray, convSwarm convArray) error {
	in, err := os.Open(orig)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(new)
	if err != nil {
		return err
	}
	defer out.Close()

	// Make sure file has 600 permissions
	if err := out.Chmod(0600); err != nil {
		return err
	}

	return convert(in, out, enableQuic, convBootstrap, convSwarm)
}

// convert converts the config from one version to another
func convert(in io.Reader, out io.Writer, enableQuic bool, convBootstrap convArray, convSwarm convArray) error {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	confMap := make(map[string]interface{})
	if err = json.Unmarshal(data, &confMap); err != nil {
		return err
	}

	// Convert bootstrap config
	convertBootstrap(confMap, convBootstrap)

	// Convert swarm config
	if enableQuic {
		// When enabling quic
		// - Remove experimental option from config
		confEnabled, ok := removeExperimentalQuic(confMap)
		// - Only convert swarm config if experimental quic option was present
		//   and not enabled
		if ok && !confEnabled {
			convertSwarm(confMap, convSwarm)
		}
	} else {
		convertSwarm(confMap, convSwarm)
	}

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

// Remove Experimental.QUIC flag
func removeExperimentalQuic(confMap map[string]interface{}) (bool, bool) {
	confExpi := confMap["Experimental"].(map[string]interface{})
	if confExpi == nil {
		return false, false
	}
	enabledi, ok := confExpi["QUIC"]
	if !ok {
		return false, false
	}

	enabled, ok := enabledi.(bool)

	delete(confExpi, "QUIC")

	return enabled, ok
}

// Convert Bootstrap addresses to/from QUIC
func convertBootstrap(confMap map[string]interface{}, conv convArray) {
	bootstrapi, _ := confMap["Bootstrap"].([]interface{})
	if bootstrapi == nil {
		log.Log("No Bootstrap field in config, skipping")
		return
	}
	bootstrap := make([]string, len(bootstrapi))
	for i := range bootstrapi {
		bootstrap[i] = bootstrapi[i].(string)
	}
	confMap["Bootstrap"] = conv(bootstrap)
}

// Convert Addresses.Swarm to/from QUIC
func convertSwarm(confMap map[string]interface{}, conv convArray) {
	addressesi, _ := confMap["Addresses"].(map[string]interface{})
	if addressesi == nil {
		log.Log("Addresses field missing or of the wrong type")
		return
	}

	swarmi, _ := addressesi["Swarm"].([]interface{})
	if swarmi == nil {
		log.Log("Addresses.Swarm field missing or of the wrong type")
		return
	}

	swarm := make([]string, len(swarmi))
	for i := range swarmi {
		swarm[i] = swarmi[i].(string)
	}
	addressesi["Swarm"] = conv(swarm)
}

// Add QUIC Bootstrap address
func ver9to10Bootstrap(bootstrap []string) []string {
	hasOld := false
	hasNew := false
	res := make([]string, 0, len(bootstrap)+1)
	for _, addr := range bootstrap {
		res = append(res, addr)
		if addr == ip4BootstrapAddr {
			hasOld = true
		} else if addr == quicBootstrapAddr {
			hasNew = true
		}
	}

	// If the config has the old IP v4 bootstrapper, add the new QUIC
	// bootstrapper
	if hasOld && !hasNew {
		res = append(res, quicBootstrapAddr)
	}

	return res
}

func ver10to9Bootstrap(bootstrap []string) []string {
	// No need to remove the QUIC bootstrapper, just leave things as they are
	return bootstrap
}

var tcpRegexp = regexp.MustCompile(`/tcp/([0-9]+)`)

// For each TCP listener, add a QUIC listener
func ver9to10Swarm(swarm []string) []string {
	res := make([]string, 0, len(swarm)*2)
	for _, addr := range swarm {
		res = append(res, addr)

		// If the old configuration already has a quic address in it, assume
		// the user has already set up their swarm addresses for quic and leave
		// things as they are
		if strings.Contains(addr, "/udp/quic") {
			return swarm
		}
	}

	// For each tcp address, add a corresponding quic address
	for _, addr := range swarm {
		if tcpRegexp.MatchString(addr) {
			res = append(res, tcpRegexp.ReplaceAllString(addr, `/udp/$1/quic`))
		}
	}

	return res
}

var quicRegexp = regexp.MustCompile(`/udp/[0-9]+/quic`)

// Remove QUIC listeners
func ver10to9Swarm(swarm []string) []string {
	// Remove quic addresses
	res := make([]string, 0, len(swarm))
	for _, addr := range swarm {
		if !quicRegexp.MatchString(addr) {
			res = append(res, addr)
		}
	}

	return res
}
