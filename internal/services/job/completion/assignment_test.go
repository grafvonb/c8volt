// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCompletionAssignmentEvidence prevents completed observations from laundering failed assignment.
func TestCompletionAssignmentEvidence(t *testing.T) {
	for _, status := range []d.CompletionSubmissionStatus{d.CompletionSubmissionUncertain, d.CompletionSubmissionRejected} {
		t.Run(string(status), func(t *testing.T) {
			b := preparedForExecution(t, 1)
			b.SetBusinessID = toolx.Ptr(" Order,42 ")
			reads, writes := 0, 0
			r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ExecutionPorts{
				Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
					reads++
					state := "CREATED"
					if reads > 1 {
						state = "COMPLETED"
					}
					return d.Job{Key: key, State: state}, nil
				},
				Submit: func(_ context.Context, job d.Job, p d.CompletionPayload) (d.CompletionSubmission, error) {
					writes++
					require.Equal(t, " Order,42 ", *p.SetBusinessID)
					return d.CompletionSubmission{Status: status, HTTPStatus: toolx.Ptr(409)}, d.ErrConflict
				},
			}, 1, services.WithNoWait())
			require.ErrorIs(t, err, d.ErrConflict)
			require.Equal(t, 1, writes)
			require.Equal(t, d.CompletionDispositionUnsatisfied, r.Outcomes[0].Disposition)
			require.Equal(t, d.CompletionObservationCompleted, r.Outcomes[0].Observation.Status)
			require.Equal(t, map[d.CompletionSubmissionStatus]string{d.CompletionSubmissionUncertain: "unknown", d.CompletionSubmissionRejected: "rejected"}[status], r.Outcomes[0].Assignment.Status)
		})
	}
}

// TestCompletionAssignmentFrozenPayload preserves copied intent, job-key deduplication, and distinct jobs in one process.
func TestCompletionAssignmentFrozenPayload(t *testing.T) {
	value := "original"
	vars := map[string]any{"nested": map[string]any{"approved": true}}
	get := func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
		return d.Job{Key: key, State: "CREATED", ProcessInstanceKey: "9"}, nil
	}
	batch, err := Prepare(context.Background(), d.CompletionRequest{SetBusinessID: &value, Variables: vars, Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1", "1", "2"}}}, "", config.BackoffConfig{}, ReadPorts{Get: get})
	require.NoError(t, err)
	require.Len(t, batch.Candidates, 2)
	value = "changed"
	vars["nested"].(map[string]any)["approved"] = false
	writes := 0
	report, err := Execute(context.Background(), batch, "", config.BackoffConfig{}, ExecutionPorts{
		Get: func(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error) {
			// Mutating the caller's prepared storage after execution starts must not alter its snapshot.
			*batch.SetBusinessID = "edited"
			batch.Variables["nested"].(map[string]any)["approved"] = false
			return get(ctx, key, opts...)
		},
		Submit: func(_ context.Context, j d.Job, p d.CompletionPayload) (d.CompletionSubmission, error) {
			writes++
			require.Equal(t, "original", *p.SetBusinessID)
			require.Equal(t, true, p.Variables["nested"].(map[string]any)["approved"])
			if j.Key == "2" {
				return d.CompletionSubmission{Status: d.CompletionSubmissionRejected, HTTPStatus: toolx.Ptr(400)}, d.ErrBadRequest
			}
			return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
		},
	}, 1)
	require.ErrorIs(t, err, d.ErrBadRequest)
	require.Equal(t, 2, writes)
	require.Equal(t, 1, report.Counts.Accepted)
	require.Equal(t, 1, report.Counts.Unsatisfied)
	require.Equal(t, "accepted", report.Outcomes[0].Assignment.Status)
	require.Equal(t, "rejected", report.Outcomes[1].Assignment.Status)
	require.Equal(t, "original", *report.RequestedBusinessID)
	require.Nil(t, report.Outcomes[0].Candidate.BusinessId)
}

// TestCompletionAssignmentNoSubmission retains explicit evidence for dry-run, completed jobs and fail-fast slots.
func TestCompletionAssignmentNoSubmission(t *testing.T) {
	b := preparedForExecution(t, 2)
	b.SetBusinessID = toolx.Ptr("order")
	preview, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ExecutionPorts{}, 1, services.WithDryRun())
	require.NoError(t, err)
	require.Equal(t, "order", *preview.RequestedBusinessID)
	require.Empty(t, preview.Outcomes)
	r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ExecutionPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
		return d.Job{Key: key, State: "COMPLETED"}, nil
	}, Submit: func(context.Context, d.Job, d.CompletionPayload) (d.CompletionSubmission, error) {
		t.Fatal("completed job submitted")
		return d.CompletionSubmission{}, nil
	}}, 1, services.WithFailFast())
	require.ErrorIs(t, err, d.ErrPrecondition)
	require.Equal(t, 1, r.Counts.Unsatisfied)
	require.Equal(t, 1, r.Counts.NotStarted)
	for _, o := range r.Outcomes {
		require.Equal(t, "not_submitted", o.Assignment.Status)
	}
}
