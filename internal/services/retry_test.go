// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/stretchr/testify/require"
)

// TestRetryCamundaMutationRetriesResourceExhaustion verifies Camunda broker throttling is retried before the caller maps HTTP status errors.
func TestRetryCamundaMutationRetriesResourceExhaustion(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	got, err := RetryCamundaMutation(context.Background(), slog.Default(), "create pi", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		if calls == 1 {
			return "", &http.Response{StatusCode: http.StatusServiceUnavailable}, []byte(`{"title":"RESOURCE_EXHAUSTED"}`), nil
		}
		return "ok", &http.Response{StatusCode: http.StatusOK}, nil, nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.Equal(t, 2, calls)
}

// TestRetryCamundaMutationDoesNotRetryConflicts keeps functional command errors immediate.
func TestRetryCamundaMutationDoesNotRetryConflicts(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	_, err := RetryCamundaMutation(context.Background(), slog.Default(), "delete pi", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		return "", &http.Response{StatusCode: http.StatusConflict}, []byte(`{"title":"CONFLICT"}`), nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

// TestRetryCamundaMutationRespectsContext verifies backoff sleep exits promptly on cancellation.
func TestRetryCamundaMutationRespectsContext(t *testing.T) {
	oldAttempts := camundaMutationRetryAttempts
	oldBaseDelay := camundaMutationRetryBaseDelay
	oldMaxDelay := camundaMutationRetryMaxDelay
	camundaMutationRetryAttempts = 3
	camundaMutationRetryBaseDelay = time.Minute
	camundaMutationRetryMaxDelay = time.Minute
	t.Cleanup(func() {
		camundaMutationRetryAttempts = oldAttempts
		camundaMutationRetryBaseDelay = oldBaseDelay
		camundaMutationRetryMaxDelay = oldMaxDelay
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RetryCamundaMutation(ctx, slog.Default(), "delete pi", func(context.Context) (string, *http.Response, []byte, error) {
		return "", &http.Response{StatusCode: http.StatusTooManyRequests}, nil, nil
	})

	require.ErrorIs(t, err, context.Canceled)
}

// TestRetryCamundaMutationStopsAfterBudget verifies persistent throttling returns the last response for normal HTTP error mapping.
func TestRetryCamundaMutationStopsAfterBudget(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	got, err := RetryCamundaMutation(context.Background(), slog.Default(), "delete pi", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		return "last", &http.Response{StatusCode: http.StatusServiceUnavailable}, []byte(`{"title":"RESOURCE_EXHAUSTED"}`), nil
	})

	require.NoError(t, err)
	require.Equal(t, "last", got)
	require.Equal(t, camundaMutationRetryAttempts, calls)
}

// TestRetryCamundaMutationRetriesTemporaryNetworkErrors covers retryable transport failures.
func TestRetryCamundaMutationRetriesTemporaryNetworkErrors(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	got, err := RetryCamundaMutation(context.Background(), slog.Default(), "deploy", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		if calls == 1 {
			return "", nil, nil, temporaryNetError{}
		}
		return "ok", &http.Response{StatusCode: http.StatusOK}, nil, nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.Equal(t, 2, calls)
}

type temporaryNetError struct{}

func (temporaryNetError) Error() string   { return "temporary" }
func (temporaryNetError) Timeout() bool   { return false }
func (temporaryNetError) Temporary() bool { return true }
func (temporaryNetError) Unwrap() error   { return errors.New("temporary") }

func withFastCamundaMutationRetry(t *testing.T) {
	t.Helper()
	oldAttempts := camundaMutationRetryAttempts
	oldBaseDelay := camundaMutationRetryBaseDelay
	oldMaxDelay := camundaMutationRetryMaxDelay
	oldLogInterval := camundaMutationRetryLogInterval
	camundaMutationRetryAttempts = 3
	camundaMutationRetryBaseDelay = time.Nanosecond
	camundaMutationRetryMaxDelay = time.Nanosecond
	camundaMutationRetryLogInterval = 0
	t.Cleanup(func() {
		camundaMutationRetryAttempts = oldAttempts
		camundaMutationRetryBaseDelay = oldBaseDelay
		camundaMutationRetryMaxDelay = oldMaxDelay
		camundaMutationRetryLogInterval = oldLogInterval
	})
}

type requestRateRetryTransportFunc func(*http.Request) (*http.Response, error)

// RoundTrip supports isolated interception of client timeout error replacement.
func (f requestRateRetryTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestRequestRateMutationLocalTimeoutNeverRetries uses the real client timeout wrapper with a still-live parent context.
func TestRequestRateMutationLocalTimeoutNeverRetries(t *testing.T) {
	withFastCamundaMutationRetry(t)
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint("timeout expired ", expired), func(t *testing.T) {
			var dispatched atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { dispatched.Add(1); w.WriteHeader(http.StatusNoContent) }))
			defer server.Close()
			service, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "30ms", RequestRate: 0.5}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			parent := service.ToContext(context.Background())
			req, err := http.NewRequestWithContext(parent, http.MethodPost, server.URL, nil)
			require.NoError(t, err)
			first, err := service.Client().Do(req)
			require.NoError(t, err)
			_ = first.Body.Close()
			base := service.Client().Transport
			if expired {
				// Admission rejects the next request early. Delay returning that unsent
				// error until Client.Timeout expires, forcing net/http's type replacement.
				service.Client().Transport = requestRateRetryTransportFunc(func(req *http.Request) (*http.Response, error) {
					resp, err := base.RoundTrip(req)
					if httpc.IsRequestRateWaitError(err) {
						<-req.Cancel // closed only after the client marks its timeout
					}
					return resp, err
				})
			}
			calls := 0
			_, err = RetryCamundaMutation(parent, nil, "cancel", func(ctx context.Context) (string, *http.Response, []byte, error) {
				calls++
				response, callErr := service.Client().Do(req.WithContext(ctx))
				if expired {
					require.False(t, httpc.IsRequestRateWaitError(callErr), "Client.Timeout must have replaced the type")
				}
				return "preserved", response, nil, callErr
			})
			require.Equal(t, 1, calls)
			require.Equal(t, int32(1), dispatched.Load(), "only the initial priming request was sent")
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.True(t, httpc.IsRequestRateWaitError(err))
			require.Contains(t, err.Error(), "request not sent")
			require.NoError(t, parent.Err())
			// A new resource must not inherit the previous local failure evidence.
			_, err = RetryCamundaMutation(parent, nil, "other resource", func(context.Context) (string, *http.Response, []byte, error) { return "ok", nil, nil, nil })
			require.NoError(t, err)
		})
	}
}

// TestRequestRateMutationBackendPolicy keeps dispatched timeout/429/eligible-503 retries and budgets unchanged.
func TestRequestRateMutationBackendPolicy(t *testing.T) {
	withFastCamundaMutationRetry(t)
	for _, mode := range []string{"timeout", "429", "eligible 503", "other 503"} {
		t.Run(mode, func(t *testing.T) {
			service, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "30s", RequestRate: 100}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			calls := 0
			var contexts []context.Context
			got, err := RetryCamundaMutation(service.ToContext(context.Background()), nil, "mutate", func(ctx context.Context) (string, *http.Response, []byte, error) {
				calls++
				contexts = append(contexts, ctx)
				switch mode {
				case "timeout":
					return "last", nil, nil, context.DeadlineExceeded
				case "429":
					return "last", &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"0"}}}, nil, nil
				case "eligible 503":
					return "last", &http.Response{StatusCode: 503}, []byte("RESOURCE_EXHAUSTED"), nil
				default:
					return "last", &http.Response{StatusCode: 503}, []byte("unavailable"), nil
				}
			})
			require.Equal(t, "last", got)
			if mode == "timeout" {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			} else {
				require.NoError(t, err)
			}
			if mode == "other 503" {
				require.Equal(t, 1, calls)
			} else {
				require.Equal(t, camundaMutationRetryAttempts, calls)
				require.NotSame(t, contexts[0], contexts[1], "each callback must receive fresh evidence")
			}
		})
	}
}

