// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package services

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
	defaultCamundaMutationRetryAttempts    = 8
	defaultCamundaMutationRetryBaseDelay   = 500 * time.Millisecond
	defaultCamundaMutationRetryMaxDelay    = 10 * time.Second
	defaultCamundaMutationRetryLogInterval = 5 * time.Second
)

var (
	camundaMutationRetryAttempts    = defaultCamundaMutationRetryAttempts
	camundaMutationRetryBaseDelay   = defaultCamundaMutationRetryBaseDelay
	camundaMutationRetryMaxDelay    = defaultCamundaMutationRetryMaxDelay
	camundaMutationRetryLogInterval = defaultCamundaMutationRetryLogInterval

	camundaMutationRetryLogMu   sync.Mutex
	camundaMutationRetryLastLog = map[string]time.Time{}
)

// RetryCamundaMutation retries Camunda write calls when the broker reports transient throttling.
func RetryCamundaMutation[T any](ctx context.Context, log *slog.Logger, operation string, fn func(context.Context) (T, *http.Response, []byte, error)) (T, error) {
	var zero T
	if fn == nil {
		return zero, errors.New("nil Camunda mutation retry function")
	}
	attempts := camundaMutationRetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		got, resp, body, err := fn(ctx)
		retry, reason := shouldRetryCamundaMutation(resp, body, err)
		if !retry {
			return got, err
		}
		lastErr = err
		if attempt == attempts {
			return got, err
		}
		delay := camundaMutationRetryDelay(attempt, resp)
		logCamundaMutationRetry(log, operation, reason, delay)
		if err := sleepCamundaMutationRetry(ctx, delay); err != nil {
			return zero, err
		}
	}
	return zero, lastErr
}

func shouldRetryCamundaMutation(resp *http.Response, body []byte, err error) (bool, string) {
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
	case http.StatusServiceUnavailable:
		if isResourceExhaustedBody(body) {
			return true, "resource exhausted"
		}
	}
	return false, ""
}

func isResourceExhaustedBody(body []byte) bool {
	return strings.Contains(strings.ToUpper(string(body)), "RESOURCE_EXHAUSTED")
}

func camundaMutationRetryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			return delay
		}
	}
	delay := camundaMutationRetryBaseDelay
	if delay <= 0 {
		delay = defaultCamundaMutationRetryBaseDelay
	}
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= camundaMutationRetryMaxDelay {
			delay = camundaMutationRetryMaxDelay
			break
		}
	}
	if camundaMutationRetryMaxDelay > 0 && delay > camundaMutationRetryMaxDelay {
		delay = camundaMutationRetryMaxDelay
	}
	if delay <= 0 {
		return 0
	}
	jitter := time.Duration(rand.Int63n(int64(delay/2) + 1))
	return delay + jitter
}

func parseRetryAfter(raw string) (time.Duration, bool) {
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

func logCamundaMutationRetry(log *slog.Logger, operation string, reason string, delay time.Duration) {
	if log == nil {
		return
	}
	now := time.Now()
	key := strings.TrimSpace(operation)
	if key == "" {
		key = "write request"
	}
	camundaMutationRetryLogMu.Lock()
	last := camundaMutationRetryLastLog[key]
	if camundaMutationRetryLogInterval > 0 && !last.IsZero() && now.Sub(last) < camundaMutationRetryLogInterval {
		camundaMutationRetryLogMu.Unlock()
		return
	}
	camundaMutationRetryLastLog[key] = now
	camundaMutationRetryLogMu.Unlock()
	if reason == "" {
		reason = "throttled"
	}
	log.Info(fmt.Sprintf("Camunda throttled %s; %s; retrying in %s", key, reason, delay.Round(time.Millisecond)))
}

func sleepCamundaMutationRetry(ctx context.Context, delay time.Duration) error {
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
