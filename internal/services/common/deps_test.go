// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/stretchr/testify/require"
)

// depsGate records whether preparation defers preflight until an actual request.
type depsGate func(context.Context)

// BeforeRequest invokes the test-owned gate.
func (f depsGate) BeforeRequest(ctx context.Context) { f(ctx) }

// depsTransport prevents requests from reaching any external service.
type depsTransport func(*http.Request) (*http.Response, error)

// RoundTrip handles the original request inside the isolated fixture.
func (f depsTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestPrepareServiceDeps preserves defaults and concrete identity while exposing lazy gated Do.
func TestPrepareServiceDeps(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{APIs: config.APIs{Camunda: config.API{BaseURL: "https://isolated.invalid"}}}
	_, err := PrepareServiceDeps(nil, nil, nil)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = PrepareServiceDeps(&config.Config{}, nil, nil)
	require.ErrorIs(t, err, ErrEmptyCamundaURL)
	defaults, err := PrepareServiceDeps(cfg, nil, nil)
	require.NoError(t, err)
	require.Same(t, http.DefaultClient, defaults.HTTPClient)
	require.Same(t, slog.Default(), defaults.Logger)
	require.NotNil(t, defaults.APIClient)

	calls, gates := 0, 0
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := &http.Client{Transport: depsTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok")), Request: req}, nil
	})}
	deps, err := PrepareServiceDeps(cfg, client, logger)
	require.NoError(t, err)
	require.Same(t, cfg, deps.Config)
	require.Same(t, client, deps.HTTPClient)
	require.Same(t, logger, deps.Logger)
	require.Zero(t, calls)
	ctx := httpc.WithBeforeRequestGate(context.Background(), depsGate(func(context.Context) { gates++ }))
	for _, requestContext := range []context.Context{context.Background(), ctx, httpc.BypassBeforeRequest(ctx)} {
		req, err := http.NewRequestWithContext(requestContext, http.MethodGet, cfg.APIs.Camunda.BaseURL, nil)
		require.NoError(t, err)
		resp, err := deps.APIClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	require.Equal(t, 3, calls)
	require.Equal(t, 1, gates)
}
