// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/stretchr/testify/require"
)

// assignmentSubmitter keeps both action wrappers subject to the same evidence contract.
type assignmentSubmitter func(context.Context, string, *slog.Logger, func(context.Context) (*http.Response, []byte, error)) (d.UserTaskAssignmentResult, error)

// assignmentStep supplies a response or a transport error without network access.
type assignmentStep struct {
	code      int
	body      string
	err       error
	malformed string
	cancel    bool
	backoff   bool
}

// TestAssignUserTaskSubmissionEvidence covers both wrappers and all meaningful retry/evidence transitions once.
func TestAssignUserTaskSubmissionEvidence(t *testing.T) {
	t.Parallel()
	transport := errors.New("POST https://distinctive.invalid/2251799813685249 DISTINCTIVE_BODY")
	for action, submit := range map[string]assignmentSubmitter{"assign": common.AssignUserTaskSubmission, "unassign": common.UnassignUserTaskSubmission} {
		t.Run(action, func(t *testing.T) {
			for _, tc := range []struct {
				name       string
				steps      []assignmentStep
				status     d.UserTaskAssignmentStatus
				submission d.UserTaskAssignmentSubmission
				cause      error
				code       int
				calls      int
				before     bool
				deadline   bool
			}{
				{name: "accepted", steps: []assignmentStep{{code: 204}}, status: "accepted", submission: "accepted", code: 204, calls: 1},
				{name: "nil response", steps: []assignmentStep{{malformed: "nil"}}, status: "failed", submission: "unknown", cause: d.ErrMalformedResponse, calls: 1},
				{name: "missing request", steps: []assignmentStep{{code: 204, malformed: "request"}}, status: "failed", submission: "unknown", cause: d.ErrMalformedResponse, code: 204, calls: 1},
				{name: "missing URL", steps: []assignmentStep{{code: 204, malformed: "url"}}, status: "failed", submission: "unknown", cause: d.ErrMalformedResponse, code: 204, calls: 1},
				{name: "zero status", steps: []assignmentStep{{code: 0}}, status: "failed", submission: "unknown", cause: d.ErrMalformedResponse, calls: 1},
				{name: "invalid status", steps: []assignmentStep{{code: 600}}, status: "failed", submission: "unknown", cause: d.ErrMalformedResponse, code: 600, calls: 1},
				{name: "undocumented success", steps: []assignmentStep{{code: 202}}, status: "failed", submission: "unknown", cause: d.ErrMalformedResponse, code: 202, calls: 1},
				{name: "partial success", steps: []assignmentStep{{code: 204, err: transport}}, status: "failed", submission: "unknown", cause: transport, code: 204, calls: 1},
				{name: "transport", steps: []assignmentStep{{malformed: "nil", err: transport}}, status: "failed", submission: "unknown", cause: transport, calls: 1},
				{name: "generic 500", steps: []assignmentStep{{code: 500}}, status: "failed", submission: "unknown", cause: d.ErrInternal, code: 500, calls: 1},
				{name: "generic 503", steps: []assignmentStep{{code: 503}}, status: "failed", submission: "unknown", cause: d.ErrUnavailable, code: 503, calls: 1},
				{name: "generic 504", steps: []assignmentStep{{code: 504}}, status: "timed_out", submission: "unknown", cause: d.ErrGatewayTimeout, code: 504, calls: 1},
				{name: "rate limit exhausted", steps: []assignmentStep{{code: 429}}, status: "failed", submission: "rejected", cause: d.ErrRateLimited, code: 429, calls: 8},
				{name: "resource exhausted", steps: []assignmentStep{{code: 503, body: `{"detail":"RESOURCE_EXHAUSTED"}`}}, status: "failed", submission: "rejected", cause: d.ErrUnavailable, code: 503, calls: 8},
				{name: "proxy marker is ambiguous", steps: []assignmentStep{{code: 503, body: `proxy RESOURCE_EXHAUSTED`}}, status: "failed", submission: "unknown", cause: d.ErrUnavailable, code: 503, calls: 8},
				{name: "throttle then success", steps: []assignmentStep{{code: 429}, {code: 204}}, status: "accepted", submission: "accepted", code: 204, calls: 2},
				{name: "exhaustion then success", steps: []assignmentStep{{code: 503, body: `{"detail":"RESOURCE_EXHAUSTED"}`}, {code: 204}}, status: "accepted", submission: "accepted", code: 204, calls: 2},
				{name: "ambiguous then rejection", steps: []assignmentStep{{err: completionNetworkError{}}, {code: 409}}, status: "failed", submission: "unknown", cause: d.ErrConflict, code: 409, calls: 2},
				{name: "ambiguous then success", steps: []assignmentStep{{err: completionNetworkError{timeout: true}}, {code: 204}}, status: "accepted", submission: "accepted", code: 204, calls: 2},
				{name: "rejection then 504", steps: []assignmentStep{{code: 429}, {code: 504}}, status: "timed_out", submission: "unknown", cause: d.ErrGatewayTimeout, code: 504, calls: 2},
				{name: "rejection then network timeout exhaustion", steps: []assignmentStep{{code: 429}, {err: completionNetworkError{timeout: true}}}, status: "timed_out", submission: "unknown", cause: d.ErrGatewayTimeout, code: 429, calls: 8},
				{name: "rejection then timeout", steps: []assignmentStep{{code: 429}, {malformed: "nil", err: context.DeadlineExceeded, cancel: true}}, status: "canceled", submission: "unknown", cause: context.Canceled, code: 429, calls: 2},
				{name: "temporary exhausted", steps: []assignmentStep{{err: completionNetworkError{}}}, status: "failed", submission: "unknown", cause: completionNetworkError{}, calls: 8},
				{name: "timeout exhausted", steps: []assignmentStep{{err: completionNetworkError{timeout: true}}}, status: "timed_out", submission: "unknown", cause: d.ErrGatewayTimeout, calls: 8},
				{name: "before cancellation", before: true, status: "not_attempted", submission: "not_submitted", cause: context.Canceled},
				{name: "before deadline", before: true, deadline: true, status: "not_attempted", submission: "not_submitted", cause: context.DeadlineExceeded},
				{name: "inflight cancellation", steps: []assignmentStep{{malformed: "nil", err: context.Canceled, cancel: true}}, status: "canceled", submission: "unknown", cause: context.Canceled, calls: 1},
				{name: "rejection then inflight cancellation", steps: []assignmentStep{{code: 429}, {malformed: "nil", err: context.Canceled, cancel: true}}, status: "canceled", submission: "unknown", cause: context.Canceled, code: 429, calls: 2},
				{name: "cancel during rejection backoff", steps: []assignmentStep{{code: 429, cancel: true, backoff: true}}, status: "canceled", submission: "rejected", cause: context.Canceled, code: 429, calls: 1},
				{name: "deadline during rejection backoff", deadline: true, steps: []assignmentStep{{code: 429, backoff: true}}, status: "timed_out", submission: "rejected", cause: context.DeadlineExceeded, code: 429, calls: 1},
				{name: "ambiguity survives backoff cancellation", steps: []assignmentStep{{err: completionNetworkError{}}, {code: 429, cancel: true, backoff: true}}, status: "canceled", submission: "unknown", cause: context.Canceled, code: 429, calls: 2},
				{name: "accepted despite cancellation", steps: []assignmentStep{{code: 204, cancel: true}}, status: "accepted", submission: "accepted", code: 204, calls: 1},
			} {
				t.Run(tc.name, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					if tc.deadline {
						timeout := 100 * time.Millisecond
						if tc.before {
							timeout = -time.Second
						}
						var stop context.CancelFunc
						ctx, stop = context.WithTimeout(ctx, timeout)
						defer stop()
					} else if tc.before {
						cancel()
					}
					calls := 0
					got, err := submit(ctx, "2251799813685249", nil, func(context.Context) (*http.Response, []byte, error) {
						i := calls
						calls++
						if i >= len(tc.steps) {
							i = len(tc.steps) - 1
						}
						require.GreaterOrEqual(t, i, 0, "unexpected submission")
						step := tc.steps[i]
						if step.cancel {
							cancel()
						}
						if step.malformed == "nil" {
							return nil, nil, step.err
						}
						response := completionResponse(step.code).HTTPResponse
						if step.malformed == "request" {
							response.Request = nil
						}
						if step.malformed == "url" {
							response.Request.URL = nil
						}
						if step.backoff {
							response.Header.Set("Retry-After", "3600")
						}
						return response, []byte(step.body), step.err
					})
					require.Equal(t, tc.calls, calls)
					require.Equal(t, "2251799813685249", got.Key)
					require.Equal(t, tc.status, got.Status)
					require.Equal(t, tc.submission, got.Submission)
					require.Equal(t, tc.code, got.StatusCode)
					if tc.cause == nil {
						require.NoError(t, err)
						require.Empty(t, got.Error)
					} else {
						require.ErrorIs(t, err, tc.cause)
						require.Equal(t, err.Error(), got.Error)
						if !tc.before {
							require.Contains(t, err.Error(), action+" user task:")
						}
					}
				})
			}
		})
	}
}

