// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// versionCheckGateFunc supplies controlled preflight behavior without cluster coupling.
type versionCheckGateFunc func(context.Context)

// BeforeRequest invokes the test's gate action synchronously.
func (f versionCheckGateFunc) BeforeRequest(ctx context.Context) { f(ctx) }

// versionCheckBody tracks ownership of a body that preflight must not consume.
type versionCheckBody struct {
	io.Reader
	closed bool
}

// Close records cleanup when cancellation prevents delegation.
func (b *versionCheckBody) Close() error { b.closed = true; return nil }

// TestVersionCheckBeforeRequest preserves the original request and delegates without a gate or through bypass.
func TestVersionCheckBeforeRequest(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"absent", "gated", "bypass"} {
		t.Run(mode, func(t *testing.T) {
			gateCalls := 0
			ctx := context.Background()
			if mode != "absent" {
				ctx = WithBeforeRequestGate(ctx, versionCheckGateFunc(func(context.Context) { gateCalls++ }))
			}
			if mode == "bypass" {
				ctx = BypassBeforeRequest(ctx)
			}
			body := &versionCheckBody{Reader: strings.NewReader("original body")}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://isolated.invalid/operation", body)
			require.NoError(t, err)
			calls := 0
			client := NewBeforeRequestClient(&http.Client{Transport: requestRateRoundTripFunc(func(got *http.Request) (*http.Response, error) {
				calls++
				require.Same(t, req, got)
				require.Same(t, body, got.Body)
				require.Equal(t, ctx, got.Context())
				require.False(t, body.closed)
				data, err := io.ReadAll(got.Body)
				require.NoError(t, err)
				require.Equal(t, "original body", string(data))
				_ = got.Body.Close()
				return newReadRetryResponse(200, "result"), nil
			})})
			response, err := client.Do(req)
			require.NoError(t, err)
			_ = response.Body.Close()
			require.Equal(t, 1, calls)
			wantGateCalls := 0
			if mode == "gated" {
				wantGateCalls = 1
			}
			require.Equal(t, wantGateCalls, gateCalls)
		})
	}
}

// TestVersionCheckCancellation closes unconsumed bodies and never sends canceled operations.
func TestVersionCheckCancellation(t *testing.T) {
	t.Parallel()
	for _, before := range []bool{true, false} {
		t.Run(map[bool]string{true: "before", false: "during"}[before], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			gateCalls := 0
			ctx = WithBeforeRequestGate(ctx, versionCheckGateFunc(func(context.Context) { gateCalls++; cancel() }))
			if before {
				cancel()
			}
			body := &versionCheckBody{Reader: strings.NewReader("unconsumed")}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://isolated.invalid/operation", body)
			require.NoError(t, err)
			client := NewBeforeRequestClient(&http.Client{Transport: requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Error("canceled operation dispatched")
				return nil, context.Canceled
			})})
			response, err := client.Do(req)
			require.Nil(t, response)
			require.ErrorIs(t, err, context.Canceled)
			require.True(t, body.closed)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			require.Equal(t, "unconsumed", string(data))
			if before {
				require.Zero(t, gateCalls)
			} else {
				require.Equal(t, 1, gateCalls)
			}
		})
	}
}

// TestVersionCheckTimeoutStartsAfterGate prevents preflight from consuming the operation's HTTP budget.
func TestVersionCheckTimeoutStartsAfterGate(t *testing.T) {
	t.Parallel()
	const timeout = 40 * time.Millisecond
	ctx := WithBeforeRequestGate(context.Background(), versionCheckGateFunc(func(ctx context.Context) {
		_, hasDeadline := ctx.Deadline()
		require.False(t, hasDeadline)
		time.Sleep(2 * timeout)
	}))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://isolated.invalid/operation", nil)
	require.NoError(t, err)
	client := NewBeforeRequestClient(&http.Client{Timeout: timeout, Transport: requestRateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		require.True(t, ok)
		require.Greater(t, time.Until(deadline), timeout/2)
		return newReadRetryResponse(200, "body"), nil
	})})
	response, err := client.Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "body", string(data))
	require.NoError(t, response.Body.Close())
	require.NoError(t, ctx.Err())
}

// TestVersionCheckDelegatedRedirects leaves redirects and response-body handling with the original client.
func TestVersionCheckDelegatedRedirects(t *testing.T) {
	t.Parallel()
	redirects, calls := 0, 0
	original := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { redirects++; return nil },
		Transport: requestRateRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path == "/first" {
				response := newReadRetryResponse(http.StatusFound, "")
				response.Header.Set("Location", "/second")
				return response, nil
			}
			return newReadRetryResponse(200, "final"), nil
		}),
	}
	req, err := http.NewRequest(http.MethodGet, "http://isolated.invalid/first", nil)
	require.NoError(t, err)
	response, err := NewBeforeRequestClient(original).Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "final", string(data))
	require.NoError(t, response.Body.Close())
	require.Equal(t, 1, redirects)
	require.Equal(t, 2, calls)
}
