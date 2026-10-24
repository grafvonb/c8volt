// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

// requestRateWindow belongs to one reporter; cumulative snapshots partition starts.
type requestRateWindow struct {
	count uint64
	at    time.Time
}

// snapshot uses actual monotonic elapsed time, retaining starts when time has not advanced.
func (w *requestRateWindow) snapshot(count uint64, at time.Time) (uint64, time.Duration, float64) {
	elapsed := at.Sub(w.at)
	if elapsed <= 0 {
		return 0, 0, 0
	}
	delta := count - w.count
	w.count, w.at = count, at
	return delta, elapsed, float64(delta) / elapsed.Seconds()
}

// requestRateReporter owns the sole reporting loop; dispatch only touches starts.
type requestRateReporter struct {
	starts atomic.Uint64
	cancel context.CancelFunc
	done   chan struct{}
}

// newRequestRateReporter starts one independently cancellable timer for an enabled logger.
func newRequestRateReporter(parent context.Context, log *slog.Logger, interval time.Duration) *requestRateReporter {
	ctx, cancel := context.WithCancel(parent)
	r := &requestRateReporter{cancel: cancel, done: make(chan struct{})}
	window := requestRateWindow{at: time.Now()}
	ticker := time.NewTicker(interval)
	go func() {
		defer close(r.done)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if ctx.Err() != nil {
					return
				}
				// Read the current time, not the potentially stale timestamp buffered by the ticker.
				count, elapsed, measured := window.snapshot(r.starts.Load(), time.Now())
				if elapsed > 0 {
					log.DebugContext(ctx, fmt.Sprintf("HTTP request rate: %d requests in %s (%.2f requests/s)", count, elapsed, measured))
				}
			}
		}
	}()
	return r
}

// WithInvocationContext ties reporting to the owning invocation's cancellation.
func WithInvocationContext(ctx context.Context) Option {
	return func(s *Service) {
		if ctx != nil {
			s.invocationContext = ctx
		}
	}
}

// Close cancels reporting and joins its goroutine without waiting for a tick.
// It is safe to repeat.
// It does not cancel requests or change ownership of the underlying HTTP transport.
func (s *Service) Close() {
	if s == nil || s.reporter == nil {
		return
	}
	s.reporter.cancel()
	<-s.reporter.done
}
