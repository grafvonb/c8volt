// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type stubProcessAPI struct {
	dryRunCancelOrDeletePlan   func(context.Context, types.Keys, ...options.FacadeOption) (process.DryRunPIKeyExpansion, error)
	cancelProcessInstances     func(context.Context, types.Keys, int, ...options.FacadeOption) (process.CancelReports, error)
	deleteProcessInstances     func(context.Context, types.Keys, int, ...options.FacadeOption) (process.DeleteReports, error)
	filterOrphanParent         func(context.Context, []process.ProcessInstance, ...options.FacadeOption) ([]process.ProcessInstance, error)
	searchProcessInstancesPage func(context.Context, process.ProcessInstanceFilter, process.ProcessInstancePageRequest, ...options.FacadeOption) (process.ProcessInstancePage, error)
}

// dryRunCancelMutationGuard fails a test if dry-run execution submits a cancel mutation.
func dryRunCancelMutationGuard(t *testing.T) func(context.Context, types.Keys, int, ...options.FacadeOption) (process.CancelReports, error) {
	t.Helper()
	return func(_ context.Context, keys types.Keys, _ int, _ ...options.FacadeOption) (process.CancelReports, error) {
		t.Fatalf("unexpected cancel mutation during dry-run test for keys %v", keys)
		return process.CancelReports{}, nil
	}
}

// dryRunDeleteMutationGuard fails a test if dry-run execution submits a delete mutation.
func dryRunDeleteMutationGuard(t *testing.T) func(context.Context, types.Keys, int, ...options.FacadeOption) (process.DeleteReports, error) {
	t.Helper()
	return func(_ context.Context, keys types.Keys, _ int, _ ...options.FacadeOption) (process.DeleteReports, error) {
		t.Fatalf("unexpected delete mutation during dry-run test for keys %v", keys)
		return process.DeleteReports{}, nil
	}
}

// requireDryRunPreviewStringSlice verifies a dry-run JSON string-slice field.
func requireDryRunPreviewStringSlice(t *testing.T, payload map[string]any, field string, want types.Keys) {
	t.Helper()

	gotRaw, ok := payload[field]
	require.Truef(t, ok, "expected dry-run preview field %q", field)
	gotItems, ok := gotRaw.([]any)
	require.Truef(t, ok, "expected dry-run preview field %q to be a JSON array", field)

	got := make([]string, len(gotItems))
	for i, item := range gotItems {
		value, ok := item.(string)
		require.Truef(t, ok, "expected dry-run preview field %q item %d to be a string", field, i)
		got[i] = value
	}
	require.Equal(t, []string(want), got)
}

// requireDryRunPreviewMissingAncestors verifies missing-ancestor details in a dry-run JSON payload.
func requireDryRunPreviewMissingAncestors(t *testing.T, payload map[string]any, want []process.MissingAncestor) {
	t.Helper()

	gotRaw, ok := payload["missingAncestors"]
	require.True(t, ok, "expected dry-run preview field missingAncestors")
	gotItems, ok := gotRaw.([]any)
	require.True(t, ok, "expected dry-run preview field missingAncestors to be a JSON array")
	require.Len(t, gotItems, len(want))

	for i, item := range gotItems {
		got, ok := item.(map[string]any)
		require.Truef(t, ok, "expected missingAncestors item %d to be a JSON object", i)
		require.Equal(t, want[i].Key, got["key"])
		require.Equal(t, want[i].StartKey, got["startKey"])
	}
}

// requireDryRunSummaryPayload verifies aggregate dry-run JSON counts and returns preview items for deeper checks.
func requireDryRunSummaryPayload(t *testing.T, payload map[string]any, operation string, requestedCount, rootCount, affectedCount int, previewCount int) []any {
	t.Helper()

	require.Equal(t, operation, payload["operation"])
	require.Equal(t, float64(requestedCount), payload["requestedCount"])
	require.Equal(t, float64(rootCount), payload["resolvedRootCount"])
	require.Equal(t, float64(affectedCount), payload["affectedCount"])
	require.Equal(t, string(process.TraversalOutcomeComplete), payload["traversalOutcome"])
	require.Equal(t, true, payload["scopeComplete"])
	require.Equal(t, false, payload["mutationSubmitted"])

	previews, ok := payload["previews"].([]any)
	require.True(t, ok, "expected dry-run summary previews to be a JSON array")
	require.Len(t, previews, previewCount)
	return previews
}

// requireDryRunEnvelopePayload decodes a successful dry-run result envelope and returns its payload.
func requireDryRunEnvelopePayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok, "expected dry-run result envelope payload")
	return payload
}

func (stubProcessAPI) SearchProcessDefinitions(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessDefinitionsLatest(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessDefinition(context.Context, string, ...options.FacadeOption) (process.ProcessDefinition, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessDefinitionXML(context.Context, string, ...options.FacadeOption) (string, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstance(context.Context, process.ProcessInstanceData, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstances(context.Context, []process.ProcessInstanceData, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstanceStateByKey(context.Context, string, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

// SearchProcessInstancesPage delegates to the optional paged-search callback used by dry-run command tests.
func (s stubProcessAPI) SearchProcessInstancesPage(ctx context.Context, filter process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, opts ...options.FacadeOption) (process.ProcessInstancePage, error) {
	if s.searchProcessInstancesPage == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstancesPage(ctx, filter, req, opts...)
}

func (stubProcessAPI) SearchProcessInstances(context.Context, process.ProcessInstanceFilter, int32, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CancelProcessInstance(context.Context, string, ...options.FacadeOption) (process.CancelReport, process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DeleteProcessInstance(context.Context, string, ...options.FacadeOption) (process.DeleteReport, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []process.ProcessInstance, opts ...options.FacadeOption) ([]process.ProcessInstance, error) {
	if s.filterOrphanParent == nil {
		panic("unexpected call")
	}
	return s.filterOrphanParent(ctx, items, opts...)
}

func (stubProcessAPI) WaitForProcessInstanceState(context.Context, string, process.States, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Ancestry(context.Context, string, ...options.FacadeOption) (string, []string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Descendants(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Family(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) AncestryResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DescendantsResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) FamilyResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstances(context.Context, types.Keys, int, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateNProcessInstances(context.Context, process.ProcessInstanceData, int, int, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
	if s.cancelProcessInstances == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (s stubProcessAPI) DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
	if s.deleteProcessInstances == nil {
		panic("unexpected call")
	}
	return s.deleteProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (stubProcessAPI) WaitForProcessInstancesState(context.Context, types.Keys, process.States, int, ...options.FacadeOption) (process.StateReports, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DryRunCancelOrDeleteGetPIKeys(context.Context, types.Keys, ...options.FacadeOption) (types.Keys, types.Keys, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) DryRunCancelOrDeletePlan(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
	if s.dryRunCancelOrDeletePlan == nil {
		panic("unexpected call")
	}
	return s.dryRunCancelOrDeletePlan(ctx, keys, opts...)
}

var _ process.API = stubProcessAPI{}
