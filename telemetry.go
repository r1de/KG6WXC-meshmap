// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const pollingPath = "/cgi-bin/sysinfo.json?link_info=1&services_local=1&topology=1"

// Most old firmware ignores optional flags. Firmware that rejects lqm gets a
// retry with the previous query; a transport failure does not trigger a retry.
func pollSysinfo(ctx context.Context, client *http.Client, host string) (Row, error) {
	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()
	
	raw, err := nodeJSON(ctx, client, host, pollingPath+"&lqm=1")
	if err == nil && nodeName(raw) != "" {
		return raw, nil
	}
	var transport *url.Error
	if ctx.Err() != nil || errors.As(err, &transport) {
		return raw, err
	}
	return nodeJSON(ctx, client, host, pollingPath)
}
func nodeName(raw Row) string {
	if n, ok := raw["node"].(string); ok {
		return n
	}
	return str(obj(raw["node"])["node"])
}
func canonicalName(s string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".local.mesh")
}

// Retain the whole LQM object, including unknown future fields, nulls, sample
// clocks and unmatched/stale trackers. Only associate trackers with already
// reported links; cached LQM entries must not become new discovery targets.
func attachTelemetry(node Row, raw Row) {
	node["lqm"] = raw["lqm"]
	links := Row{}
	for ip, v := range obj(node["link_info"]) {
		link := Row{}
		for k, value := range obj(v) {
			if k != "lqm_trackers" {
				link[k] = value
			}
		}
		links[ip] = link
	}
	node["link_info"] = links
	for id, v := range obj(obj(obj(raw["lqm"])["info"])["trackers"]) {
		tracker := obj(v)
		if len(tracker) == 0 {
			continue
		}
		peer := ""
		for _, key := range []string{"canonical_ip", "ip"} {
			ip := str(tracker[key])
			if ip != "" {
				if _, ok := links[ip]; ok {
					peer = ip
					break
				}
			}
		}
		if peer == "" && canonicalName(str(tracker["hostname"])) != "" {
			matches := []string{}
			for ip, value := range links {
				link := obj(value)
				if canonicalName(str(link["hostname"])) != canonicalName(str(tracker["hostname"])) {
					continue
				}
				iface := str(link["interface"])
				if iface == "" {
					iface = str(link["olsrInterface"])
				}
				if iface != "" && str(tracker["device"]) != "" && iface != str(tracker["device"]) {
					continue
				}
				matches = append(matches, ip)
			}
			if len(matches) == 1 {
				peer = matches[0]
			}
		}
		if peer == "" {
			continue
		}
		link := obj(links[peer])
		associated := obj(link["lqm_trackers"])
		associated[id] = v
		link["lqm_trackers"] = associated
	}
}
