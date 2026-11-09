// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

//go:embed web/index.html web/app.js web/app.css web/metrics.js web/logo.png web/NOTICE.txt web/vendor
var embeddedWeb embed.FS

type webOptions struct {
	Address, Cert, Key string
	Interval           time.Duration
}
type webSnapshot struct {
	Body []byte
	ETag string
}
type snapshotCache struct{ Current atomic.Pointer[webSnapshot] }

func (s *snapshotCache) reload(path string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 64*1024*1024+1))
	if e != nil {
		return e
	}
	if len(b) > 64*1024*1024 {
		return fmt.Errorf("public snapshot exceeds 64 MiB")
	}
	var shape struct {
		Schema int   `json:"schema_version"`
		Nodes  []Row `json:"nodes"`
	}
	if e = json.Unmarshal(b, &shape); e != nil {
		return e
	}
	if shape.Schema != 1 || shape.Nodes == nil {
		return fmt.Errorf("unsupported public snapshot")
	}
	sum := sha256.Sum256(b)
	s.Current.Store(&webSnapshot{b, fmt.Sprintf("\"%x\"", sum)})
	return nil
}
// Polling status indicator
//func webHandler(cache *snapshotCache) http.Handler {
func webHandler(cache *snapshotCache, d *diagnostics, pollingEnabled bool) http.Handler {
	assets, _ := fs.Sub(embeddedWeb, "web")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https: http:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		//Polling Status Indicator
		if r.URL.Path == "/api/status" {
			status := Row{"state": "serve-only"}
			
			if pollingEnabled {
				status["state"] = "waiting"
				
				if d == nil {
					status["state"] = "unknown"
				} else if active := d.active.Load(); active != nil {
					status["state"] = "polling"
					status["completed"] = active.completed.Load()
					status["targets"] = active.targets.Load()
					status["elapsed_seconds"] = int(time.Since(active.started).Seconds())
					
					if phase := active.phase.Load(); phase != nil {
						status["phase"] = phase.name
					}
				} else {
					next := d.nextScheduled.Load()
					if manual := d.nextManual.Load(); manual != nil &&
						(next == nil || manual.Before(*next)) {
							next = manual
						}
						if next != nil {
							status["next_poll_in_seconds"] = max(0, time.Until(*next).Seconds())
						}
				}
			}
			
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			
			if r.Method == http.MethodGet {
				json.NewEncoder(w).Encode(status)
			}
			return
		}
		
		if r.URL.Path == "/api/map" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			snapshot := cache.Current.Load()
			if snapshot == nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				if r.Method == "GET" {
					io.WriteString(w, "{\"error\":\"No completed polling snapshot yet. The map will retry automatically.\"}")
				}
				return
			}
			w.Header().Set("ETag", snapshot.ETag)
			if r.Header.Get("If-None-Match") == snapshot.ETag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			if r.Method == "GET" {
				w.Write(snapshot.Body)
			}
			return
		}
		// Embedded assets only: never a directory rooted at the install/output path.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if !fs.ValidPath(path) || strings.HasPrefix(path, ".") || strings.Contains(path, "/.") {
			http.NotFound(w, r)
			return
		}
		data, e := fs.ReadFile(assets, path)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		switch filepath.Ext(path) {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".txt":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method == "GET" {
			w.Write(data)
		}
	})
}
func runWeb(parent context.Context, c Config, options webOptions, poller func(context.Context) error) error {
	if (options.Cert == "") != (options.Key == "") {
		return fmt.Errorf("tls-cert and tls-key must be supplied together")
	}
	var tlsConfig *tls.Config
	if options.Cert != "" {
		pair, e := tls.LoadX509KeyPair(options.Cert, options.Key)
		if e != nil {
			return e
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}}
	}
	listener, e := net.Listen("tcp", options.Address)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	cache := &snapshotCache{}
	snapshotPath := filepath.Join(c.Output, "public_map.json")
	if e = cache.reload(snapshotPath); e != nil && !os.IsNotExist(e) {
		diagnosticEvent(ctx, slog.LevelWarn, "no readable public snapshot", "error", e)
	}
	srv := &http.Server{Handler: webHandler(cache, diagnosticFrom(ctx), poller != nil), ErrorLog: httpDiagnosticLogger(ctx), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024, TLSConfig: tlsConfig}
	done := make(chan error, 1)
	go func() {
		if tlsConfig != nil {
			done <- srv.ServeTLS(listener, "", "")
		} else {
			done <- srv.Serve(listener)
		}
	}()
	scheme := "http"
	if tlsConfig != nil {
		scheme = "https"
	}
	diagnosticEvent(ctx, slog.LevelInfo, "HTTP server listening", "address", scheme+"://"+listener.Addr().String())
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		if poller == nil {
			return
		}
