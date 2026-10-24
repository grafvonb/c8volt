// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"net/http"

	"golang.org/x/time/rate"
)

// requestRateWaiter allows isolated tests to force HTTP timeout races at admission.
type requestRateWaiter interface{ Wait(context.Context) error }

// RequestRateTransport admits starts with one shared burst-one allowance. It
// never holds a lock over delegated I/O, so in-flight requests can overlap.
type RequestRateTransport struct {
	base    http.RoundTripper
	limiter requestRateWaiter
}

// newRequestRateTransport is used only for a validated positive rate.
func newRequestRateTransport(base http.RoundTripper, perSecond float64) *RequestRateTransport {
	return &RequestRateTransport{base: base, limiter: rate.NewLimiter(rate.Limit(perSecond), 1)}
}

// RoundTrip includes local waiting in the existing request deadline. A rejected
// admission is not a dispatched attempt and never enters HTTP diagnostics.
func (t *RequestRateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if err := t.limiter.Wait(ctx); err != nil {
		cause := ctx.Err()
		if cause == nil {
			// Burst is fixed at one, so a live context can only fail because its
			// remaining deadline is insufficient for the next admission.
			cause = context.DeadlineExceeded
		}
		return nil, recordRequestRateFailure(ctx, cause)
	}
	if err := ctx.Err(); err != nil {
		return nil, recordRequestRateFailure(ctx, err)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
