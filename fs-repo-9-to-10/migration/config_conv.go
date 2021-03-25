package mg9

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/ipfs/fs-repo-migrations/fs-repo-9-to-10/atomicfile"
	log "github.com/ipfs/fs-repo-migrations/tools/stump"
)

var (
	ip4BootstrapAddr  = "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
	quicBootstrapAddr = "/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
)

// convArray does an inplace conversion of an array of strings
// configuration from one version to another
type convArray func([]string) []string

// convAddrs does an inplace conversion of the swarm, announce
// and noAnnounce arrays of strings from one version to another
type convAddrs func([]string, []string, []string) ([]string, []string, []string)

// convertFile converts a config file from one version to another
func convertFile(path string, convBootstrap convArray, convAddresses convAddrs) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}

	// Create a temp file to write the output to on success
	out, err := atomicfile.New(path, 0600)
	if err != nil {
		in.Close()
		return err
	}

	err = convert(in, out, convBootstrap, convAddresses)

	in.Close()

	if err != nil {
		// There was an error so abort writing the output and clean up temp file
		out.Abort()
	} else {
		// Write the output and clean up temp file
		out.Close()
	}

	return err
}

// convert converts the config from one version to another
func convert(in io.Reader, out io.Writer, convBootstrap convArray, convAddresses convAddrs) error {
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

	// Convert addresses config
	convertAddresses(confMap, convAddresses)

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

// Convert Bootstrap addresses to/from QUIC
func convertBootstrap(confMap map[string]interface{}, conv convArray) {
	bootstrapi, _ := confMap["Bootstrap"].([]interface{})
	if bootstrapi == nil {
		log.Log("No Bootstrap field in config, skipping")
		return
	}
	confMap["Bootstrap"] = conv(toStringArray(bootstrapi))
}

// Convert Addresses.Swarm, Addresses.Announce, Addresses.NoAnnounce to/from QUIC
func convertAddresses(confMap map[string]interface{}, conv convAddrs) {
	addressesi, _ := confMap["Addresses"].(map[string]interface{})
	if addressesi == nil {
		log.Log("Addresses field missing or of the wrong type")
		return
	}

	swarm := toStringArray(addressesi["Swarm"])
	announce := toStringArray(addressesi["Announce"])
	noAnnounce := toStringArray(addressesi["NoAnnounce"])

	s, a, na := conv(swarm, announce, noAnnounce)
	addressesi["Swarm"] = s
	addressesi["Announce"] = a
	addressesi["NoAnnounce"] = na
}

func toStringArray(el interface{}) []string {
	listi, _ := el.([]interface{})
	if listi == nil {
		return []string{}
	}

	list := make([]string, len(listi))
	for i := range listi {
		list[i] = listi[i].(string)
	}
	return list
}

// Add QUIC Bootstrap address
func ver9to10Bootstrap(bootstrap []string) []string {
	hasOld := false
	hasNew := false
	res := make([]string, 0, len(bootstrap)+1)
	for _, addr := range bootstrap {
		// Upgrade /ipfs & /p2p. This should have happened in migration
		// 7-to-8, but that migration wouldn't run at all if we already
		// had the new bootstrappers.
		addr = strings.Replace(addr, "/ipfs/Qm", "/p2p/Qm", -1)
		addr = strings.Replace(addr, "/ipfs/1", "/p2p/1", -1)

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

// For each TCP address, add a QUIC address
func ver9to10Addresses(swarm, announce, noAnnounce []string) ([]string, []string, []string) {
	for _, addr := range append(swarm, append(announce, noAnnounce...)...) {
		// If the old configuration already has a quic address in it, assume
		// the user has already set up their addresses for quic and leave
		// things as they are
		if strings.HasSuffix(addr, "/quic") {
			return swarm, announce, noAnnounce
		}
	}

	return addQuic(swarm), addQuic(announce), addQuic(noAnnounce)
}

var tcpRegexp = regexp.MustCompile(`/tcp/([0-9]+)$`)

func addQuic(addrs []string) []string {
	res := make([]string, 0, len(addrs)*2)
	for _, addr := range addrs {
		res = append(res, addr)
	}

	// For each tcp address, add a corresponding quic address
	for _, addr := range addrs {
		if tcpRegexp.MatchString(addr) {
			res = append(res, tcpRegexp.ReplaceAllString(addr, `/udp/$1/quic`))
		}
	}

	return res
}
