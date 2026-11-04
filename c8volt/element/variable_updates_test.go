// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// elementPlannerFunc isolates the facade contract from backend mechanics.
type elementPlannerFunc func(context.Context, typex.Keys, map[string]any, ...services.CallOption) (d.ElementVariableUpdatePlan, error)

// PlanElementVariableUpdates supplies a test-owned planning response.
func (f elementPlannerFunc) PlanElementVariableUpdates(ctx context.Context, keys typex.Keys, values map[string]any, opts ...services.CallOption) (d.ElementVariableUpdatePlan, error) {
	return f(ctx, keys, values, opts...)
}

// TestElementVariableUpdatesPlanFacade proves one delegation, option mapping,
// complete field conversion, nested isolation, and explicit empty JSON arrays.
func TestElementVariableUpdatesPlanFacade(t *testing.T) {
	t.Parallel()
	input := map[string]any{"x": map[string]any{"list": []any{float64(1)}}}
	keys := typex.Keys{"E", "E"}
	base, _ := d.NewExplicitKeysTenantContext("configured")
	tenant, _ := d.WithTenantEvidence(base, []string{"a", "b"}, 0)
	response := d.ElementVariableUpdatePlan{
		RequestedKeys: []string{"E"}, RequestedValues: map[string]any{"x": []any{true}},
		Elements: []d.ElementVariablePlan{{ElementKey: "E", ProcessInstanceKey: "P", ScopeKey: "E", TenantId: "a",
			Additions:          []d.ElementVariablePlannedValue{{Name: "x", Value: map[string]any{"list": []any{true}}}},
			Changes:            []d.ElementVariablePlannedChange{{Name: "c", Before: []any{false}, After: []any{true}, APITruncated: true}},
			UnchangedRequested: []d.ElementVariablePlannedValue{{Name: "u", Value: []any{true}}},
			Untouched:          []d.ElementVariablePlannedValue{{Name: "t", Value: []any{false}, APITruncated: true}},
			MutationVariables:  map[string]any{"x": []any{true}}, VerificationValues: map[string]any{"x": []any{true}},
		}}, RequestedCount: 1, UpdateCount: 1, VariableAddCount: 1, VariableChangeCount: 1, VariableUnchangedCount: 1, VariableUntouchedCount: 1, TenantContext: tenant,
	}
	calls := 0
	api := NewWithVariableUpdates(nil, nil, nil, elementPlannerFunc(func(_ context.Context, gotKeys typex.Keys, gotValues map[string]any, opts ...services.CallOption) (d.ElementVariableUpdatePlan, error) {
		calls++
		require.Equal(t, keys, gotKeys)
		require.Equal(t, input, gotValues)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.IgnoreTenant)
		require.True(t, cfg.FailFast)
		gotKeys[0] = "mutated"
		gotValues["x"].(map[string]any)["list"].([]any)[0] = float64(9)
		return response, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), keys, input, foptions.WithIgnoreTenant(), foptions.WithFailFast())
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, "E", keys[0])
	require.Equal(t, float64(1), input["x"].(map[string]any)["list"].([]any)[0])
	require.Equal(t, 1, plan.RequestedCount)
	require.Equal(t, 1, plan.UpdateCount)
	require.Equal(t, 1, plan.VariableAddCount)
	require.Equal(t, 1, plan.VariableChangeCount)
	require.Equal(t, 1, plan.VariableUnchangedCount)
	require.Equal(t, 1, plan.VariableUntouchedCount)
	require.False(t, plan.MutationSubmitted)
	require.Equal(t, "E", plan.Elements[0].ElementKey)
	require.Equal(t, "P", plan.Elements[0].ProcessInstanceKey)
	require.Equal(t, "E", plan.Elements[0].ScopeKey)
	require.Equal(t, "a", plan.Elements[0].TenantId)
	require.True(t, plan.Elements[0].Changes[0].APITruncated)
	require.True(t, plan.Elements[0].Untouched[0].APITruncated)
	before, err := json.Marshal(plan)
	require.NoError(t, err)
	response.RequestedKeys[0] = "changed"
	response.RequestedValues["x"].([]any)[0] = false
	response.TenantContext.ResolvedTenantIDs[0] = "changed"
	response.TenantContext.Warnings[0].Message = "changed"
	p := &response.Elements[0]
	p.Additions[0].Value.(map[string]any)["list"].([]any)[0] = false
	p.Changes[0].Before.([]any)[0] = true
	p.Changes[0].After.([]any)[0] = false
	p.UnchangedRequested[0].Value.([]any)[0] = false
	p.Untouched[0].Value.([]any)[0] = true
	p.MutationVariables["x"].([]any)[0] = false
	p.VerificationValues["x"].([]any)[0] = false
	after, err := json.Marshal(plan)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
	empty := fromDomainElementVariableUpdatePlan(d.ElementVariableUpdatePlan{RequestedValues: map[string]any{}, Elements: []d.ElementVariablePlan{{ElementKey: "E", ProcessInstanceKey: "P", ScopeKey: "E", Additions: []d.ElementVariablePlannedValue{}, Changes: []d.ElementVariablePlannedChange{}, UnchangedRequested: []d.ElementVariablePlannedValue{}, Untouched: []d.ElementVariablePlannedValue{}, VerificationValues: map[string]any{}}}})
	raw, err := json.Marshal(empty)
	require.NoError(t, err)
	for _, field := range []string{"additions", "changes", "unchangedRequested", "untouched"} {
		require.Contains(t, string(raw), `"`+field+`":[]`)
	}
	require.Contains(t, string(raw), `"mutationSubmitted":false`)
}

