// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type processInstanceTypedValues map[string]processInstanceTypedList

type processInstanceTypedList []string

type processInstanceVariablePlanningAPI struct {
	instances []d.ProcessInstance
	err       error
	keys      typex.Keys
	options   *services.CallCfg
}

// GetProcessInstances records stable selection and returns controlled identity facts.
func (a *processInstanceVariablePlanningAPI) GetProcessInstances(_ context.Context, keys typex.Keys, _ int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	a.keys = append(typex.Keys(nil), keys...)
	a.options = services.ApplyCallOptions(opts)
	return append([]d.ProcessInstance(nil), a.instances...), a.err
}

type processInstanceRootVariablePlanningAPI struct {
	variables map[string][]d.ProcessInstanceVariable
	errors    map[string]error
	calls     []string
}

// SearchProcessInstanceVariables records complete root reads and returns fixtures.
func (a *processInstanceRootVariablePlanningAPI) SearchProcessInstanceVariables(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	a.calls = append(a.calls, key)
	if err := a.errors[key]; err != nil {
		return nil, err
	}
	return append([]d.ProcessInstanceVariable(nil), a.variables[key]...), nil
}

// TestPlanProcessInstanceVariableUpdatesBuildsCompleteRootPlan verifies stable
// deduplication, root isolation, classifications, truncation, counts, and tenant evidence.
func TestPlanProcessInstanceVariableUpdatesBuildsCompleteRootPlan(t *testing.T) {
	t.Parallel()

	instances := &processInstanceVariablePlanningAPI{instances: []d.ProcessInstance{
		{Key: "pi-b", TenantId: "tenant-b"},
		{Key: "pi-a", TenantId: "tenant-a"},
	}}
	variables := &processInstanceRootVariablePlanningAPI{variables: map[string][]d.ProcessInstanceVariable{
		"pi-b": {
			{Name: "equal", Value: `{"count":1}`, VariableKey: "v-equal", ProcessInstanceKey: "pi-b", ScopeKey: "pi-b", TenantId: "tenant-b"},
			{Name: "change", Value: `"before"`, VariableKey: "v-change", ProcessInstanceKey: "pi-b", ScopeKey: "pi-b", TenantId: "tenant-b"},
			{Name: "truncated", Value: `{"large":`, VariableKey: "v-truncated", ProcessInstanceKey: "pi-b", ScopeKey: "pi-b", TenantId: "tenant-b", APITruncated: true},
			{Name: "untouched", Value: `false`, VariableKey: "v-untouched", ProcessInstanceKey: "pi-b", ScopeKey: "pi-b", TenantId: "tenant-b"},
			{Name: "equal", Value: `"descendant"`, VariableKey: "v-child", ProcessInstanceKey: "pi-b", ScopeKey: "child-b", TenantId: "tenant-b"},
		},
		"pi-a": {
			{Name: "equal", Value: `{"count":2}`, VariableKey: "v-equal-a", ProcessInstanceKey: "pi-a", ScopeKey: "pi-a", TenantId: "tenant-a"},
		},
	}}
	cfg := config.New()
	cfg.App.Tenant = "configured-not-filtering"
	input := map[string]any{
		"addition":  map[string]any{"nested": []any{float64(1)}},
		"change":    "after",
		"equal":     map[string]any{"count": 1},
		"truncated": map[string]any{"large": true},
	}

	plan, err := NewVariableUpdates(instances, variables, cfg, nil).PlanProcessInstanceVariableUpdates(
		context.Background(), typex.Keys{"pi-b", "pi-a", "pi-b"}, input, services.WithFailFast(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{"pi-b", "pi-a"}, plan.RequestedKeys)
	require.Equal(t, typex.Keys{"pi-b", "pi-a"}, instances.keys)
	require.True(t, instances.options.FailFast)
	require.Equal(t, []string{"pi-b", "pi-a"}, variables.calls)
	require.Equal(t, 2, plan.RequestedCount)
	require.Equal(t, 2, plan.UpdateCount)
	require.Equal(t, 4, plan.VariableAddCount)
	require.Equal(t, 3, plan.VariableChangeCount)
	require.Equal(t, 1, plan.VariableUnchangedCount)
	require.Equal(t, 1, plan.VariableUntouchedCount)
	require.False(t, plan.MutationSubmitted)
	require.Equal(t, d.TenantContextModeExplicitKeys, plan.TenantContext.Mode)
	require.Equal(t, "configured-not-filtering", plan.TenantContext.ConfiguredTenantID)
	require.Equal(t, []string{"tenant-a", "tenant-b"}, plan.TenantContext.ResolvedTenantIDs)
	require.True(t, plan.TenantContext.CrossTenant)

	require.Len(t, plan.Instances, 2)
	first := plan.Instances[0]
	require.Equal(t, "pi-b", first.ProcessInstanceKey)
	require.Equal(t, "pi-b", first.ScopeKey)
	require.Equal(t, []string{"addition"}, processInstancePlannedValueNames(first.Additions))
	require.Equal(t, []string{"change", "truncated"}, processInstancePlannedChangeNames(first.Changes))
	require.Equal(t, []string{"equal"}, processInstancePlannedValueNames(first.UnchangedRequested))
	require.Equal(t, []string{"untouched"}, processInstancePlannedValueNames(first.Untouched))
	require.True(t, first.Changes[1].APITruncated)
	require.Equal(t, `{"large":`, first.Changes[1].Before)
	require.Equal(t, map[string]any{
		"addition": map[string]any{"nested": []any{float64(1)}},
		"change":   "after", "truncated": map[string]any{"large": true},
	}, first.MutationVariables)
	require.Equal(t, input, first.VerificationValues)

	input["addition"].(map[string]any)["nested"].([]any)[0] = float64(99)
	require.Equal(t, float64(1), plan.RequestedValues["addition"].(map[string]any)["nested"].([]any)[0])
	require.Equal(t, float64(1), first.MutationVariables["addition"].(map[string]any)["nested"].([]any)[0])
	require.Equal(t, float64(1), first.VerificationValues["addition"].(map[string]any)["nested"].([]any)[0])
}

// TestPlanProcessInstanceVariableUpdatesSnapshotsTypedAndRawJSON verifies
// accepted typed collections and raw JSON cannot be changed through caller aliases.
func TestPlanProcessInstanceVariableUpdatesSnapshotsTypedAndRawJSON(t *testing.T) {
	t.Parallel()

	const key = "pi-snapshot"
	typed := processInstanceTypedValues{"items": {"original"}}
	raw := json.RawMessage(`{"raw":"original"}`)
	instances := &processInstanceVariablePlanningAPI{instances: []d.ProcessInstance{{Key: key}}}
	variables := &processInstanceRootVariablePlanningAPI{variables: map[string][]d.ProcessInstanceVariable{key: {}}}

	plan, err := NewVariableUpdates(instances, variables, config.New(), nil).
		PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{key}, map[string]any{"typed": typed, "raw": raw})

	require.NoError(t, err)
	typed["items"][0] = "mutated"
	raw[8] = 'X'
	require.Equal(t, "original", plan.RequestedValues["typed"].(processInstanceTypedValues)["items"][0])
	require.JSONEq(t, `{"raw":"original"}`, string(plan.RequestedValues["raw"].(json.RawMessage)))
	require.Equal(t, "original", plan.Instances[0].Additions[1].Value.(processInstanceTypedValues)["items"][0])
	require.JSONEq(t, `{"raw":"original"}`, string(plan.Instances[0].MutationVariables["raw"].(json.RawMessage)))
	require.Equal(t, "original", plan.Instances[0].VerificationValues["typed"].(processInstanceTypedValues)["items"][0])
}

