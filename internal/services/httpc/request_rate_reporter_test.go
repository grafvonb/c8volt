// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestRequestRateWindows uses actual snapshot times, including delayed and empty windows.
func TestRequestRateWindows(t *testing.T) {
	t.Parallel()
	start := time.Now()
	window := requestRateWindow{at: start}
	count, elapsed, measured := window.snapshot(7, start.Add(2*time.Second))
	require.Equal(t, uint64(7), count)
	require.Equal(t, 2*time.Second, elapsed)
	require.Equal(t, 3.5, measured)
	count, elapsed, measured = window.snapshot(7, start.Add(5*time.Second))
	require.Zero(t, count)
	require.Equal(t, 3*time.Second, elapsed)
	require.Zero(t, measured)
	_, elapsed, measured = window.snapshot(9, start.Add(5*time.Second))
	require.Zero(t, elapsed)
	require.Zero(t, measured)
	count, _, measured = window.snapshot(9, start.Add(6*time.Second))
	require.Equal(t, uint64(2), count, "nonpositive windows must not consume starts")
	require.Equal(t, 2.0, measured)
}

// TestRequestRateConcurrentWindowAccounting assigns every concurrent start to exactly one window.
func TestRequestRateConcurrentWindowAccounting(t *testing.T) {
	t.Parallel()
	var counter atomic.Uint64
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 10000 {
				counter.Add(1)
			}
		})
	}
	window := requestRateWindow{at: time.Now()}
	var total uint64
	for range 1000 {
		count, _, _ := window.snapshot(counter.Load(), time.Now())
		total += count
	}
	wg.Wait()
	count, _, _ := window.snapshot(counter.Load(), time.Now().Add(time.Nanosecond))
	require.Equal(t, uint64(80000), total+count)
}

// TestRequestRateReporterLifetime checks activation, zero-rate observation bypass and prompt idempotent close.
func TestRequestRateReporterLifetime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval, level string
		enabled         bool
	}{
		{"5s", "debug", true}, {"10s", "debug", true}, {"0s", "debug", false}, {"5s", "info", false},
	} {
		t.Run(tc.interval+tc.level, func(t *testing.T) {
			level := slog.LevelInfo
			if tc.level == "debug" {
				level = slog.LevelDebug
			}
			log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: level}))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			svc, err := New(&config.Config{HTTP: config.HTTP{Timeout: "1s", RequestRateReportInterval: tc.interval}}, log, WithInvocationContext(ctx))
			require.NoError(t, err)
			defer svc.Close()
			require.Equal(t, tc.enabled, svc.reporter != nil)
			if !tc.enabled {
				require.Nil(t, svc.requestRate, "disabled controls must retain the original transport path")
			}
			attemptCtx := svc.ToContext(ctx)
			got, observation := BeginRequestRateAttempt(attemptCtx)
			require.Same(t, attemptCtx, got)
			require.Nil(t, observation)
			if tc.enabled {
				require.Nil(t, svc.requestRate.limiter, "measurement-only mode cannot wait on a limiter")
				cancel()
				select {
				case <-svc.reporter.done:
				case <-time.After(time.Second):
					t.Fatal("cancellation waited for interval")
				}
			}
			svc.Close()
			svc.Close()
		})
	}
}

// TestRequestRateDispatchCounting counts retries and failed attempts, excluding unsent cancellations.
func TestRequestRateDispatchCounting(t *testing.T) {
	t.Parallel()
	for _, perSecond := range []float64{0, 1000} {
		t.Run(fmt.Sprint(perSecond), func(t *testing.T) {
			var calls atomic.Uint64
			var count atomic.Uint64
			transport := newRequestRateTransport(requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return newReadRetryResponse(http.StatusServiceUnavailable, ""), nil
			}), perSecond)
			transport.counter = &count
			retry := &ReadRetryTransport{base: transport, policy: readRetryPolicy{attempts: 3, baseDelay: time.Nanosecond, maxDelay: time.Nanosecond}}
			response, err := retry.RoundTrip(newReadRetryRequest(t, http.MethodGet))
			require.NoError(t, err)
			response.Body.Close()
			require.Equal(t, uint64(3), calls.Load())
			require.Equal(t, calls.Load(), count.Load())
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err = transport.RoundTrip(newReadRetryRequest(t, http.MethodPost).WithContext(ctx))
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, uint64(3), count.Load())
			if perSecond > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
				defer cancel()
				_, err = transport.RoundTrip(newReadRetryRequest(t, http.MethodPost).WithContext(ctx))
				require.Error(t, err)
				require.Equal(t, uint64(3), count.Load())
			}
			transport.base = requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })
			_, err = transport.RoundTrip(newReadRetryRequest(t, http.MethodPost))
			require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			require.Equal(t, uint64(4), count.Load(), "dispatched transport failures count")
		})
	}
}

