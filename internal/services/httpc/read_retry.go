// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultCamundaReadRetryAttempts  = 4
	defaultCamundaReadRetryBaseDelay = 500 * time.Millisecond
	defaultCamundaReadRetryMaxDelay  = 10 * time.Second
	defaultCamundaReadRetryJitter    = true
	defaultCamundaReadRetryLogEvery  = 5 * time.Second
)

type readRetryPolicy struct {
	attempts    int
	baseDelay   time.Duration
	maxDelay    time.Duration
	jitter      bool
	logInterval time.Duration
}

// ReadRetryTransport owns bounded retry mechanics for safe Camunda read requests.
type ReadRetryTransport struct {
	base       http.RoundTripper
	policy     readRetryPolicy
	log        *slog.Logger
	logMu      sync.Mutex
	lastLogFor map[string]time.Time
}

// RoundTrip retries safe Camunda read requests after transient transport or server failures.
func (t *ReadRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || !isReadRetryMethod(req.Method) {
		return t.rt().RoundTrip(req)
	}
	policy := normalizeReadRetryPolicy(t.policy)
	var lastResp *http.Response
	var lastErr error
	for attempt := 1; attempt <= policy.attempts; attempt++ {
		resp, err := t.rt().RoundTrip(req)
		retry, reason := shouldRetryReadResult(req.Context(), resp, err)
		if !retry {
			return resp, err
		}
		lastResp, lastErr = resp, err
		if attempt == policy.attempts {
			return resp, err
		}
		delay := readRetryDelay(attempt, resp, policy)
		t.logRetry(req, responseRetryReason(resp, reason), delay, policy)
		closeRetryResponse(resp)
		if err := sleepReadRetry(req.Context(), delay); err != nil {
			return nil, err
		}
	}
	return lastResp, lastErr
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
		attempts:    defaultCamundaReadRetryAttempts,
		baseDelay:   defaultCamundaReadRetryBaseDelay,
		maxDelay:    defaultCamundaReadRetryMaxDelay,
		jitter:      defaultCamundaReadRetryJitter,
		logInterval: defaultCamundaReadRetryLogEvery,
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
	if policy.logInterval < 0 {
		policy.logInterval = defaults.logInterval
	}
	return policy
}

func isReadRetryMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

func shouldRetryReadResult(ctx context.Context, resp *http.Response, err error) (bool, string) {
	if ctx != nil && ctx.Err() != nil {
		return false, ""
	}
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			return true, "temporary network error"
		}
		return false, ""
	}
	if resp == nil {
		return false, ""
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return true, "rate limited"
	case http.StatusInternalServerError:
		return true, "internal server error"
	case http.StatusBadGateway:
		return true, "bad gateway"
	case http.StatusServiceUnavailable:
		return true, "unavailable"
	case http.StatusGatewayTimeout:
		return true, "gateway timeout"
	default:
		return false, ""
	}
}

func responseRetryReason(resp *http.Response, fallback string) string {
	if resp == nil {
		return fallback
	}
	if strings.TrimSpace(resp.Status) != "" {
		return resp.Status
	}
	if text := http.StatusText(resp.StatusCode); text != "" {
		return fmt.Sprintf("%d %s", resp.StatusCode, text)
	}
	return fallback
}

func closeRetryResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_ = resp.Body.Close()
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

func (t *ReadRetryTransport) logRetry(req *http.Request, reason string, delay time.Duration, policy readRetryPolicy) {
	if t.log == nil {
		return
	}
	operation := strings.TrimSpace(httpActivityMessage(req))
	if operation == "" {
		operation = "loading Camunda API data"
	}
	if t.suppressRetryLog(operation, policy.logInterval) {
		return
	}
	if reason == "" {
		reason = "transient failure"
	}
	t.log.Info(fmt.Sprintf("Camunda read failed %s; %s; retrying in %s", operation, reason, delay.Round(time.Millisecond)))
}

func (t *ReadRetryTransport) suppressRetryLog(operation string, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}
	now := time.Now()
	t.logMu.Lock()
	defer t.logMu.Unlock()
	if t.lastLogFor == nil {
		t.lastLogFor = map[string]time.Time{}
	}
	last := t.lastLogFor[operation]
	if !last.IsZero() && now.Sub(last) < interval {
		return true
	}
	t.lastLogFor[operation] = now
	return false
}