// TestElementVariableUpdatesPlanFacadeErrors converts service failures and
// preserves legacy constructors without publishing partial plans.
func TestElementVariableUpdatesPlanFacadeErrors(t *testing.T) {
	t.Parallel()
	for _, api := range []API{New(nil, nil), NewWithListeners(nil, nil, nil), NewWithEnrichment(nil, nil, nil, nil)} {
		plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{})
		require.ErrorIs(t, err, ferrors.ErrLocalPrecondition)
		require.Equal(t, ElementVariableUpdatePlan{}, plan)
	}
	for _, tc := range []struct{ domain, public error }{{d.ErrValidation, ferrors.ErrInvalidInput}, {d.ErrUnauthorized, ferrors.ErrLocalPrecondition}, {d.ErrNotFound, ferrors.ErrNotFound}, {context.Canceled, context.Canceled}} {
		api := NewWithVariableUpdates(nil, nil, nil, elementPlannerFunc(func(context.Context, typex.Keys, map[string]any, ...services.CallOption) (d.ElementVariableUpdatePlan, error) {
			return d.ElementVariableUpdatePlan{RequestedCount: 5}, tc.domain
		}), nil)
		plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{})
		require.ErrorIs(t, err, tc.public)
		require.Equal(t, ElementVariableUpdatePlan{}, plan)
	}
}

// TestElementVariableUpdatesPlanFacadeInvalidJSON rejects cyclic and unsupported
// payload storage before copying or invoking the service.
func TestElementVariableUpdatesPlanFacadeInvalidJSON(t *testing.T) {
	t.Parallel()
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	api := NewWithVariableUpdates(nil, nil, nil, elementPlannerFunc(func(context.Context, typex.Keys, map[string]any, ...services.CallOption) (d.ElementVariableUpdatePlan, error) {
		t.Fatal("invalid JSON reached planner")
		return d.ElementVariableUpdatePlan{}, nil
	}), nil)
	for _, input := range []map[string]any{cyclic, {"bad": make(chan int)}} {
		_, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, input)
		require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	}
}

// facadeMarshalOnlyValue exposes private mutable storage as a JSON array.
type facadeMarshalOnlyValue struct{ values []string }

// MarshalJSON deliberately has no corresponding typed unmarshaller.
func (v facadeMarshalOnlyValue) MarshalJSON() ([]byte, error) { return json.Marshal(v.values) }

