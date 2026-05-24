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

	req, err := http.NewRequest(http.MethodPost, "https://camunda.example.test/v2/process-instances/search", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"searching process instances"}, msgs)
}

func TestHTTPActivityMessageUsesEndpointLabelsWithoutHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		url    string
		want   string
	}{
		{name: "process instance deletion", method: http.MethodPost, url: "https://camunda.example.test/v2/process-instances/2251799813685250/deletion", want: "submitting process-instance deletion"},
		{name: "incident resolution", method: http.MethodPost, url: "https://camunda.example.test/v2/incidents/2251799813685251/resolution", want: "submitting incident resolution"},
		{name: "job update", method: http.MethodPatch, url: "https://camunda.example.test/v2/jobs/2251799813685252", want: "updating job"},
		{name: "fallback", method: http.MethodGet, url: "https://camunda.example.test/v2/unknown", want: "loading Camunda API data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, httpActivityMessage(req))
			require.NotContains(t, httpActivityMessage(req), "camunda.example.test")
		})
	}
}
