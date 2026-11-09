// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const pollRequestsDir = "poll-requests"

func queuePollRequest(logDir string, due time.Time) error {
	dir := filepath.Join(logDir, pollRequestsDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}

	// The filename contains the entire request, so creating it publishes it.
	name := fmt.Sprintf("%d-%x.request", due.UnixNano(), id)
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	return file.Close()
}

func takeDuePollRequests(dir string, now time.Time) (bool, *time.Time, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	
	var taken bool
	var nextManual *time.Time
	var removeErr error
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".request") {
			continue
		}
		
		stamp, _, _ := strings.Cut(entry.Name(), "-")
		due, err := strconv.ParseInt(stamp, 10, 64)
		if err != nil {
			continue
		}
		
		if due > now.UnixNano() {
			when := time.Unix(0, due)
			if nextManual == nil || when.Before(*nextManual) {
				nextManual = &when
			}
			continue
		}
		
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			if !os.IsNotExist(err) {
				removeErr = errors.Join(removeErr, err)
			}
			continue
		}
		taken = true
	}
	return taken, nextManual, removeErr
}

func runPollScheduler(ctx context.Context, logDir string, interval time.Duration, poller func(context.Context) error, reload func() error) {
	schedulePath := filepath.Join(logDir, "poll-schedule.json")
	requestDir := filepath.Join(logDir, pollRequestsDir)
	var nextScheduled time.Time

	d := diagnosticFrom(ctx)
	setNextScheduled := func(when time.Time) {
		nextScheduled = when
		if d != nil {
			d.nextScheduled.Store(&when)
		}
	}

	if b, err := os.ReadFile(schedulePath); err == nil {
		var lastFinished time.Time
		if err := json.Unmarshal(b, &lastFinished); err != nil {
			diagnosticEvent(ctx, slog.LevelWarn, "invalid polling schedule", "error", err)
		} else if !lastFinished.IsZero() {
/*			nextScheduled = lastFinished.Add(interval) */
			setNextScheduled(lastFinished.Add(interval))
			diagnosticEvent(ctx, slog.LevelInfo, "poll schedule restored", "next_poll", nextScheduled)
		}
	} else if !os.IsNotExist(err) {
		diagnosticEvent(ctx, slog.LevelWarn, "cannot read polling schedule", "error", err)
	}

	var lastQueueError string
	for ctx.Err() == nil {
		manual, nextManual, queueErr := takeDuePollRequests(requestDir, time.Now())
		if d != nil {
			d.nextManual.Store(nextManual)
		}
		if queueErr == nil {
			lastQueueError = ""
		} else if queueErr.Error() != lastQueueError {
			lastQueueError = queueErr.Error()
			diagnosticEvent(ctx, slog.LevelWarn, "cannot process poll requests", "error", queueErr)
		}

		scheduled := !time.Now().Before(nextScheduled)
		if !scheduled && !manual {
			timer := time.NewTimer(min(time.Second, time.Until(nextScheduled)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				continue
			}
		}

		err := poller(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if scheduled {
				diagnosticEvent(ctx, slog.LevelError, "polling failed; scheduler will retry",
					"error", err, "retry_in", interval)
			} else {
				diagnosticEvent(ctx, slog.LevelError, "manual poll failed", "error", err)
			}
		} else if err := reload(); err != nil {
			diagnosticEvent(ctx, slog.LevelWarn, "snapshot reload failed", "error", err)
		}

		if scheduled {
			lastFinished := time.Now()
/*			nextScheduled = lastFinished.Add(interval) */
			setNextScheduled(lastFinished.Add(interval))
			if err := jsonWrite(schedulePath, lastFinished.UTC()); err != nil {
				diagnosticEvent(ctx, slog.LevelWarn, "cannot save polling schedule", "error", err)
			}
			diagnosticEvent(ctx, slog.LevelInfo, "next poll scheduled",
				"delay", interval, "next_poll", nextScheduled)
		}
	}
}