// requestRateBlockingWriter isolates a stalled report sink from dispatch and exposes report windows.
type requestRateBlockingWriter struct {
	entered chan string
	release chan struct{}
}

// Write allows the test to control logger delay without a reporting lock in dispatch.
func (w *requestRateBlockingWriter) Write(p []byte) (int, error) {
	w.entered <- string(p)
	<-w.release
	return len(p), nil
}

// TestRequestRateBlockedReporterNeverBlocksDispatch proves logging is outside RoundTrip.
func TestRequestRateBlockedReporterNeverBlocksDispatch(t *testing.T) {
	t.Parallel()
	sink := &requestRateBlockingWriter{entered: make(chan string, 10), release: make(chan struct{})}
	log := slog.New(slog.NewTextHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := newRequestRateReporter(context.Background(), log, time.Millisecond)
	defer func() { close(sink.release); r.cancel(); <-r.done }()
	select {
	case message := <-sink.entered:
		require.Contains(t, message, "0 requests in")
	case <-time.After(time.Second):
		t.Fatal("reporter did not tick")
	}
	transport := newRequestRateTransport(requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) { return newReadRetryResponse(200, ""), nil }), 0)
	transport.counter = &r.starts
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for range 100 {
			response, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodPost))
			if err != nil {
				t.Error(err)
				return
			}
			response.Body.Close()
		}
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("dispatch waited for reporting")
	}
	require.Equal(t, uint64(100), r.starts.Load())
}

// TestRequestRateReporterCadence uses a virtual clock to prove default/custom intervals,
// empty windows, and shutdown without real-time sleeps or scheduler assumptions.
func TestRequestRateReporterCadence(t *testing.T) {
	for _, interval := range []time.Duration{5 * time.Second, 10 * time.Second, 250 * time.Millisecond} {
		t.Run(interval.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var output diagnosticCallbackWriter
				log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
				r := newRequestRateReporter(context.Background(), log, interval)
				defer func() { r.cancel(); <-r.done }()
				r.starts.Add(2)
				time.Sleep(interval - time.Nanosecond)
				synctest.Wait()
				require.Empty(t, output.String())
				time.Sleep(time.Nanosecond)
				synctest.Wait()
				require.Equal(t, 1, strings.Count(output.String(), "HTTP request rate:"))
				require.Contains(t, output.String(), fmt.Sprintf("2 requests in %s (%.2f requests/s)", interval, 2/interval.Seconds()))
				time.Sleep(interval)
				synctest.Wait()
				require.Contains(t, output.String(), fmt.Sprintf("0 requests in %s (0.00 requests/s)", interval))
				r.cancel()
				<-r.done
				before := output.String()
				time.Sleep(interval * 2)
				synctest.Wait()
				require.Equal(t, before, output.String())
			})
		})
	}
}

// TestRequestRateReportingPause suppresses all windows, preserves accounting and
// supports nested/idempotent resume without resetting the existing limiter.
func TestRequestRateReportingPause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var output diagnosticCallbackWriter
		log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
		r := newRequestRateReporter(context.Background(), log, time.Second)
		s := &Service{reporter: r}
		defer s.Close()
		r.starts.Add(2)
		time.Sleep(250 * time.Millisecond)
		resume := s.PauseRequestRateReporting()
		time.Sleep(time.Second)
		resumeNested := s.PauseRequestRateReporting()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Empty(t, output.String(), "nonzero pending traffic must also remain silent")
		resume()
		resume()
		time.Sleep(time.Second)
		synctest.Wait()
		require.Empty(t, output.String(), "the nested pause still owns the prompt")
		r.starts.Add(1)
		resumeNested()
		time.Sleep(750 * time.Millisecond)
		synctest.Wait()
		require.Contains(t, output.String(), "3 requests in 1s (3.00 requests/s)", "retain pre-prompt starts and exclude the four-second overlapping pause exactly once")
		time.Sleep(time.Second)
		synctest.Wait()
		require.Contains(t, output.String(), "0 requests in 1s (0.00 requests/s)")
		resumeAfterClose := s.PauseRequestRateReporting()
		s.Close()
		before := output.String()
		resumeAfterClose()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Equal(t, before, output.String(), "resume must not restart a closed reporter")
	})
	var absent *Service
	absent.PauseRequestRateReporting()()
	(&Service{}).PauseRequestRateReporting()()
}

