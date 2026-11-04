// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Target struct {
	IP    string
	Hops  any
	Links Row
}
type Result struct {
	Target Target
	Node   Row
	Err    error
	Raw    Row
}

func newClient(timeout time.Duration, workers int) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: timeout}).DialContext, MaxIdleConns: workers * 2, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, ResponseHeaderTimeout: timeout}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		return nil
	}}
}
func endpoint(host, path string, legacy bool) (string, error) {
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	u, e := url.Parse(host)
	if e != nil || u.Hostname() == "" || u.User != nil || u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid node address")
	}
	if legacy && u.Port() == "" {
		u.Host = net.JoinHostPort(u.Hostname(), "8080")
	}
	p, e := url.Parse(path)
	if e != nil {
		return "", e
	}
	u.Path = p.Path
	u.RawQuery = p.RawQuery
	return u.String(), nil
}
func fetch(ctx context.Context, client *http.Client, host, path string, legacy bool) (body []byte, err error) {
	u, e := endpoint(host, path, legacy)
	if e != nil {
		return nil, e
	}
	started := time.Now()
	status := 0
	diagnosticEvent(ctx, slog.LevelDebug, "HTTP request started", "url", u)
	defer func() {
		diagnosticEvent(ctx, slog.LevelDebug, "HTTP request finished", "url", u, "duration", time.Since(started),
			"status", status, "bytes", len(body), "error", err)
	}()
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return nil, e
	}
	resp, e := client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", e)
	}
	defer resp.Body.Close()
	status = resp.StatusCode
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 8*1024*1024 {
		return nil, fmt.Errorf("response exceeds 8 MiB")
	}
	return b, nil
}
func fetchJSON(ctx context.Context, client *http.Client, host, path string, legacy bool) (Row, error) {
	b, e := fetch(ctx, client, host, path, legacy)
	if e != nil {
		return nil, e
	}
	var r Row
	e = json.Unmarshal(b, &r)
	if e != nil || r == nil {
		return nil, fmt.Errorf("invalid JSON object")
	}
	return r, nil
}
func nodeJSON(ctx context.Context, client *http.Client, host, path string) (Row, error) {
	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	paths := []string{path}
	if strings.HasPrefix(path, "/cgi-bin/sysinfo.json") {
		q := strings.TrimPrefix(path, "/cgi-bin/sysinfo.json")
		paths = []string{"/a/sysinfo" + q, path}
	}
	var last error
	for _, legacy := range []bool{false, true} {
		for _, p := range paths {
			r, e := fetchJSON(ctx, client, host, p, legacy)
			if e == nil {
				return r, nil
			}
			last = e
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Another URL path cannot repair a connection failure on this same port.
			var transportError *url.Error
			if errors.As(e, &transportError) {
				break
			}
		}
	}
	return nil, last
}
func topology(raw Row) ([]Target, error) {
	v, ok := raw["topology"]
	if !ok {
		v, ok = obj(obj(obj(raw["pages"])["mesh"])["topology"])["topology"]
	}
	if !ok {
		return nil, fmt.Errorf("response has no topology")
	}
	var links []any
	switch x := v.(type) {
	case []any:
		links = x
	case map[string]any:
		for _, l := range x {
			links = append(links, l)
		}
	default:
		return nil, fmt.Errorf("invalid topology")
	}
	targets := map[string]*Target{}
	for _, v := range links {
		l := obj(v)
		ip, dest := str(l["lastHopIP"]), str(l["destinationIP"])
		if net.ParseIP(ip) == nil || net.ParseIP(dest) == nil {
			return nil, fmt.Errorf("topology contains invalid IP address")
		}
		t := targets[ip]
		if t == nil {
			t = &Target{IP: ip, Links: Row{}}
			targets[ip] = t
		}
		t.Hops = l["hops"]
		t.Links[dest] = l
	}
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IP < out[j].IP })
	return out, nil
}

