// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func versions(ctx context.Context, client *http.Client, c Config) Row {
	v := Row{"AREDN_STABLE_VERSION": c.s("current_stable_fw_version"), "AREDN_NIGHTLY_VERSION": ""}
	if !c.b("aredn_firmware_check") {
		return v
	}
	u := c.s("AREDN_VERSION_CHECK_URL")
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return v
	}
	resp, e := client.Do(req)
	if e != nil {
		return v
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return v
	}
	var buf strings.Builder // downloads are bounded just like node responses
	b := make([]byte, 4096)
	for buf.Len() < 1024*1024 {
		n, e := resp.Body.Read(b)
		buf.Write(b[:n])
		if e != nil {
			break
		}
	}
	re := regexp.MustCompile(`(?m)versions:\s*(\{[^\n]*\})`)
	m := re.FindStringSubmatch(buf.String())
	if m == nil {
		return v
	}
	var parsed map[string]any
	if json.Unmarshal([]byte(strings.ReplaceAll(m[1], "'", "\"")), &parsed) != nil {
		return v
	}
	stable := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
	for k := range parsed {
		field := "AREDN_NIGHTLY_VERSION"
		if stable.MatchString(k) {
			field = "AREDN_STABLE_VERSION"
		}
		if versionGreater(k, str(v[field])) {
			v[field] = k
		}
	}
	return v
}
func versionGreater(a, b string) bool {
	re := regexp.MustCompile(`\d+`)
	aa, bb := re.FindAllString(a, -1), re.FindAllString(b, -1)
	for i := 0; i < len(aa) || i < len(bb); i++ {
		x, y := 0, 0
		if i < len(aa) {
			x, _ = strconv.Atoi(aa[i])
		}
		if i < len(bb) {
			y, _ = strconv.Atoi(bb[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}
func atomicWrite(path string, b []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".meshmap-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0644); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func jsonWrite(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return atomicWrite(path, append(b, '\n'))
}
func numeric(v any) any {
	switch x := v.(type) {
	case Row:
		r := Row{}
		for k, v := range x {
			if k == "lqm" || k == "lqm_trackers" {
				r[k] = v
			} else {
				r[k] = numeric(v)
			}
		}
		return r
	case map[string]any:
		return numeric(Row(x))
	case []any:
		r := make([]any, len(x))
		for i, v := range x {
			r[i] = numeric(v)
		}
		return r
	case string:
		if n, e := strconv.ParseFloat(x, 64); e == nil {
			return n
		}
	}
	return v
}
func generate(c Config, nodes []Row, stats, vers Row, origin string, errs, noloc []Row) error {
	if e := os.MkdirAll(c.Output, 0755); e != nil {
		return e
	}
	groups := Row{}
	for _, k := range []string{"noRF", "supernode", "900", "2ghz", "3ghz", "5ghz"} {
		groups[k] = []any{}
	}
	report := []any{}
	telemetry := []any{}
	old, total := 0, 0
	for _, n := range nodes {
		if nonempty(n["node"]) {
			telemetry = append(telemetry, Row{"node": n["node"], "wlan_ip": n["wlan_ip"], "last_seen": n["last_seen"], "lqm": n["lqm"], "link_info": n["link_info"]})
		}
		if n["lat"] != nil {
			total++
		}
		t, e := time.Parse("2006-01-02 15:04:05", str(n["last_seen"]))
		if e != nil || time.Since(t) > 7*24*time.Hour {
			old++
		}
		if n["node"] != nil {
			r := Row{}
			for k, v := range legacyNode(n) {
				switch k {
				case "loadavg", "services", "link_info":
					switch v.(type) {
					case Row, map[string]any, []any:
						v = php(v)
					}
				}
				r[k] = v
			}
			r["location"] = str(n["lat"]) + ", " + str(n["lon"])
			report = append(report, r)
		}
		if !located(n) {
			continue
		}
		key := band(n)
		if truth(n["mesh_supernode"]) {
			key = "supernode"
		} else if str(n["meshRF"]) == "off" || strings.EqualFold(str(n["channel"]), "none") {
			key = "noRF"
		}
		if key != "" {
			display := Row{}
			for k, v := range legacyNode(n) {
				display[k] = v
			}
			if display["hopsAway"] == nil {
				display["hopsAway"] = "unknown"
			}
			groups[key] = append(groups[key].([]any), numeric(display))
		}
	}
	coords := c.Arrays["map_center_coordinates"]
	mapInfo := Row{"localnode": origin, "mapTileServers": c.Arrays["mapTileServers"], "title": c.s("pageTitle"), "attribution": c.s("attribution"), "mapContact": c.s("mapContact"), "kilometers": c.b("kilometers"), "webpageDataDir": "", "mapCenterCoords": []any{num(coords["lat"]), num(coords["lon"])}, "mapInitialZoom": num(c.s("map_initial_zoom_level")), "totalNodesInDB": total, "weekPlusOld": old}
	var js strings.Builder
	for _, item := range []struct {
		name string
		v    any
	}{{"arednVersions", numeric(vers)}, {"mapInfo", mapInfo}, {"pollingInfo", stats}, {"allDevices", groups}} {
		b, e := json.MarshalIndent(item.v, "", "  ")
		if e != nil {
			return e
		}
		fmt.Fprintf(&js, "%s = %s;\n", item.name, b)
	}
	if e := atomicWrite(filepath.Join(c.Output, "map_data.js"), []byte(js.String())); e != nil {
		return e
	}
	for _, item := range []struct {
		name string
		v    any
	}{{"node_report_data.json", Row{"data": report}}, {"link_telemetry.json", Row{"schema_version": 1, "data": telemetry}}, {"error_report.json", Row{"data": errs}}, {"no_location.json", Row{"data": noloc}}} {
		if e := jsonWrite(filepath.Join(c.Output, item.name), item.v); e != nil {
			return e
		}
	}
	return writePublicSnapshot(c, nodes, stats, vers, origin)
}

/*
	func upload(ctx context.Context, c Config) error {
		if !c.b("uploadToCloud") {
			return nil
		}
		keys := []string{}
		for k := range c.Arrays["cloudServer"] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if strings.Contains(k, "Example") {
				continue
			}
			dest := c.Arrays["cloudServer"][k]
			if strings.HasPrefix(dest, "-") {
				return fmt.Errorf("invalid cloud destination")
			}
			for _, f := range []string{"map_data.js", "node_report_data.json", "error_report.json", "no_location.json", "link_telemetry.json"} {
				cmd := exec.CommandContext(ctx, "scp", "-B", "-i", c.s("cloudSSHKeyFile"), "--", filepath.Join(c.Output, f), strings.TrimRight(dest, "/")+"/"+f)
				if b, e := cmd.CombinedOutput(); e != nil {
					return fmt.Errorf("upload %s: %w: %s", k, e, b)
				}
			}
		}
		return nil
	}
*/
func upload(ctx context.Context, c Config) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if !c.b("uploadToCloud") {
		return nil
	}

	// Quote a path safely for the remote Linux shell.
	quote := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}

	keys := []string{}
	for k := range c.Arrays["cloudServer"] {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if strings.Contains(k, "Example") {
			continue
		}

		dest := c.Arrays["cloudServer"][k]
		host, dir, ok := strings.Cut(dest, ":")
		if !ok || host == "" || strings.HasPrefix(host, "-") ||
			!strings.HasPrefix(dir, "/") {
			return fmt.Errorf(
				"cloud destination %s must be user@hostname:/absolute/directory",
				k,
			)
		}

		dir = strings.TrimRight(dir, "/")
		finalPath := dir + "/public_map.json"
		tempPath := fmt.Sprintf(
			"%s/.public_map.json.%d.%d.tmp",
			dir, os.Getpid(), time.Now().UnixNano(),
		)

		// Upload to a unique temporary filename first.
		started := time.Now()
		diagnosticEvent(ctx, slog.LevelInfo, "cloud upload started", "destination", k)
		cmd := exec.CommandContext(
			ctx, "scp",
			"-B",
			"-i", c.s("cloudSSHKeyFile"),
			"--",
			filepath.Join(c.Output, "public_map.json"),
			host+":"+tempPath,
		)
		if b, err := cmd.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("upload %s: %w: %s", k, ctx.Err(), b)
			}
			return fmt.Errorf("upload %s: %w: %s", k, err, b)
		}
		diagnosticEvent(ctx, slog.LevelDebug, "cloud copy finished", "destination", k, "duration", time.Since(started))

		// Publish only after the upload has completed successfully.
		cmd = exec.CommandContext(
			ctx, "ssh",
			"-o", "BatchMode=yes",
			"-i", c.s("cloudSSHKeyFile"),
			"--", host,
			"mv -f -- "+quote(tempPath)+" "+quote(finalPath),
		)
		if b, err := cmd.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("publish %s: %w: %s", k, ctx.Err(), b)
			}
			return fmt.Errorf("publish %s: %w: %s", k, err, b)
		}
		diagnosticEvent(ctx, slog.LevelInfo, "cloud upload published", "destination", k, "duration", time.Since(started))
	}

	return nil
}
func logs(c Config, results []Result) ([]Row, []Row, error) {
	errs, noloc := []Row{}, []Row{}
	var out, errlog, loclog strings.Builder
	for _, r := range results {
		date := time.Now().Format("Jan 2 15:04:05")
		if r.Err != nil {
			row := Row{"date": date, "error": "sysinfo.json was not returned or parsed", "node": r.Target.IP, "reason": r.Err.Error()}
			errs = append(errs, row)
			fmt.Fprintf(&errlog, "%s : %s : %s : %s\n", date, row["error"], r.Target.IP, r.Err)
			continue
		}
		fmt.Fprintf(&out, "%s %s (%s)\n", date, r.Node["node"], r.Target.IP)
		if !located(r.Node) {
			row := Row{"date": date, "error": "no usable location info", "node": r.Target.IP + " (" + str(r.Node["node"]) + ")", "reason": "Location Info Not Found!"}
			noloc = append(noloc, row)
			fmt.Fprintf(&loclog, "%s : %s : %s : %s\n", date, row["error"], row["node"], row["reason"])
		}
	}
	if e := os.MkdirAll(c.Logs, 0755); e != nil {
		return nil, nil, e
	}
	for name, b := range map[string]string{"polling_output.log": out.String(), "polling_errors.log": errlog.String(), "no_location.log": loclog.String()} {
		if e := atomicWrite(filepath.Join(c.Logs, name), []byte(b)); e != nil {
			return nil, nil, e
		}
	}
	return errs, noloc, nil
}

// Regeneration retains the previous origin and reports without querying nodes.
func previousReports(c Config) (string, []Row, []Row) {
	origin := c.s("localnode")
	if b, e := os.ReadFile(filepath.Join(c.Output, "node_observations.json")); e == nil {
		var r Row
		if json.Unmarshal(b, &r) == nil && str(r["origin"]) != "" {
			origin = str(r["origin"])
		}
	}
	read := func(name string) []Row {
		r := struct {
			Data []Row `json:"data"`
		}{Data: []Row{}}
		if b, e := os.ReadFile(filepath.Join(c.Output, name)); e == nil {
			json.Unmarshal(b, &r)
		}
		return r.Data
	}
	return origin, read("error_report.json"), read("no_location.json")
}

// The existing webpage does not consume telemetry. Keep its payload small;
// full measurements are provided separately in link_telemetry.json.
func legacyNode(node Row) Row {
	out := Row{}
	for k, v := range node {
		if k != "lqm" {
			out[k] = v
		}
	}
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
	out["link_info"] = links
	return out
}
