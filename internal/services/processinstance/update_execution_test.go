// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type processInstanceVariableExecutionAPI struct {
	mu       sync.Mutex
	writes   []processInstanceVariableExecutionWrite
	write    func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
	search   func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error)
	searches []string
}

type processInstanceVariableExecutionWrite struct {
	scope     string
	variables map[string]any
	options   *services.CallCfg
}

// SearchProcessInstanceVariables records complete verification reads and delegates controlled observations.
func (a *processInstanceVariableExecutionAPI) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	a.mu.Lock()
	a.searches = append(a.searches, key)
	a.mu.Unlock()
	if a.search != nil {
		return a.search(ctx, key, opts...)
	}
	return nil, nil
}

// UpdateScopeVariables records exact payload copies before delegating controlled mutation outcomes.
func (a *processInstanceVariableExecutionAPI) UpdateScopeVariables(ctx context.Context, scope string, variables map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
	a.mu.Lock()
	a.writes = append(a.writes, processInstanceVariableExecutionWrite{
		scope: scope, variables: copyProcessInstanceVariableMap(variables), options: services.ApplyCallOptions(opts),
	})
	a.mu.Unlock()
	if a.write != nil {
		return a.write(ctx, scope, variables, opts...)
	}
	return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true, StatusCode: 204, Status: "accepted"}, nil
}

// TestExecuteProcessInstanceVariableUpdatesSubmitsOnlyFrozenChanges verifies
// stable results, exact root-local payloads, unchanged outcomes, and no-wait behavior.
func TestExecuteProcessInstanceVariableUpdatesSubmitsOnlyFrozenChanges(t *testing.T) {
	t.Parallel()

	api := &processInstanceVariableExecutionAPI{}
	plan := processInstanceExecutionPlan()
	results, err := NewVariableUpdates(nil, api, config.New(), nil).
		ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 2,
			services.WithNoWait(), services.WithFailFast(), services.WithNoWorkerLimit(), services.WithSuppressProcessInstanceDetailLogs())

	require.NoError(t, err)
	require.Len(t, api.writes, 1)
	require.Equal(t, "pi-change", api.writes[0].scope)
	require.Equal(t, map[string]any{"add": map[string]any{"nested": []any{float64(1)}}, "change": "after"}, api.writes[0].variables)
	require.True(t, api.writes[0].options.NoWait)
	require.True(t, api.writes[0].options.FailFast)
	require.True(t, api.writes[0].options.NoWorkerLimit)
	require.True(t, api.writes[0].options.SuppressProcessInstanceDetailLogs)
	require.Empty(t, api.searches)
	require.Equal(t, []string{"pi-unchanged", "pi-change"}, []string{results.Items[0].Key, results.Items[1].Key})
	require.Equal(t, d.ProcessInstanceVariableUpdateStatusUnchanged, results.Items[0].Status)
	require.False(t, results.Items[0].MutationAccepted)
	require.Equal(t, d.ProcessInstanceVariableUpdateStatusSubmitted, results.Items[1].Status)
	require.True(t, results.Items[1].MutationAccepted)
	require.Equal(t, "skipped", results.Items[1].ConfirmationStatus)
	require.Equal(t, plan.Instances[1].VerificationValues, results.Items[1].Variables)
}