// TestRequestRateReportingRepeatedPauses excludes every manual-input pause but
// retains ordinary elapsed time and does not subtract old pauses in later windows.
func TestRequestRateReportingRepeatedPauses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var output diagnosticCallbackWriter
		log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
		r := newRequestRateReporter(context.Background(), log, time.Second)
		s := &Service{reporter: r}
		defer s.Close()
		r.starts.Add(2)
		time.Sleep(250 * time.Millisecond)
		resume := s.PauseRequestRateReporting()
		time.Sleep(250 * time.Millisecond)
		resume()
		time.Sleep(250 * time.Millisecond)
		resume = s.PauseRequestRateReporting()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Empty(t, output.String())
		resume()
		time.Sleep(250 * time.Millisecond)
		synctest.Wait()
		require.Contains(t, output.String(), "2 requests in 750ms (2.67 requests/s)")
		r.starts.Add(3)
		time.Sleep(250 * time.Millisecond)
		resume = s.PauseRequestRateReporting()
		time.Sleep(time.Second)
		resume()
		time.Sleep(750 * time.Millisecond)
		synctest.Wait()
		require.Contains(t, output.String(), "3 requests in 1s (3.00 requests/s)")
		time.Sleep(time.Second)
		synctest.Wait()
		require.Contains(t, output.String(), "0 requests in 1s (0.00 requests/s)", "ordinary idle or polling waits still belong to the window")
		require.Equal(t, 3, strings.Count(output.String(), "HTTP request rate:"))
	})
}

// TestRequestRateReportingZeroActiveTime retains counts until a positive duration
// exists, even if the entire first window was spent waiting for manual input.
func TestRequestRateReportingZeroActiveTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var output diagnosticCallbackWriter
		log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
		r := &requestRateReporter{}
		s := &Service{reporter: r}
		window := requestRateWindow{at: time.Now()}
		r.starts.Add(4)
		resume := s.PauseRequestRateReporting()
		time.Sleep(3 * time.Second)
		resume()
		r.report(context.Background(), log, &window)
		require.Empty(t, output.String())
		time.Sleep(500 * time.Millisecond)
		r.report(context.Background(), log, &window)
		require.Contains(t, output.String(), "4 requests in 500ms (8.00 requests/s)")
	})
}

// TestRequestRatePauseWaitsForEmission closes the check/write race: once pause
// returns a previously admitted summary cannot appear after the prompt.
func TestRequestRatePauseWaitsForEmission(t *testing.T) {
	t.Parallel()
	sink := &requestRateBlockingWriter{entered: make(chan string, 100), release: make(chan struct{})}
	log := slog.New(slog.NewTextHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := newRequestRateReporter(context.Background(), log, time.Millisecond)
	s := &Service{reporter: r}
	release := sync.OnceFunc(func() { close(sink.release) })
	defer func() { release(); s.Close() }()
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("reporter did not start a write")
	}
	paused := make(chan func(), 1)
	go func() { paused <- s.PauseRequestRateReporting() }()
	// A mutex wait is not durably blocked in synctest; use a bounded real wait
	// while the controlled sink keeps the existing write demonstrably in flight.
	select {
	case <-paused:
		t.Fatal("pause returned while a summary was still being written")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	select {
	case resume := <-paused:
		defer resume()
	case <-time.After(time.Second):
		t.Fatal("pause did not finish after the write completed")
	}
	// An overdue tick may finish before pause takes ownership; only later writes
	// violate the boundary, so discard records completed before pause returned.
	for len(sink.entered) > 0 {
		<-sink.entered
	}
	select {
	case <-sink.entered:
		t.Fatal("a summary crossed the paused prompt boundary")
	case <-time.After(20 * time.Millisecond):
	}
}
