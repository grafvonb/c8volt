// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// elementValidationPlan supplies all four classifications and independent owners.
func elementValidationPlan(t *testing.T) d.ElementVariableUpdatePlan {
	t.Helper()
	api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
		return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P-" + key, TenantId: key}, nil
	}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		return []d.ProcessInstanceVariable{
			{Name: "change", Value: `{"old":[1]}`, ScopeKey: q.ScopeKey, ProcessInstanceKey: q.ProcessInstanceKey},
			{Name: "same", Value: `{"same":[1]}`, ScopeKey: q.ScopeKey, ProcessInstanceKey: q.ProcessInstanceKey},
			{Name: "untouched", Value: `{"keep":[1]}`, ScopeKey: q.ScopeKey, ProcessInstanceKey: q.ProcessInstanceKey},
		}, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E2", "E1"}, map[string]any{"add": map[string]any{"new": []any{float64(1)}}, "change": map[string]any{"new": []any{float64(1)}}, "same": map[string]any{"same": []any{float64(1)}}})
	require.NoError(t, err)
	return plan
}

// TestElementVariableUpdatesPlanValidation rejects independent tampering of every
// destination, count, category, value map and tenant-context invariant.
func TestElementVariableUpdatesPlanValidation(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*d.ElementVariableUpdatePlan){
		"submitted":       func(p *d.ElementVariableUpdatePlan) { p.MutationSubmitted = true },
		"requested count": func(p *d.ElementVariableUpdatePlan) { p.RequestedCount++ },
		"element count":   func(p *d.ElementVariableUpdatePlan) { p.Elements = p.Elements[:1] },
		"nil request":     func(p *d.ElementVariableUpdatePlan) { p.RequestedValues = nil },
		"invalid request": func(p *d.ElementVariableUpdatePlan) { p.RequestedValues["bad"] = make(chan int) },
		"blank key":       func(p *d.ElementVariableUpdatePlan) { p.RequestedKeys[0] = " " },
		"duplicate keys": func(p *d.ElementVariableUpdatePlan) {
			p.RequestedKeys[1] = p.RequestedKeys[0]
			p.Elements[1] = p.Elements[0]
		},
		"reordered":       func(p *d.ElementVariableUpdatePlan) { p.Elements[0], p.Elements[1] = p.Elements[1], p.Elements[0] },
		"element key":     func(p *d.ElementVariableUpdatePlan) { p.Elements[0].ElementKey = "wrong" },
		"root redirect":   func(p *d.ElementVariableUpdatePlan) { p.Elements[0].ScopeKey = p.Elements[0].ProcessInstanceKey },
		"missing owner":   func(p *d.ElementVariableUpdatePlan) { p.Elements[0].ProcessInstanceKey = " " },
		"updates":         func(p *d.ElementVariableUpdatePlan) { p.UpdateCount++ },
		"add count":       func(p *d.ElementVariableUpdatePlan) { p.VariableAddCount++ },
		"change count":    func(p *d.ElementVariableUpdatePlan) { p.VariableChangeCount++ },
		"unchanged count": func(p *d.ElementVariableUpdatePlan) { p.VariableUnchangedCount++ },
		"untouched count": func(p *d.ElementVariableUpdatePlan) { p.VariableUntouchedCount++ },
		"blank name":      func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Additions[0].Name = " " },
		"duplicate category": func(p *d.ElementVariableUpdatePlan) {
			p.Elements[0].Untouched = append(p.Elements[0].Untouched, p.Elements[0].Additions[0])
		},
		"missing category": func(p *d.ElementVariableUpdatePlan) {
			p.Elements[0].UnchangedRequested = nil
			p.VariableUnchangedCount--
		},
		"extra addition":      func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Additions[0].Name = "not-requested" },
		"addition value":      func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Additions[0].Value = false },
		"addition truncated":  func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Additions[0].APITruncated = true },
		"change after":        func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Changes[0].After = false },
		"change before":       func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Changes[0].Before = make(chan int) },
		"equal change":        func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Changes[0].Before = p.Elements[0].Changes[0].After },
		"unchanged unequal":   func(p *d.ElementVariableUpdatePlan) { p.Elements[0].UnchangedRequested[0].Value = false },
		"unchanged truncated": func(p *d.ElementVariableUpdatePlan) { p.Elements[0].UnchangedRequested[0].APITruncated = true },
		"untouched requested": func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Untouched[0].Name = "same" },
		"untouched invalid":   func(p *d.ElementVariableUpdatePlan) { p.Elements[0].Untouched[0].Value = make(chan int) },
		"mutation missing":    func(p *d.ElementVariableUpdatePlan) { delete(p.Elements[0].MutationVariables, "add") },
		"mutation extra": func(p *d.ElementVariableUpdatePlan) {
			p.Elements[0].MutationVariables["same"] = p.RequestedValues["same"]
		},
		"mutation unequal":     func(p *d.ElementVariableUpdatePlan) { p.Elements[0].MutationVariables["add"] = false },
		"verification missing": func(p *d.ElementVariableUpdatePlan) { delete(p.Elements[0].VerificationValues, "same") },
		"verification extra":   func(p *d.ElementVariableUpdatePlan) { p.Elements[0].VerificationValues["extra"] = true },
		"verification unequal": func(p *d.ElementVariableUpdatePlan) { p.Elements[0].VerificationValues["same"] = false },
		"element tenant":       func(p *d.ElementVariableUpdatePlan) { p.Elements[0].TenantId = "other" },
		"tenant mode":          func(p *d.ElementVariableUpdatePlan) { p.TenantContext.Mode = d.TenantContextModeDiscovery },
		"tenant filter":        func(p *d.ElementVariableUpdatePlan) { p.TenantContext.Filter = d.TenantContextFilterNamed },
		"tenant target":        func(p *d.ElementVariableUpdatePlan) { p.TenantContext.TargetTenantID = "other" },
		"tenant resolved":      func(p *d.ElementVariableUpdatePlan) { p.TenantContext.ResolvedTenantIDs = []string{"other"} },
		"tenant unknown":       func(p *d.ElementVariableUpdatePlan) { p.TenantContext.UnknownTargetCount++ },
		"tenant cross":         func(p *d.ElementVariableUpdatePlan) { p.TenantContext.CrossTenant = false },
		"tenant warning":       func(p *d.ElementVariableUpdatePlan) { p.TenantContext.Warnings[0].Message = "other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			plan := elementValidationPlan(t)
			mutate(&plan)
			frozen, err := validateAndCopyElementVariableUpdatePlan(plan)
			require.ErrorIs(t, err, d.ErrValidation)
			require.Equal(t, d.ElementVariableUpdatePlan{}, frozen)
		})
	}
}