// TestExecuteProcessInstanceVariableUpdatesIsolatesWriterMutations verifies a
// writer cannot corrupt later payloads, verification values, or returned results.
func TestExecuteProcessInstanceVariableUpdatesIsolatesWriterMutations(t *testing.T) {
	t.Parallel()

	typed := processInstanceTypedValues{"items": {"original"}}
	raw := json.RawMessage(`{"raw":"original"}`)
	requested := map[string]any{"raw": raw, "typed": typed}
	plan := d.ProcessInstanceVariableUpdatePlan{
		RequestedKeys: []string{"pi-first", "pi-second"}, RequestedValues: requested,
		RequestedCount: 2, UpdateCount: 2, VariableAddCount: 4,
		Instances: []d.ProcessInstanceVariablePlan{
			{ProcessInstanceKey: "pi-first", ScopeKey: "pi-first",
				Additions:         []d.ProcessInstanceVariablePlannedValue{{Name: "raw", Value: raw}, {Name: "typed", Value: typed}},
				MutationVariables: requested, VerificationValues: requested},
			{ProcessInstanceKey: "pi-second", ScopeKey: "pi-second",
				Additions:         []d.ProcessInstanceVariablePlannedValue{{Name: "raw", Value: raw}, {Name: "typed", Value: typed}},
				MutationVariables: requested, VerificationValues: requested},
		},
	}
	writes := 0
	var secondTyped string
	var secondRaw json.RawMessage
	api := &processInstanceVariableExecutionAPI{
		write: func(_ context.Context, _ string, variables map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			writes++
			if writes == 1 {
				variables["typed"].(processInstanceTypedValues)["items"][0] = "writer-mutated"
				variables["raw"].(json.RawMessage)[8] = 'X'
			} else {
				secondTyped = variables["typed"].(processInstanceTypedValues)["items"][0]
				secondRaw = append(json.RawMessage(nil), variables["raw"].(json.RawMessage)...)
			}
			return d.ScopeVariableUpdateResponse{Accepted: true, StatusCode: 204, Status: "accepted"}, nil
		},
		search: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			return []d.ProcessInstanceVariable{
				{Name: "raw", Value: `{"raw":"original"}`, ProcessInstanceKey: key, ScopeKey: key},
				{Name: "typed", Value: `{"items":["original"]}`, ProcessInstanceKey: key, ScopeKey: key},
			}, nil
		},
	}

	results, err := NewVariableUpdates(nil, api, config.New(), nil).
		ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1)

	require.NoError(t, err)
	require.Equal(t, 2, writes)
	require.Equal(t, "original", secondTyped)
	require.JSONEq(t, `{"raw":"original"}`, string(secondRaw))
	for _, item := range results.Items {
		require.Equal(t, d.ProcessInstanceVariableUpdateStatusConfirmed, item.Status)
		require.Equal(t, "original", item.Variables["typed"].(processInstanceTypedValues)["items"][0])
		require.JSONEq(t, `{"raw":"original"}`, string(item.Variables["raw"].(json.RawMessage)))
	}
}

// TestExecuteProcessInstanceVariableUpdatesRejectsForgedPlansBeforeWrites verifies
// classification, verification, payload, and destination inconsistencies are atomic failures.
func TestExecuteProcessInstanceVariableUpdatesRejectsForgedPlansBeforeWrites(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		forge func(*d.ProcessInstanceVariableUpdatePlan)
	}{
		{name: "destination", forge: func(plan *d.ProcessInstanceVariableUpdatePlan) { plan.Instances[1].ScopeKey = "other" }},
		{name: "payload", forge: func(plan *d.ProcessInstanceVariableUpdatePlan) { plan.Instances[1].MutationVariables["same"] = true }},
		{name: "verification", forge: func(plan *d.ProcessInstanceVariableUpdatePlan) { plan.Instances[1].VerificationValues["same"] = false }},
		{name: "classification", forge: func(plan *d.ProcessInstanceVariableUpdatePlan) {
			plan.Instances[1].UnchangedRequested[0].APITruncated = true
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &processInstanceVariableExecutionAPI{}
			plan := processInstanceExecutionPlan()
			tt.forge(&plan)

			results, err := NewVariableUpdates(nil, api, config.New(), nil).
				ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1, services.WithNoWait())

			require.ErrorIs(t, err, d.ErrValidation)
			require.Empty(t, results.Items)
			require.Empty(t, api.writes)
		})
	}
}