// TestElementVariableUpdatesPlanFacadeTypedValues preserves raw JSON, typed
// collections, number precision and marshal-only payloads at both boundaries.
func TestElementVariableUpdatesPlanFacadeTypedValues(t *testing.T) {
	t.Parallel()
	type typedMap map[string][]string
	typed := typedMap{"x": {"original"}}
	raw := json.RawMessage(`9007199254740993`)
	custom := facadeMarshalOnlyValue{values: []string{"original"}}
	input := map[string]any{"typed": typed, "raw": raw, "custom": custom, "number": json.Number("9007199254740993")}
	var received map[string]any
	api := NewWithVariableUpdates(nil, nil, nil, elementPlannerFunc(func(_ context.Context, _ typex.Keys, values map[string]any, _ ...services.CallOption) (d.ElementVariableUpdatePlan, error) {
		received = values
		return d.ElementVariableUpdatePlan{RequestedValues: values}, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, input)
	require.NoError(t, err)
	typed["x"][0] = "caller"
	raw[0] = '1'
	custom.values[0] = "caller"
	require.Equal(t, "original", received["typed"].(typedMap)["x"][0])
	require.Equal(t, []any{"original"}, received["custom"])
	received["typed"].(typedMap)["x"][0] = "service"
	received["raw"].(json.RawMessage)[0] = '2'
	received["custom"].([]any)[0] = "service"
	require.Equal(t, "original", plan.RequestedValues["typed"].(typedMap)["x"][0])
	require.Equal(t, "9007199254740993", string(plan.RequestedValues["raw"].(json.RawMessage)))
	require.Equal(t, json.Number("9007199254740993"), plan.RequestedValues["number"])
	require.Equal(t, []any{"original"}, plan.RequestedValues["custom"])
}

// ExecuteElementVariableUpdates keeps planning-only doubles explicit about unsupported execution.
func (f elementPlannerFunc) ExecuteElementVariableUpdates(context.Context, d.ElementVariableUpdatePlan, int, ...services.CallOption) (d.ElementVariableUpdateResults, error) {
	return d.ElementVariableUpdateResults{}, d.ErrPrecondition
}

// elementExecutorStub isolates execution mapping while retaining the planner interface.
type elementExecutorStub struct {
	elementPlannerFunc
	execute func(context.Context, d.ElementVariableUpdatePlan, int, ...services.CallOption) (d.ElementVariableUpdateResults, error)
}

// ExecuteElementVariableUpdates captures one delegated frozen plan and returns partial evidence.
func (s elementExecutorStub) ExecuteElementVariableUpdates(ctx context.Context, p d.ElementVariableUpdatePlan, w int, opts ...services.CallOption) (d.ElementVariableUpdateResults, error) {
	return s.execute(ctx, p, w, opts...)
}

// TestElementVariableUpdatesExecuteFacade preserves every mapped plan/result field,
// converts failures without dropping partial evidence, and isolates mutable storage.
func TestElementVariableUpdatesExecuteFacade(t *testing.T) {
	t.Parallel()
	base, _ := d.NewExplicitKeysTenantContext("configured")
	tenant, _ := d.WithTenantEvidence(base, []string{"actual", "other"}, 0)
	domainPlan := d.ElementVariableUpdatePlan{RequestedKeys: []string{"E"}, RequestedValues: map[string]any{"x": []any{true}}, RequestedCount: 1, UpdateCount: 1, VariableAddCount: 1, VariableChangeCount: 2, VariableUnchangedCount: 3, VariableUntouchedCount: 4, TenantContext: tenant, Elements: []d.ElementVariablePlan{{ElementKey: "E", ProcessInstanceKey: "P", ScopeKey: "E", TenantId: "actual", Additions: []d.ElementVariablePlannedValue{{Name: "a", Value: []any{true}}}, Changes: []d.ElementVariablePlannedChange{{Name: "c", Before: []any{false}, After: []any{true}, APITruncated: true}}, UnchangedRequested: []d.ElementVariablePlannedValue{{Name: "u", Value: []any{true}}}, Untouched: []d.ElementVariablePlannedValue{{Name: "t", Value: []any{false}, APITruncated: true}}, MutationVariables: map[string]any{"x": []any{true}}, VerificationValues: map[string]any{"x": []any{true}}}}}
	plan := fromDomainElementVariableUpdatePlan(domainPlan)
	before, err := json.Marshal(plan)
	require.NoError(t, err)
	response := d.ElementVariableUpdateResults{Items: []d.ElementVariableUpdateResult{{Key: "E", ProcessInstanceKey: "P", ScopeKey: "E", Status: d.ElementVariableUpdateStatusConfirmationFailed, SubmissionStatus: d.ScopeVariableSubmissionAccepted, MutationAccepted: true, ConfirmationStatus: "failed", StatusCode: 204, Message: "accepted", Error: "read failed", Variables: map[string]any{"x": []any{true}}}}}
	calls := 0
	api := NewWithVariableUpdates(nil, nil, nil, elementExecutorStub{execute: func(_ context.Context, p d.ElementVariableUpdatePlan, workers int, opts ...services.CallOption) (d.ElementVariableUpdateResults, error) {
		calls++
		require.Equal(t, 7, workers)
		require.Equal(t, domainPlan, p)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.IgnoreTenant)
		require.True(t, cfg.FailFast)
		require.True(t, cfg.NoWorkerLimit)
		require.True(t, cfg.NoWait)
		p.RequestedKeys[0] = "changed"
		p.RequestedValues["x"].([]any)[0] = false
		p.TenantContext.ResolvedTenantIDs[0] = "changed"
		p.TenantContext.Warnings[0].Message = "changed"
		e := &p.Elements[0]
		e.Additions[0].Value.([]any)[0] = false
		e.Changes[0].Before.([]any)[0] = true
		e.Changes[0].After.([]any)[0] = false
		e.UnchangedRequested[0].Value.([]any)[0] = false
		e.Untouched[0].Value.([]any)[0] = true
		e.MutationVariables["x"].([]any)[0] = false
		e.VerificationValues["x"].([]any)[0] = false
		return response, d.ErrNotFound
	}}, nil)
	result, err := api.ExecuteElementVariableUpdates(context.Background(), plan, 7, foptions.WithIgnoreTenant(), foptions.WithFailFast(), foptions.WithNoWorkerLimit(), foptions.WithNoWait())
	require.ErrorIs(t, err, ferrors.ErrNotFound)
	require.Equal(t, 1, calls)
	require.Len(t, result.Items, 1)
	after, err := json.Marshal(plan)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
	item := result.Items[0]
	require.Equal(t, "E", item.Key)
	require.Equal(t, "P", item.ProcessInstanceKey)
	require.Equal(t, "E", item.ScopeKey)
	require.Equal(t, ElementVariableUpdateStatusConfirmationFailed, item.Status)
	require.Equal(t, VariableSubmissionAccepted, item.SubmissionStatus)
	require.True(t, item.MutationAccepted)
	require.Equal(t, "failed", item.ConfirmationStatus)
	require.Equal(t, 204, item.StatusCode)
	require.Equal(t, "accepted", item.Message)
	require.Equal(t, "read failed", item.Error)
	require.False(t, item.OK())
	response.Items[0].Variables["x"].([]any)[0] = false
	require.Equal(t, []any{true}, item.Variables["x"])
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[{"key":"E","processInstanceKey":"P","scopeKey":"E","status":"confirmation_failed","submissionStatus":"accepted","mutationAccepted":true,"confirmationStatus":"failed","statusCode":204,"message":"accepted","error":"read failed","variables":{"x":[true]}}]}`, string(raw))
}

// TestElementVariableUpdatesExecuteFacadeErrors rejects unsupported compositions
// and cyclic plans before copying, while preserving each service failure class.
func TestElementVariableUpdatesExecuteFacadeErrors(t *testing.T) {
	t.Parallel()
	for _, api := range []API{New(nil, nil), NewWithListeners(nil, nil, nil), NewWithEnrichment(nil, nil, nil, nil)} {
		result, err := api.ExecuteElementVariableUpdates(context.Background(), ElementVariableUpdatePlan{}, 1)
		require.ErrorIs(t, err, ferrors.ErrLocalPrecondition)
		require.Empty(t, result.Items)
	}
	for _, tc := range []struct{ internal, public error }{{d.ErrValidation, ferrors.ErrInvalidInput}, {d.ErrPrecondition, ferrors.ErrLocalPrecondition}, {d.ErrUnauthorized, ferrors.ErrLocalPrecondition}, {d.ErrUpstream, ferrors.ErrInternal}, {context.Canceled, context.Canceled}, {context.DeadlineExceeded, ferrors.ErrTimeout}} {
		api := NewWithVariableUpdates(nil, nil, nil, elementExecutorStub{execute: func(context.Context, d.ElementVariableUpdatePlan, int, ...services.CallOption) (d.ElementVariableUpdateResults, error) {
			return d.ElementVariableUpdateResults{Items: []d.ElementVariableUpdateResult{{Key: "E"}}}, tc.internal
		}}, nil)
		result, err := api.ExecuteElementVariableUpdates(context.Background(), ElementVariableUpdatePlan{}, 1)
		require.ErrorIs(t, err, tc.public)
		require.Len(t, result.Items, 1)
	}
	cycle := map[string]any{}
	cycle["cycle"] = cycle
	api := NewWithVariableUpdates(nil, nil, nil, elementExecutorStub{execute: func(context.Context, d.ElementVariableUpdatePlan, int, ...services.CallOption) (d.ElementVariableUpdateResults, error) {
		t.Fatal("cyclic plan delegated")
		return d.ElementVariableUpdateResults{}, nil
	}}, nil)
	_, err := api.ExecuteElementVariableUpdates(context.Background(), ElementVariableUpdatePlan{RequestedValues: cycle}, 1)
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
}

// TestElementVariableUpdatesResultSuccess distinguishes deliberate no-wait from
// accepted work whose confirmation was interrupted before starting.
func TestElementVariableUpdatesResultSuccess(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status       ElementVariableUpdateStatus
		confirmation string
		accepted     bool
		success      bool
	}{
		{ElementVariableUpdateStatusUnchanged, "not_required", false, true},
		{ElementVariableUpdateStatusSkipped, "not_required", false, false},
		{ElementVariableUpdateStatusMutationFailed, "not_required", false, false},
		{ElementVariableUpdateStatusSubmitted, "skipped", true, true},
		{ElementVariableUpdateStatusSubmitted, "not_started", true, false},
		{ElementVariableUpdateStatusConfirmed, "confirmed", true, true},
		{ElementVariableUpdateStatusConfirmationFailed, "failed", true, false},
	} {
		evidence := VariableSubmissionNotSent
		if tc.accepted {
			evidence = VariableSubmissionAccepted
		}
		r := ElementVariableUpdateResult{Status: tc.status, SubmissionStatus: evidence, MutationAccepted: tc.accepted, ConfirmationStatus: tc.confirmation}
		require.Equal(t, tc.success, r.OK())
		r.Error = "failure"
		require.False(t, r.OK())
	}
}
