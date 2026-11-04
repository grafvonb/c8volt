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

// TestCompletionTimeoutUpdated covers the observed 8.10 state across selection,
// the execution recheck and independent observation without a live backend.
func TestCompletionTimeoutUpdated(t *testing.T) {
	t.Run("search selection", func(t *testing.T) {
		request := d.CompletionRequest{Selection: d.CompletionSelection{
			Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters,
			Filter: d.CompletionFilter{State: "timeout_updated", Type: "work"},
		}}
		calls := 0
		batch, err := Prepare(context.Background(), request, "tenant", config.BackoffConfig{}, ReadPorts{
			Search: func(_ context.Context, query d.JobSearchQuery, page d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
				calls++
				require.Equal(t, "TIMEOUT_UPDATED", query.State)
				return d.JobSearchPage{Request: page, Items: []d.Job{{Key: "1", State: query.State, Type: "work", TenantId: "tenant"}}, RawItemCount: 1, NextFrom: 1, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, d.CompletionPreparationComplete, batch.Status)
		require.Len(t, batch.Candidates, 1)
		require.Equal(t, "TIMEOUT_UPDATED", batch.Candidates[0].State)
	})

	t.Run("business ID completion after timeout update", func(t *testing.T) {
		batch := preparedForExecution(t, 1) // Discovered as CREATED before the timeout changed.
		batch.SetBusinessID = toolx.Ptr(" Order-42 ")
		batch.Variables = map[string]any{"approved": true}
		reads, writes := 0, 0
		report, err := Execute(context.Background(), batch, "tenant", config.BackoffConfig{}, ExecutionPorts{
			Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				reads++
				return d.Job{Key: key, State: "TIMEOUT_UPDATED", Kind: "BPMN_ELEMENT", Type: "work", TenantId: "tenant"}, nil
			},
			Submit: func(_ context.Context, job d.Job, payload d.CompletionPayload) (d.CompletionSubmission, error) {
				writes++
				require.Equal(t, "1", job.Key)
				require.Equal(t, "TIMEOUT_UPDATED", job.State)
				require.Equal(t, batch.SetBusinessID, payload.SetBusinessID)
				require.Equal(t, batch.Variables, payload.Variables)
				return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
			},
		}, 1, services.WithNoWait())
		require.NoError(t, err)
		require.Equal(t, 1, reads)
		require.Equal(t, 1, writes)
		require.Equal(t, 1, report.Counts.Accepted)
		require.Equal(t, "accepted", report.Outcomes[0].Assignment.Status)
		require.Equal(t, d.CompletionObservationSkipped, report.Outcomes[0].Observation.Status)
	})

	t.Run("observation is known and unfinished", func(t *testing.T) {
		_, observation, err := observeJob(context.Background(), "1", "CREATED", d.CompletionPhaseReconciliation,
			func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				return d.Job{Key: key, State: "TIMEOUT_UPDATED"}, nil
			})
		require.NoError(t, err)
		require.Equal(t, "TIMEOUT_UPDATED", observation.State)
		require.Equal(t, d.CompletionObservationNotCompleted, observation.Status)
	})
}