// TestExecuteProcessInstanceVariableUpdatesWaitsOnceForAllRequestedValues verifies
// accepted writes use one waiter workflow whose complete reads include initially unchanged values.
func TestExecuteProcessInstanceVariableUpdatesWaitsOnceForAllRequestedValues(t *testing.T) {
	t.Parallel()

	api := &processInstanceVariableExecutionAPI{search: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		return []d.ProcessInstanceVariable{
			{Name: "add", Value: `{"nested":[1]}`, ProcessInstanceKey: key, ScopeKey: key},
			{Name: "change", Value: `"after"`, ProcessInstanceKey: key, ScopeKey: key},
			{Name: "same", Value: `true`, ProcessInstanceKey: key, ScopeKey: key},
		}, nil
	}}
	cfg := config.New()
	cfg.App.Backoff.InitialDelay = time.Millisecond
	cfg.App.Backoff.MaxRetries = 1
	results, err := NewVariableUpdates(nil, api, cfg, nil).
		ExecuteProcessInstanceVariableUpdates(context.Background(), processInstanceExecutionPlan(), 1)

	require.NoError(t, err)
	require.Equal(t, []string{"pi-change"}, api.searches)
	require.Equal(t, d.ProcessInstanceVariableUpdateStatusConfirmed, results.Items[1].Status)
	require.Equal(t, "confirmed", results.Items[1].ConfirmationStatus)
}

// TestExecuteProcessInstanceVariableUpdatesMarksUnstartedFailFastWorkSkipped verifies
// failed, skipped, and already unchanged outcomes retain selected-key order.
func TestExecuteProcessInstanceVariableUpdatesMarksUnstartedFailFastWorkSkipped(t *testing.T) {
	t.Parallel()

	api := &processInstanceVariableExecutionAPI{write: func(_ context.Context, scope string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
		if scope == "pi-change" {
			return d.ScopeVariableUpdateResponse{ScopeKey: scope, StatusCode: 500, Status: "rejected"}, d.ErrUnavailable
		}
		return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true}, nil
	}}
	plan := processInstanceExecutionPlan()
	plan.RequestedKeys = append(plan.RequestedKeys, "pi-later")
	plan.RequestedCount++
	plan.UpdateCount++
	plan.VariableAddCount++
	plan.VariableChangeCount++
	plan.VariableUnchangedCount++
	plan.Instances = append(plan.Instances, d.ProcessInstanceVariablePlan{
		ProcessInstanceKey: "pi-later", ScopeKey: "pi-later",
		Additions: []d.ProcessInstanceVariablePlannedValue{{Name: "add", Value: plan.RequestedValues["add"]}},
		Changes: []d.ProcessInstanceVariablePlannedChange{
			{Name: "change", Before: "before", After: "after"},
		},
		UnchangedRequested: []d.ProcessInstanceVariablePlannedValue{{Name: "same", Value: true}},
		MutationVariables:  map[string]any{"add": plan.RequestedValues["add"], "change": "after"},
		VerificationValues: copyProcessInstanceVariableMap(plan.RequestedValues),
	})

	results, err := NewVariableUpdates(nil, api, config.New(), nil).
		ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1, services.WithNoWait(), services.WithFailFast())

	require.ErrorIs(t, err, d.ErrUnavailable)
	require.Equal(t, []d.ProcessInstanceVariableUpdateStatus{
		d.ProcessInstanceVariableUpdateStatusUnchanged,
		d.ProcessInstanceVariableUpdateStatusMutationFailed,
		d.ProcessInstanceVariableUpdateStatusSkipped,
	}, []d.ProcessInstanceVariableUpdateStatus{results.Items[0].Status, results.Items[1].Status, results.Items[2].Status})
}

// TestExecuteProcessInstanceVariableUpdatesPreservesAcceptedConfirmationFailure verifies
// waiter errors do not erase evidence that the mutation was accepted.
func TestExecuteProcessInstanceVariableUpdatesPreservesAcceptedConfirmationFailure(t *testing.T) {
	t.Parallel()

	api := &processInstanceVariableExecutionAPI{search: func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		return nil, errors.New("verification unavailable")
	}}
	results, err := NewVariableUpdates(nil, api, config.New(), nil).
		ExecuteProcessInstanceVariableUpdates(context.Background(), processInstanceExecutionPlan(), 1)

	require.Error(t, err)
	require.Equal(t, d.ProcessInstanceVariableUpdateStatusConfirmationFailed, results.Items[1].Status)
	require.True(t, results.Items[1].MutationAccepted)
	require.Equal(t, "failed", results.Items[1].ConfirmationStatus)
}

