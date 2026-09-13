// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsRetryAttemptsPreserveMethodPolicy verifies each observed retry boundary gets one record without broadening retries to mutations.
func TestAPIDiagnosticsRetryAttemptsPreserveMethodPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		method       string
		results      []readRetryRoundTripResult
		wantCalls    int
		wantComplete []string
	}{
		{
			name:   "get retries and leaves discarded response incomplete",
			method: http.MethodGet,
			results: []readRetryRoundTripResult{
				{resp: newReadRetryResponse(http.StatusServiceUnavailable, "discarded")},
				{resp: newReadRetryResponse(http.StatusOK, "ok")},
			},
			wantCalls:    2,
			wantComplete: []string{"status=503", "response-bytes=0 response-complete=false", "status=200", "response-bytes=2 response-complete=true"},
		},
		{
			name:   "head retries as a bodyless read",
			method: http.MethodHead,
			results: []readRetryRoundTripResult{
				{resp: newReadRetryResponse(http.StatusGatewayTimeout, "")},
				{resp: newReadRetryResponse(http.StatusOK, "")},
			},
			wantCalls:    2,
			wantComplete: []string{"status=504", "status=200", "response-bytes=0 response-complete=true"},
		},
		{
			name:   "mutation remains a single higher-level attempt",
			method: http.MethodPost,
			results: []readRetryRoundTripResult{
				{resp: newReadRetryResponse(http.StatusServiceUnavailable, "not retried")},
				{resp: newReadRetryResponse(http.StatusOK, "unexpected")},
			},
			wantCalls:    1,
			wantComplete: []string{"status=503", "response-bytes=11 response-complete=true"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			sequence := &readRetrySequenceTransport{t: t, results: tt.results}
			transport := &ReadRetryTransport{
				base:   &DiagnosticsTransport{base: sequence, collector: newAPIDiagnosticCollector(t, &output)},
				policy: fastReadRetryPolicy(),
			}
			req := newReadRetryRequest(t, tt.method)

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			if tt.method != http.MethodHead {
				_, err = io.ReadAll(resp.Body)
				require.NoError(t, err)
			}
			require.NoError(t, resp.Body.Close())

			require.Equal(t, tt.wantCalls, sequence.Calls())
			require.Equal(t, tt.wantCalls, strings.Count(output.String(), "api #"))
			for _, fragment := range tt.wantComplete {
				require.Contains(t, output.String(), fragment)
			}
		})
	}
}

// TestAPIDiagnosticsRedirectsCountClientRoundTripBoundaries verifies net/http redirects retain one diagnostic record per public transport call.
func TestAPIDiagnosticsRedirectsCountClientRoundTripBoundaries(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.URL.Path == "/start" {
			http.Redirect(w, req, "/final", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "done")
	}))
	defer server.Close()

	client := &http.Client{Transport: &DiagnosticsTransport{
		base:      http.DefaultTransport,
		collector: newAPIDiagnosticCollector(t, &output),
	}}
	resp, err := client.Get(server.URL + "/start")
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, 2, requests)
	require.Equal(t, 2, strings.Count(output.String(), "api #"))
	require.Contains(t, output.String(), "api #1 GET /start: status=302")
	require.Contains(t, output.String(), "api #2 GET /final: status=200")
}
