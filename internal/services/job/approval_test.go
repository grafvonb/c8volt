// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestScalarCompletionApproval freezes exact values and rejects edited plan identity
// or attachments without treating later request-payload edits as new approval.
func TestScalarCompletionApproval(t *testing.T) {
	t.Parallel()
	request := d.JobWorkerOutcomeRequest{Key: "job", Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: toolx.Ptr(" Order,😀 "), Variables: map[string]any{"nested": []any{json.Number("9007199254740993")}}}
	frozen, err := FreezeCompletionPayload("job", d.JobWorkerOutcomeCompletion, request)
	require.NoError(t, err)
	*request.SetBusinessID = "later"
	request.Variables["nested"].([]any)[0] = false
	require.Equal(t, " Order,😀 ", *frozen.SetBusinessID)
	require.Equal(t, json.Number("9007199254740993"), frozen.Variables["nested"].([]any)[0])
	variables, err := toolx.CopyJSONMap(frozen.Variables)
	require.NoError(t, err)
	plan := ScalarCompletionApproval{Key: "job", Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: toolx.CopyPtr(frozen.SetBusinessID), Variables: variables, Approved: &frozen}
	resolved, err := PrepareJobWorkerOutcomeRequest(request, &plan)
	require.NoError(t, err)
	require.Equal(t, frozen, resolved)
	*resolved.SetBusinessID = "adapter"
	resolved.Variables["nested"].([]any)[0] = true
	require.Equal(t, " Order,😀 ", *frozen.SetBusinessID)
	require.Equal(t, json.Number("9007199254740993"), frozen.Variables["nested"].([]any)[0])
	for _, edit := range []func(*ScalarCompletionApproval){
		func(p *ScalarCompletionApproval) { p.Key = "other" },
		func(p *ScalarCompletionApproval) { p.Mode = d.JobWorkerOutcomeBPMNError },
		func(p *ScalarCompletionApproval) { p.SetBusinessID = toolx.Ptr("other") },
		func(p *ScalarCompletionApproval) { p.Variables = map[string]any{"edited": true} },
		func(p *ScalarCompletionApproval) { p.Approved = nil },
	} {
		changed := plan
		edit(&changed)
		_, err := PrepareJobWorkerOutcomeRequest(request, &changed)
		require.ErrorIs(t, err, d.ErrValidation)
	}
	for _, changed := range []d.JobWorkerOutcomeRequest{
		{Key: "other", Mode: request.Mode, SetBusinessID: request.SetBusinessID},
		{Key: request.Key, Mode: d.JobWorkerOutcomeBPMNError, SetBusinessID: request.SetBusinessID},
	} {
		_, err := PrepareJobWorkerOutcomeRequest(changed, &plan)
		require.ErrorIs(t, err, d.ErrValidation)
	}
}

// TestScalarCompletionApprovalGuards protects request-free local validation,
// optional assignment, ordinary outcomes and invalid JSON before backend dispatch.
func TestScalarCompletionApprovalGuards(t *testing.T) {
	t.Parallel()
	for _, request := range []d.JobWorkerOutcomeRequest{
		{Key: "job", Mode: d.JobWorkerOutcomeBPMNError, SetBusinessID: toolx.Ptr("order")},
		{Key: "job", Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: toolx.Ptr(" ")},
		{Key: "job", Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: toolx.Ptr("order"), Variables: map[string]any{"invalid": make(chan int)}},
	} {
		_, err := PrepareJobWorkerOutcomeRequest(request, nil)
		require.ErrorIs(t, err, d.ErrValidation)
	}
	_, err := FreezeCompletionPayload("other", d.JobWorkerOutcomeCompletion, d.JobWorkerOutcomeRequest{Key: "job", Mode: d.JobWorkerOutcomeCompletion})
	require.ErrorIs(t, err, d.ErrValidation)
	_, err = FreezeCompletionPayload("job", d.JobWorkerOutcomeBPMNError, d.JobWorkerOutcomeRequest{Key: "job", Mode: d.JobWorkerOutcomeCompletion})
	require.ErrorIs(t, err, d.ErrValidation)
	_, err = FreezeCompletionPayload("job", d.JobWorkerOutcomeCompletion, d.JobWorkerOutcomeRequest{Key: "job", Mode: d.JobWorkerOutcomeCompletion, Variables: map[string]any{"invalid": make(chan int)}})
	require.ErrorIs(t, err, d.ErrValidation)
	ordinary := d.JobWorkerOutcomeRequest{Key: "job", Mode: d.JobWorkerOutcomeTechnicalFailure, Message: "exact", Retries: toolx.Ptr(int32(0)), SkipConfirmation: true}
	got, err := PrepareJobWorkerOutcomeRequest(ordinary, nil)
	require.NoError(t, err)
	require.Equal(t, ordinary, got)
	direct := d.JobWorkerOutcomeRequest{Key: "job", Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: toolx.Ptr("order")}
	got, err = PrepareJobWorkerOutcomeRequest(direct, nil)
	require.NoError(t, err)
	require.NotSame(t, direct.SetBusinessID, got.SetBusinessID)
}

// TestAdministrativeJobAssignmentGuard retains the domain class and bounded
// established wording without requiring an API, context or backend request.
func TestAdministrativeJobAssignmentGuard(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateAdministrativeAssignment(false))
	err := ValidateAdministrativeAssignment(true)
	require.ErrorIs(t, err, d.ErrValidation)
	require.EqualError(t, err, "business-ID assignment requires worker completion")
}
