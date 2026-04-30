// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestLogTransport_StartsAndStopsActivityAroundRequest verifies HTTP waits are bracketed by activity calls.
func TestLogTransport_StartsAndStopsActivityAroundRequest(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	transport := &LogTransport{
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Activity: sink,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			time.Sleep(5 * time.Millisecond)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/v2/process-instances", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0], "waiting for GET camunda.example.test")
}
