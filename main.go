// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		var reported *loggedError
		if !errors.As(e, &reported) {
			fmt.Fprintln(os.Stderr, "meshmap:", e)
		}
		os.Exit(1)
	}
}

// Avoid echoing an error to the console after diagnostics has already routed
// it to error.log (and to the console when debug mode is enabled).
type loggedError struct{ error }

func (e *loggedError) Unwrap() error { return e.error }

func run() (runErr error) {
	config := flag.String("config", "settings/user-settings.ini", "settings file (missing defaults are added automatically)")
	timeout := flag.Duration("timeout", 30*time.Second, "per-node HTTP budget and database I/O timeout")
	noSQL := flag.Bool("test-mode-no-sql", false, "poll and generate files without accessing SQL")
	verbose := flag.Bool("test-mode-with-sql", false, "poll once (summary is printed when debug mode is enabled)")
	debugMode := flag.Bool("debug", false, "write detailed polling diagnostics and memory heartbeats to debug.log")
	filesOnly := flag.Bool("data-files-only", false, "regenerate files without polling or changing stored node records")
	fixture := flag.String("fixture", "", "offline fixture JSON (requires --test-mode-no-sql)")
	initConfig := flag.Bool("init", false, "create initial settings or add missing defaults without changing existing values")
	serve := flag.Bool("serve", false, "serve the embedded map and poll on a schedule")
	serveOnly := flag.Bool("serve-only", false, "serve the latest completed map without polling")
	listen := flag.String("listen", "127.0.0.1:8088", "HTTP(S) listening address")
//	interval := flag.Duration("poll-interval", 10*time.Minute, "delay between completed polling runs")
	interval := flag.Duration("poll-interval", 4*time.Hour, "delay between completed polling runs")
	cert := flag.String("tls-cert", "", "HTTPS certificate PEM file")
	key := flag.String("tls-key", "", "HTTPS private key PEM file")
	showVersion := flag.Bool("version", false, "print executable version")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return nil
	}
	if *initConfig {
		if e := os.MkdirAll(filepath.Dir(*config), 0755); e != nil {
			return e
		}
		
		defaults, e := configDefaults(*config)
			if e != nil {
				return e
			}
			
			// Create the file only if it does not already exist.
			f, e := os.OpenFile(
				*config,
		       os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		       0600,
			)
			
			if e == nil {
				_, writeErr := f.Write(defaultSettings)
				closeErr := f.Close()
				
				if writeErr != nil {
					return writeErr
				}
				if closeErr != nil {
					return closeErr
				}
			} else if !errors.Is(e, os.ErrExist) {
				return e
			}
			
			// Add missing settings using the normal startup upgrade helper.
			_, e = upgradeConfig(*config, defaults)
			return e
	}
	firstRun := false
	var diagnostics *diagnostics
	defer func() {
		if p := recover(); p != nil {
			if diagnostics != nil {
				diagnostics.logger.Error("unhandled panic; process will exit", "panic", fmt.Sprint(p))
				diagnostics.logger.Debug("panic stack", "stack", string(debug.Stack()))
			}
			// Preserve normal Go crash behavior and optional debug.log capture.
			panic(p)
		}
		if diagnostics == nil {
			if runErr != nil {
				if firstRun {
					// Help was already printed. Skip logging and
					// prevent main() from printing the error again.
					runErr = &loggedError{runErr}
					return
				}
				
				if recordStartupError(*config, runErr, *debugMode) {
					runErr = &loggedError{runErr}
				}
			}
			return
		}
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			diagnostics.logger.Error("process stopped with error", "error", runErr)
		} else {
			diagnostics.logger.Info("process stopped", "error", runErr)
		}
		if runErr != nil {
			runErr = &loggedError{runErr}
		}
		diagnostics.Close()
	}()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if *noSQL && *filesOnly {
		return fmt.Errorf("test-mode-no-sql and data-files-only are mutually exclusive")
	}
	if *fixture != "" && !*noSQL {
		return fmt.Errorf("fixture requires test-mode-no-sql")
	}
	// One-shot maintenance/test commands stay one-shot unless serving is
	// explicitly requested, even when run_mode=serve is saved in the INI.
	explicitServerMode := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "serve" || f.Name == "serve-only" {
			explicitServerMode = true
		}
	})
	overrides := map[string]string{}
	if !explicitServerMode && (*filesOnly || *noSQL || *verbose) {
		overrides["run_mode"] = "poll"
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "debug":
			overrides["debug_mode"] = strconv.FormatBool(*debugMode)
		case "timeout":
			overrides["http_timeout"] = timeout.String()
		case "listen":
			overrides["listen_address"] = *listen
		case "poll-interval":
			overrides["poll_interval"] = interval.String()
		case "tls-cert":
			overrides["tls_cert"] = *cert
		case "tls-key":
			overrides["tls_key"] = *key
		case "serve", "serve-only":
			overrides["run_mode"] = "poll"
		}
	})
	if *serve {
		overrides["run_mode"] = "serve"
	}
	if *serveOnly {
		overrides["run_mode"] = "serve-only"
	}
	if *serve && *serveOnly {
		return fmt.Errorf("choose either serve or serve-only")
	}
	c, e := loadConfigOverrides(*config, 0, overrides)
	if e != nil {
		fmt.Fprintf(os.Stderr, "MeshMap cannot load the settings file!\n   %v\n", e)
		
		if errors.Is(e, os.ErrNotExist) {
			firstRun = true
			fmt.Fprintf(os.Stderr,
				"\nIf this is a first run, create a settings file with:\n"+
				"   %[1]s --init\n\n"+
				"That will create or update a settings file in: %[2]q\n"+
				"You can then customize that settings file to your site.\n\n"+
				"You can also change the location of the config file with:\n"+
				"   %[1]s --config %[2]q\n\n"+
				"Or create or update a new file in a different location with:\n"+
				"   %[1]s --config %[2]q --init\n\n",
			os.Args[0], *config)
		}
		
		fmt.Fprint(os.Stderr, "\nRun with --help for available options.\n\n")
		return e
	}
	*serve = c.Mode == "serve"
	*serveOnly = c.Mode == "serve-only"
	*interval = c.PollInterval
	*listen = c.ListenAddress
	*cert = c.TLSCert
	*key = c.TLSKey
	diagnostics, e = newDiagnostics(c, os.Stderr)
	if e != nil {
		return fmt.Errorf("initialize diagnostics: %w", e)
	}
	if e = diagnostics.captureCrashes(c); e != nil {
		return fmt.Errorf("capture runtime crashes: %w", e)
	}
	base := context.WithValue(context.Background(), diagnosticsKey{}, diagnostics)
	ctx, cancel := context.WithCancel(base)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	signalDone := make(chan struct{})
	go func() {
		defer close(signalDone)
		select {
		case sig := <-signals:
			diagnosticEvent(ctx, slog.LevelInfo, "shutdown requested", "signal", sig.String())
			cancel()
		case <-ctx.Done():
		}
	}()
	stop := func() { signal.Stop(signals); cancel(); <-signalDone }
	defer stop()
	stopHeartbeat := diagnostics.startHeartbeat(ctx, c.DebugHeartbeat)
	defer stopHeartbeat()
	executable, _ := os.Executable()
	workingDir, _ := os.Getwd()
	diagnosticEvent(ctx, slog.LevelInfo, "process started", "version", version, "mode", c.Mode,
		"executable", executable, "working_directory", workingDir, "go", runtime.Version(),
		"workers", c.Workers, "timeout", c.Timeout, "poll_interval", c.PollInterval, "debug", c.Debug, "logs", c.Logs)
	if c.Debug {
		diagnostics.heartbeat(ctx)
	}
	if *serve && *serveOnly {
		return fmt.Errorf("choose either serve or serve-only")
	}
	if *serve || *serveOnly {
		if *interval <= 0 {
			return fmt.Errorf("poll-interval must be positive")
		}
		if *serveOnly && (*noSQL || *fixture != "" || *filesOnly) {
			return fmt.Errorf("serve-only cannot be combined with polling test modes")
		}
		if *serve && *filesOnly {
			return fmt.Errorf("serve cannot be combined with data-files-only")
		}
		var poller func(context.Context) error
		if !*serveOnly {
			first := true
			poller = func(ctx context.Context) error {
				if first {
					first = false
					if !*noSQL {
						if err := pollOnce(ctx, c, false, false, true, ""); err != nil {
							diagnosticEvent(ctx, slog.LevelError, "stored map preload failed", "error", err)
						}
					}
				}
				return pollOnce(ctx, c, *noSQL, *verbose, false, *fixture)
			}
		}
		return runWeb(ctx, c, webOptions{Address: *listen, Cert: *cert, Key: *key, Interval: *interval}, poller)
	}
	return pollOnce(ctx, c, *noSQL, *verbose, *filesOnly, *fixture)
}

