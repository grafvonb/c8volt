// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	// emission protects pause timing and complete logger writes, never dispatch.
	emission  sync.Mutex
	pauses    int
	pausedAt  time.Time
	pausedFor time.Duration
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
				r.report(ctx, log, &window)
			}
		}
	}()
	return r
}

// report keeps the window intact while paused, so the next emitted rate retains
// every start while excluding time spent waiting for manual input.
func (r *requestRateReporter) report(ctx context.Context, log *slog.Logger, window *requestRateWindow) {
	r.emission.Lock()
	defer r.emission.Unlock()
	if r.pauses > 0 || ctx.Err() != nil {
		return
	}
	// Shift each snapshot by cumulative paused time. Differences then exclude only
	// pauses since the previous snapshot, retaining ordinary idle/polling time.
	count, elapsed, measured := window.snapshot(r.starts.Load(), time.Now().Add(-r.pausedFor))
	if elapsed > 0 {
		log.DebugContext(ctx, fmt.Sprintf("HTTP request rate: %d requests in %s (%.2f requests/s)", count, elapsed, measured))
	}
}

// PauseRequestRateReporting finishes any in-flight summary before returning.
// Its idempotent resume restores periodic output without changing request pacing,
// counting, or deadlines. Disabled reporting is a no-op; overlapping pauses nest.
func (s *Service) PauseRequestRateReporting() func() {
	if s == nil || s.reporter == nil {
		return func() {}
	}
	r := s.reporter
	r.emission.Lock()
	if r.pauses == 0 {
		r.pausedAt = time.Now()
	}
	r.pauses++
	r.emission.Unlock()
	return sync.OnceFunc(func() {
		r.emission.Lock()
		defer r.emission.Unlock()
		r.pauses--
		if r.pauses == 0 {
			r.pausedFor += time.Since(r.pausedAt)
		}
	})
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
