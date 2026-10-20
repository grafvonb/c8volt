// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestBulkCompletionReconciliationEvidence retains original submission facts across completed, missing and terminal observations.
func TestBulkCompletionReconciliationEvidence(t *testing.T) {
	t.Parallel()
	for _, submission := range []struct {
		name   string
		status d.CompletionSubmissionStatus
		http   int
		err    error
	}{
		{"not found", d.CompletionSubmissionRejected, 404, d.ErrNotFound},
		{"conflict", d.CompletionSubmissionRejected, 409, d.ErrConflict},
		{"transport", d.CompletionSubmissionUncertain, 0, errors.New("connection lost")},
		{"server", d.CompletionSubmissionUncertain, 502, d.ErrUpstream},
	} {
		for _, state := range []string{"COMPLETED", "MISSING", "CANCELED", "ERROR_THROWN", "CREATED", "UNKNOWN", "UNREADABLE", "WRONG_KEY"} {
			t.Run(submission.name+"/"+state, func(t *testing.T) {
				b := preparedForExecution(t, 1)
				reads, writes := 0, 0
				evidence := d.CompletionSubmission{Status: submission.status, Diagnostic: submission.err.Error()}
				if submission.http != 0 {
					status := submission.http
					evidence.HTTPStatus = &status
				}
				ports := ExecutionPorts{Get: func(_ context.Context, key string, opts ...services.CallOption) (d.Job, error) {
					reads++
					require.True(t, services.ApplyCallOptions(opts).NoWait)
					if reads == 1 {
						return d.Job{Key: key, State: "CREATED"}, nil
					}
					switch state {
					case "MISSING":
						return d.Job{}, d.ErrNotFound
					case "UNREADABLE":
						return d.Job{}, d.ErrUnavailable
					case "WRONG_KEY":
						return d.Job{Key: "999", State: "COMPLETED"}, nil
					}
					return d.Job{Key: key, State: state}, nil
				}, Submit: func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
					writes++
					return evidence, submission.err
				}}
				r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 1, services.WithNoWait())
				require.Equal(t, 1, writes)
				require.Equal(t, 2, reads)
				assertExecutionCounts(t, b, r)
				out := r.Outcomes[0]
				require.Equal(t, evidence, out.Submission)
				require.Equal(t, d.CompletionPhaseReconciliation, out.Observation.Phase)
				if state == "COMPLETED" {
					require.NoError(t, err)
					require.Equal(t, d.CompletionDispositionCompleted, out.Disposition)
					require.Empty(t, out.Reason)
				} else {
					require.Error(t, err)
					require.ErrorIs(t, err, submission.err)
					require.Equal(t, d.CompletionDispositionUnsatisfied, out.Disposition)
					if state == "MISSING" {
						require.Equal(t, d.CompletionObservationMissing, out.Observation.Status)
						require.Equal(t, "CREATED", out.Observation.State)
					}
					if state == "WRONG_KEY" {
						require.Equal(t, d.CompletionObservationUnreadable, out.Observation.Status)
						require.ErrorIs(t, err, d.ErrMalformedResponse)
					}
					if state == "CANCELED" || state == "ERROR_THROWN" {
						require.Equal(t, d.CompletionOutcomeOtherTerminal, out.Reason)
					} else if submission.status == d.CompletionSubmissionUncertain {
						require.Equal(t, d.CompletionOutcomeUncertain, out.Reason)
					} else {
						require.Equal(t, d.CompletionOutcomeRejected, out.Reason)
					}
				}
			})
		}
	}
}

// TestBulkCompletionReconciliationPolling retries observations only, retains stale visibility, and respects all configured bounds.
func TestBulkCompletionReconciliationPolling(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"eventual", "max retries", "timeout", "caller deadline", "caller cancellation", "cancel after submission"} {
		t.Run(mode, func(t *testing.T) {
			b := preparedForExecution(t, 1)
			reads, writes := 0, 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "caller deadline" {
				var timeoutCancel context.CancelFunc
				ctx, timeoutCancel = context.WithTimeout(ctx, 30*time.Millisecond)
				defer timeoutCancel()
			}
			backoff := config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Millisecond, Timeout: time.Second}
			if mode == "max retries" {
				backoff.MaxRetries = 2
			}
			if mode == "timeout" {
				backoff.Timeout = 20 * time.Millisecond
			}
			ports := ExecutionPorts{Get: func(readCtx context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				reads++
				if reads == 1 {
					return d.Job{Key: key, State: "CREATED"}, nil
				}
				_, hasDeadline := readCtx.Deadline()
				require.True(t, hasDeadline)
				if mode == "eventual" && reads == 4 {
					return d.Job{Key: key, State: "COMPLETED"}, nil
				}
				if mode == "caller cancellation" {
					cancel()
					return d.Job{}, readCtx.Err()
				}
				if mode == "timeout" || mode == "caller deadline" {
					<-readCtx.Done()
					return d.Job{}, readCtx.Err()
				}
				if reads == 2 {
					return d.Job{}, d.ErrNotFound
				}
				return d.Job{Key: key, State: "CREATED"}, nil
			}, Submit: func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
				writes++
				if mode == "cancel after submission" {
					cancel()
				}
				return d.CompletionSubmission{Status: d.CompletionSubmissionUncertain, Diagnostic: "connection lost"}, d.ErrUnavailable
			}}
			r, err := Execute(ctx, b, "tenant", backoff, ports, 1)
			require.Equal(t, 1, writes)
			assertExecutionCounts(t, b, r)
			require.Equal(t, d.CompletionSubmissionUncertain, r.Outcomes[0].Submission.Status)
			switch mode {
			case "eventual":
				require.NoError(t, err)
				require.Equal(t, 4, reads)
				require.Equal(t, 1, r.Counts.Completed)
			case "max retries":
				require.Error(t, err)
				require.Equal(t, 3, reads)
				require.Equal(t, d.CompletionBatchFinished, r.Status)
			case "timeout":
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Equal(t, 2, reads)
				require.Equal(t, d.CompletionBatchFinished, r.Status)
			case "caller deadline":
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Equal(t, d.CompletionBatchInterrupted, r.Status)
			default:
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, d.CompletionBatchInterrupted, r.Status)
			}
			if mode == "cancel after submission" {
				require.Equal(t, 1, reads)
			}
		})
	}
}

// TestBulkCompletionNoUnnecessaryReconciliation skips confirmation for acceptance and unrelated explicit rejections.
func TestBulkCompletionNoUnnecessaryReconciliation(t *testing.T) {
	t.Parallel()
	for _, code := range []int{204, 400, 401, 403, 422, 429, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			b := preparedForExecution(t, 1)
			reads, writes := 0, 0
			ports := ExecutionPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				reads++
				return d.Job{Key: key, State: "CREATED"}, nil
			}, Submit: func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
				writes++
				if code == 204 {
					return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted, HTTPStatus: &code}, nil
				}
				return d.CompletionSubmission{Status: d.CompletionSubmissionRejected, HTTPStatus: &code}, d.ErrRateLimited
			}}
			r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 1)
			require.Equal(t, 1, reads)
			require.Equal(t, 1, writes)
			if code == 204 {
				require.NoError(t, err)
				require.Equal(t, d.CompletionObservationSkipped, r.Outcomes[0].Observation.Status)
			} else {
				require.ErrorIs(t, err, d.ErrRateLimited)
			}
		})
	}
}
