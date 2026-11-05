// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// Sidecar lock files have no data. Lock byte zero, even in an empty file.
// os.OpenFile opens a synchronous handle, so LockFileEx waits unless asked
// to fail immediately. Windows also releases the lock when the handle closes.
func lockFile(f *os.File, wait bool) error {
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK)
	if !wait {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &windows.Overlapped{})
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{})
}

func preserveFileOwner(_ *os.File, _ os.FileInfo) error {
	// Windows has no Unix UID/GID. The temporary file inherits directory ACLs.
	return nil
}

func shutdownSignals() []os.Signal {
	// Go reports Ctrl+C as Interrupt and console close/shutdown as SIGTERM.
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
