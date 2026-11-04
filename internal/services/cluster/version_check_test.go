// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	clustercommon "github.com/grafvonb/c8volt/internal/services/cluster/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// versionCheckTransport prevents any test fixture from reaching a real endpoint.
type versionCheckTransport func(*http.Request) (*http.Response, error)

// RoundTrip dispatches only to the test-owned handler.
func (f versionCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// versionCheckConfig constructs an effective version and an unreachable fake-only URL.
func versionCheckConfig(version toolx.CamundaVersion) *config.Config {
	return &config.Config{App: config.App{CamundaVersion: version}, APIs: config.APIs{Camunda: config.API{BaseURL: "https://isolated.invalid/v2"}}, HTTP: config.HTTP{Timeout: "1s", RequestRateReportInterval: "0s"}}
}

// versionCheckResponse supplies generated-client-compatible responses with owned bodies.
func versionCheckResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}
}

// versionCheckLogger discards diagnostics so tests need no process-global logger.
func versionCheckLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// TestVersionCheckOutcomes uses real topology adapters to assert safe, retained, exact advisory facts.
func TestVersionCheckOutcomes(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, tc := range []struct {
			name, observed, body string
			status               int
			reason               d.VersionCheckFailure
			transportErr         error
		}{
			{name: "match", observed: version.String() + ".3"},
			{name: "patch build", observed: version.String() + ".7+build.2"},
			{name: "prerelease", observed: "8.10.0-alpha4"},
			{name: "numeric minor", observed: "8.10.3"},
			{name: "major", observed: "9.0.1"},
			{name: "trim", observed: " v8.9.3 "},
			{name: "boundary", observed: "8.9+" + strings.Repeat("a", 124)},
			{name: "missing", body: `{}`, reason: d.VersionCheckInvalidVersion},
			{name: "null", body: `null`, reason: d.VersionCheckInvalidVersion},
			{name: "malformed JSON", body: `{`, reason: d.VersionCheckInvalidVersion},
			{name: "wrong JSON type", body: `{"gatewayVersion":42}`, reason: d.VersionCheckInvalidVersion},
			{name: "empty body", reason: d.VersionCheckInvalidVersion},
			{name: "unsafe", observed: "8.9\nPOST https://secret.invalid body key", reason: d.VersionCheckInvalidVersion},
			{name: "long", observed: "8.9+" + strings.Repeat("a", 125), reason: d.VersionCheckInvalidVersion},
			{name: "bad version", observed: "8.9.garbage", reason: d.VersionCheckInvalidVersion},
			{name: "unauthorized", status: 401, body: `{}`, reason: d.VersionCheckAccessDenied},
			{name: "forbidden", status: 403, body: `{"detail":"GET https://secret.invalid key body"}`, reason: d.VersionCheckAccessDenied},
			{name: "backend", status: 500, body: `{}`, reason: d.VersionCheckUnavailable},
			{name: "gateway timeout", status: 504, body: `{}`, reason: d.VersionCheckTimeout},
			{name: "transport", transportErr: errors.New("GET https://secret.invalid body key credentials"), reason: d.VersionCheckUnavailable},
		} {
			t.Run(version.String()+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				requests, callbacks := 0, 0
				body := tc.body
				if tc.observed != "" {
					encoded, err := json.Marshal(map[string]any{"gatewayVersion": tc.observed, "brokers": []map[string]string{{"version": "99.99"}}})
					require.NoError(t, err)
					body = string(encoded)
				}
				client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
					requests++
					require.Equal(t, "/v2/topology", req.URL.Path)
					if tc.transportErr != nil {
						return nil, tc.transportErr
					}
					status := tc.status
					if status == 0 {
						status = 200
					}
					return versionCheckResponse(req, status, body), nil
				})}
				var callbackOutcome d.VersionCheckOutcome
				checker := NewVersionCheck(versionCheckConfig(version), client, versionCheckLogger(), func(outcome d.VersionCheckOutcome) { callbacks++; callbackOutcome = outcome })
				require.Zero(t, requests)
				require.Zero(t, callbacks)
				_, complete := checker.Outcome()
				require.False(t, complete)
				checker.BeforeRequest(context.Background())
				checker.BeforeRequest(context.Background())
				got, complete := checker.Outcome()
				require.True(t, complete)
				require.Equal(t, 1, requests)
				require.Equal(t, 1, callbacks)
				require.Equal(t, callbackOutcome, got)
				want := d.VersionCheckOutcome{ConfiguredVersion: version.String()}
				if tc.observed != "" {
					want.BrokerCount = 1
					want.BrokerVersions = []string{"99.99"}
				}
				if tc.reason != "" {
					want.Status, want.FailureReason = d.VersionCheckFailed, tc.reason
				} else {
					want.Status, want.ObservedVersion = d.VersionCheckMatched, strings.TrimSpace(tc.observed)
					parsed, err := toolx.ParseCamundaRelease(tc.observed)
					require.NoError(t, err)
					configured, err := toolx.ParseCamundaRelease(version.String())
					require.NoError(t, err)
					if parsed.Major != configured.Major || parsed.Minor != configured.Minor {
						want.Status = d.VersionCheckMismatched
					}
				}
				require.Equal(t, want, got)
				require.Equal(t, want.Status != d.VersionCheckMatched, got.WarningNeeded())
			})
		}
	}
}