/*
		for {
			if e := poller(ctx); e != nil {
				if ctx.Err() != nil {
					return
				}
				diagnosticEvent(ctx, slog.LevelError, "polling failed; scheduler will retry", "error", e, "retry_in", options.Interval)
			} else if err := cache.reload(snapshotPath); err != nil {
				diagnosticEvent(ctx, slog.LevelWarn, "snapshot reload failed", "error", err)
			}
			diagnosticEvent(ctx, slog.LevelInfo, "next poll scheduled", "delay", options.Interval, "next_poll", time.Now().Add(options.Interval))
			timer := time.NewTimer(options.Interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
*/
/*
		schedulePath := filepath.Join(c.Logs, "poll-schedule.json")
		var lastFinished time.Time

		if b, err := os.ReadFile(schedulePath); err == nil {
			if err := json.Unmarshal(b, &lastFinished); err != nil {
				lastFinished = time.Time{}
				diagnosticEvent(ctx, slog.LevelWarn, "invalid polling schedule", "error", err)
			}
		} else if !os.IsNotExist(err) {
			diagnosticEvent(ctx, slog.LevelWarn, "cannot read polling schedule", "error", err)
		}

		for ctx.Err() == nil {
			if !lastFinished.IsZero() {
				nextPoll := lastFinished.Add(options.Interval)
				if delay := time.Until(nextPoll); delay > 0 {
					diagnosticEvent(ctx, slog.LevelInfo, "next poll scheduled",
						"delay", delay, "next_poll", nextPoll)
			
					timer := time.NewTimer(delay)
					select {
					case <-ctx.Done():
						timer.Stop()
						return
					case <-timer.C:
					}
				}
			}
			if ctx.Err() != nil {
				return
			}
	
			err := poller(ctx)
			if ctx.Err() != nil {
				return
			}
	
			if err != nil {
				diagnosticEvent(ctx, slog.LevelError, "polling failed; scheduler will retry",
					"error", err, "retry_in", options.Interval)
			} else if err := cache.reload(snapshotPath); err != nil {
				diagnosticEvent(ctx, slog.LevelWarn, "snapshot reload failed", "error", err)
			}
	
			lastFinished = time.Now()
			if err := jsonWrite(schedulePath, lastFinished.UTC()); err != nil {
				diagnosticEvent(ctx, slog.LevelWarn, "cannot save polling schedule", "error", err)
			}
		}
	}()
*/
		runPollScheduler(ctx, c.Logs, options.Interval, poller, func() error {
			return cache.reload(snapshotPath)
		})
	}()
	// Serve-only also follows snapshots published by a separate scheduled poller.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		var previous time.Time
		lastFailure := ""
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if info, e := os.Stat(snapshotPath); e == nil && info.ModTime() != previous {
					if err := cache.reload(snapshotPath); err == nil {
						previous = info.ModTime()
						lastFailure = ""
					} else if err.Error() != lastFailure {
						diagnosticEvent(ctx, slog.LevelWarn, "snapshot watcher retained last good map", "error", err)
						lastFailure = err.Error()
					}
				} else if e != nil && !os.IsNotExist(e) && e.Error() != lastFailure {
					diagnosticEvent(ctx, slog.LevelWarn, "snapshot watcher cannot stat map", "error", e)
					lastFailure = e.Error()
				}
			}
		}
	}()
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-done:
	}
	cancel()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	if e = srv.Shutdown(shutdownCtx); e != nil {
		diagnosticEvent(ctx, slog.LevelWarn, "HTTP shutdown deadline exceeded", "error", e)
		srv.Close()
	}
	select {
	case <-workerDone:
	case <-shutdownCtx.Done():
		diagnosticEvent(ctx, slog.LevelWarn, "polling worker still active at shutdown deadline")
	}
	diagnosticEvent(ctx, slog.LevelInfo, "HTTP server stopped", "error", serveErr)
	if serveErr != nil && serveErr != http.ErrServerClosed {
		return serveErr
	}
	return nil
}