// TestElementVariableUpdatesPlanValidationCopies freezes every mutable plan field,
// retaining numeric representations and independent nested classifications.
func TestElementVariableUpdatesPlanValidationCopies(t *testing.T) {
	t.Parallel()
	plan := elementValidationPlan(t)
	frozen, err := validateAndCopyElementVariableUpdatePlan(plan)
	require.NoError(t, err)
	expected := elementValidationPlan(t)
	plan.RequestedKeys[0] = "mutated"
	plan.RequestedValues["add"].(map[string]any)["new"].([]any)[0] = 9
	plan.TenantContext.ResolvedTenantIDs[0] = "mutated"
	plan.TenantContext.Warnings[0].Message = "mutated"
	for i := range plan.Elements {
		p := &plan.Elements[i]
		p.Additions[0].Value.(map[string]any)["new"].([]any)[0] = 9
		p.Changes[0].Before.(map[string]any)["old"].([]any)[0] = 9
		p.Changes[0].After.(map[string]any)["new"].([]any)[0] = 9
		p.UnchangedRequested[0].Value.(map[string]any)["same"].([]any)[0] = 9
		p.Untouched[0].Value.(map[string]any)["keep"].([]any)[0] = 9
		p.MutationVariables["add"].(map[string]any)["new"].([]any)[0] = 9
		p.VerificationValues["same"].(map[string]any)["same"].([]any)[0] = 9
	}
	require.Equal(t, expected, frozen)
	require.Equal(t, json.Number("1"), frozen.Elements[0].Changes[0].Before.(map[string]any)["old"].([]any)[0])
}

// TestElementVariableUpdatesPlanTypedPayload preserves established numbers and
// snapshots named collections and RawMessage values instead of caller storage.
func TestElementVariableUpdatesPlanTypedPayload(t *testing.T) {
	t.Parallel()
	type typedList []string
	type typedMap map[string]typedList
	typed := typedMap{"items": {"original"}}
	raw := json.RawMessage(`{"raw":9007199254740993}`)
	api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
		return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P"}, nil
	}}, scopeReaderFunc(func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		return nil, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{"typed": typed, "raw": raw, "number": json.Number("9007199254740993")})
	require.NoError(t, err)
	typed["items"][0] = "changed"
	raw[7] = '1'
	require.Equal(t, "original", plan.RequestedValues["typed"].(typedMap)["items"][0])
	require.Equal(t, `{"raw":9007199254740993}`, string(plan.Elements[0].MutationVariables["raw"].(json.RawMessage)))
	require.Equal(t, json.Number("9007199254740993"), plan.RequestedValues["number"])
}

// elementMarshalOnlyValue models an accepted custom JSON value with private storage.
type elementMarshalOnlyValue struct{ values map[string]any }

// MarshalJSON exposes the JSON payload without a matching decoder.
func (v elementMarshalOnlyValue) MarshalJSON() ([]byte, error) { return json.Marshal(v.values) }

// TestElementVariableUpdatesPlanMarshalOnlyPayload freezes JSON meaning even
// when a custom marshaler cannot round-trip into its original concrete type.
func TestElementVariableUpdatesPlanMarshalOnlyPayload(t *testing.T) {
	t.Parallel()
	input := elementMarshalOnlyValue{values: map[string]any{"x": []any{true}}}
	api := NewVariableUpdates(stubElementAPI{get: func(context.Context, string, ...services.CallOption) (d.Element, error) {
		return d.Element{ElementInstanceKey: "E", ProcessInstanceKey: "P"}, nil
	}}, scopeReaderFunc(func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		return nil, nil
	}), nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), typex.Keys{"E"}, map[string]any{"custom": input})
	require.NoError(t, err)
	input.values["x"].([]any)[0] = false
	require.Equal(t, map[string]any{"custom": map[string]any{"x": []any{true}}}, plan.RequestedValues)
	require.Equal(t, plan.RequestedValues, plan.Elements[0].MutationVariables)
}
