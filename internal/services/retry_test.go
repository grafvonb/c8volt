// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package services

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRetryCamundaMutationRetriesResourceExhaustion verifies Camunda broker throttling is retried before the caller maps HTTP status errors.
func TestRetryCamundaMutationRetriesResourceExhaustion(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	got, err := RetryCamundaMutation(context.Background(), slog.Default(), "create pi", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		if calls == 1 {
			return "", &http.Response{StatusCode: http.StatusServiceUnavailable}, []byte(`{"title":"RESOURCE_EXHAUSTED"}`), nil
		}
		return "ok", &http.Response{StatusCode: http.StatusOK}, nil, nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.Equal(t, 2, calls)
}

// TestRetryCamundaMutationDoesNotRetryConflicts keeps functional command errors immediate.
func TestRetryCamundaMutationDoesNotRetryConflicts(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	_, err := RetryCamundaMutation(context.Background(), slog.Default(), "delete pi", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		return "", &http.Response{StatusCode: http.StatusConflict}, []byte(`{"title":"CONFLICT"}`), nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

// TestRetryCamundaMutationRespectsContext verifies backoff sleep exits promptly on cancellation.
func TestRetryCamundaMutationRespectsContext(t *testing.T) {
	oldAttempts := camundaMutationRetryAttempts
	oldBaseDelay := camundaMutationRetryBaseDelay
	oldMaxDelay := camundaMutationRetryMaxDelay
	camundaMutationRetryAttempts = 3
	camundaMutationRetryBaseDelay = time.Minute
	camundaMutationRetryMaxDelay = time.Minute
	t.Cleanup(func() {
		camundaMutationRetryAttempts = oldAttempts
		camundaMutationRetryBaseDelay = oldBaseDelay
		camundaMutationRetryMaxDelay = oldMaxDelay
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RetryCamundaMutation(ctx, slog.Default(), "delete pi", func(context.Context) (string, *http.Response, []byte, error) {
		return "", &http.Response{StatusCode: http.StatusTooManyRequests}, nil, nil
	})

	require.ErrorIs(t, err, context.Canceled)
}

// TestRetryCamundaMutationStopsAfterBudget verifies persistent throttling returns the last response for normal HTTP error mapping.
func TestRetryCamundaMutationStopsAfterBudget(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	got, err := RetryCamundaMutation(context.Background(), slog.Default(), "delete pi", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		return "last", &http.Response{StatusCode: http.StatusServiceUnavailable}, []byte(`{"title":"RESOURCE_EXHAUSTED"}`), nil
	})

	require.NoError(t, err)
	require.Equal(t, "last", got)
	require.Equal(t, camundaMutationRetryAttempts, calls)
}

// TestRetryCamundaMutationRetriesTemporaryNetworkErrors covers retryable transport failures.
func TestRetryCamundaMutationRetriesTemporaryNetworkErrors(t *testing.T) {
	withFastCamundaMutationRetry(t)
	calls := 0

	got, err := RetryCamundaMutation(context.Background(), slog.Default(), "deploy", func(context.Context) (string, *http.Response, []byte, error) {
		calls++
		if calls == 1 {
			return "", nil, nil, temporaryNetError{}
		}
		return "ok", &http.Response{StatusCode: http.StatusOK}, nil, nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.Equal(t, 2, calls)
}

type temporaryNetError struct{}

func (temporaryNetError) Error() string   { return "temporary" }
func (temporaryNetError) Timeout() bool   { return false }
func (temporaryNetError) Temporary() bool { return true }
func (temporaryNetError) Unwrap() error   { return errors.New("temporary") }

func withFastCamundaMutationRetry(t *testing.T) {
	t.Helper()
	oldAttempts := camundaMutationRetryAttempts
	oldBaseDelay := camundaMutationRetryBaseDelay
	oldMaxDelay := camundaMutationRetryMaxDelay
	oldLogInterval := camundaMutationRetryLogInterval
	camundaMutationRetryAttempts = 3
	camundaMutationRetryBaseDelay = time.Nanosecond
	camundaMutationRetryMaxDelay = time.Nanosecond
	camundaMutationRetryLogInterval = 0
	t.Cleanup(func() {
		camundaMutationRetryAttempts = oldAttempts
		camundaMutationRetryBaseDelay = oldBaseDelay
		camundaMutationRetryMaxDelay = oldMaxDelay
		camundaMutationRetryLogInterval = oldLogInterval
	})
}
