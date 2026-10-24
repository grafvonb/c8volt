// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/stretchr/testify/require"
)

// completionResponseFixture holds only the HTTP evidence consumed by the shared submission path.
type completionResponseFixture struct {
	HTTPResponse *http.Response
	Body         []byte
}

// completionResponse supplies valid request metadata and immediate deterministic retries.
func completionResponse(code int) *completionResponseFixture {
	request, _ := http.NewRequest(http.MethodPost, "https://distinctive.invalid/v2/user-tasks/2251799813685249/completion", nil)
	return &completionResponseFixture{
		HTTPResponse: &http.Response{StatusCode: code, Request: request, Header: http.Header{"Retry-After": []string{"0"}}},
		Body:         []byte(`{"detail":"DISTINCTIVE_BODY RESOURCE_EXHAUSTED"}`),
	}
}

// completionSubmission runs shared behavior directly, without a generated client or version adapter.
func completionSubmission(submit func(context.Context) (*completionResponseFixture, error)) func(context.Context, string) (d.UserTaskCompletionResult, error) {
	return func(ctx context.Context, key string) (d.UserTaskCompletionResult, error) {
		return common.CompleteUserTaskSubmission(ctx, key, nil, func(ctx context.Context) (*http.Response, []byte, error) {
			response, err := submit(ctx)
			if response == nil {
				return nil, nil, err
			}
			return response.HTTPResponse, response.Body, err
		})
	}
}

// TestCompleteUserTaskFailures preserves classified machine evidence and the finite retry policy.
func TestCompleteUserTaskFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code       int
		cause      error
		submission d.UserTaskCompletionSubmission
		status     d.UserTaskCompletionStatus
		attempts   int
	}{
		{400, d.ErrBadRequest, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{401, d.ErrUnauthorized, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{403, d.ErrForbidden, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{404, d.ErrNotFound, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{409, d.ErrConflict, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{429, d.ErrRateLimited, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 8},
		{503, d.ErrUnavailable, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 8},
		{500, d.ErrInternal, d.UserTaskCompletionSubmissionUnknown, d.UserTaskCompletionStatusFailed, 1},
		{504, d.ErrGatewayTimeout, d.UserTaskCompletionSubmissionUnknown, d.UserTaskCompletionStatusTimedOut, 1},
	} {
		t.Run(fmt.Sprint(tc.code), func(t *testing.T) {
			calls := 0
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				return completionResponse(tc.code), nil
			})
			got, err := complete(context.Background(), "2251799813685249")
			require.ErrorIs(t, err, tc.cause)
			require.Equal(t, tc.attempts, calls)
			require.Equal(t, tc.status, got.Status)
			require.Equal(t, tc.submission, got.Submission)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, err.Error(), got.Error)
			require.Contains(t, got.Error, "DISTINCTIVE_BODY")
			require.Contains(t, got.Error, "POST")
			require.Contains(t, got.Error, "https://distinctive.invalid/")
		})
	}
}

// TestCompleteUserTaskMalformedAndTransport never interprets missing, unexpected or incomplete responses as acceptance.
func TestCompleteUserTaskMalformedAndTransport(t *testing.T) {
	t.Parallel()
	transportErr := errors.New("DISTINCTIVE_TRANSPORT")
	for _, tc := range []struct {
		name     string
		response *completionResponseFixture
		err      error
		cause    error
		code     int
	}{
		{"nil", nil, nil, d.ErrMalformedResponse, 0},
		{"no HTTP", &completionResponseFixture{}, nil, d.ErrMalformedResponse, 0},
		{"zero status", completionResponse(0), nil, d.ErrMalformedResponse, 0},
		{"unexpected 200", completionResponse(200), nil, d.ErrMalformedResponse, 200},
		{"unexpected 202", completionResponse(202), nil, d.ErrMalformedResponse, 202},
		{"transport", nil, transportErr, transportErr, 0},
		{"incomplete 204", completionResponse(204), transportErr, transportErr, 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				return tc.response, tc.err
			})
			got, err := complete(context.Background(), "2251799813685249")
			require.ErrorIs(t, err, tc.cause)
			require.Equal(t, 1, calls)
			require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
			require.Equal(t, tc.code, got.StatusCode)
		})
	}
}

