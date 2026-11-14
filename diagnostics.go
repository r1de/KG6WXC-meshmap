// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const version = "3.1.5"

// rotatingLog writes immediately (no userspace buffer) and serializes writers
// from polling workers, HTTP handlers and the heartbeat goroutine.
type rotatingLog struct {
	mu      sync.Mutex
	path    string
	max     int64
	backups int
	file    *os.File
	size    int64
	closed  bool
	crash   bool
	warning sync.Once
}

func openRotatingLog(path string, max int64, backups int) (*rotatingLog, error) {
	w := &rotatingLog{path: path, max: max, backups: backups}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *rotatingLog) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	// The runtime duplicates this descriptor. Reattach after rotation so a
	// subsequent crash lands in the current debug.log, not an old archive.
	if w.crash {
		if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
			f.Close()
			return err
		}
	}
	w.file, w.size = f, info.Size()
	return nil
}

func rotateLogFiles(path string, backups int) error {
	oldest := path
	if backups > 0 {
		oldest = fmt.Sprintf("%s.%d", path, backups)
	}
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := backups - 1; i >= 1; i-- {
		if err := os.Rename(fmt.Sprintf("%s.%d", path, i), fmt.Sprintf("%s.%d", path, i+1)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if backups > 0 {
		if err := os.Rename(path, path+".1"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (w *rotatingLog) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer func() {
		if err != nil {
			w.warning.Do(func() { fmt.Fprintf(os.Stderr, "meshmap: diagnostic log %s: %v\n", w.path, err) })
		}
	}()
	if w.closed {
		return 0, os.ErrClosed
	}
	if w.file == nil {
		if err = w.open(); err != nil {
			return 0, err
		}
	}
	if w.size > 0 && w.size+int64(len(p)) > w.max {
		// Release the runtime's duplicate handle before renaming the file.
		// Windows cannot rename a log while that handle remains open.
		if w.crash && runtime.GOOS == "windows" {
			if err = debug.SetCrashOutput(nil, debug.CrashOptions{}); err != nil {
				return 0, err
			}
		}
		if err = w.file.Close(); err != nil {
			w.file = nil
			return 0, err
		}
		w.file = nil
		if err = rotateLogFiles(w.path, w.backups); err != nil {
			return 0, err
		}
		if err = w.open(); err != nil {
			return 0, err
		}
	}
	n, err = w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingLog) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	if w.crash {
		debug.SetCrashOutput(nil, debug.CrashOptions{})
		w.crash = false
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *rotatingLog) captureCrashes() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.file == nil {
		return os.ErrClosed
	}
	if err := debug.SetCrashOutput(w.file, debug.CrashOptions{}); err != nil {
		return err
	}
	w.crash = true
	return nil
}

type logFanout []slog.Handler

func (h logFanout) Enabled(ctx context.Context, level slog.Level) bool {
	for _, next := range h {
		if next.Enabled(ctx, level) {
			return true
		}
	}
	return false
}
func (h logFanout) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, next := range h {
		if next.Enabled(ctx, r.Level) {
			errs = append(errs, next.Handle(ctx, r.Clone()))
		}
	}
	return errors.Join(errs...)
}
func (h logFanout) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make(logFanout, len(h))
	for i := range h {
		next[i] = h[i].WithAttrs(attrs)
	}
	return next
}
func (h logFanout) WithGroup(name string) slog.Handler {
	next := make(logFanout, len(h))
	for i := range h {
		next[i] = h[i].WithGroup(name)
	}
	return next
}

type diagnosticsKey struct{}
type pollTraceKey struct{}
type pollPhase struct {
	name  string
	since time.Time
}
type pollTrace struct {
	id        uint64
	started   time.Time
	phase     atomic.Pointer[pollPhase]
	completed atomic.Int64
	targets   atomic.Int64
}
type diagnostics struct {
	logger    *slog.Logger
	files     []*rotatingLog
	debug     bool
	started   time.Time
	active    atomic.Pointer[pollTrace]
	nextScheduled atomic.Pointer[time.Time]
	nextManual atomic.Pointer[time.Time]
	cycles    atomic.Uint64
	debugFile *rotatingLog
}

func newDiagnostics(c Config, stderr io.Writer) (*diagnostics, error) {
	if err := os.MkdirAll(c.Logs, 0700); err != nil {
		return nil, err
	}
	d := &diagnostics{debug: c.Debug, started: time.Now()}
	max := c.LogMaxBytes
	if max <= 0 {
		max = 10 * 1024 * 1024
	}
	// Never dump the configuration or DSN into logs. Redact known credentials
	// as a further safeguard when a driver returns them inside an error.
	secrets := []string{}
	if c.Driver == "mysql" {
		secrets = append(secrets, c.DSN)
	}
	secrets = append(secrets, c.s("sql_passwd"))
	replace := func(_ []string, a slog.Attr) slog.Attr {
		if err, ok := a.Value.Any().(error); ok {
			a.Value = slog.StringValue(err.Error())
		}
		if a.Value.Kind() == slog.KindString {
			s := a.Value.String()
			for _, secret := range secrets {
				if secret != "" {
					s = strings.ReplaceAll(s, secret, "[redacted]")
				}
			}
			if len(s) > 16*1024 {
				s = s[:16*1024] + "...[truncated; enable debug mode for native panic output in debug.log]"
			}
			a.Value = slog.StringValue(s)
		}
		return a
	}
	handlers := logFanout{}
	for _, sink := range []struct {
		name  string
		level slog.Level
	}{{"error.log", slog.LevelWarn}, {"debug.log", slog.LevelDebug}} {
		if sink.name == "debug.log" && !c.Debug {
			continue
		}
		file, err := openRotatingLog(filepath.Join(c.Logs, sink.name), max, c.LogBackups)
		if err != nil {
			d.Close()
			return nil, err
		}
		d.files = append(d.files, file)
		if sink.name == "debug.log" {
			d.debugFile = file
		}
		handlers = append(handlers, slog.NewTextHandler(file, &slog.HandlerOptions{Level: sink.level, ReplaceAttr: replace}))
	}
	// Only debug mode mirrors diagnostics to the console. Persist first in case
	// the terminal/pipe has disconnected.
	if c.Debug {
		handlers = append(handlers, slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: replace}))
	}
	d.logger = slog.New(handlers).With("pid", os.Getpid())
	return d, nil
}