// API 2.0 advertises canonical nodes, not the broader list of LAN hosts.
// OLSR topology edges are retained only when the API actually supplies them.
func discoveryTargets(raw Row) ([]Target, error) {
	index := map[string]Target{}
	valid := false
	if nodes, ok := raw["nodes"].([]any); ok {
		valid = true
		for _, v := range nodes {
			n := obj(v)
			ip := str(n["ip"])
			if net.ParseIP(ip) == nil {
				return nil, fmt.Errorf("invalid advertised node IP %q", ip)
			}
			index[ip] = Target{IP: ip, Links: Row{}}
		}
	}
	if ts, e := topology(raw); e == nil {
		valid = true
		for _, t := range ts {
			index[t.IP] = t
			for dest := range t.Links {
				if _, ok := index[dest]; !ok {
					index[dest] = Target{IP: dest, Links: Row{}}
				}
			}
		}
	}
	if !valid {
		return nil, fmt.Errorf("API supplies neither nodes nor OLSR topology")
	}
	if ip := str(raw["ip"]); net.ParseIP(ip) != nil {
		if _, ok := index[ip]; !ok {
			index[ip] = Target{IP: ip, Hops: 0, Links: Row{}}
		}
	}
	out := make([]Target, 0, len(index))
	for _, t := range index {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IP < out[j].IP })
	return out, nil
}
func discover(ctx context.Context, client *http.Client, host string) ([]Target, error) {
	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	var last error
	for _, p := range []struct {
		path   string
		legacy bool
	}{{"/a/sysinfo?nodes=1&topology=1", false}, {"/cgi-bin/sysinfo.json?nodes=1&topology=1", false}, {"/cgi-bin/sysinfo.json?nodes=1&topology=1", true}, {"/cgi-bin/api?mesh=topology", true}} {
		r, e := fetchJSON(ctx, client, host, p.path, p.legacy)
		if e == nil {
			t, e := discoveryTargets(r)
			if e == nil {
				return t, nil
			}
			last = e
		} else {
			last = e
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("cannot discover network: %w", last)
}
func poll(ctx context.Context, client *http.Client, targets []Target, workers int) []Result {
	jobs := make(chan Target)
	results := make(chan Result)
	var wg sync.WaitGroup
	if workers > len(targets) {
		workers = len(targets)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				started := time.Now()
				diagnosticEvent(ctx, slog.LevelDebug, "node poll started", "node", t.IP)
				raw, e := pollSysinfo(ctx, client, t.IP)
				r := Result{Target: t, Err: e, Raw: raw}
				if e == nil {
					links := Row{}
					for k, v := range t.Links {
						links[k] = v
					}
					for k, v := range obj(raw["link_info"]) {
						merged := obj(links[k])
						for a, b := range obj(v) {
							merged[a] = b
						}
						links[k] = merged
					}
					raw["link_info"] = links
					raw["hopsAway"] = t.Hops
					r.Node = parseNode(raw, t.IP)
					if str(r.Node["node"]) == "" {
						r.Err = fmt.Errorf("sysinfo has no node name")
						r.Node = nil
					}
				}
				if trace := traceFrom(ctx); trace != nil {
					trace.completed.Add(1)
				}
				diagnosticEvent(ctx, slog.LevelDebug, "node poll finished", "node", t.IP, "duration", time.Since(started), "error", r.Err)
				results <- r
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, t := range targets {
			jobs <- t
		}
	}()
	go func() { wg.Wait(); close(results) }()
	out := make([]Result, 0, len(targets))
	for r := range results {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Target.IP < out[j].Target.IP })
	return out
}

// Follow direct neighbors and older topology from successful responses. A failed
// node is attempted once, and the same bounded pool handles each expansion round.
func pollNetwork(ctx context.Context, client *http.Client, targets []Target, workers, limit int) ([]Target, []Result, error) {
	return pollNetworkProgress(ctx, client, targets, workers, limit, nil)
}
func pollNetworkProgress(ctx context.Context, client *http.Client, targets []Target, workers, limit int, progress func(int, int)) ([]Target, []Result, error) {
	if len(targets) > limit {
		return nil, nil, fmt.Errorf("discovery exceeds max_nodes (%d)", limit)
	}
	seen := map[string]bool{}
	for _, t := range targets {
		seen[t.IP] = true
	}
	all := append([]Target{}, targets...)
	if trace := traceFrom(ctx); trace != nil {
		trace.targets.Store(int64(len(all)))
	}
	results := []Result{}
	batch := targets
	for len(batch) > 0 {
		res := poll(ctx, client, batch, workers)
		results = append(results, res...)
		next := []Target{}
		add := func(t Target) error {
			if seen[t.IP] {
				return nil
			}
			if len(all) >= limit {
				return fmt.Errorf("discovery exceeds max_nodes (%d)", limit)
			}
			seen[t.IP] = true
			all = append(all, t)
			if trace := traceFrom(ctx); trace != nil {
				trace.targets.Store(int64(len(all)))
			}
			next = append(next, t)
			return nil
		}
		for _, r := range res {
			if r.Err != nil {
				continue
			}
			if ts, e := topology(r.Raw); e == nil {
				for _, t := range ts {
					if e = add(t); e != nil {
						return nil, nil, e
					}
				}
			}
			for ip := range obj(r.Node["link_info"]) {
				if net.ParseIP(ip) != nil {
					if e := add(Target{IP: ip, Links: Row{}}); e != nil {
						return nil, nil, e
					}
				}
			}
		}
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if progress != nil {
			progress(len(results), len(next))
		}
		diagnosticEvent(ctx, slog.LevelDebug, "discovery round completed", "completed", len(results), "discovered", len(all), "new_targets", len(next))
		batch = next
	}
	return all, results, nil
}