// completionNetworkError exercises the retry helper's existing network policy without using network traffic.
type completionNetworkError struct{ timeout bool }

// Error includes sensitive markers that must never enter ordinary retry messages.
func (completionNetworkError) Error() string {
	return "POST https://distinctive.invalid/2251799813685249 DISTINCTIVE_BODY"
}

// Timeout distinguishes timeout classification from temporary connection failures.
func (e completionNetworkError) Timeout() bool { return e.timeout }

// Temporary permits the existing finite retry policy to handle the fixture.
func (completionNetworkError) Temporary() bool { return true }

// TestCompleteUserTaskRetryEvidence keeps earlier ambiguity until a later documented acceptance supersedes it.
func TestCompleteUserTaskRetryEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		first      int
		transport  error
		final      int
		submission d.UserTaskCompletionSubmission
	}{
		{"throttle then accepted", 429, nil, 204, d.UserTaskCompletionSubmissionAccepted},
		{"resource exhaustion then accepted", 503, nil, 204, d.UserTaskCompletionSubmissionAccepted},
		{"temporary then rejected", 0, completionNetworkError{}, 409, d.UserTaskCompletionSubmissionUnknown},
		{"timeout then accepted", 0, completionNetworkError{timeout: true}, 204, d.UserTaskCompletionSubmissionAccepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				if calls == 1 {
					return completionResponse(tc.first), tc.transport
				}
				return completionResponse(tc.final), nil
			})
			got, err := complete(context.Background(), "2251799813685249")
			if tc.final == 204 {
				require.NoError(t, err)
				require.Equal(t, d.UserTaskCompletionStatusAccepted, got.Status)
			} else {
				require.ErrorIs(t, err, d.ErrConflict)
				require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			}
			require.Equal(t, 2, calls)
			require.Equal(t, tc.final, got.StatusCode)
			require.Equal(t, tc.submission, got.Submission)
		})
	}
}

// TestCompleteUserTaskCancellation retains submission evidence and the last status even when retry backoff returns no response.
func TestCompleteUserTaskCancellation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		before         bool
		response       int
		transport      bool
		timeout        bool
		wantStatus     d.UserTaskCompletionStatus
		wantSubmission d.UserTaskCompletionSubmission
	}{
		{"before request", true, 0, false, false, d.UserTaskCompletionStatusNotAttempted, d.UserTaskCompletionSubmissionNotSubmitted},
		{"during request", false, 0, true, false, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionUnknown},
		{"after rejection", false, 429, false, false, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionRejected},
		{"after ambiguous timeout", false, 503, true, true, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionUnknown},
		{"accepted despite cancellation", false, 204, false, false, d.UserTaskCompletionStatusAccepted, d.UserTaskCompletionSubmissionAccepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.before {
				cancel()
			}
			calls := 0
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				cancel()
				if tc.timeout {
					return completionResponse(tc.response), context.DeadlineExceeded
				}
				if tc.transport {
					return nil, context.Canceled
				}
				return completionResponse(tc.response), nil
			})
			got, err := complete(ctx, "2251799813685249")
			if tc.response == 204 {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, context.Canceled)
			}
			require.Equal(t, tc.wantStatus, got.Status)
			require.Equal(t, tc.wantSubmission, got.Submission)
			require.Equal(t, tc.response, got.StatusCode)
			if tc.before {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
		})
	}
}

// TestCompleteUserTaskDeadlineBeforeSubmission performs no native request for an expired caller deadline.
func TestCompleteUserTaskDeadlineBeforeSubmission(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
		t.Fatal("must not submit")
		return nil, nil
	})
	got, err := complete(ctx, "2251799813685249")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, d.UserTaskCompletionStatusNotAttempted, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionNotSubmitted, got.Submission)
}