// TestExecuteProcessInstanceVariableUpdatesAcceptsPlannerNumericEvidence verifies
// a mixed planner-produced plan retains precise unchanged and nested numeric values.
func TestExecuteProcessInstanceVariableUpdatesAcceptsPlannerNumericEvidence(t *testing.T) {
	t.Parallel()

	const key = "pi-precise"
	instances := &processInstanceVariablePlanningAPI{instances: []d.ProcessInstance{{Key: key}}}
	api := &processInstanceVariableExecutionAPI{}
	variables := &processInstanceRootVariablePlanningAPI{variables: map[string][]d.ProcessInstanceVariable{
		key: {
			{Name: "exact", Value: `9007199254740993`, ProcessInstanceKey: key, ScopeKey: key},
			{Name: "mismatch", Value: `9007199254740993`, ProcessInstanceKey: key, ScopeKey: key},
			{Name: "nested", Value: `{"value":9007199254740993}`, ProcessInstanceKey: key, ScopeKey: key},
		},
	}}
	service := NewVariableUpdates(instances, variables, config.New(), nil)
	plan, err := service.PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{key}, map[string]any{
		"exact":    json.Number("9007199254740993"),
		"mismatch": json.Number("9007199254740992"),
		"nested":   map[string]any{"value": json.Number("9007199254740993")},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"exact", "nested"}, processInstancePlannedValueNames(plan.Instances[0].UnchangedRequested))
	require.Equal(t, []string{"mismatch"}, processInstancePlannedChangeNames(plan.Instances[0].Changes))
	require.Equal(t, json.Number("9007199254740993"), plan.Instances[0].UnchangedRequested[0].Value)
	require.Equal(t, json.Number("9007199254740993"), plan.Instances[0].UnchangedRequested[1].Value.(map[string]any)["value"])

	executor := NewVariableUpdates(nil, api, config.New(), nil)
	results, err := executor.ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1, services.WithNoWait())

	require.NoError(t, err)
	require.Len(t, api.writes, 1)
	require.Equal(t, map[string]any{"mismatch": json.Number("9007199254740992")}, api.writes[0].variables)
	require.Equal(t, d.ProcessInstanceVariableUpdateStatusSubmitted, results.Items[0].Status)
}

// processInstanceExecutionPlan builds one unchanged and one mixed changed root.
func processInstanceExecutionPlan() d.ProcessInstanceVariableUpdatePlan {
	requested := map[string]any{"add": map[string]any{"nested": []any{float64(1)}}, "change": "after", "same": true}
	return d.ProcessInstanceVariableUpdatePlan{
		RequestedKeys: []string{"pi-unchanged", "pi-change"}, RequestedValues: copyProcessInstanceVariableMap(requested),
		RequestedCount: 2, UpdateCount: 1, VariableAddCount: 1, VariableChangeCount: 1, VariableUnchangedCount: 4,
		Instances: []d.ProcessInstanceVariablePlan{
			{ProcessInstanceKey: "pi-unchanged", ScopeKey: "pi-unchanged",
				UnchangedRequested: []d.ProcessInstanceVariablePlannedValue{{Name: "add", Value: requested["add"]}, {Name: "change", Value: "after"}, {Name: "same", Value: true}},
				VerificationValues: copyProcessInstanceVariableMap(requested)},
			{ProcessInstanceKey: "pi-change", ScopeKey: "pi-change",
				Additions:          []d.ProcessInstanceVariablePlannedValue{{Name: "add", Value: requested["add"]}},
				Changes:            []d.ProcessInstanceVariablePlannedChange{{Name: "change", Before: "before", After: "after"}},
				UnchangedRequested: []d.ProcessInstanceVariablePlannedValue{{Name: "same", Value: true}},
				MutationVariables:  map[string]any{"add": requested["add"], "change": "after"}, VerificationValues: copyProcessInstanceVariableMap(requested)},
		},
	}
}
