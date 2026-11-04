// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

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

// TestElementVariableUpdatesPlanLocalScope proves ordered deduplication, independent
// owner/scope, local-only classifications, tenant binding, and isolated frozen values.
func TestElementVariableUpdatesPlanLocalScope(t *testing.T) {
	t.Parallel()
	gets, reads := []string{}, []string{}
	input := map[string]any{"add": map[string]any{"nested": []any{float64(1)}}, "change": "new", "equal": float64(1), "truncated": "visible", "null": nil}
	keys := typex.Keys{"E2", "E1", "E2"}
	cfg := config.New()
	cfg.App.Tenant = "discovery"
	api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, opts ...services.CallOption) (d.Element, error) {
		gets = append(gets, key)
		require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
		require.True(t, services.ApplyCallOptions(opts).FailFast)
		return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P", TenantId: "actual-" + key}, nil
	}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		reads = append(reads, q.ScopeKey)
		require.Equal(t, "P", q.ProcessInstanceKey)
		require.Equal(t, "actual-"+q.ScopeKey, *q.ExpectedTenantId)
		require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
		require.True(t, services.ApplyCallOptions(opts).FailFast)
		return []d.ProcessInstanceVariable{
			{Name: "add", Value: `{"nested":[1]}`, ProcessInstanceKey: "P", ScopeKey: "parent"},
			{Name: "change", Value: `"old"`, ProcessInstanceKey: "P", ScopeKey: q.ScopeKey},
			{Name: "equal", Value: `1`, ProcessInstanceKey: "P", ScopeKey: q.ScopeKey},
			{Name: "truncated", Value: `"visible"`, APITruncated: true, ProcessInstanceKey: "P", ScopeKey: q.ScopeKey},
			{Name: "untouched", Value: `{"keep":[true]}`, ProcessInstanceKey: "P", ScopeKey: q.ScopeKey},
		}, nil
	}), cfg)
	plan, err := api.PlanElementVariableUpdates(context.Background(), keys, input, services.WithFailFast())
	require.NoError(t, err)
	require.Equal(t, []string{"E2", "E1"}, plan.RequestedKeys)
	require.Equal(t, plan.RequestedKeys, gets)
	require.Equal(t, gets, reads)
	require.Equal(t, 2, plan.RequestedCount)
	require.Equal(t, 2, plan.UpdateCount)
	require.Equal(t, 4, plan.VariableAddCount)
	require.Equal(t, 4, plan.VariableChangeCount)
	require.Equal(t, 2, plan.VariableUnchangedCount)
	require.Equal(t, 2, plan.VariableUntouchedCount)
	require.False(t, plan.MutationSubmitted)
	require.Equal(t, d.TenantContextModeExplicitKeys, plan.TenantContext.Mode)
	require.Equal(t, d.TenantContextFilterNotApplied, plan.TenantContext.Filter)
	require.Equal(t, "discovery", plan.TenantContext.ConfiguredTenantID)
	require.Equal(t, []string{"actual-E1", "actual-E2"}, plan.TenantContext.ResolvedTenantIDs)
	require.True(t, plan.TenantContext.CrossTenant)
	first := plan.Elements[0]
	require.Equal(t, "E2", first.ElementKey)
	require.Equal(t, "E2", first.ScopeKey)
	require.Equal(t, "P", first.ProcessInstanceKey)
	require.Equal(t, []d.ElementVariablePlannedValue{{Name: "add", Value: input["add"]}, {Name: "null", Value: nil}}, first.Additions)
	require.Equal(t, []d.ElementVariablePlannedChange{{Name: "change", Before: "old", After: "new"}, {Name: "truncated", Before: "visible", After: "visible", APITruncated: true}}, first.Changes)
	require.Equal(t, []d.ElementVariablePlannedValue{{Name: "equal", Value: json.Number("1")}}, first.UnchangedRequested)
	require.Equal(t, input, first.VerificationValues)
	require.NotContains(t, first.MutationVariables, "equal")
	require.NotContains(t, first.MutationVariables, "untouched")
	input["add"].(map[string]any)["nested"].([]any)[0] = float64(99)
	keys[0] = "changed"
	for _, values := range []map[string]any{plan.RequestedValues, first.MutationVariables, first.VerificationValues, plan.Elements[1].MutationVariables} {
		require.Equal(t, float64(1), values["add"].(map[string]any)["nested"].([]any)[0])
	}
	first.MutationVariables["add"].(map[string]any)["nested"].([]any)[0] = float64(55)
	require.Equal(t, float64(1), first.Additions[0].Value.(map[string]any)["nested"].([]any)[0])
	require.Equal(t, float64(1), first.VerificationValues["add"].(map[string]any)["nested"].([]any)[0])
}

