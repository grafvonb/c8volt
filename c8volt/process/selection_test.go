// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"encoding/json"
	tr "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"io"
	"log/slog"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDSelectionFacade retains copied selection evidence, option mapping and original failure classification.
func TestBusinessIDSelectionFacade(t *testing.T) {
	t.Parallel()
	for _, ignore := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			bid := " exact,* Ω "
			observed := "different"
			calls := 0
			api := stubProcessInstanceAPI{searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
				calls++
				require.Equal(t, ignore, services.ApplyCallOptions(opts).IgnoreTenant)
				require.Equal(t, bid, *filter.BusinessId)
				if calls == 2 {
					return d.ProcessInstancePage{}, d.ErrForbidden
				}
				overflow := d.ProcessInstanceOverflowStateNoMore
				if fail {
					overflow = d.ProcessInstanceOverflowStateHasMore
				}
				return d.ProcessInstancePage{Request: page, Items: []d.ProcessInstance{{Key: "1", BusinessId: &observed, TenantId: "tenant-a", Variables: map[string]any{"nested": map[string]any{"value": "original"}}}}, OverflowState: overflow}, nil
			}}
			cli := New(nil, api, stubIncidentAPI{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			opts := []options.FacadeOption{}
			if ignore {
				opts = append(opts, options.WithIgnoreTenant())
			}
			result, err := cli.ResolveProcessInstanceSelection(context.Background(), ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: &bid}}}, opts...)
			if fail {
				require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
				require.Equal(t, ProcessInstanceSearchCompletionIncomplete, result.Completion)
			} else {
				require.NoError(t, err)
				require.Equal(t, ProcessInstanceSearchCompletionExhausted, result.Completion)
			}
			bid = "changed"
			observed = "changed"
			require.Equal(t, " exact,* Ω ", *result.Request.SearchRequest.Filter.BusinessId)
			require.Equal(t, "different", *result.Items[0].BusinessId)
			require.Equal(t, "search", result.Origin)
			require.Equal(t, ignore, result.IgnoreTenant)
			require.Equal(t, []string{"tenant-a"}, result.TenantEvidence.ResolvedTenantIDs)
		}
	}
}

// TestBusinessIDCancelFacade preserves independent plan copies, JSON provenance and original partial execution errors.
func TestBusinessIDCancelFacade(t *testing.T) {
	t.Parallel()
	observed := d.ProcessInstance{Key: "101", TenantId: "tenant-a", State: d.StateActive, BusinessId: toolx.Ptr("observed"), Variables: map[string]any{"nested": map[string]any{"value": "original"}}}
	searches := 0
	mutations := 0
	api := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, f d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			searches++
			require.False(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{observed}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
		},
		ancestryResult: func(_ context.Context, key string, opts ...services.CallOption) (tr.Result, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return tr.Result{StartKey: key, RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: observed}, Outcome: tr.OutcomeComplete}, nil
		},
		descendantsResult: func(_ context.Context, key string, opts ...services.CallOption) (tr.Result, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return tr.Result{RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: observed}, Outcome: tr.OutcomeComplete}, nil
		},
		cancelProcessInstance: func(_ context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			mutations++
			require.Equal(t, "101", key)
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return d.CancelResponse{}, nil, d.ErrForbidden
		},
	}
	cli := New(nil, api, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := ProcessInstanceSelectedMutationRequest{Operation: "cancel", Selection: ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: toolx.Ptr(" exact,Ω\n ")}}}}
	plan, err := cli.PlanProcessInstanceSelectedMutation(context.Background(), request)
	require.NoError(t, err)
	*request.Selection.SearchRequest.Filter.BusinessId = "changed"
	*observed.BusinessId = "changed"
	observed.Variables["nested"].(map[string]any)["value"] = "changed"
	require.Equal(t, " exact,Ω\n ", *plan.Selection.Request.SearchRequest.Filter.BusinessId)
	require.Equal(t, "observed", *plan.Families[0].Items[0].BusinessId)
	raw, err := json.Marshal(plan)
	require.NoError(t, err)
	var decoded ProcessInstanceSelectedMutationPlan
	require.NoError(t, json.Unmarshal(raw, &decoded))
	domain := toDomainSelectedMutationPlan(plan)
	plan.Selection.Items[0].Variables["nested"].(map[string]any)["value"] = "caller mutation"
	plan.Families[0].Items[0].Variables["nested"].(map[string]any)["value"] = "caller mutation"
	require.Equal(t, "original", domain.Selection.Items[0].Variables["nested"].(map[string]any)["value"])
	require.Equal(t, "original", domain.Families[0].Items[0].Variables["nested"].(map[string]any)["value"])
	result, err := cli.ExecuteProcessInstanceCancellationPlan(context.Background(), decoded, options.WithNoWait())
	require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
	require.Len(t, result.Items, 1)
	require.Equal(t, "101", result.Items[0].Key)
	require.Equal(t, 1, searches)
	require.Equal(t, 1, mutations)
}

