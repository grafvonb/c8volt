// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

type requestRateRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip adapts isolated delegates without any network access.
func (f requestRateRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type requestRateWaitFunc func(context.Context) error

// Wait supplies controlled admission/cancellation for deadline edge cases.
func (f requestRateWaitFunc) Wait(ctx context.Context) error { return f(ctx) }

// TestRequestRateSharedConcurrentStarts proves one burst-one budget and overlapping in-flight calls.
func TestRequestRateSharedConcurrentStarts(t *testing.T) {
	t.Parallel()
	const count = 6
	const perSecond = 25.0
	release := make(chan struct{})
	var wg sync.WaitGroup
	defer func() { close(release); wg.Wait() }()
	starts := make(chan time.Time, count)
	transport := newRequestRateTransport(requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) {
		starts <- time.Now()
		<-release
		return newReadRetryResponse(http.StatusOK, ""), nil
	}), perSecond)
	for range count {
		wg.Go(func() {
			response, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodPost))
			if err != nil {
				t.Error(err)
				return
			}
			_ = response.Body.Close()
		})
	}
	times := make([]time.Time, 0, count)
	for range count {
		select {
		case started := <-starts:
			times = append(times, started)
		case <-time.After(3 * time.Second):
			t.Fatal("pacing serialized in-flight responses")
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	// Allow 5ms of timer/scheduler variation at the observable delegate boundary.
	for i, started := range times {
		elapsed := started.Sub(times[0]) + 5*time.Millisecond
		require.LessOrEqual(t, float64(i+1), 1+perSecond*elapsed.Seconds())
	}
}

// TestRequestRateIdleBurstNeverAccumulates permits only one immediate request after idle time.
func TestRequestRateIdleBurstNeverAccumulates(t *testing.T) {
	t.Parallel()
	transport := newRequestRateTransport(requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) { return newReadRetryResponse(200, ""), nil }), 20)
	time.Sleep(120 * time.Millisecond)
	first, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodPost))
	require.NoError(t, err)
	_ = first.Body.Close()
	started := time.Now()
	second, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodPost))
	require.NoError(t, err)
	_ = second.Body.Close()
	require.GreaterOrEqual(t, time.Since(started), 45*time.Millisecond)
}

// TestRequestRateLocalFailures preserves typed cancellation/deadline identity without dispatch.
func TestRequestRateLocalFailures(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"already canceled", "queued cancellation", "early deadline", "after admission"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			transport := newRequestRateTransport(requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) { calls.Add(1); return newReadRetryResponse(200, ""), nil }), 0.5)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := context.Canceled
			switch mode {
			case "already canceled":
				cancel()
			case "queued cancellation", "early deadline":
				response, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodPost))
				require.NoError(t, err)
				_ = response.Body.Close()
				calls.Store(0)
				if mode == "early deadline" {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(context.Background(), time.Second)
					defer stop()
					want = context.DeadlineExceeded
				} else {
					time.AfterFunc(20*time.Millisecond, cancel)
				}
			case "after admission":
				transport.limiter = requestRateWaitFunc(func(context.Context) error { cancel(); return nil })
			}
			response, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodPost).WithContext(ctx))
			require.Nil(t, response)
			require.ErrorIs(t, err, want)
			var local *RequestRateWaitError
			require.ErrorAs(t, err, &local)
			require.Contains(t, err.Error(), "request not sent")
			require.Zero(t, calls.Load())
			if mode == "early deadline" {
				require.NoError(t, ctx.Err())
			}
		})
	}
}

// TestRequestRateClientTimeoutRetainsObservation exercises net/http's replacement of the typed transport error.
func TestRequestRateClientTimeoutRetainsObservation(t *testing.T) {
	t.Parallel()
	service, err := New(&config.Config{HTTP: config.HTTP{RequestRateReportInterval: "5s", Timeout: "20ms", RequestRate: 1}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	parent := service.ToContext(context.Background())
	attempt, observation := BeginRequestRateAttempt(parent)
	transport := service.requestRate
	// Force expiry rather than the limiter's usual early deadline rejection.
	transport.limiter = requestRateWaitFunc(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	var calls atomic.Int32
	transport.base = requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("unexpected dispatch")
	})
	// net/http has independent context and legacy client timeout timers for
	// custom transports. Wait for its Cancel channel to prove the latter fired
	// before returning; context cancellation alone races error replacement.
	base := service.Client().Transport
	service.Client().Transport = requestRateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		response, err := base.RoundTrip(req)
		<-req.Cancel
		return response, err
	})
	_, err = service.Client().Do(newReadRetryRequest(t, http.MethodPost).WithContext(attempt))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	var local *RequestRateWaitError
	require.False(t, errors.As(err, &local), "actual Client.Timeout must replace the original type in this regression")
	require.ErrorAs(t, observation.Failure(), &local)
	require.ErrorIs(t, observation.Failure(), context.DeadlineExceeded)
	require.NoError(t, parent.Err())
	require.Zero(t, calls.Load())
	_, next := BeginRequestRateAttempt(parent)
	require.NoError(t, next.Failure(), "attempt evidence cannot leak")
}

// TestRequestRateServiceWiring keeps zero-rate reporting unpaced and diagnostics traversable.
func TestRequestRateServiceWiring(t *testing.T) {
	t.Parallel()
	for _, value := range []float64{0, 0.5} {
		cfg := &config.Config{HTTP: config.HTTP{RequestRateReportInterval: "5s", Timeout: "30s", RequestRate: value}}
		log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
		service, err := New(cfg, log, WithDiagnostics())
		require.NoError(t, err)
		defer service.Close()
		service.InstallAuthEditor(nil)
		require.NotNil(t, unwrapLogTransport(service.Client().Transport))
		collector := diagnosticCollectorFromTransport(service.Client().Transport)
		require.NotNil(t, collector)
		other := &http.Client{}
		ShareDiagnostics(service.Client(), other)
		require.Same(t, collector, diagnosticCollectorFromTransport(other.Transport))
		require.IsType(t, &DiagnosticsTransport{}, other.Transport, "independent OAuth clients share diagnostics only")
		ctx := service.ToContext(context.Background())
		attempt, observation := BeginRequestRateAttempt(ctx)
		if value == 0 {
			require.Nil(t, service.requestRate.limiter)
			require.Same(t, ctx, attempt)
			require.Nil(t, observation)
			require.NoError(t, observation.Failure())
		} else {
			require.NotNil(t, service.requestRate)
			require.NotSame(t, ctx, attempt)
			require.NotNil(t, observation)
		}
	}
}
