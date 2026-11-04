// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"net/http"
)

// BeforeRequestGate completes invocation preflight before the HTTP timeout starts.
// Implementations must honor caller cancellation and keep advisory errors internal.
type BeforeRequestGate interface {
	BeforeRequest(context.Context)
}

// beforeRequestKey and bypassBeforeRequestKey keep gate state scoped to the context.
type beforeRequestKey struct{}
type bypassBeforeRequestKey struct{}

// WithBeforeRequestGate shares preflight across operational clients in an invocation.
func WithBeforeRequestGate(ctx context.Context, gate BeforeRequestGate) context.Context {
	return context.WithValue(ctx, beforeRequestKey{}, gate)
}

// BypassBeforeRequest prevents the preflight's own requests from reentering its gate.
func BypassBeforeRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassBeforeRequestKey{}, true)
}

// BeforeRequestClient preserves the concrete client's semantics after preflight.
type BeforeRequestClient struct {
	client *http.Client
}

// NewBeforeRequestClient wraps Do, outside the delegated client's timeout boundary.
func NewBeforeRequestClient(client *http.Client) *BeforeRequestClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &BeforeRequestClient{client: client}
}

// Do consults the invocation gate without replacing the request, body, or context.
// An early canceled return owns closing the body, just as http.Client.Do would.
func (c *BeforeRequestClient) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return c.client.Do(req)
	}
	ctx := req.Context()
	gate, ok := ctx.Value(beforeRequestKey{}).(BeforeRequestGate)
	if !ok || ctx.Value(bypassBeforeRequestKey{}) == true {
		return c.client.Do(req)
	}
	if ctx.Err() == nil {
		gate.BeforeRequest(ctx)
	}
	if err := ctx.Err(); err != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, err
	}
	return c.client.Do(req)
}
