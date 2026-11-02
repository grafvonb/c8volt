// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"encoding/json"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDActionsFacadeCopies preserves mutable provenance across all plan conversions and JSON round trips.
func TestBusinessIDActionsFacadeCopies(t *testing.T) {
	t.Parallel()
	selection := ProcessInstanceSelection{Request: ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: toolx.Ptr(" order,Ω\n ")}}}, Origin: "search", Pages: 1, Completion: ProcessInstanceSearchCompletionExhausted, Keys: []string{"2251799813685101"}, Items: []ProcessInstance{{Key: "2251799813685101", BusinessId: toolx.Ptr("observed"), Variables: map[string]any{"nested": map[string]any{"value": "original"}}}}}
	update := ProcessInstanceVariableUpdatePlan{Selection: &selection, SelectionTenant: "tenant-a"}
	suspend := ProcessInstanceSuspensionPlan{Selection: &selection, SelectionTenant: "tenant-a", Exact: true, TargetKeys: selection.Keys, RequestedKeys: selection.Keys}
	resume := ProcessInstanceResumptionPlan{Selection: &selection, SelectionTenant: "tenant-a", Exact: true, TargetKeys: selection.Keys, RequestedKeys: selection.Keys}
	ud, sd, rd := toDomainProcessInstanceVariableUpdatePlan(update), toDomainProcessInstanceSuspensionPlan(suspend), toDomainProcessInstanceResumptionPlan(resume)
	*selection.Request.SearchRequest.Filter.BusinessId = "changed"
	selection.Items[0].Variables["nested"].(map[string]any)["value"] = "changed"
	selection.Keys[0] = "changed"
	for _, s := range []*d.ProcessInstanceSelection{ud.Selection, sd.Selection, rd.Selection} {
		require.Equal(t, " order,Ω\n ", *s.Request.SearchRequest.Filter.BusinessId)
		require.Equal(t, "original", s.Items[0].Variables["nested"].(map[string]any)["value"])
		require.Equal(t, "2251799813685101", s.Keys[0])
	}
	up, sp, rp := fromDomainProcessInstanceVariableUpdatePlan(ud), fromDomainProcessInstanceSuspensionPlan(sd), fromDomainProcessInstanceResumptionPlan(rd)
	for _, plan := range []any{up, sp, rp} {
		raw, err := json.Marshal(plan)
		require.NoError(t, err)
		require.Contains(t, string(raw), `order,Ω\n `)
		require.Contains(t, string(raw), `"selectionTenant":"tenant-a"`)
	}
	var roundTrip ProcessInstanceSuspensionPlan
	raw, err := json.Marshal(sp)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &roundTrip))
	require.Equal(t, sp, roundTrip)
	*sd.Selection.Request.SearchRequest.Filter.BusinessId = "domain changed"
	require.Equal(t, " order,Ω\n ", *sp.Selection.Request.SearchRequest.Filter.BusinessId)
}

// TestBusinessIDActionsFacadeVariablePlan delegates selection once, preserves partial errors, and isolates execution input.
func TestBusinessIDActionsFacadeVariablePlan(t *testing.T) {
	t.Parallel()
	searches := 0
	api := stubProcessInstanceAPI{searchForProcessInstancesPage: func(_ context.Context, _ d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
		searches++
		require.False(t, services.ApplyCallOptions(opts).IgnoreTenant)
		return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{{Key: "2251799813685101"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}}
	updates := &facadeProcessInstanceVariableUpdateAPI{plan: d.ProcessInstanceVariableUpdatePlan{RequestedKeys: []string{"2251799813685101"}}}
	cli := &client{piApi: api, updateApi: updates}
	bid := "original"
	values := map[string]any{"x": map[string]any{"y": true}}
	plan, err := cli.PlanSelectedProcessInstanceVariableUpdates(context.Background(), ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: &bid}}}, values)
	require.NoError(t, err)
	require.Equal(t, 1, searches)
	require.True(t, updates.callConfig.IgnoreTenant)
	bid = "changed"
	values["x"].(map[string]any)["y"] = false
	require.Equal(t, "original", *plan.Selection.Request.SearchRequest.Filter.BusinessId)
	require.Equal(t, true, updates.variables["x"].(map[string]any)["y"])
	updates.err = d.ErrForbidden
	updates.results = d.ProcessInstanceVariableUpdateResults{Items: []d.ProcessInstanceVariableUpdateResult{{Key: "2251799813685101", MutationAccepted: true}}}
	result, err := cli.ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 3, options.WithNoWait())
	require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
	require.Len(t, result.Items, 1)
	require.Equal(t, 3, updates.workers)
	require.True(t, updates.callConfig.NoWait)
	*plan.Selection.Request.SearchRequest.Filter.BusinessId = "caller edit"
	require.Equal(t, "original", *updates.executed.Selection.Request.SearchRequest.Filter.BusinessId)
	require.Equal(t, 1, searches)
}