// TestBusinessIDDeleteFacade preserves independent plan copies, JSON provenance and original partial execution errors.
func TestBusinessIDDeleteFacade(t *testing.T) {
	t.Parallel()
	observed := d.ProcessInstance{Key: "101", TenantId: "tenant-a", State: d.StateCompleted, BusinessId: toolx.Ptr("observed"), Variables: map[string]any{"nested": map[string]any{"value": "original"}}}
	searches := 0
	mutations := 0
	api := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, f d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			searches++
			require.False(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{observed}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
		},
		ancestryResult: func(_ context.Context, key string, opts ...services.CallOption) (tr.Result, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return tr.Result{StartKey: key, RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: observed}, Outcome: tr.OutcomeComplete}, nil
		},
		descendantsResult: func(_ context.Context, key string, opts ...services.CallOption) (tr.Result, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return tr.Result{RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: observed}, Outcome: tr.OutcomeComplete}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
			mutations++
			require.Equal(t, "101", key)
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return d.DeleteResponse{}, d.ErrForbidden
		},
	}
	cli := New(nil, api, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := ProcessInstanceSelectedMutationRequest{Operation: "delete", Selection: ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: toolx.Ptr(" exact,Ω\n ")}}}}
	plan, err := cli.PlanProcessInstanceSelectedMutation(context.Background(), request)
	require.NoError(t, err)
	*request.Selection.SearchRequest.Filter.BusinessId = "changed"
	*observed.BusinessId = "changed"
	observed.Variables["nested"].(map[string]any)["value"] = "changed"
	require.Equal(t, " exact,Ω\n ", *plan.Selection.Request.SearchRequest.Filter.BusinessId)
	require.Equal(t, "observed", *plan.Families[0].Items[0].BusinessId)
	raw, err := json.Marshal(plan)
	require.NoError(t, err)
	var decoded ProcessInstanceSelectedMutationPlan
	require.NoError(t, json.Unmarshal(raw, &decoded))
	domain := toDomainSelectedMutationPlan(plan)
	plan.Selection.Items[0].Variables["nested"].(map[string]any)["value"] = "caller mutation"
	plan.Families[0].Items[0].Variables["nested"].(map[string]any)["value"] = "caller mutation"
	require.Equal(t, "original", domain.Selection.Items[0].Variables["nested"].(map[string]any)["value"])
	require.Equal(t, "original", domain.Families[0].Items[0].Variables["nested"].(map[string]any)["value"])
	result, err := cli.ExecuteProcessInstanceDeletionPlan(context.Background(), decoded, options.WithNoWait())
	require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
	require.Len(t, result.Items, 1)
	require.Len(t, result.Members, 1)
	require.False(t, result.Members[0].Ok)
	require.Equal(t, "101", result.Items[0].Key)
	require.Equal(t, 1, searches)
	require.Equal(t, 1, mutations)
}