// versionCheckWaitContext signals when a waiter reaches its cancellation/completion select.
type versionCheckWaitContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

// Done proves that the caller joined the in-flight attempt without sleeping to infer ordering.
func (c *versionCheckWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.entered) })
	return c.Context.Done()
}

// awaitVersionCheck makes synchronization failures bounded instead of hanging a test process.
func awaitVersionCheck(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("version check synchronization timed out")
	}
}

// TestVersionCheckConcurrentCompletion publishes only after callback and shares one attempt with cancelable waiters.
func TestVersionCheckConcurrentCompletion(t *testing.T) {
	t.Parallel()
	probeEntered, releaseProbe := make(chan struct{}), make(chan struct{})
	callbackEntered, releaseCallback := make(chan struct{}), make(chan struct{})
	var releaseProbeOnce, releaseCallbackOnce sync.Once
	defer releaseProbeOnce.Do(func() { close(releaseProbe) })
	defer releaseCallbackOnce.Do(func() { close(releaseCallback) })
	var requests, callbacks atomic.Int32
	client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		close(probeEntered)
		<-releaseProbe
		return versionCheckResponse(req, 200, `{"gatewayVersion":"8.10.3"}`), nil
	})}
	var checker *VersionCheck
	checker = NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), func(d.VersionCheckOutcome) {
		callbacks.Add(1)
		_, complete := checker.Outcome()
		if complete {
			t.Error("outcome published before callback completed")
		}
		close(callbackEntered)
		<-releaseCallback
	})
	winnerDone := make(chan struct{})
	go func() { checker.BeforeRequest(context.Background()); close(winnerDone) }()
	awaitVersionCheck(t, probeEntered)
	const waiters = 12
	waiterDone := make(chan struct{}, waiters)
	for range waiters {
		ctx := &versionCheckWaitContext{Context: context.Background(), entered: make(chan struct{})}
		go func() { checker.BeforeRequest(ctx); waiterDone <- struct{}{} }()
		awaitVersionCheck(t, ctx.entered)
	}
	canceled, cancel := context.WithCancel(context.Background())
	waiting := &versionCheckWaitContext{Context: canceled, entered: make(chan struct{})}
	canceledDone := make(chan struct{})
	go func() { checker.BeforeRequest(waiting); close(canceledDone) }()
	awaitVersionCheck(t, waiting.entered)
	cancel()
	awaitVersionCheck(t, canceledDone)
	releaseProbeOnce.Do(func() { close(releaseProbe) })
	awaitVersionCheck(t, callbackEntered)
	_, complete := checker.Outcome()
	require.False(t, complete)
	select {
	case <-waiterDone:
		t.Fatal("waiter proceeded before callback completion")
	default:
	}
	releaseCallbackOnce.Do(func() { close(releaseCallback) })
	awaitVersionCheck(t, winnerDone)
	for range waiters {
		awaitVersionCheck(t, waiterDone)
	}
	outcome, complete := checker.Outcome()
	require.True(t, complete)
	require.Equal(t, d.VersionCheckMismatched, outcome.Status)
	require.EqualValues(t, 1, requests.Load())
	require.EqualValues(t, 1, callbacks.Load())
	// Later operations share no long-lived execution lock after the attempt completes.
	checker.BeforeRequest(context.Background())
	require.EqualValues(t, 1, requests.Load())
}

