// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
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
