// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

type readRetryRoundTripResult struct {
	resp *http.Response
	err  error
}

type readRetrySequenceTransport struct {
	t       *testing.T
	results []readRetryRoundTripResult
	calls   int
}

// RoundTrip returns queued responses so retry tests can assert exact attempt counts.
func (t *readRetrySequenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.t.Helper()
	if t.calls >= len(t.results) {
		t.t.Fatalf("unexpected HTTP retry attempt %d", t.calls+1)
	}
	result := t.results[t.calls]
	t.calls++
	if result.resp != nil && result.resp.Request == nil {
		result.resp.Request = req
	}
	return result.resp, result.err
}

// Calls reports how many delegate round trips were attempted.
func (t *readRetrySequenceTransport) Calls() int {
	return t.calls
}

// fastReadRetryPolicy removes backoff cost while preserving multiple-attempt behavior in tests.
func fastReadRetryPolicy() readRetryPolicy {
	return readRetryPolicy{
		attempts:  3,
		baseDelay: time.Nanosecond,
		maxDelay:  time.Nanosecond,
		jitter:    false,
	}
}

// newReadRetryResponse creates a minimal HTTP response with a readable body for transport tests.
func newReadRetryResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// newReadRetryRequest creates a Camunda-shaped request so retry logs can reuse existing activity wording.
func newReadRetryRequest(t *testing.T, method string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, "https://camunda.example/v2/process-instances/123", nil)
	require.NoError(t, err)
	return req
}

// TestReadRetryTransportRetriesGETServerFailure verifies a transient GET status is hidden when a retry succeeds.
func TestReadRetryTransportRetriesGETServerFailure(t *testing.T) {
	t.Parallel()
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{resp: newReadRetryResponse(http.StatusInternalServerError, "transient")},
			{resp: newReadRetryResponse(http.StatusOK, "ok")},
		},
	}
	transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}

	resp, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodGet))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
	require.Equal(t, 2, seq.Calls())
}

// TestReadRetryTransportRetriesHEADUnavailable verifies safe HEAD reads use the same transient retry path.
func TestReadRetryTransportRetriesHEADUnavailable(t *testing.T) {
	t.Parallel()
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{resp: newReadRetryResponse(http.StatusServiceUnavailable, "")},
			{resp: newReadRetryResponse(http.StatusOK, "")},
		},
	}
	transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}

	resp, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodHead))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 2, seq.Calls())
}

// TestReadRetryTransportRetriesTemporaryNetworkError verifies retryable transport errors are recovered.
func TestReadRetryTransportRetriesTemporaryNetworkError(t *testing.T) {
	t.Parallel()
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{err: temporaryReadRetryError{}},
			{resp: newReadRetryResponse(http.StatusOK, "ok")},
		},
	}
	transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}

	resp, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodGet))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 2, seq.Calls())
}

// TestReadRetryTransportLogsCompactRetryLine keeps operator retry diagnostics in the established activity wording.
func TestReadRetryTransportLogsCompactRetryLine(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{resp: newReadRetryResponse(http.StatusInternalServerError, "transient")},
			{resp: newReadRetryResponse(http.StatusOK, "ok")},
		},
	}
	transport := &ReadRetryTransport{
		base:   seq,
		policy: fastReadRetryPolicy(),
		log:    slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo)),
	}

	resp, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodGet))

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	gotLog := logBuf.String()
	require.Contains(t, gotLog, "Camunda read failed loading process instance")
	require.Contains(t, gotLog, "500 Internal Server Error")
	require.Contains(t, gotLog, "retrying in")
}

// TestReadRetryTransportDoesNotRetrySemanticReadResponses preserves business outcomes that callers interpret directly.
func TestReadRetryTransportDoesNotRetrySemanticReadResponses(t *testing.T) {
	t.Parallel()
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
	}
	for _, status := range statuses {
		status := status
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			seq := &readRetrySequenceTransport{
				t: t,
				results: []readRetryRoundTripResult{
					{resp: newReadRetryResponse(status, "semantic failure")},
				},
			}
			transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}

			resp, err := transport.RoundTrip(newReadRetryRequest(t, http.MethodGet))

			require.NoError(t, err)
			require.Equal(t, status, resp.StatusCode)
			require.Equal(t, 1, seq.Calls())
		})
	}
}

// TestReadRetryTransportPreservesFinalExhaustedResponse keeps diagnostics readable after retry budget exhaustion.
func TestReadRetryTransportPreservesFinalExhaustedResponse(t *testing.T) {
	t.Parallel()
	finalResp := newReadRetryResponse(http.StatusInternalServerError, "final diagnostic")
	finalResp.Header.Set("X-Camunda-Diagnostic", "preserved")
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{resp: newReadRetryResponse(http.StatusInternalServerError, "first transient")},
			{resp: newReadRetryResponse(http.StatusBadGateway, "second transient")},
			{resp: finalResp},
		},
	}
	transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}
	req := newReadRetryRequest(t, http.MethodGet)

	resp, err := transport.RoundTrip(req)

	require.NoError(t, err)
	require.Same(t, finalResp, resp)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Same(t, req, resp.Request)
	require.Equal(t, "preserved", resp.Header.Get("X-Camunda-Diagnostic"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "final diagnostic", string(body))
	require.Equal(t, 3, seq.Calls())
}