// TestElementVariableUpdatesPlanNoOp still authorizes and reads every target for
// empty objects and unchanged values, retaining allocated classification arrays.
func TestElementVariableUpdatesPlanNoOp(t *testing.T) {
	t.Parallel()
	for _, input := range []map[string]any{{}, {"same": nil}} {
		gets, reads := 0, 0
		api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
			gets++
			return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P"}, nil
		}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			reads++
			require.Nil(t, q.ExpectedTenantId)
			return []d.ProcessInstanceVariable{{Name: "same", Value: `null`, ProcessInstanceKey: "P", ScopeKey: q.ScopeKey}}, nil
		}), nil)
		plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, input)
		require.NoError(t, err)
		require.Equal(t, 1, gets)
		require.Equal(t, 1, reads)
		require.Zero(t, plan.UpdateCount)
		require.Equal(t, 1, plan.TenantContext.UnknownTargetCount)
		require.NotNil(t, plan.RequestedValues)
		require.NotNil(t, plan.Elements[0].Additions)
		require.NotNil(t, plan.Elements[0].Changes)
		require.NotNil(t, plan.Elements[0].UnchangedRequested)
		require.NotNil(t, plan.Elements[0].Untouched)
		_, err = validateAndCopyElementVariableUpdatePlan(plan)
		require.NoError(t, err)
	}
}

// TestElementVariableUpdatesPlanInvalidInput rejects invalid selections and
// payloads before any remote access, including null versus empty objects.
func TestElementVariableUpdatesPlanInvalidInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		keys   typex.Keys
		values map[string]any
	}{
		{nil, map[string]any{}}, {typex.Keys{" "}, map[string]any{}}, {typex.Keys{"E"}, nil},
		{typex.Keys{"E"}, map[string]any{" ": true}}, {typex.Keys{"E"}, map[string]any{"x": make(chan int)}},
		{typex.Keys{"E"}, map[string]any{"x": json.RawMessage(`{"broken":`)}},
	}
	for _, tc := range cases {
		plan, err := NewVariableUpdates(nil, nil, nil).PlanElementVariableUpdates(context.Background(), tc.keys, tc.values)
		require.ErrorIs(t, err, d.ErrValidation)
		require.Equal(t, d.ElementVariableUpdatePlan{}, plan)
	}
}

// TestElementVariableUpdatesPlanLateFailure never publishes a partial plan when
// a later identity, authorization, complete read, or observation fails.
func TestElementVariableUpdatesPlanLateFailure(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name            string
		element         d.Element
		observations    []d.ProcessInstanceVariable
		getErr, readErr error
	}{
		{name: "missing", getErr: d.ErrNotFound}, {name: "denied", getErr: d.ErrUnauthorized},
		{name: "forbidden read", readErr: d.ErrForbidden}, {name: "incomplete read", readErr: d.ErrMalformedResponse},
		{name: "wrong element", element: d.Element{ElementInstanceKey: "other", ProcessInstanceKey: "P"}},
		{name: "missing owner", element: d.Element{ElementInstanceKey: "E2"}},
		{name: "wrong owner", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "other"}}},
		{name: "missing scope", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ProcessInstanceKey: "P"}}},
		{name: "missing owner observation", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ScopeKey: "E2"}}},
		{name: "blank name", observations: []d.ProcessInstanceVariable{{Name: " ", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P"}}},
		{name: "invalid JSON", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `oops`, ScopeKey: "E2", ProcessInstanceKey: "P"}}},
		{name: "tenant conflict", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P", TenantId: "wrong"}}},
		{name: "duplicate value", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P"}, {Name: "x", Value: `2`, ScopeKey: "E2", ProcessInstanceKey: "P"}}},
		{name: "duplicate key", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P", VariableKey: "v1"}, {Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P", VariableKey: "v2"}}},
		{name: "duplicate completeness", observations: []d.ProcessInstanceVariable{{Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P"}, {Name: "x", Value: `1`, ScopeKey: "E2", ProcessInstanceKey: "P", APITruncated: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
				calls = append(calls, key)
				if key == "E2" {
					if tc.getErr != nil {
						return d.Element{}, tc.getErr
					}
					if tc.element != (d.Element{}) {
						return tc.element, nil
					}
				}
				return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P", TenantId: "known"}, nil
			}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				if q.ScopeKey == "E2" {
					return tc.observations, tc.readErr
				}
				return nil, nil
			}), nil)
			plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E1", "E2"}, map[string]any{"x": 1})
			require.Error(t, err)
			require.Equal(t, d.ElementVariableUpdatePlan{}, plan)
			require.Equal(t, []string{"E1", "E2"}, calls)
			if tc.getErr != nil {
				require.ErrorIs(t, err, tc.getErr)
			}
			if tc.readErr != nil {
				require.ErrorIs(t, err, tc.readErr)
			}
		})
	}
}

