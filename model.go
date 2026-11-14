// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
)

type Row map[string]any

//go:embed hardware.json
var hardwareJSON []byte
var hardware map[string]string

func init() {
	if err := json.Unmarshal(hardwareJSON, &hardware); err != nil {
		panic(err)
	}
}
func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func num(v any) float64 { n, _ := strconv.ParseFloat(str(v), 64); return n }
func obj(v any) Row {
	switch m := v.(type) {
	case map[string]any:
		return Row(m)
	case Row:
		return m
	}
	return Row{}
}
func truth(v any) bool    { s := strings.ToLower(str(v)); return s == "true" || s == "1" || s == "on" }
func nonempty(v any) bool { return v != nil && strings.TrimSpace(str(v)) != "" }
func parseNode(raw Row, ip string) Row {
	r := Row{"meshRF": "on", "ethInf": "eth0", "wlanInf": "wlan0", "node": "", "wlan_ip": ip, "uptime": "Not Available", "loadavg": "Not Available", "model": "Not Available", "firmware_version": "Not Available", "ssid": "None", "channel": "None", "chanbw": "None", "tunnel_installed": "false", "active_tunnel_count": "0", "lat": 0.0, "lon": 0.0, "wifi_mac_address": "", "api_version": "0.0.0", "board_id": "Not Available", "firmware_mfg": "Not Available", "grid_square": "Not Available", "lan_ip": "Not Available", "services": "Not Available", "description": "", "mesh_supernode": "false", "mesh_gateway": "false", "freq": "None", "link_info": Row{}, "hopsAway": 0, "antGain": 0, "antBeam": 0, "antDesc": "Not Available", "antBuiltin": "false", "hardware": "Unknown", "eth3975": 0}
	copyFields := func(m Row) {
		for k, v := range m {
			switch v.(type) {
			case Row, map[string]any, []any:
				if k != "loads" && k != "loadavg" && k != "link_info" && k != "services" {
					continue
				}
			}
			if !nonempty(v) {
				continue
			}
			switch k {
			case "status":
				r["meshRF"] = v
			case "mesh_gateway", "mesh_supernode":
				r[k] = strconv.FormatBool(truth(v))
			case "loads":
				r["loadavg"] = v
			default:
				if _, ok := r[k]; ok && k != "interfaces" {
					r[k] = v
				}
			}
		}
	}
	// Old API releases used different section names. Read scalar fields from
	// every section before applying the current section names and root overrides.
	keys := []string{}
	for k := range raw {
		if k != "interfaces" && k != "link_info" && k != "services_local" && k != "meshrf1" && k != "lqm" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		copyFields(obj(raw[k]))
	}
	// Explicit order avoids depending on Go map iteration order.
	for _, k := range []string{"sysinfo", "system", "node_details", "meshrf", "tunnels", "location"} {
		copyFields(obj(raw[k]))
	}
	copyFields(raw)
	if ip := str(raw["ip"]); net.ParseIP(ip) != nil {
		r["wlan_ip"] = ip
	}
	if v, ok := raw["gridsquare"]; ok {
		r["grid_square"] = v
	}
	if v, ok := raw["services_local"]; ok {
		r["services"] = v
	}
	if v, ok := raw["link_info"]; ok {
		r["link_info"] = v
	}
	if truth(r["meshRF"]) {
		a := obj(obj(raw["meshrf"])["antenna"])
		for k, d := range map[string]string{"gain": "antGain", "beamwidth": "antBeam", "description": "antDesc", "builtin": "antBuiltin"} {
			if v, ok := a[k]; ok {
				r[d] = v
			}
		}
	}
	var interfaces []Row
	switch v := raw["interfaces"].(type) {
	case []any:
		for _, i := range v {
			interfaces = append(interfaces, obj(i))
		}
	case map[string]any:
		for name, i := range v {
			m := obj(i)
			m["name"] = name
			interfaces = append(interfaces, m)
		}
	}
	priority := 0
	if net.ParseIP(str(raw["ip"])) != nil {
		priority = 10
	}
	for _, i := range interfaces {
		name := str(i["name"])
		if name == "wlan0" && nonempty(i["mac"]) {
			r["wifi_mac_address"] = i["mac"]
		}
		if name == "br-lan" && nonempty(i["ip"]) {
			r["lan_ip"] = i["ip"]
		}
		p := 0
		switch name {
		case "wlan0", "wlan1", "br-nomesh":
			p = 1
		case "eth0.3975", "eth1.3975":
			p = 2
		case "br0":
			if strings.HasPrefix(str(i["ip"]), "10.") {
				p = 3
			}
		}
		if p > priority && nonempty(i["ip"]) && str(i["ip"]) != "none" {
			r["wlan_ip"] = i["ip"]
			priority = p
			if p == 2 {
				r["eth3975"] = 1
			}
		}
	}
	id := str(r["board_id"])
	if id == "0x0000" {
		id = str(r["model"])
	}
	if h, ok := hardware[id]; ok {
		r["hardware"] = h
	}
	for _, k := range []string{"lat", "lon"} {
		r[k] = num(r[k])
	}
	//trying to get rid of duplicates!
	r["node"] = canonicalName(str(r["node"]))
	
	attachTelemetry(r, raw)
	return r
}
func band(r Row) string {
	f := num(r["freq"])
	switch {
	case f >= 800 && f <= 1000:
		return "900"
	case f >= 2300 && f <= 2500:
		return "2ghz"
	case f >= 3300 && f <= 3500:
		return "3ghz"
	case f >= 4900 && f <= 5900:
		return "5ghz"
	}
	switch str(r["board_id"]) {
	case "0xe009", "0xe1b9", "0xe239":
		return "900"
	}
	c, e := strconv.ParseFloat(str(r["channel"]), 64)
	if e != nil {
		return ""
	}
	if strings.EqualFold(str(r["channel"]), "none") {
		return ""
	}
	switch {
	case c >= -2 && c <= 11:
		return "2ghz"
	case c >= 76 && c <= 99 || c >= 3380 && c <= 3495:
		return "3ghz"
	case c >= 37 && c <= 64 || c >= 100 && c <= 184:
		return "5ghz"
	}
	return ""
}
func located(r Row) bool {
	return num(r["lat"]) != 0 && num(r["lon"]) != 0 && math.Abs(num(r["lat"])) <= 90 && math.Abs(num(r["lon"])) <= 180
}
func distance(a, b Row) Row {
	rad := math.Pi / 180
	la, lb := num(a["lat"])*rad, num(b["lat"])*rad
	dl := (num(b["lon"]) - num(a["lon"])) * rad
	h := math.Pow(math.Sin((lb-la)/2), 2) + math.Cos(la)*math.Cos(lb)*math.Pow(math.Sin(dl/2), 2)
	angle := 2 * math.Asin(math.Sqrt(math.Min(1, math.Max(0, h))))
	bearing := math.Mod(math.Atan2(math.Sin(dl)*math.Cos(lb), math.Cos(la)*math.Sin(lb)-math.Sin(la)*math.Cos(lb)*math.Cos(dl))/rad+360, 360)
	return Row{"distanceKM": math.Round(angle*6371*100) / 100, "distanceMiles": math.Round(angle*3959*100) / 100, "bearing": math.Floor(bearing)}
}
func enrich(nodes []Row) int {
	index := map[string]Row{}
	for _, r := range nodes {
		index[str(r["wlan_ip"])] = r
	}
	count := 0
	for _, r := range nodes {
		for ip, v := range obj(r["link_info"]) {
			l := obj(v)
			for _, k := range []string{"linkLat", "linkLon", "distanceKM", "distanceMiles", "bearing"} {
				delete(l, k)
			}
			other := index[ip]
			if located(r) && located(other) {
				l["linkLat"] = num(other["lat"])
				l["linkLon"] = num(other["lon"])
				if str(l["linkType"]) == "RF" {
					for k, v := range distance(r, other) {
						l[k] = v
					}
				}
				count++
			}
		}
	}
	return count
}