// TestVersionCheckCancellation consumes winner cancellation but leaves pre-canceled invocations idle.
func TestVersionCheckCancellation(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	entered := make(chan struct{})
	client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		close(entered)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	callbacks := 0
	checker := NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), func(d.VersionCheckOutcome) { callbacks++ })
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	checker.BeforeRequest(canceled)
	_, complete := checker.Outcome()
	require.False(t, complete)
	require.Zero(t, requests.Load())
	winner, stop := context.WithCancel(context.Background())
	defer stop()
	done := make(chan struct{})
	go func() { checker.BeforeRequest(winner); close(done) }()
	awaitVersionCheck(t, entered)
	stop()
	awaitVersionCheck(t, done)
	checker.BeforeRequest(context.Background())
	got, complete := checker.Outcome()
	require.True(t, complete)
	require.Equal(t, d.VersionCheckOutcome{Status: d.VersionCheckFailed, ConfiguredVersion: "8.9", FailureReason: d.VersionCheckCanceled}, got)
	require.EqualValues(t, 1, requests.Load())
	require.Equal(t, 1, callbacks)
}

// TestVersionCheckBudgets inspects bounded child deadlines without waiting for the two-second cap.
func TestVersionCheckBudgets(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		timeout, parent, budget time.Duration
	}{
		{"cap", 0, 0, 2 * time.Second}, {"long client", 10 * time.Second, 0, 2 * time.Second},
		{"short client", 100 * time.Millisecond, 0, 100 * time.Millisecond}, {"parent", time.Second, 80 * time.Millisecond, 80 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := context.Background()
			if tc.parent > 0 {
				var cancel context.CancelFunc
				parent, cancel = context.WithTimeout(parent, tc.parent)
				defer cancel()
			}
			start := time.Now()
			client := &http.Client{Timeout: tc.timeout, Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
				deadline, ok := req.Context().Deadline()
				require.True(t, ok)
				require.LessOrEqual(t, deadline.Sub(start), tc.budget+10*time.Millisecond)
				require.Greater(t, time.Until(deadline), tc.budget/2)
				return versionCheckResponse(req, 200, `{"gatewayVersion":"8.9"}`), nil
			})}
			checker := NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), nil)
			checker.BeforeRequest(parent)
			got, _ := checker.Outcome()
			require.Equal(t, d.VersionCheckMatched, got.Status)
			require.NoError(t, parent.Err())
		})
	}
}

// TestVersionCheckShortTimeoutContinues proves a failed bounded probe does not cancel the original operational request.
func TestVersionCheckShortTimeoutContinues(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	client := &http.Client{Timeout: 30 * time.Millisecond, Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		if req.URL.Path == "/v2/topology" {
			<-req.Context().Done()
			return nil, req.Context().Err()
		}
		deadline, ok := req.Context().Deadline()
		require.True(t, ok)
		require.Greater(t, time.Until(deadline), 15*time.Millisecond)
		return versionCheckResponse(req, 200, `{"ok":true}`), nil
	})}
	checker := NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), nil)
	ctx := httpc.WithBeforeRequestGate(context.Background(), checker)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://isolated.invalid/operation", strings.NewReader("body"))
	require.NoError(t, err)
	response, err := httpc.NewBeforeRequestClient(client).Do(req)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.NoError(t, ctx.Err())
	got, _ := checker.Outcome()
	require.Equal(t, d.VersionCheckOutcome{Status: d.VersionCheckFailed, ConfiguredVersion: "8.9", FailureReason: d.VersionCheckTimeout}, got)
	require.EqualValues(t, 2, requests.Load())
}