// TestRequestRateMutationZeroPassesOriginalContext ensures disabled pacing adds no attempt context.
func TestRequestRateMutationZeroPassesOriginalContext(t *testing.T) {
	withFastCamundaMutationRetry(t)
	service, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "30s"}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	parent := service.ToContext(context.Background())
	_, err = RetryCamundaMutation(parent, nil, "mutate", func(ctx context.Context) (string, *http.Response, []byte, error) {
		require.Same(t, parent, ctx)
		return "ok", nil, nil, nil
	})
	require.NoError(t, err)
}

// TestRequestRateMutationDispatchedRetriesShareAllowance proves each actual write retry re-enters shared pacing.
func TestRequestRateMutationDispatchedRetriesShareAllowance(t *testing.T) {
	withFastCamundaMutationRetry(t)
	// testx imports services/common, so this package uses a local collector.
	var starts []time.Time
	var startsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startsMu.Lock()
		starts = append(starts, time.Now())
		startsMu.Unlock()
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	service, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "2s", RequestRate: 20}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	ctx := service.ToContext(context.Background())
	_, err = RetryCamundaMutation(ctx, nil, "cancel", func(ctx context.Context) (string, *http.Response, []byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, nil)
		require.NoError(t, err)
		response, err := service.Client().Do(req)
		if response != nil {
			_ = response.Body.Close()
		}
		return "", response, nil, err
	})
	require.NoError(t, err)
	startsMu.Lock()
	times := append([]time.Time(nil), starts...)
	startsMu.Unlock()
	require.Len(t, times, camundaMutationRetryAttempts)
	// Test-owned loopback I/O may shift timestamps slightly from delegation.
	for i, started := range times {
		require.LessOrEqual(t, float64(i+1), 1+20*(started.Sub(times[0])+5*time.Millisecond).Seconds())
	}
}