func (d *diagnostics) captureCrashes(c Config) error {
	if d.debugFile == nil {
		return nil
	}
	if err := d.debugFile.captureCrashes(); err != nil {
		return err
	}
	debug.SetTraceback("all")
	d.logger.Debug("native crash capture enabled", "version", version, "mode", c.Mode)
	return nil
}

func (d *diagnostics) Close() {
	for _, file := range d.files {
		file.Close()
	}
}

func diagnosticFrom(ctx context.Context) *diagnostics {
	d, _ := ctx.Value(diagnosticsKey{}).(*diagnostics)
	return d
}
func traceFrom(ctx context.Context) *pollTrace {
	t, _ := ctx.Value(pollTraceKey{}).(*pollTrace)
	return t
}
func diagnosticEvent(ctx context.Context, level slog.Level, message string, attrs ...any) {
	if d := diagnosticFrom(ctx); d != nil {
		if t := traceFrom(ctx); t != nil {
			attrs = append(attrs, "poll_id", t.id)
		}
		d.logger.Log(ctx, level, message, attrs...)
	}
}
func pollStage(ctx context.Context, name string) {
	if t := traceFrom(ctx); t != nil {
		t.phase.Store(&pollPhase{name: name, since: time.Now()})
	}
	diagnosticEvent(ctx, slog.LevelInfo, "poll phase", "phase", name)
	if d := diagnosticFrom(ctx); d != nil && d.debug {
		d.heartbeat(ctx)
	}
}

// Keep the public entry point used by serving and one-shot commands. The
// polling implementation and its original discovery queue live in pollCycle.
func pollOnce(ctx context.Context, c Config, noSQL, verbose, filesOnly bool, fixture string) error {
	d := diagnosticFrom(ctx)
	if d == nil {
		return pollCycle(ctx, c, noSQL, verbose, filesOnly, fixture)
	}
	t := &pollTrace{id: d.cycles.Add(1), started: time.Now()}
	t.phase.Store(&pollPhase{name: "starting", since: t.started})
	d.active.Store(t)
	defer d.active.CompareAndSwap(t, nil)
	ctx = context.WithValue(ctx, pollTraceKey{}, t)
	diagnosticEvent(ctx, slog.LevelInfo, "poll started", "files_only", filesOnly, "workers", c.Workers, "timeout", c.Timeout)
	err := pollCycle(ctx, c, noSQL, verbose, filesOnly, fixture)
	level, message := slog.LevelInfo, "poll completed"
	if err != nil {
		level, message = slog.LevelError, "poll failed"
		if ctx.Err() != nil {
			level, message = slog.LevelInfo, "poll canceled"
		}
	}
	diagnosticEvent(ctx, level, message, "duration", time.Since(t.started), "phase", t.phase.Load().name,
		"completed", t.completed.Load(), "targets", t.targets.Load(), "error", err)
	return err
}

func (d *diagnostics) heartbeat(ctx context.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	active := d.active.Load()
	state := "idle"
	if active != nil {
		state = "polling"
	}
	attrs := []any{"uptime", time.Since(d.started), "goroutines", runtime.NumGoroutine(),
		"heap_mb", mem.HeapAlloc / (1024 * 1024), "heap_inuse_mb", mem.HeapInuse / (1024 * 1024),
		"runtime_sys_mb", mem.Sys / (1024 * 1024), "gc_cycles", mem.NumGC, "state", state}
	if t := active; t != nil {
		phase := t.phase.Load()
		attrs = append(attrs, "poll_id", t.id, "phase", phase.name, "phase_elapsed", time.Since(phase.since),
			"poll_elapsed", time.Since(t.started), "completed", t.completed.Load(), "targets", t.targets.Load())
	}
	d.logger.DebugContext(ctx, "heartbeat", attrs...)
}

func (d *diagnostics) startHeartbeat(ctx context.Context, interval time.Duration) func() {
	if !d.debug {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.heartbeat(ctx)
			}
		}
	}()
	return func() { cancel(); wg.Wait() }
}

func httpDiagnosticLogger(ctx context.Context) *log.Logger {
	if d := diagnosticFrom(ctx); d != nil {
		return slog.NewLogLogger(d.logger.Handler(), slog.LevelError)
	}
	return log.Default()
}

func recordStartupError(configPath string, err error, debugMode bool) bool {
	c := Config{Logs: filepath.Join(filepath.Dir(filepath.Dir(configPath)), "logs"), Debug: debugMode, LogMaxBytes: 10 * 1024 * 1024, LogBackups: 3}
	d, openErr := newDiagnostics(c, os.Stderr)
	if openErr != nil {
		fmt.Fprintf(os.Stderr, "meshmap: cannot record startup error: %v\n", openErr)
		return false
	}
	defer d.Close()
	d.logger.Error("startup failed", "config", configPath, "error", err)
	return true
}
