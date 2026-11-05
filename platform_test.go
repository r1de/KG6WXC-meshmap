// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Run in a separate process so the checks exercise real OS lock contention.
func TestFileLockHelperProcess(t *testing.T) {
	path := os.Getenv("MESHMAP_TEST_LOCK_PATH")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Println("ready")
	if err = lockFile(f, os.Getenv("MESHMAP_TEST_LOCK_WAIT") == "true"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	defer unlockFile(f)
	fmt.Println("locked")
}

func TestFileLockContentionAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meshmap.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = lockFile(f, false); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := func(wait bool) *exec.Cmd {
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestFileLockHelperProcess$")
		cmd.Env = append(os.Environ(), "MESHMAP_TEST_LOCK_PATH="+path,
			fmt.Sprintf("MESHMAP_TEST_LOCK_WAIT=%t", wait))
		return cmd
	}
	output, err := child(false).CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 3 {
		t.Fatalf("another process acquired the polling lock: %v\n%s", err, output)
	}

	cmd := child(true)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || scanner.Text() != "ready" {
		cancel()
		cmd.Wait()
		t.Fatalf("waiting process did not start: %s", stderr.String())
	}
	if err = unlockFile(f); err != nil {
		cancel()
		cmd.Wait()
		t.Fatal(err)
	}
	if !scanner.Scan() || scanner.Text() != "locked" {
		cancel()
		cmd.Wait()
		t.Fatalf("waiting process did not acquire released lock: %s", stderr.String())
	}
	for scanner.Scan() {
	}
	if err = cmd.Wait(); err != nil {
		t.Fatalf("waiting process failed: %v\n%s", err, stderr.String())
	}
	if err = lockFile(f, false); err != nil {
		t.Fatalf("lock not released by exiting process: %v", err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if output, err = child(false).CombinedOutput(); err != nil {
		t.Fatalf("closing the handle did not release the lock: %v\n%s", err, output)
	}
}

func TestPlatformSettingsUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user settings.ini")
	original := []byte("; Keep my settings\r\n[user-settings]\r\npoll_interval=7h\r\n")
	if err := os.WriteFile(path, original, 0640); err != nil {
		t.Fatal(err)
	}
	defaults := []settingDefault{
		{name: "poll_interval", value: "4h"},
		{name: "new_setting", value: "enabled"},
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := upgradeConfig(path, defaults); err != nil {
				t.Errorf("concurrent settings upgrade: %v", err)
			}
		}()
	}
	wg.Wait()
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(updated, original) || bytes.Count(updated, []byte("new_setting")) != 1 {
		t.Fatalf("settings lost or added more than once: %s", updated)
	}
	again, err := upgradeConfig(path, defaults)
	if err != nil || !bytes.Equal(updated, again) {
		t.Fatalf("repeated upgrade changed settings: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0640 {
			t.Fatalf("settings permissions changed: %v", info.Mode())
		}
	}
}

func TestPlatformOutputReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "public_map.json")
	for _, body := range []string{`{"nodes":[]}`, `{"nodes":[{"node":"test"}]}`} {
		if err := atomicWrite(path, []byte(body)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != body {
			t.Fatalf("map replacement failed: %s, %v", got, err)
		}
	}
}

func TestPlatformSQLitePersistence(t *testing.T) {
	ctx := context.Background()
	c := Config{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "data with spaces", "map.sqlite"), Table: "node_info"}
	s, err := openStore(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	if _, err = s.DB.ExecContext(ctx, "INSERT INTO node_info (wlan_ip, node) VALUES (?, ?)", "10.1.1.1", "test-node"); err != nil {
		t.Fatal(err)
	}
	if err = s.DB.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = openStore(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	nodes, err := s.load(ctx)
	if err != nil || len(nodes) != 1 || str(nodes[0]["node"]) != "test-node" {
		t.Fatalf("SQLite did not preserve the node after reopening: %v, %v", nodes, err)
	}
}

func TestPlatformCrashLogRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	w, err := openRotatingLog(path, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = w.captureCrashes(); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"old\n", "new\n", "end\n"} {
		if _, err = w.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	for name, want := range map[string]string{path: "end\n", path + ".1": "new\n"} {
		got, err := os.ReadFile(name)
		if err != nil || string(got) != want {
			t.Fatalf("incorrect rotated log %s: %q, %v", name, got, err)
		}
	}
}

func TestCrashLogHelperProcess(t *testing.T) {
	dir := os.Getenv("MESHMAP_TEST_CRASH_DIR")
	if dir == "" {
		return
	}
	w, err := openRotatingLog(filepath.Join(dir, "debug.log"), 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.captureCrashes(); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"old\n", "new\n"} {
		if _, err = w.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	panic("meshmap crash capture after rotation")
}

func TestPlatformCrashCapturedAfterRotation(t *testing.T) {
	dir := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestCrashLogHelperProcess$")
	cmd.Env = append(os.Environ(), "MESHMAP_TEST_CRASH_DIR="+dir)
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("expected subprocess panic: %v\n%s", err, output)
	}
	got, err := os.ReadFile(filepath.Join(dir, "debug.log"))
	if err != nil || !bytes.Contains(got, []byte("meshmap crash capture after rotation")) {
		t.Fatalf("panic missing from current debug.log: %v\n%s\n%s", err, got, output)
	}
	archive, err := os.ReadFile(filepath.Join(dir, "debug.log.1"))
	if err != nil || string(archive) != "old\n" {
		t.Fatalf("panic went into old archive: %v\n%s", err, archive)
	}
}