// TestPlanProcessInstanceVariableUpdatesReturnsInitializedEmptyPlan verifies
// valid service-level no-work input does not require backend dependencies.
func TestPlanProcessInstanceVariableUpdatesReturnsInitializedEmptyPlan(t *testing.T) {
	t.Parallel()

	plan, err := NewVariableUpdates(nil, nil, config.New(), nil).
		PlanProcessInstanceVariableUpdates(context.Background(), nil, map[string]any{"value": []any{"x"}})

	require.NoError(t, err)
	require.NotNil(t, plan.RequestedKeys)
	require.NotNil(t, plan.Instances)
	require.Equal(t, map[string]any{"value": []any{"x"}}, plan.RequestedValues)
}

// TestPlanProcessInstanceVariableUpdatesRejectsInvalidInputsBeforeReads verifies
// blank keys, blank names, and non-JSON values fail before remote discovery.
func TestPlanProcessInstanceVariableUpdatesRejectsInvalidInputsBeforeReads(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keys      typex.Keys
		variables map[string]any
	}{
		{name: "blank key", keys: typex.Keys{" "}, variables: map[string]any{"value": true}},
		{name: "blank name", keys: typex.Keys{"pi-a"}, variables: map[string]any{" ": true}},
		{name: "invalid value", keys: typex.Keys{"pi-a"}, variables: map[string]any{"value": make(chan int)}},
		{name: "invalid raw JSON", keys: typex.Keys{"pi-a"}, variables: map[string]any{"value": json.RawMessage(`{"broken":`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instances := &processInstanceVariablePlanningAPI{}
			plan, err := NewVariableUpdates(instances, &processInstanceRootVariablePlanningAPI{}, config.New(), nil).
				PlanProcessInstanceVariableUpdates(context.Background(), tt.keys, tt.variables)

			require.ErrorIs(t, err, d.ErrValidation)
			require.Equal(t, d.ProcessInstanceVariableUpdatePlan{}, plan)
			require.Nil(t, instances.keys)
		})
	}
}