func pollCycle(ctx context.Context, c Config, noSQL, verbose, filesOnly bool, fixture string) error {
	pollStage(ctx, "poll lock")
	var e error
	if e = os.MkdirAll(c.Logs, 0755); e != nil {
		return e
	}
	lock, e := os.OpenFile(filepath.Join(c.Logs, "meshmap.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		return fmt.Errorf("another polling run holds the lock")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	client := newClient(c.Timeout, c.Workers)
	defer client.CloseIdleConnections()
	var store *Store
	if !noSQL {
		pollStage(ctx, "database open")
		store, e = openStore(ctx, c)
		if e != nil {
			return e
		}
		defer store.DB.Close()
	}
	if filesOnly {
		pollStage(ctx, "stored map load")
		nodes, e := store.load(ctx)
		if e != nil {
			return e
		}
		stats, vers, e := store.metadata(ctx)
		if e != nil {
			return e
		}
		enrich(nodes)
		origin, errs, noloc := previousReports(c)
		pollStage(ctx, "stored map export")
		return generate(c, nodes, stats, vers, origin, errs, noloc)
	}
	var targets []Target
	var results []Result
	origin := ""
	vers := Row{"AREDN_STABLE_VERSION": "", "AREDN_NIGHTLY_VERSION": ""}
	start := time.Now()
	if fixture != "" {
		pollStage(ctx, "fixture load")
		b, e := os.ReadFile(fixture)
		if e != nil {
			return e
		}
		var f struct {
			Origin   string         `json:"origin"`
			Topology Row            `json:"topology"`
			Nodes    map[string]Row `json:"nodes"`
		}
		if e = json.Unmarshal(b, &f); e != nil {
			return e
		}
		origin = f.Origin
		targets, e = discoveryTargets(f.Topology)
		if e != nil {
			return e
		}
		for _, t := range targets {
			raw, ok := f.Nodes[t.IP]
			if !ok {
				results = append(results, Result{Target: t, Err: fmt.Errorf("node missing from fixture")})
				continue
			}
			merged := Row{}
			for k, v := range t.Links {
				merged[k] = v
			}
			for k, v := range obj(raw["link_info"]) {
				m := obj(merged[k])
				for a, b := range obj(v) {
					m[a] = b
				}
				merged[k] = m
			}
			raw["link_info"] = merged
			raw["hopsAway"] = t.Hops
			n := parseNode(raw, t.IP)
			r := Result{Target: t, Node: n, Raw: raw}
			if str(n["node"]) == "" {
				r.Err = fmt.Errorf("sysinfo has no node name")
			}
			results = append(results, r)
		}
	} else {
		pollStage(ctx, "origin lookup")
		local, e := nodeJSON(ctx, client, c.s("localnode"), "/cgi-bin/sysinfo.json")
		if e != nil {
			return fmt.Errorf("map origin: %w", e)
		}
		origin = str(local["node"])
		if origin == "" {
			return fmt.Errorf("origin node has no name")
		}
		pollStage(ctx, "network discovery")
		targets, e = discover(ctx, client, c.s("localnode"))
		if e != nil {
			return e
		}
		pollStage(ctx, "firmware version check")
		vers = versions(ctx, client, c)
		start = time.Now()
		// Optional additional seeds allow discovery on both sides of a mixed network.
		for _, seed := range strings.Split(c.s("discovery_seeds"), ",") {
			seed = strings.TrimSpace(seed)
			if seed == "" {
				continue
			}
			pollStage(ctx, "additional seed discovery")
			diagnosticEvent(ctx, slog.LevelDebug, "discovery seed", "node", seed)
			ts, err := discover(ctx, client, seed)
			if err != nil {
				return fmt.Errorf("seed %s: %w", seed, err)
			}
			index := map[string]bool{}
			for _, t := range targets {
				index[t.IP] = true
			}
			for _, t := range ts {
				if !index[t.IP] {
					targets = append(targets, t)
					index[t.IP] = true
				}
			}
		}
		limit := 10000
		if v := c.s("max_nodes"); v != "" {
			limit, e = strconv.Atoi(v)
			if e != nil || limit < 1 {
				return fmt.Errorf("max_nodes must be positive")
			}
		}
		var progress func(int, int)
		if c.Debug && (verbose || noSQL) {
			fmt.Fprintf(os.Stderr, "Discovered %d addresses; polling with %d workers\n", len(targets), c.Workers)
			progress = func(done, queued int) {
				fmt.Fprintf(os.Stderr, "Polled %d addresses; %d newly discovered\n", done, queued)
			}
		}
		pollStage(ctx, "node polling")
		targets, results, e = pollNetworkProgress(ctx, client, targets, c.Workers, limit, progress)
		if e != nil {
			return e
		}
	}
	elapsed := time.Since(start).Seconds()
	if trace := traceFrom(ctx); trace != nil {
		trace.targets.Store(int64(len(targets)))
		trace.completed.Store(int64(len(results)))
	}
	diagnosticEvent(ctx, slog.LevelInfo, "polling responses collected", "attempted", len(results), "targets", len(targets), "polling_seconds", elapsed)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	pollStage(ctx, "poll report logs")
	errs, noloc, e := logs(c, results)
	if e != nil {
		return e
	}
	nodes := []Row{}
	if store != nil {
		pollStage(ctx, "database save")
		if e = store.savePoll(ctx, c, targets, results); e != nil {
			return fmt.Errorf("save polling results: %w", e)
		}
		pollStage(ctx, "database reload")
		nodes, e = store.load(ctx)
		if e != nil {
			return e
		}
	} else {
		for _, r := range results {
			if r.Err == nil {
				r.Node["last_seen"] = time.Now().UTC().Format("2006-01-02 15:04:05")
				nodes = append(nodes, r.Node)
			}
		}
	}
	pollStage(ctx, "link enrichment")
	linkCount := enrich(nodes)
	highest := 0.0
	for _, t := range targets {
		if num(t.Hops) > highest {
			highest = num(t.Hops)
		}
	}
	stats := Row{"numParallelThreads": c.Workers, "nodeTotal": len(targets), "garbageReturned": len(errs), "highestHops": highest, "totalPolled": len(results) - len(errs), "noLocation": len(noloc), "mappableNodes": len(results) - len(errs) - len(noloc), "mappableLinks": linkCount, "pollingTimeSec": elapsed, "lastPollingRun": time.Now().UTC().Format("2006-01-02 15:04:05")}
	if store != nil {
		pollStage(ctx, "database metadata")
		_, previous, e := store.metadata(ctx)
		if e != nil {
			return e
		}
		for k, v := range vers {
			if str(v) == "" {
				vers[k] = previous[k]
			}
		}
		if e = store.saveMeta(ctx, nodes, stats, vers); e != nil {
			return fmt.Errorf("save metadata: %w", e)
		}
	}
	pollStage(ctx, "map export")
	if e = generate(c, nodes, stats, vers, origin, errs, noloc); e != nil {
		return e
	}
	observations := []any{}
	for _, r := range results {
		if r.Err == nil {
			observations = append(observations, Row{"polled_address": r.Target.IP, "sysinfo": r.Raw})
		}
	}
	pollStage(ctx, "raw observations export")
	if e = jsonWrite(filepath.Join(c.Output, "node_observations.json"), Row{"origin": origin, "captured_at": stats["lastPollingRun"], "nodes": observations}); e != nil {
		return e
	}
	if !noSQL {
		pollStage(ctx, "cloud upload")
		if e = upload(ctx, c); e != nil {
			return e
		}
	}
	if c.Debug && (verbose || noSQL) {
		b, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(b))
	}
	return nil
}
