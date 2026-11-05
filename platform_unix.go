// SPDX-License-Identifier: GPL-3.0-or-later
//go:build unix

package main

import (
	"os"
	"syscall"
)

// Polling uses a nonblocking lock; settings upgrades wait for other writers.
func lockFile(f *os.File, wait bool) error {
	flags := syscall.LOCK_EX
	if !wait {
		flags |= syscall.LOCK_NB
	}
	return syscall.Flock(int(f.Fd()), flags)
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

func preserveFileOwner(f *os.File, info os.FileInfo) error {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return f.Chown(int(stat.Uid), int(stat.Gid))
	}
	return nil
}

func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
