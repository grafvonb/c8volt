// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCompletionAssignmentScalarApproval freezes approval independently of mutable request and preview storage.
func TestCompletionAssignmentScalarApproval(t *testing.T) {
	value := " Order,😀 "
	vars := map[string]any{"nested": map[string]any{"approved": true}}
	request := WorkerOutcomeRequest{Key: "1", Mode: WorkerOutcomeCompletion, SetBusinessID: &value, Variables: vars}
	plan := UpdatePlan{Key: "1", Mode: MutationModeCompletion}
	require.NoError(t, plan.FreezeCompletionPayload(request))
	request.OutcomePlan = &plan
	value = "changed"
	vars["nested"].(map[string]any)["approved"] = false
	input, err := toDomainWorkerOutcomeRequest(request)
	require.NoError(t, err)
	require.Equal(t, " Order,😀 ", *input.SetBusinessID)
	require.Equal(t, true, input.Variables["nested"].(map[string]any)["approved"])
	*input.SetBusinessID = "service"
	input.Variables["nested"].(map[string]any)["approved"] = false
	input, err = toDomainWorkerOutcomeRequest(request)
	require.NoError(t, err)
	require.Equal(t, " Order,😀 ", *input.SetBusinessID)
	require.Equal(t, true, input.Variables["nested"].(map[string]any)["approved"])
	for _, change := range []func(*WorkerOutcomeRequest){
		func(r *WorkerOutcomeRequest) { r.Key = "2" }, func(r *WorkerOutcomeRequest) { r.Mode = WorkerOutcomeBPMNError },
		func(r *WorkerOutcomeRequest) { p := *r.OutcomePlan; p.Key = "2"; r.OutcomePlan = &p },
		func(r *WorkerOutcomeRequest) {
			p := *r.OutcomePlan
			p.SetBusinessID = toolx.Ptr("replacement")
			r.OutcomePlan = &p
		},
		func(r *WorkerOutcomeRequest) {
			p := *r.OutcomePlan
			p.Variables = map[string]any{"replaced": true}
			r.OutcomePlan = &p
		},
	} {
		r := request
		change(&r)
		_, err := toDomainWorkerOutcomeRequest(r)
		require.ErrorIs(t, err, d.ErrValidation)
	}
}

// TestCompletionAssignmentFacadeCopies keeps request, preparation and returned failure evidence independently owned.
func TestCompletionAssignmentFacadeCopies(t *testing.T) {
	value := "original"
	request := CompletionRequest{SetBusinessID: &value, Variables: map[string]any{"nested": []any{true}}}
	input, err := toDomainCompletionRequest(request)
	require.NoError(t, err)
	value = "changed"
	require.Equal(t, "original", *input.SetBusinessID)
	batch, err := fromDomainPreparedCompletionBatch(d.PreparedCompletionBatch{SetBusinessID: input.SetBusinessID, Variables: input.Variables})
	require.NoError(t, err)
	*input.SetBusinessID = "service"
	require.Equal(t, "original", *batch.SetBusinessID)
	execution, err := toDomainPreparedCompletionBatch(batch)
	require.NoError(t, err)
	*batch.SetBusinessID = "caller"
	require.Equal(t, "original", *execution.SetBusinessID)
	original := d.JobWorkerOutcomeResult{RequestedBusinessID: toolx.Ptr("order"), Assignment: &d.CompletionAssignment{Status: "unknown", HTTPStatus: toolx.Ptr(503)}, Submission: &d.CompletionSubmission{Status: d.CompletionSubmissionUncertain}, Observation: &d.CompletionObservation{Status: d.CompletionObservationCompleted}, MutationError: "lost"}
	api := New(fakeJobService{outcome: func(context.Context, d.JobWorkerOutcomeRequest, ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
		return original, d.ErrUnavailable
	}}, nil)
	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{Key: "1", Mode: WorkerOutcomeCompletion, SetBusinessID: toolx.Ptr("order")})
	require.ErrorIs(t, err, ferrors.ErrUnavailable)
	require.Equal(t, "unknown", result.Assignment.Status)
	require.Equal(t, CompletionObservationCompleted, result.Observation.Status)
	*original.RequestedBusinessID = "later"
	*original.Assignment.HTTPStatus = 400
	require.Equal(t, "order", *result.RequestedBusinessID)
	require.Equal(t, 503, *result.Assignment.HTTPStatus)
	raw, err := json.Marshal(fromDomainWorkerOutcomeResult(d.JobWorkerOutcomeResult{}))
	require.NoError(t, err)
	require.NotContains(t, string(raw), "assignment")
	require.NotContains(t, string(raw), "requestedBusinessId")
}

// TestCompletionAssignmentPublicGuards rejects invalid values and non-completion modes without service dispatch.
func TestCompletionAssignmentPublicGuards(t *testing.T) {
	api := New(fakeJobService{outcome: func(context.Context, d.JobWorkerOutcomeRequest, ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
		t.Fatal("invalid request dispatched")
		return d.JobWorkerOutcomeResult{}, nil
	}}, nil)
	for _, v := range []string{"", " \t", string([]byte{0xff}), strings.Repeat("a", 257), strings.Repeat("😀", 129)} {
		_, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{Key: "1", Mode: WorkerOutcomeCompletion, SetBusinessID: &v})
		require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	}
	for _, r := range []UpdateRequest{
		{Key: "1", Retries: toolx.Ptr(int32(1)), WorkerOutcome: &WorkerOutcomeRequest{SetBusinessID: toolx.Ptr("order")}},
		{Key: "1", TimeoutMillis: toolx.Ptr(int64(100)), UpdatePlan: &UpdatePlan{SetBusinessID: toolx.Ptr("order")}},
	} {
		_, err := api.UpdateJob(context.Background(), r)
		require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	}
	for _, mode := range []WorkerOutcomeMode{WorkerOutcomeTechnicalFailure, WorkerOutcomeBPMNError, "update", "timeout"} {
		_, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{Key: "1", Mode: mode, SetBusinessID: toolx.Ptr("order")})
		require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	}
}
