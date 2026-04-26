// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type fakeActivitySink struct {
	mu      sync.Mutex
	started int
	stopped int
	msgs    []string
}

func (s *fakeActivitySink) StartActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started++
	s.msgs = append(s.msgs, msg)
}

func (s *fakeActivitySink) StopActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped++
}

func TestLogTransport_StartsAndStopsActivityAroundRequest(t *testing.T) {
	t.Parallel()

	sink := &fakeActivitySink{}
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

	sink.mu.Lock()
	defer sink.mu.Unlock()
	require.Equal(t, 1, sink.started)
	require.Equal(t, 1, sink.stopped)
	require.Len(t, sink.msgs, 1)
	require.Contains(t, sink.msgs[0], "waiting for GET camunda.example.test")
}
