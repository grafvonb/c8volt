// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// validateAndCopyElementVariableUpdatePlan rejects any plan whose
// collection, classification, destination, or payload facts no longer agree.
func validateAndCopyElementVariableUpdatePlan(plan d.ElementVariableUpdatePlan) (d.ElementVariableUpdatePlan, error) {
	if plan.MutationSubmitted {
		return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("plan already records mutation submission")
	}
	if plan.RequestedCount != len(plan.RequestedKeys) || len(plan.Elements) != len(plan.RequestedKeys) {
		return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("requested element counts are inconsistent")
	}
	if err := validateElementVariableUpdateInput(plan.RequestedKeys, plan.RequestedValues); err != nil {
		return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("requested values are invalid: %v", err)
	}
	seenKeys := make(map[string]struct{}, len(plan.RequestedKeys))
	additions, changes, unchanged, untouched, updates := 0, 0, 0, 0, 0
	for i, key := range plan.RequestedKeys {
		if strings.TrimSpace(key) == "" {
			return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("requested element key must not be blank")
		}
		if _, exists := seenKeys[key]; exists {
			return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("requested element key %s is duplicated", key)
		}
		seenKeys[key] = struct{}{}
		element := plan.Elements[i]
		if element.ElementKey != key || element.ScopeKey != key || strings.TrimSpace(element.ProcessInstanceKey) == "" {
			return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("element %s has an invalid local destination", key)
		}
		if err := validateElementVariableClassifications(element, plan.RequestedValues); err != nil {
			return d.ElementVariableUpdatePlan{}, err
		}
		additions += len(element.Additions)
		changes += len(element.Changes)
		unchanged += len(element.UnchangedRequested)
		untouched += len(element.Untouched)
		if len(element.MutationVariables) > 0 {
			updates++
		}
	}
	if plan.UpdateCount != updates || plan.VariableAddCount != additions || plan.VariableChangeCount != changes ||
		plan.VariableUnchangedCount != unchanged || plan.VariableUntouchedCount != untouched {
		return d.ElementVariableUpdatePlan{}, elementExecutionPlanValidationError("plan aggregate counts are inconsistent")
	}
	if err := validateElementPlanTenantContext(plan); err != nil {
		return d.ElementVariableUpdatePlan{}, err
	}
	return copyElementVariableUpdatePlan(plan), nil
}

// validateElementVariableClassifications enforces complete disjoint
// requested classifications and exact mutation and verification maps.
func validateElementVariableClassifications(element d.ElementVariablePlan, requested map[string]any) error {
	seen := make(map[string]struct{}, len(requested)+len(element.Untouched))
	expectedMutation := make(map[string]any, len(element.Additions)+len(element.Changes))
	for _, value := range element.Additions {
		if err := validateElementPlannedName(element.ElementKey, value.Name, seen); err != nil {
			return err
		}
		expected, exists := requested[value.Name]
		if !exists || value.APITruncated || !normalizedRequestedElementValuesEqual(expected, value.Value) {
			return elementExecutionPlanValidationError("element %s addition %q differs from requested values", element.ElementKey, value.Name)
		}
		expectedMutation[value.Name] = value.Value
	}
	for _, value := range element.Changes {
		if err := validateElementPlannedName(element.ElementKey, value.Name, seen); err != nil {
			return err
		}
		expected, exists := requested[value.Name]
		if !exists || !normalizedRequestedElementValuesEqual(expected, value.After) {
			return elementExecutionPlanValidationError("element %s change %q differs from requested values", element.ElementKey, value.Name)
		}
		if _, ok := normalizeElementRequestedValue(value.Before); !ok {
			return elementExecutionPlanValidationError("element %s change %q has invalid before evidence", element.ElementKey, value.Name)
		}
		if !value.APITruncated && normalizedRequestedElementValuesEqual(value.Before, value.After) {
			return elementExecutionPlanValidationError("element %s change %q has complete equal values", element.ElementKey, value.Name)
		}
		expectedMutation[value.Name] = value.After
	}
	for _, value := range element.UnchangedRequested {
		if err := validateElementPlannedName(element.ElementKey, value.Name, seen); err != nil {
			return err
		}
		expected, exists := requested[value.Name]
		if !exists || value.APITruncated || !normalizedRequestedElementValuesEqual(expected, value.Value) {
			return elementExecutionPlanValidationError("element %s unchanged variable %q has inconsistent evidence", element.ElementKey, value.Name)
		}
	}
	for _, value := range element.Untouched {
		if err := validateElementPlannedName(element.ElementKey, value.Name, seen); err != nil {
			return err
		}
		if _, ok := normalizeElementRequestedValue(value.Value); !ok {
			return elementExecutionPlanValidationError("element %s untouched variable %q has invalid value", element.ElementKey, value.Name)
		}
		if _, exists := requested[value.Name]; exists {
			return elementExecutionPlanValidationError("element %s untouched variable %q was requested", element.ElementKey, value.Name)
		}
	}
	for name := range requested {
		if _, exists := seen[name]; !exists {
			return elementExecutionPlanValidationError("element %s does not classify requested variable %q", element.ElementKey, name)
		}
	}
	if !equalElementVariableMaps(expectedMutation, element.MutationVariables) {
		return elementExecutionPlanValidationError("element %s mutation payload differs from classifications", element.ElementKey)
	}
	if !equalElementVariableMaps(requested, element.VerificationValues) {
		return elementExecutionPlanValidationError("element %s verification values differ from requested values", element.ElementKey)
	}
	return nil
}

