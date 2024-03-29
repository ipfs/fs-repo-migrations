package mg12

import (
	log "github.com/ipfs/fs-repo-migrations/tools/stump"
)

// convertQuicAddrs converts quic multiaddrs to v1 and enables webtransport listener
// https://github.com/ipfs/kubo/issues/9410
// https://github.com/ipfs/kubo/issues/9292
func convertQuicAddrs(confMap map[string]any) {
	// run this first to avoid having both quic and quic-v1 webtransport addresses
	runOnAllAddressFields(confMap, multiaddrPatternReplace(false, "/quic/webtransport", "/quic-v1/webtransport"))

	runOnAllAddressFields(confMap, multiaddrPatternReplace(true, "/quic", "/quic-v1", "/p2p-circuit"))
	runOnAllAddressFields(confMap, multiaddrPatternReplace(true, "/quic-v1", "/quic-v1/webtransport", "/p2p-circuit", "/webtransport"))
}

// convertRouting converts Routing.Type to implicit default
// https://github.com/ipfs/kubo/pull/9475
func convertRouting(confMap map[string]any) {
	routing, _ := confMap["Routing"].(map[string]any)
	if routing == nil {
		log.Log("No Routing field in config, skipping")
		return
	}

	routers, ok := routing["Routers"].(map[string]any)
	if len(routers) > 0 {
		log.Log("Custom Routing.Routers in config, skipping")
		return
	}
	methods, ok := routing["Methods"].(map[string]any)
	if len(methods) > 0 {
		log.Log("Custom Routing.Methods in config, skipping")
		return
	}

	rType, ok := routing["Type"].(string)
	if !ok {
		log.Log("No Routing.Type field in config, skipping")
		return
	}
	if rType == "dht" || rType == "" {
		delete(routing, "Type")
	} else {
		log.Log("Routing.Type settings is different than the old default, skipping")
	}
}

// convertReprovider converts Reprovider to implicit defaults
// https://github.com/ipfs/kubo/pull/9326
func convertReprovider(confMap map[string]any) {
	reprovider, _ := confMap["Reprovider"].(map[string]any)
	if reprovider == nil {
		log.Log("No Reprovider field in config, skipping")
		return
	}

	interval, ok := reprovider["Interval"].(string)
	if !ok {
		log.Log("No Reprovider.Interval field in config, skipping")
		return
	}

	strategy, ok := reprovider["Strategy"].(string)
	if !ok {
		log.Log("No Reprovider.Strategy field in config, skipping")
		return
	}

	if interval == "12h" && strategy == "all" {
		delete(reprovider, "Strategy")
		delete(reprovider, "Interval")
	} else {
		log.Log("Reprovider settings are different than the old default, skipping")
	}
}

// convertConnMgr converts Swarm.ConnMgr to implicit defaults
// https://github.com/ipfs/kubo/pull/9467
func convertConnMgr(confMap map[string]any) {
	swarm, _ := confMap["Swarm"].(map[string]any)
	if swarm == nil {
		log.Log("No Swarm field in config, skipping")
		return
	}
	connmgr, _ := swarm["ConnMgr"].(map[string]any)
	if connmgr == nil {
		log.Log("No Swarm.ConnMgr field in config, skipping")
		return
	}
	cmType, ok := connmgr["Type"].(string)
	if !ok {
		log.Log("No Swarm.ConnMgr.Type field in config, skipping")
		return
	}
	cmLowWater, ok := connmgr["LowWater"].(float64)
	if !ok {
		log.Log("No Swarm.ConnMgr.LowWater field in config, skipping")
		return
	}
	cmHighWater, ok := connmgr["HighWater"].(float64)
	if !ok {
		log.Log("No Swarm.ConnMgr.HighWater field in config, skipping")
		return
	}
	cmGrace, ok := connmgr["GracePeriod"].(string)
	if !ok {
		log.Log("No Swarm.ConnMgr.GracePeriod field in config, skipping")
		return
	}

	if cmType == "basic" && int(cmLowWater) == 600 && int(cmHighWater) == 900 && cmGrace == "20s" {
		delete(connmgr, "Type")
		delete(connmgr, "GracePeriod")
		delete(connmgr, "LowWater")
		delete(connmgr, "HighWater")
	} else {
		log.Log("Swarm.ConnMgr settings are different than the old defaults, skipping")
	}
}