// TestCompleteUserTaskTimeoutAndExhaustion retains uncertainty and classifies exhausted temporary network errors accurately.
func TestCompleteUserTaskTimeoutAndExhaustion(t *testing.T) {
	t.Parallel()
	for _, timeout := range []bool{false, true} {
		t.Run(fmt.Sprint(timeout), func(t *testing.T) {
			calls := 0
			networkErr := completionNetworkError{timeout: timeout}
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				return completionResponse(0), networkErr
			})
			got, err := complete(context.Background(), "2251799813685249")
			require.ErrorIs(t, err, networkErr)
			require.Equal(t, 8, calls)
			require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
			if timeout {
				require.Equal(t, d.UserTaskCompletionStatusTimedOut, got.Status)
				require.ErrorIs(t, err, d.ErrGatewayTimeout)
			} else {
				require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			}
		})
	}
}

// TestCompleteUserTaskBackoffEvidence cancels after a response exists, before retry waiting can submit another request.
func TestCompleteUserTaskBackoffEvidence(t *testing.T) {
	t.Parallel()
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprint(ambiguous), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				if ambiguous && calls == 1 {
					return completionResponse(0), completionNetworkError{}
				}
				// Cancellation occurs after the backend response exists, so backoff must retain it.
				cancel()
				response := completionResponse(429)
				response.HTTPResponse.Header.Set("Retry-After", "3600")
				return response, nil
			})
			got, err := complete(ctx, "2251799813685249")
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, d.UserTaskCompletionStatusCanceled, got.Status)
			require.Equal(t, 429, got.StatusCode)
			if ambiguous {
				require.Equal(t, 2, calls)
				require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
			} else {
				require.Equal(t, 1, calls)
				require.Equal(t, d.UserTaskCompletionSubmissionRejected, got.Submission)
			}
		})
	}
}

// TestCompleteUserTaskDeadlineDuringBackoff retains explicit rejection when the caller's deadline interrupts retry waiting.
func TestCompleteUserTaskDeadlineDuringBackoff(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
		calls++
		response := completionResponse(429)
		response.HTTPResponse.Header.Set("Retry-After", "3600")
		return response, nil
	})
	got, err := complete(ctx, "2251799813685249")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, calls)
	require.Equal(t, d.UserTaskCompletionStatusTimedOut, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionRejected, got.Submission)
	require.Equal(t, 429, got.StatusCode)
}

// TestCompleteUserTaskLastStatusSurvivesTransportCancellation keeps the most recent actual HTTP evidence without treating it as final rejection.
func TestCompleteUserTaskLastStatusSurvivesTransportCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
		calls++
		if calls == 1 {
			return completionResponse(429), nil
		}
		cancel()
		return nil, context.Canceled
	})
	got, err := complete(ctx, "2251799813685249")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 2, calls)
	require.Equal(t, d.UserTaskCompletionStatusCanceled, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
	require.Equal(t, 429, got.StatusCode)
}

// TestCompleteUserTaskMalformedMetadataAndGeneric503 never accepts incomplete metadata or treats a proxy error as definitive rejection.
func TestCompleteUserTaskMalformedMetadataAndGeneric503(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"missing request", "missing URL", "generic 503", "proxy resource marker"} {
		t.Run(name, func(t *testing.T) {
			response := completionResponse(204)
			switch name {
			case "missing request":
				response.HTTPResponse.Request = nil
			case "missing URL":
				response.HTTPResponse.Request.URL = nil
			case "generic 503":
				response = completionResponse(503)
				response.Body = []byte(`{"detail":"proxy unavailable"}`)
			case "proxy resource marker":
				response = completionResponse(503)
				response.Body = []byte("proxy RESOURCE_EXHAUSTED page")
			}
			calls := 0
			complete := completionSubmission(func(context.Context) (*completionResponseFixture, error) {
				calls++
				return response, nil
			})
			got, err := complete(context.Background(), "2251799813685249")
			if response.HTTPResponse.StatusCode == 204 {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
			} else {
				require.ErrorIs(t, err, d.ErrUnavailable)
			}
			if name == "proxy resource marker" {
				require.Equal(t, 8, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
		})
	}
}