// TestVersionCheckLazyConstruction captures effective config and retains factory failures as advisory outcomes.
func TestVersionCheckLazyConstruction(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"nil config", "empty URL", "invalid URL", "unsupported"} {
		t.Run(kind, func(t *testing.T) {
			cfg := versionCheckConfig(toolx.V89)
			switch kind {
			case "nil config":
				cfg = nil
			case "empty URL":
				cfg.APIs.Camunda.BaseURL = ""
			case "invalid URL":
				cfg.APIs.Camunda.BaseURL = "%"
			case "unsupported":
				cfg.App.CamundaVersion = "0.0"
			}
			client := &http.Client{Transport: versionCheckTransport(func(*http.Request) (*http.Response, error) {
				t.Error("invalid construction reached transport")
				return nil, errors.New("unexpected")
			})}
			callbacks := 0
			checker := NewVersionCheck(cfg, client, versionCheckLogger(), func(d.VersionCheckOutcome) { callbacks++ })
			require.Zero(t, callbacks)
			checker.BeforeRequest(context.Background())
			checker.BeforeRequest(context.Background())
			got, complete := checker.Outcome()
			require.True(t, complete)
			require.Equal(t, d.VersionCheckUnavailable, got.FailureReason)
			require.Equal(t, d.VersionCheckFailed, got.Status)
			require.Equal(t, 1, callbacks)
		})
	}
	cfg := versionCheckConfig(toolx.V89)
	calls := 0
	client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		return versionCheckResponse(req, 200, `{"gatewayVersion":"8.9"}`), nil
	})}
	first := NewVersionCheck(cfg, client, versionCheckLogger(), nil)
	second := NewVersionCheck(cfg, client, versionCheckLogger(), nil)
	cfg.App.CamundaVersion = toolx.V810
	first.BeforeRequest(context.Background())
	second.BeforeRequest(context.Background())
	for _, checker := range []*VersionCheck{first, second} {
		got, _ := checker.Outcome()
		require.Equal(t, "8.9", got.ConfiguredVersion)
		require.Equal(t, d.VersionCheckMatched, got.Status)
	}
	require.Equal(t, 2, calls)
}

// TestVersionCheckProbeCopy refuses redirects without mutating shared cookie, transport, or redirect behavior.
func TestVersionCheckProbeCopy(t *testing.T) {
	t.Parallel()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	endpoint, err := url.Parse("https://isolated.invalid")
	require.NoError(t, err)
	jar.SetCookies(endpoint, []*http.Cookie{{Name: "session", Value: "shared"}})
	calls, redirects := 0, 0
	transport := versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		cookie, err := req.Cookie("session")
		require.NoError(t, err)
		require.Equal(t, "shared", cookie.Value)
		response := versionCheckResponse(req, 302, "")
		response.Header.Set("Location", "https://redirect.invalid/target")
		return response, nil
	})
	client := &http.Client{Transport: transport, Jar: jar, Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return http.ErrUseLastResponse }}
	checker := NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), nil)
	checker.BeforeRequest(context.Background())
	got, _ := checker.Outcome()
	require.Equal(t, d.VersionCheckUnavailable, got.FailureReason)
	require.Equal(t, 1, calls)
	require.Zero(t, redirects)
	require.Same(t, jar, client.Jar)
	require.Equal(t, time.Second, client.Timeout)
	require.ErrorIs(t, client.CheckRedirect(nil, nil), http.ErrUseLastResponse)
	require.Equal(t, 1, redirects)
}