// TestElementVariableUpdatesPlanCancellationAndDependencies verifies failures
// before and after reads cannot escape as successful frozen plans.
func TestElementVariableUpdatesPlanCancellationAndDependencies(t *testing.T) {
	t.Parallel()
	for _, api := range []VariableUpdateAPI{NewVariableUpdates(nil, nil, nil), NewVariableUpdates(stubElementAPI{}, nil, nil)} {
		_, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{})
		require.ErrorIs(t, err, d.ErrPrecondition)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewVariableUpdates(stubElementAPI{get: func(context.Context, string, ...services.CallOption) (d.Element, error) {
		t.Fatal("canceled planning performed a lookup")
		return d.Element{}, nil
	}}, scopeReaderFunc(nil), nil).PlanElementVariableUpdates(ctx, typex.Keys{"E"}, map[string]any{})
	require.ErrorIs(t, err, context.Canceled)
	ctx, cancel = context.WithCancel(context.Background())
	api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
		return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P"}, nil
	}}, scopeReaderFunc(func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		cancel()
		return nil, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(ctx, typex.Keys{"E"}, map[string]any{})
	require.True(t, errors.Is(err, context.Canceled))
	require.Equal(t, d.ElementVariableUpdatePlan{}, plan)
}

// TestElementVariableUpdatesPlanOptionalEvidence preserves unknown evidence,
// learns known tenant/key facts, and detects conflicts across three duplicate rows.
func TestElementVariableUpdatesPlanOptionalEvidence(t *testing.T) {
	t.Parallel()
	for _, conflict := range []string{"", "key", "tenant"} {
		t.Run(conflict, func(t *testing.T) {
			rows := []d.ProcessInstanceVariable{
				{Name: "value", Value: `1`, ProcessInstanceKey: "P", ScopeKey: "E"},
				{Name: "value", Value: `1`, ProcessInstanceKey: "P", ScopeKey: "E", VariableKey: "v1", TenantId: "learned"},
				{Name: "value", Value: `1`, ProcessInstanceKey: "P", ScopeKey: "E", VariableKey: "v1", TenantId: "learned"},
			}
			if conflict == "key" {
				rows[2].VariableKey = "v2"
			}
			if conflict == "tenant" {
				rows[2].TenantId = "different"
			}
			api := NewVariableUpdates(stubElementAPI{get: func(context.Context, string, ...services.CallOption) (d.Element, error) {
				return d.Element{ElementInstanceKey: "E", ProcessInstanceKey: "P"}, nil
			}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				require.Nil(t, q.ExpectedTenantId)
				return rows, nil
			}), nil)
			plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{"value": 1})
			if conflict != "" {
				require.ErrorIs(t, err, d.ErrValidation)
				require.Equal(t, d.ElementVariableUpdatePlan{}, plan)
				return
			}
			require.NoError(t, err)
			require.Zero(t, plan.UpdateCount)
			require.Equal(t, "learned", plan.Elements[0].TenantId)
			require.Equal(t, []string{"learned"}, plan.TenantContext.ResolvedTenantIDs)
			require.Zero(t, plan.TenantContext.UnknownTargetCount)
		})
	}
}

// TestElementVariableUpdatesPlanNumericAndTruncatedEvidence retains existing
// number-spelling comparisons and incomplete raw fragments without false equality.
func TestElementVariableUpdatesPlanNumericAndTruncatedEvidence(t *testing.T) {
	t.Parallel()
	api := NewVariableUpdates(stubElementAPI{get: func(context.Context, string, ...services.CallOption) (d.Element, error) {
		return d.Element{ElementInstanceKey: "E", ProcessInstanceKey: "P"}, nil
	}}, scopeReaderFunc(func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		return []d.ProcessInstanceVariable{
			{Name: "integer", Value: `9007199254740993`, ProcessInstanceKey: "P", ScopeKey: "E"},
			{Name: "spelling", Value: `1.0`, ProcessInstanceKey: "P", ScopeKey: "E"},
			{Name: "partial", Value: `{"large":`, APITruncated: true, ProcessInstanceKey: "P", ScopeKey: "E"},
			{Name: "untouched", Value: `{"partial":`, APITruncated: true, ProcessInstanceKey: "P", ScopeKey: "E"},
		}, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{"integer": json.Number("9007199254740993"), "spelling": float64(1), "partial": map[string]any{"large": true}})
	require.NoError(t, err)
	require.Equal(t, 1, plan.VariableUnchangedCount)
	require.Equal(t, 2, plan.VariableChangeCount)
	require.Equal(t, `{"large":`, plan.Elements[0].Changes[0].Before)
	require.True(t, plan.Elements[0].Changes[0].APITruncated)
	require.Equal(t, `{"partial":`, plan.Elements[0].Untouched[0].Value)
	require.True(t, plan.Elements[0].Untouched[0].APITruncated)
}