// TestReadRetryTransportStopsRetrySleepOnContextCancel verifies cancellation interrupts backoff before another request is sent.
func TestReadRetryTransportStopsRetrySleepOnContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{resp: newReadRetryResponse(http.StatusServiceUnavailable, "retry later")},
			{resp: newReadRetryResponse(http.StatusOK, "unexpected")},
		},
	}
	transport := &ReadRetryTransport{
		base: seq,
		policy: readRetryPolicy{
			attempts:  3,
			baseDelay: 200 * time.Millisecond,
			maxDelay:  200 * time.Millisecond,
			jitter:    false,
		},
	}
	req := newReadRetryRequest(t, http.MethodGet).WithContext(ctx)

	time.AfterFunc(10*time.Millisecond, cancel)
	start := time.Now()
	resp, err := transport.RoundTrip(req)

	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, resp)
	require.Equal(t, 1, seq.Calls())
	require.Less(t, time.Since(start), 100*time.Millisecond)
}

// TestReadRetryTransportDoesNotRetryPOSTSearch keeps request-body search calls outside the generic read retry layer.
func TestReadRetryTransportDoesNotRetryPOSTSearch(t *testing.T) {
	t.Parallel()
	seq := &readRetrySequenceTransport{
		t: t,
		results: []readRetryRoundTripResult{
			{resp: newReadRetryResponse(http.StatusServiceUnavailable, "search temporarily unavailable")},
			{resp: newReadRetryResponse(http.StatusOK, "unexpected retry")},
		},
	}
	transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}
	req, err := http.NewRequest(http.MethodPost, "https://camunda.example/v2/process-instances/search", strings.NewReader(`{"filter":{}}`))
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.Equal(t, 1, seq.Calls())
}

// TestReadRetryTransportDoesNotRetryUnsafeMethods protects mutation-style requests from generic replay.
func TestReadRetryTransportDoesNotRetryUnsafeMethods(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		url    string
	}{
		{
			name:   "delete",
			method: http.MethodDelete,
			url:    "https://camunda.example/v2/process-instances/123",
		},
		{
			name:   "patch",
			method: http.MethodPatch,
			url:    "https://camunda.example/v2/jobs/456",
		},
		{
			name:   "put",
			method: http.MethodPut,
			url:    "https://camunda.example/v2/resources/789",
		},
		{
			name:   "non-search post",
			method: http.MethodPost,
			url:    "https://camunda.example/v2/process-instances/123/cancellation",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seq := &readRetrySequenceTransport{
				t: t,
				results: []readRetryRoundTripResult{
					{resp: newReadRetryResponse(http.StatusInternalServerError, "mutation transient")},
					{resp: newReadRetryResponse(http.StatusOK, "unexpected retry")},
				},
			}
			transport := &ReadRetryTransport{base: seq, policy: fastReadRetryPolicy()}
			req, err := http.NewRequest(tt.method, tt.url, strings.NewReader(`{}`))
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)

			require.NoError(t, err)
			require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			require.Equal(t, 1, seq.Calls())
		})
	}
}

type temporaryReadRetryError struct{}

func (temporaryReadRetryError) Error() string   { return "temporary" }
func (temporaryReadRetryError) Timeout() bool   { return false }
func (temporaryReadRetryError) Temporary() bool { return true }
func (temporaryReadRetryError) Unwrap() error   { return errors.New("temporary") }

// TestUnwrapLogTransportFindsLogTransportBehindReadRetry keeps activity wiring compatible with the future retry transport chain.
func TestUnwrapLogTransportFindsLogTransportBehindReadRetry(t *testing.T) {
	t.Parallel()

	logTransport := &LogTransport{}
	got := unwrapLogTransport(&ReadRetryTransport{
		base: &AuthTransport{base: logTransport},
	})

	require.Same(t, logTransport, got)
}

// TestNewInstallsReadRetryTransport verifies the shared service client path owns Camunda read retries.
func TestNewInstallsReadRetryTransport(t *testing.T) {
	t.Parallel()

	svc, err := New(&config.Config{HTTP: config.HTTP{Timeout: "30s"}}, nil)

	require.NoError(t, err)
	_, ok := svc.Client().Transport.(*ReadRetryTransport)
	require.True(t, ok)
	require.NotNil(t, unwrapLogTransport(svc.Client().Transport))
}