// validateElementPlannedName enforces one category per nonblank name.
func validateElementPlannedName(key, name string, seen map[string]struct{}) error {
	if strings.TrimSpace(name) == "" {
		return elementExecutionPlanValidationError("element %s has a blank planned variable name", key)
	}
	if _, exists := seen[name]; exists {
		return elementExecutionPlanValidationError("element %s repeats planned variable %q", key, name)
	}
	seen[name] = struct{}{}
	return nil
}

// equalElementVariableMaps compares exact JSON-compatible name/value sets.
func equalElementVariableMaps(expected, actual map[string]any) bool {
	if len(expected) != len(actual) {
		return false
	}
	for name, value := range expected {
		other, exists := actual[name]
		if !exists || !normalizedRequestedElementValuesEqual(value, other) {
			return false
		}
	}
	return true
}

// normalizedRequestedElementValuesEqual compares two requested JSON values.
func normalizedRequestedElementValuesEqual(a, b any) bool {
	normalizedA, okA := normalizeElementRequestedValue(a)
	normalizedB, okB := normalizeElementRequestedValue(b)
	return okA && okB && reflect.DeepEqual(normalizedA, normalizedB)
}

// copyElementVariableUpdatePlan freezes every mutable collection before workers start.
func copyElementVariableUpdatePlan(plan d.ElementVariableUpdatePlan) d.ElementVariableUpdatePlan {
	out := plan
	out.TenantContext.ResolvedTenantIDs = slices.Clone(plan.TenantContext.ResolvedTenantIDs)
	out.TenantContext.Warnings = slices.Clone(plan.TenantContext.Warnings)
	out.RequestedKeys = append([]string(nil), plan.RequestedKeys...)
	out.RequestedValues = copyElementVariableMap(plan.RequestedValues)
	out.Elements = append([]d.ElementVariablePlan(nil), plan.Elements...)
	for i := range out.Elements {
		out.Elements[i].Additions = copyElementPlannedValues(plan.Elements[i].Additions)
		out.Elements[i].Changes = slices.Clone(plan.Elements[i].Changes)
		for j := range out.Elements[i].Changes {
			out.Elements[i].Changes[j].Before = copyElementVariableValue(plan.Elements[i].Changes[j].Before)
			out.Elements[i].Changes[j].After = copyElementVariableValue(plan.Elements[i].Changes[j].After)
		}
		out.Elements[i].UnchangedRequested = copyElementPlannedValues(plan.Elements[i].UnchangedRequested)
		out.Elements[i].Untouched = copyElementPlannedValues(plan.Elements[i].Untouched)
		out.Elements[i].MutationVariables = copyElementVariableMap(plan.Elements[i].MutationVariables)
		out.Elements[i].VerificationValues = copyElementVariableMap(plan.Elements[i].VerificationValues)
	}
	return out
}

// copyElementPlannedValues deep-copies planned values.
func copyElementPlannedValues(values []d.ElementVariablePlannedValue) []d.ElementVariablePlannedValue {
	out := slices.Clone(values)
	for i := range out {
		out[i].Value = copyElementVariableValue(values[i].Value)
	}
	return out
}

// elementExecutionPlanValidationError classifies unsafe caller-supplied plans.
func elementExecutionPlanValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: invalid element variable update plan: %s", d.ErrValidation, fmt.Sprintf(format, args...))
}

// validateElementPlanTenantContext checks every derived evidence field against
// the frozen destinations, including warning text and unknown target counts.
func validateElementPlanTenantContext(plan d.ElementVariableUpdatePlan) error {
	base, err := d.NewExplicitKeysTenantContext(plan.TenantContext.ConfiguredTenantID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(plan.Elements))
	unknown := 0
	for _, item := range plan.Elements {
		if item.TenantId == "" {
			unknown++
		} else {
			ids = append(ids, item.TenantId)
		}
	}
	expected, err := d.WithTenantEvidence(base, ids, unknown)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, plan.TenantContext) {
		return elementExecutionPlanValidationError("tenant context differs from selected element evidence")
	}
	return nil
}
