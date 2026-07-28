// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// TestUnwrapLogTransportFindsLogTransportBehindReadRetry keeps activity wiring compatible with the future retry transport chain.
func TestUnwrapLogTransportFindsLogTransportBehindReadRetry(t *testing.T) {
	t.Parallel()

	logTransport := &LogTransport{}
	got := unwrapLogTransport(&ReadRetryTransport{
		base: &AuthTransport{base: logTransport},
	})

	require.Same(t, logTransport, got)
}