// TestVersionCheckNilTopologyResponse classifies the existing adapter's nil-response sentinel without exposing its detail.
func TestVersionCheckNilTopologyResponse(t *testing.T) {
	t.Parallel()
	_, err := clustercommon.GetClusterTopology(context.Background(), versionCheckLogger(), "https://isolated.invalid", nil,
		func(context.Context) (clustercommon.PayloadResponse[struct{}], error) {
			return clustercommon.PayloadResponse[struct{}]{}, nil
		},
		func(struct{}) d.Topology { return d.Topology{} })
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Equal(t, d.VersionCheckInvalidVersion, versionCheckFailure(context.Background(), fmt.Errorf("wrapped: %w", err)))
}

// TestVersionCheckNoRetry uses the actual shared retry transport and proves only the advisory disables retries.
func TestVersionCheckNoRetry(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		err    error
		reason d.VersionCheckFailure
	}{
		{"rate limited", 429, nil, d.VersionCheckUnavailable},
		{"unavailable", 503, nil, d.VersionCheckUnavailable},
		{"temporary network", 0, &net.DNSError{Err: "temporary", IsTemporary: true}, d.VersionCheckUnavailable},
		{"network timeout", 0, &net.DNSError{Err: "timeout", IsTimeout: true}, d.VersionCheckTimeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := versionCheckConfig(toolx.V89)
			calls := 0
			svc, err := httpc.New(cfg, versionCheckLogger(), func(s *httpc.Service) {
				s.Client().Transport = versionCheckTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 3 {
						return versionCheckResponse(req, 200, `{"gatewayVersion":"8.9"}`), nil
					}
					if tc.err != nil {
						return nil, tc.err
					}
					response := versionCheckResponse(req, tc.status, `{}`)
					response.Header.Set("Retry-After", "0")
					return response, nil
				})
			})
			require.NoError(t, err)
			checker := NewVersionCheck(cfg, svc.Client(), versionCheckLogger(), nil)
			checker.BeforeRequest(svc.ToContext(context.Background()))
			got, _ := checker.Outcome()
			require.Equal(t, d.VersionCheckOutcome{Status: d.VersionCheckFailed, ConfiguredVersion: "8.9", FailureReason: tc.reason}, got)
			require.Equal(t, 1, calls)
			// A subsequent ordinary read still retries through the same shared transport.
			req, err := http.NewRequest(http.MethodGet, "https://isolated.invalid/v2/topology", nil)
			require.NoError(t, err)
			response, err := svc.Client().Do(req)
			require.NoError(t, err)
			require.Equal(t, 200, response.StatusCode)
			require.NoError(t, response.Body.Close())
			require.Equal(t, 3, calls)
		})
	}
}

// TestVersionCheckRecursiveBypass simulates the generated client's future pre-Do boundary without wiring adapters early.
func TestVersionCheckRecursiveBypass(t *testing.T) {
	t.Parallel()
	calls := 0
	inner := httpc.NewBeforeRequestClient(&http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		return versionCheckResponse(req, 200, `{"gatewayVersion":"8.9"}`), nil
	})})
	client := &http.Client{Timeout: 100 * time.Millisecond, Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) { return inner.Do(req) })}
	checker := NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), nil)
	ctx := httpc.WithBeforeRequestGate(context.Background(), checker)
	checker.BeforeRequest(ctx)
	got, complete := checker.Outcome()
	require.True(t, complete)
	require.Equal(t, d.VersionCheckMatched, got.Status)
	require.Equal(t, 1, calls)
}

