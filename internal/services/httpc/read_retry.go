// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultCamundaReadRetryAttempts  = 4
	defaultCamundaReadRetryBaseDelay = 500 * time.Millisecond
	defaultCamundaReadRetryMaxDelay  = 10 * time.Second
	defaultCamundaReadRetryJitter    = true
)

type readRetryPolicy struct {
	attempts  int
	baseDelay time.Duration
	maxDelay  time.Duration
	jitter    bool
}

// ReadRetryTransport owns bounded retry mechanics for safe Camunda read requests.
type ReadRetryTransport struct {
	base   http.RoundTripper
	policy readRetryPolicy
}

// RoundTrip currently preserves normal transport behavior while the retry policy surface is introduced.
func (t *ReadRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.rt().RoundTrip(req)
}

// rt returns the configured delegate or the process default transport.
func (t *ReadRetryTransport) rt() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}

// defaultReadRetryPolicy centralizes the bounded GET/HEAD retry timing defaults.
func defaultReadRetryPolicy() readRetryPolicy {
	return readRetryPolicy{
		attempts:  defaultCamundaReadRetryAttempts,
		baseDelay: defaultCamundaReadRetryBaseDelay,
		maxDelay:  defaultCamundaReadRetryMaxDelay,
		jitter:    defaultCamundaReadRetryJitter,
	}
}

// normalizeReadRetryPolicy keeps invalid test or caller policy values from disabling the finite retry budget.
func normalizeReadRetryPolicy(policy readRetryPolicy) readRetryPolicy {
	defaults := defaultReadRetryPolicy()
	if policy.attempts < 1 {
		policy.attempts = defaults.attempts
	}
	if policy.baseDelay < 0 {
		policy.baseDelay = defaults.baseDelay
	}
	if policy.maxDelay < 0 {
		policy.maxDelay = defaults.maxDelay
	}
	return policy
}

// readRetryDelay applies Retry-After first, then exponential backoff with optional jitter.
func readRetryDelay(attempt int, resp *http.Response, policy readRetryPolicy) time.Duration {
	policy = normalizeReadRetryPolicy(policy)
	if resp != nil {
		if delay, ok := parseReadRetryAfter(resp.Header.Get("Retry-After")); ok {
			return delay
		}
	}
	delay := policy.baseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if policy.maxDelay > 0 && delay >= policy.maxDelay {
			delay = policy.maxDelay
			break
		}
	}
	if policy.maxDelay > 0 && delay > policy.maxDelay {
		delay = policy.maxDelay
	}
	if delay <= 0 || !policy.jitter {
		return delay
	}
	return delay + time.Duration(rand.Int63n(int64(delay/2)+1))
}

// parseReadRetryAfter accepts both seconds and HTTP-date Retry-After header values.
func parseReadRetryAfter(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds < 0 {
			seconds = 0
		}
		return time.Duration(seconds) * time.Second, true
	}
	when, err := http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	delay := time.Until(when)
	if delay < 0 {
		delay = 0
	}
	return delay, true
}

// sleepReadRetry waits for the retry delay while honoring request cancellation.
func sleepReadRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