// TestPlanProcessInstanceVariableUpdatesRejectsPartialOrConflictingDiscovery
// verifies final-target failures and duplicate conflicts expose no executable plan.
func TestPlanProcessInstanceVariableUpdatesRejectsPartialOrConflictingDiscovery(t *testing.T) {
	t.Parallel()

	t.Run("last target fails", func(t *testing.T) {
		instances := &processInstanceVariablePlanningAPI{instances: []d.ProcessInstance{{Key: "pi-a"}, {Key: "pi-b"}}}
		variables := &processInstanceRootVariablePlanningAPI{
			variables: map[string][]d.ProcessInstanceVariable{"pi-a": {{Name: "value", Value: `1`, ProcessInstanceKey: "pi-a", ScopeKey: "pi-a"}}},
			errors:    map[string]error{"pi-b": d.ErrUnavailable},
		}

		plan, err := NewVariableUpdates(instances, variables, config.New(), nil).
			PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{"pi-a", "pi-b"}, map[string]any{"value": 2})

		require.ErrorIs(t, err, d.ErrUnavailable)
		require.Equal(t, d.ProcessInstanceVariableUpdatePlan{}, plan)
		require.Equal(t, []string{"pi-a", "pi-b"}, variables.calls)
	})

	t.Run("conflicting duplicate", func(t *testing.T) {
		instances := &processInstanceVariablePlanningAPI{instances: []d.ProcessInstance{{Key: "pi-a"}}}
		variables := &processInstanceRootVariablePlanningAPI{variables: map[string][]d.ProcessInstanceVariable{
			"pi-a": {
				{Name: "value", Value: `1`, VariableKey: "v-1", ProcessInstanceKey: "pi-a", ScopeKey: "pi-a"},
				{Name: "value", Value: `2`, VariableKey: "v-2", ProcessInstanceKey: "pi-a", ScopeKey: "pi-a"},
			},
		}}

		plan, err := NewVariableUpdates(instances, variables, config.New(), nil).
			PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{"pi-a"}, map[string]any{"value": 2})

		require.ErrorIs(t, err, d.ErrValidation)
		require.Equal(t, d.ProcessInstanceVariableUpdatePlan{}, plan)
	})
}

// TestPlanProcessInstanceVariableUpdatesChecksDependencies verifies legacy
// compositions fail clearly only when a nonempty new planning operation is used.
func TestPlanProcessInstanceVariableUpdatesChecksDependencies(t *testing.T) {
	t.Parallel()

	_, err := NewVariableUpdates(nil, nil, config.New(), nil).
		PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{"pi-a"}, map[string]any{"value": true})
	require.ErrorIs(t, err, d.ErrPrecondition)

	instances := &processInstanceVariablePlanningAPI{instances: []d.ProcessInstance{{Key: "pi-a"}}}
	_, err = NewVariableUpdates(instances, nil, config.New(), nil).
		PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{"pi-a"}, map[string]any{"value": true})
	require.ErrorIs(t, err, d.ErrPrecondition)
}

// TestPlanProcessInstanceVariableUpdatesPropagatesInstanceReadFailure verifies
// collection identity discovery errors remain classified and produce no plan.
func TestPlanProcessInstanceVariableUpdatesPropagatesInstanceReadFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("lookup failed")
	plan, err := NewVariableUpdates(&processInstanceVariablePlanningAPI{err: wantErr}, &processInstanceRootVariablePlanningAPI{}, config.New(), nil).
		PlanProcessInstanceVariableUpdates(context.Background(), typex.Keys{"pi-a"}, map[string]any{"value": true})

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, d.ProcessInstanceVariableUpdatePlan{}, plan)
}

// processInstancePlannedValueNames extracts deterministic value-category names.
func processInstancePlannedValueNames(values []d.ProcessInstanceVariablePlannedValue) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	return names
}

// processInstancePlannedChangeNames extracts deterministic change-category names.
func processInstancePlannedChangeNames(values []d.ProcessInstanceVariablePlannedChange) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	return names
}