// TestVersionCheckRequestRateIsolation gives probe admission its own evidence and preserves the parent's submitted attempt.
func TestVersionCheckRequestRateIsolation(t *testing.T) {
	t.Parallel()
	cfg := versionCheckConfig(toolx.V89)
	cfg.HTTP.RequestRate = 1
	calls := 0
	svc, err := httpc.New(cfg, versionCheckLogger(), func(s *httpc.Service) {
		s.Client().Transport = versionCheckTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			return versionCheckResponse(req, 200, `{}`), nil
		})
	})
	require.NoError(t, err)
	ctx, observation := httpc.BeginRequestRateAttempt(svc.ToContext(context.Background()))
	require.NotNil(t, observation)
	// Exhaust the shared limiter's initial slot; no sleep or external network is needed.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://isolated.invalid/mutation", nil)
	require.NoError(t, err)
	response, err := svc.Client().Do(req)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	svc.Client().Timeout = 20 * time.Millisecond
	checker := NewVersionCheck(cfg, svc.Client(), versionCheckLogger(), nil)
	checker.BeforeRequest(ctx)
	got, _ := checker.Outcome()
	require.Equal(t, d.VersionCheckTimeout, got.FailureReason)
	require.Equal(t, 1, calls, "probe must not dispatch without admission")
	require.NoError(t, observation.Failure(), "probe must not mark the submitted parent attempt as unsent")
	require.NoError(t, ctx.Err())
	checker.BeforeRequest(ctx)
	require.Equal(t, 1, calls)
}

// TestVersionCheckParentDeadline prevents dispatch after the probe exhausts the original caller's budget.
func TestVersionCheckParentDeadline(t *testing.T) {
	t.Parallel()
	calls := 0
	client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/v2/topology", req.URL.Path)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	checker := NewVersionCheck(versionCheckConfig(toolx.V89), client, versionCheckLogger(), nil)
	parent, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	ctx := httpc.WithBeforeRequestGate(parent, checker)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://isolated.invalid/operation", nil)
	require.NoError(t, err)
	response, err := httpc.NewBeforeRequestClient(client).Do(req)
	require.Nil(t, response)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, calls)
	got, _ := checker.Outcome()
	require.Equal(t, d.VersionCheckTimeout, got.FailureReason)
}

// TestVersionCheckTopologySnapshot retains safe operational facts without letting
// broker data affect compatibility or callers mutate the published snapshot.
func TestVersionCheckTopologySnapshot(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		t.Run(version.String(), func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(map[string]any{
				"gatewayVersion": version.String() + ".3", "clusterSize": 5,
				"partitionsCount": 12, "replicationFactor": 3,
				"brokers": []map[string]string{
					{"version": "8.9.3"}, {"version": " 8.10.2 "}, {"version": "8.9.3"},
					{"version": "8.9\nPOST https://private.invalid secret"}, {"version": ""},
				},
			})
			require.NoError(t, err)
			requests := 0
			client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				return versionCheckResponse(req, 200, string(body)), nil
			})}
			checker := NewVersionCheck(versionCheckConfig(version), client, versionCheckLogger(), func(outcome d.VersionCheckOutcome) {
				require.Equal(t, []string{"8.10.2", "8.9.3", "unknown"}, outcome.BrokerVersions)
				outcome.BrokerVersions[0] = "callback mutation"
			})
			checker.BeforeRequest(context.Background())
			checker.BeforeRequest(context.Background())
			outcome, complete := checker.Outcome()
			require.True(t, complete)
			require.Equal(t, 1, requests)
			require.Equal(t, d.VersionCheckMatched, outcome.Status)
			require.Equal(t, []string{"8.10.2", "8.9.3", "unknown"}, outcome.BrokerVersions)
			require.Equal(t, 5, outcome.BrokerCount)
			require.EqualValues(t, 5, outcome.ClusterSize)
			require.EqualValues(t, 12, outcome.PartitionsCount)
			require.EqualValues(t, 3, outcome.ReplicationFactor)
			outcome.BrokerVersions[0] = "reader mutation"
			again, _ := checker.Outcome()
			require.Equal(t, []string{"8.10.2", "8.9.3", "unknown"}, again.BrokerVersions)
		})
	}
}
