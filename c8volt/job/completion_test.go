// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestBulkCompletionFacadePreparation preserves partial evidence, options, context, and independent mutable values.
func TestBulkCompletionFacadePreparation(t *testing.T) {
	t.Parallel()
	retries := int32(0)
	request := CompletionRequest{Selection: CompletionSelection{Mode: CompletionSelectionExplicit, InputForm: CompletionInputCollection, Keys: []string{"9007199254740993", "2"}, Filter: CompletionFilter{Retries: &retries}}, Variables: map[string]any{"nested": map[string]any{"items": []any{"original"}}}}
	var returned d.PreparedCompletionBatch
	api := New(fakeJobService{prepare: func(ctx context.Context, input d.CompletionRequest, opts ...services.CallOption) (d.PreparedCompletionBatch, error) {
		require.Equal(t, context.Background(), ctx)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.DryRun)
		require.True(t, cfg.NoWait)
		require.True(t, cfg.FailFast)
		require.True(t, cfg.NoWorkerLimit)
		require.True(t, cfg.IgnoreTenant)
		require.True(t, cfg.Force)
		require.True(t, cfg.NoStateCheck)
		require.Equal(t, d.CompletionSelectionExplicit, input.Selection.Mode)
		require.Zero(t, *input.Selection.Filter.Retries)
		require.Equal(t, request.Selection.Keys, input.Selection.Keys)
		input.Selection.Keys[0] = "3"
		*input.Selection.Filter.Retries = 9
		input.Variables["nested"].(map[string]any)["items"].([]any)[0] = "service"
		returned = d.PreparedCompletionBatch{Selection: input.Selection, Variables: input.Variables, Status: d.CompletionPreparationIncomplete, DiscoveryTenant: "configured", TenantIDs: []string{"other"}, PageCount: 3, ObservedCount: 7, Limited: true, Candidates: []d.CompletionCandidate{{Key: "3", ProcessInstanceKey: "4", ElementInstanceKey: "5", ElementId: "task", TenantId: "other", Type: "work", Kind: "TASK_LISTENER", ListenerEventType: "CANCELING", State: "CREATED", Worker: "worker", Retries: 0}}}
		return returned, d.ErrNotFound
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	batch, err := api.PrepareJobCompletionBatch(context.Background(), request, foptions.WithDryRun(), foptions.WithNoWait(), foptions.WithFailFast(), foptions.WithNoWorkerLimit(), foptions.WithIgnoreTenant(), foptions.WithForce(), foptions.WithNoStateCheck())
	require.ErrorIs(t, err, ferrors.ErrNotFound)
	require.Equal(t, "9007199254740993", request.Selection.Keys[0])
	require.Zero(t, retries)
	require.Equal(t, "original", request.Variables["nested"].(map[string]any)["items"].([]any)[0])
	require.Equal(t, CompletionPreparationIncomplete, batch.Status)
	require.Equal(t, CompletionCandidate(returned.Candidates[0]), batch.Candidates[0])
	require.Equal(t, int64(3), batch.PageCount)
	require.Equal(t, int64(7), batch.ObservedCount)
	require.True(t, batch.Limited)
	require.Equal(t, "configured", batch.DiscoveryTenant)
	returned.Candidates[0].Key = "changed"
	returned.TenantIDs[0] = "changed"
	returned.Selection.Keys[0] = "changed"
	*returned.Selection.Filter.Retries = 3
	returned.Variables["nested"].(map[string]any)["items"].([]any)[0] = "changed"
	require.Equal(t, "3", batch.Candidates[0].Key)
	require.Equal(t, "other", batch.TenantIDs[0])
	require.Equal(t, "3", batch.Selection.Keys[0])
	require.Equal(t, int32(9), *batch.Selection.Filter.Retries)
	require.Equal(t, "service", batch.Variables["nested"].(map[string]any)["items"].([]any)[0])
}

// TestBulkCompletionReportJSON fixes empty-array, omitted-collection and numeric zero count contracts for every phase.
func TestBulkCompletionReportJSON(t *testing.T) {
	t.Parallel()
	for _, status := range []CompletionBatchStatus{CompletionBatchPreview, CompletionBatchPreparationFailed, CompletionBatchFinished, CompletionBatchAborted, CompletionBatchInterrupted} {
		t.Run(string(status), func(t *testing.T) {
			raw, err := json.Marshal(CompletionBatchReport{Status: status})
			require.NoError(t, err)
			var result map[string]any
			require.NoError(t, json.Unmarshal(raw, &result))
			applicable, inapplicable := "outcomes", "candidates"
			if status == CompletionBatchPreview || status == CompletionBatchPreparationFailed {
				applicable, inapplicable = inapplicable, applicable
			}
			require.Equal(t, []any{}, result[applicable])
			require.NotContains(t, result, inapplicable)
			require.Equal(t, map[string]any{"selected": float64(0), "accepted": float64(0), "completed": float64(0), "unsatisfied": float64(0), "notStarted": float64(0)}, result["counts"])
		})
	}
	raw, err := json.Marshal(CompletionOutcome{Disposition: CompletionDispositionUnsatisfied, Reason: CompletionOutcomeUncertain, Submission: CompletionSubmission{Status: CompletionSubmissionUncertain}, Observation: CompletionObservation{Status: CompletionObservationMissing, Phase: CompletionPhaseReconciliation}})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"status":"uncertain"`)
	require.Contains(t, string(raw), `"status":"missing"`)
	require.NotContains(t, string(raw), "httpStatus")
}

// TestBulkCompletionFacadeEmptyAndInvalidPayload keeps empty preparation complete
// and refuses non-JSON input before delegation, while retaining public error classes.
func TestBulkCompletionFacadeEmptyAndInvalidPayload(t *testing.T) {
	t.Parallel()
	calls := 0
	api := New(fakeJobService{prepare: func(_ context.Context, r d.CompletionRequest, _ ...services.CallOption) (d.PreparedCompletionBatch, error) {
		calls++
		return d.PreparedCompletionBatch{Selection: r.Selection, Status: d.CompletionPreparationComplete, CompletionReason: d.CompletionReasonExplicitInputsRead, Candidates: []d.CompletionCandidate{}, TenantIDs: []string{}}, nil
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := CompletionRequest{Selection: CompletionSelection{Mode: CompletionSelectionExplicit, InputForm: CompletionInputStdin}}
	batch, err := api.PrepareJobCompletionBatch(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, CompletionPreparationComplete, batch.Status)
	require.Equal(t, CompletionReasonExplicitInputsRead, batch.CompletionReason)
	require.NotNil(t, batch.Candidates)
	request.Variables = map[string]any{"invalid": make(chan int)}
	batch, err = api.PrepareJobCompletionBatch(context.Background(), request)
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	require.Equal(t, CompletionPreparationIncomplete, batch.Status)
	require.Equal(t, 1, calls)
}

// TestBulkCompletionFacadeExecution preserves reports on error and copies mutable values in both directions.
func TestBulkCompletionFacadeExecution(t *testing.T) {
	t.Parallel()
	retries := int32(0)
	batch := PreparedCompletionBatch{Selection: CompletionSelection{Mode: CompletionSelectionExplicit, InputForm: CompletionInputStdin, Keys: []string{"1"}, Filter: CompletionFilter{Retries: &retries}}, Status: CompletionPreparationComplete, CompletionReason: CompletionReasonExplicitInputsRead, DiscoveryTenant: "tenant", TenantIDs: []string{"other"}, Candidates: []CompletionCandidate{{Key: "1", State: "CREATED"}}, ObservedCount: 1, Variables: map[string]any{"nested": []any{"original"}}}
	var returned d.CompletionBatchReport
	api := New(fakeJobService{execute: func(ctx context.Context, input d.PreparedCompletionBatch, workers int, opts ...services.CallOption) (d.CompletionBatchReport, error) {
		require.Equal(t, context.Background(), ctx)
		require.Equal(t, 7, workers)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.DryRun)
		require.True(t, cfg.NoWait)
		require.True(t, cfg.FailFast)
		require.True(t, cfg.NoWorkerLimit)
		require.True(t, cfg.Force)
		require.True(t, cfg.NoStateCheck)
		require.True(t, cfg.IgnoreTenant)
		require.Equal(t, d.CompletionPreparationComplete, input.Status)
		require.Equal(t, d.CompletionReasonExplicitInputsRead, input.CompletionReason)
		require.Equal(t, "tenant", input.DiscoveryTenant)
		require.Equal(t, int64(1), input.ObservedCount)
		input.Selection.Keys[0] = "2"
		*input.Selection.Filter.Retries = 4
		input.TenantIDs[0] = "service"
		input.Candidates[0].Key = "2"
		input.Variables["nested"].([]any)[0] = "service"
		status := 409
		returned = d.CompletionBatchReport{Status: d.CompletionBatchFinished, Selection: input.Selection, DiscoveryTenant: input.DiscoveryTenant, TenantIDs: input.TenantIDs, CompletionReason: input.CompletionReason, PageCount: 2, ObservedCount: 3, Limited: true, Counts: d.CompletionCounts{Selected: 1, Unsatisfied: 1}, Outcomes: []d.CompletionOutcome{{Candidate: input.Candidates[0], Disposition: d.CompletionDispositionUnsatisfied, Submission: d.CompletionSubmission{Status: d.CompletionSubmissionRejected, HTTPStatus: &status, Diagnostic: "conflict"}, Observation: d.CompletionObservation{Status: d.CompletionObservationMissing, State: "CREATED", Phase: d.CompletionPhaseReconciliation, Diagnostic: "missing"}, Reason: d.CompletionOutcomeRejected}}}
		return returned, d.ErrConflict
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	report, err := api.ExecuteJobCompletionBatch(context.Background(), batch, 7, foptions.WithDryRun(), foptions.WithNoWait(), foptions.WithFailFast(), foptions.WithNoWorkerLimit(), foptions.WithForce(), foptions.WithNoStateCheck(), foptions.WithIgnoreTenant())
	require.ErrorIs(t, err, ferrors.ErrConflict)
	require.Equal(t, "1", batch.Selection.Keys[0])
	require.Zero(t, retries)
	require.Equal(t, "other", batch.TenantIDs[0])
	require.Equal(t, "1", batch.Candidates[0].Key)
	require.Equal(t, "original", batch.Variables["nested"].([]any)[0])
	require.Equal(t, CompletionBatchFinished, report.Status)
	require.Equal(t, CompletionCounts{Selected: 1, Unsatisfied: 1}, report.Counts)
	require.Equal(t, int64(2), report.PageCount)
	require.Equal(t, int64(3), report.ObservedCount)
	require.True(t, report.Limited)
	out := report.Outcomes[0]
	require.Equal(t, CompletionDispositionUnsatisfied, out.Disposition)
	require.Equal(t, CompletionOutcomeRejected, out.Reason)
	require.Equal(t, CompletionObservationMissing, out.Observation.Status)
	require.Equal(t, CompletionPhaseReconciliation, out.Observation.Phase)
	require.Equal(t, "CREATED", out.Observation.State)
	require.Equal(t, "missing", out.Observation.Diagnostic)
	require.Equal(t, "conflict", out.Submission.Diagnostic)
	require.Equal(t, 409, *out.Submission.HTTPStatus)
	returned.Selection.Keys[0] = "changed"
	*returned.Selection.Filter.Retries = 9
	returned.TenantIDs[0] = "changed"
	returned.Outcomes[0].Candidate.Key = "changed"
	*returned.Outcomes[0].Submission.HTTPStatus = 500
	require.Equal(t, "2", report.Selection.Keys[0])
	require.Equal(t, int32(4), *report.Selection.Filter.Retries)
	require.Equal(t, "service", report.TenantIDs[0])
	require.Equal(t, "2", report.Outcomes[0].Candidate.Key)
	require.Equal(t, 409, *report.Outcomes[0].Submission.HTTPStatus)
	batch.Variables = map[string]any{"invalid": make(chan int)}
	report, err = api.ExecuteJobCompletionBatch(context.Background(), batch, 1)
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	require.Equal(t, CompletionBatchPreparationFailed, report.Status)
}