// TestBusinessIDActionsFacadeState preserves frozen selection and error conversion through both public execution methods.
func TestBusinessIDActionsFacadeState(t *testing.T) {
	t.Parallel()
	for _, resume := range []bool{false, true} {
		searches, writes := 0, 0
		base := stubProcessInstanceAPI{searchForProcessInstancesPage: func(_ context.Context, _ d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			searches++
			require.False(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{{Key: "2251799813685101", BusinessId: toolx.Ptr("observed")}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
		}}
		request := ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: toolx.Ptr(" ORDER,Ω ")}}}
		if resume {
			api := facadeResumptionStub{API: base, call: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				if services.ApplyCallOptions(opts).DryRun {
					return d.ProcessInstanceResumptionResult{Key: key}, nil
				}
				writes++
				return d.ProcessInstanceResumptionResult{Key: key, Submission: d.ProcessInstanceResumptionSubmissionAccepted}, d.ErrForbidden
			}}
			cli := New(nil, api, nil, nil)
			plan, err := cli.PlanSelectedProcessInstanceResumption(context.Background(), request, false)
			require.NoError(t, err)
			raw, err := json.Marshal(plan)
			require.NoError(t, err)
			var frozen ProcessInstanceResumptionPlan
			require.NoError(t, json.Unmarshal(raw, &frozen))
			*request.SearchRequest.Filter.BusinessId = "changed"
			result, err := cli.ExecuteProcessInstanceResumptionPlan(context.Background(), frozen, 1, options.WithNoWait())
			require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
			require.Equal(t, ResumptionSubmissionAccepted, result.Items[0].Submission)
			require.Equal(t, " ORDER,Ω ", *result.Selection.Request.SearchRequest.Filter.BusinessId)
		} else {
			api := facadeSuspensionStub{API: base, call: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				if services.ApplyCallOptions(opts).DryRun {
					return d.ProcessInstanceSuspensionResult{Key: key}, nil
				}
				writes++
				return d.ProcessInstanceSuspensionResult{Key: key, Submission: d.ProcessInstanceSuspensionSubmissionAccepted}, d.ErrForbidden
			}}
			cli := New(nil, api, nil, nil)
			plan, err := cli.PlanSelectedProcessInstanceSuspension(context.Background(), request, false)
			require.NoError(t, err)
			raw, err := json.Marshal(plan)
			require.NoError(t, err)
			var frozen ProcessInstanceSuspensionPlan
			require.NoError(t, json.Unmarshal(raw, &frozen))
			*request.SearchRequest.Filter.BusinessId = "changed"
			result, err := cli.ExecuteProcessInstanceSuspensionPlan(context.Background(), frozen, 1, options.WithNoWait())
			require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
			require.Equal(t, SuspensionSubmissionAccepted, result.Items[0].Submission)
			require.Equal(t, " ORDER,Ω ", *result.Selection.Request.SearchRequest.Filter.BusinessId)
		}
		require.Equal(t, 1, searches)
		require.Equal(t, 1, writes)
	}
}
