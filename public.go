// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"path/filepath"
	"strings"
	"time"
)

// Explicit public projection. Configuration, raw observations, error messages and
// unknown telemetry fields are never reachable from the public server.
func selectFields(source Row, fields string) Row {
	out := Row{}
	for _, key := range strings.Fields(fields) {
		if v, ok := source[key]; ok {
			out[key] = v
		}
	}
	return out
}
func publicSnapshot(c Config, nodes []Row, stats, vers Row, origin string) Row {
	publicNodes := []any{}
	for _, n := range nodes {
		if !nonempty(n["node"]) {
			continue
		}
		p := selectFields(n, "node wlan_ip last_seen uptime model hardware firmware_version ssid channel chanbw lat lon api_version firmware_mfg grid_square description mesh_supernode mesh_gateway meshRF freq antGain antBeam antDesc antBuiltin hopsAway")
		p["band"] = band(n)
		if truth(n["mesh_supernode"]) {
			p["band"] = "supernode"
		} else if str(n["meshRF"]) == "off" || strings.EqualFold(str(n["channel"]), "none") {
			p["band"] = "noRF"
		}
		if p["band"] == "" {
			p["band"] = "other"
		}
		services := []any{}
		if list, ok := n["services"].([]any); ok {
			for _, service := range list {
				services = append(services, selectFields(obj(service), "name link protocol"))
			}
		}
		p["services"] = services
		links := Row{}
		for ip, value := range obj(n["link_info"]) {
			link := selectFields(obj(value), "hostname linkType interface olsrInterface linkLat linkLon distanceKM distanceMiles bearing tx_rate rx_rate expected_throughput signal noise linkCost pathCost linkQuality neighborLinkQuality")
			trackers := Row{}
			for id, t := range obj(obj(value)["lqm_trackers"]) {
				trackers[id] = selectFields(obj(t), "device subdevice type mac ipv6ll canonical_ip ip hostname band lastseen lastup rev_lastseen routable signal noise snr avg_snr rev_snr tx_bitrate rx_bitrate avg_tx_bitrate avg_rx_bitrate tx_packets tx_retries tx_fail avg_tx_packets avg_tx_retries avg_tx_fail lq avg_lq rev_lq quality rev_quality tx_quality ping_quality rev_ping_quality ping_success_time rev_ping_success_time rxcost txcost rtt metric route_count babel_metric babel_route_count hello_interval update_interval")
			}
			link["lqm_trackers"] = trackers
			links[ip] = link
		}
		p["link_info"] = links
		info := obj(obj(n["lqm"])["info"])
		p["lqm_clock"] = selectFields(info, "now start")
		publicNodes = append(publicNodes, p)
	}
	return Row{"schema_version": 1, "generated_at": time.Now().UTC().Format(time.RFC3339), "mapInfo": Row{"version": version, "title": c.s("pageTitle"), "localnode": origin, "attribution": c.s("attribution"), "kilometers": c.b("kilometers"), "mapCenterCoords": []any{num(c.Arrays["map_center_coordinates"]["lat"]), num(c.Arrays["map_center_coordinates"]["lon"])}, "mapInitialZoom": num(c.s("map_initial_zoom_level")), "mapTileServers": c.Arrays["mapTileServers"]}, "pollingInfo": stats, "arednVersions": vers, "nodes": publicNodes}
}
func writePublicSnapshot(c Config, nodes []Row, stats, vers Row, origin string) error {
	return jsonWrite(filepath.Join(c.Output, "public_map.json"), publicSnapshot(c, nodes, stats, vers, origin))
}