// TestUnassignUserTaskRejections retains native rejection classes and machine evidence for either action.
func TestUnassignUserTaskRejections(t *testing.T) {
	t.Parallel()
	for action, submit := range map[string]assignmentSubmitter{"assign": common.AssignUserTaskSubmission, "unassign": common.UnassignUserTaskSubmission} {
		for code, cause := range map[int]error{400: d.ErrBadRequest, 401: d.ErrUnauthorized, 403: d.ErrForbidden, 404: d.ErrNotFound, 409: d.ErrConflict} {
			t.Run(fmt.Sprintf("%s/%d", action, code), func(t *testing.T) {
				calls := 0
				got, err := submit(context.Background(), "2251799813685249", nil, func(context.Context) (*http.Response, []byte, error) {
					calls++
					response := completionResponse(code)
					return response.HTTPResponse, response.Body, nil
				})
				require.ErrorIs(t, err, cause)
				require.Equal(t, 1, calls)
				require.Equal(t, d.UserTaskAssignmentStatusFailed, got.Status)
				require.Equal(t, d.UserTaskAssignmentSubmissionRejected, got.Submission)
				require.Equal(t, code, got.StatusCode)
				for _, marker := range []string{"POST", "https://distinctive.invalid/", "DISTINCTIVE_BODY"} {
					require.Contains(t, got.Error, marker)
				}
			})
		}
	}
}
