// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"fmt"
	"reflect"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// validateAndCopyProcessInstanceVariableUpdatePlan rejects any plan whose
// collection, classification, destination, or payload facts no longer agree.
func validateAndCopyProcessInstanceVariableUpdatePlan(plan d.ProcessInstanceVariableUpdatePlan) (d.ProcessInstanceVariableUpdatePlan, error) {
	if plan.MutationSubmitted {
		return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("plan already records mutation submission")
	}
	if plan.RequestedCount != len(plan.RequestedKeys) || len(plan.Instances) != len(plan.RequestedKeys) {
		return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("requested process-instance counts are inconsistent")
	}
	if err := validateProcessInstanceVariableUpdateInput(plan.RequestedKeys, plan.RequestedValues); err != nil {
		return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("requested values are invalid: %v", err)
	}
	seenKeys := make(map[string]struct{}, len(plan.RequestedKeys))
	additions, changes, unchanged, untouched, updates := 0, 0, 0, 0, 0
	for i, key := range plan.RequestedKeys {
		if strings.TrimSpace(key) == "" {
			return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("requested process-instance key must not be blank")
		}
		if _, exists := seenKeys[key]; exists {
			return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("requested process-instance key %s is duplicated", key)
		}
		seenKeys[key] = struct{}{}
		instance := plan.Instances[i]
		if instance.ProcessInstanceKey != key || instance.ScopeKey != key || strings.TrimSpace(instance.ScopeKey) == "" {
			return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("process-instance %s has an invalid root destination", key)
		}
		if err := validateProcessInstanceVariableClassifications(instance, plan.RequestedValues); err != nil {
			return d.ProcessInstanceVariableUpdatePlan{}, err
		}
		additions += len(instance.Additions)
		changes += len(instance.Changes)
		unchanged += len(instance.UnchangedRequested)
		untouched += len(instance.Untouched)
		if len(instance.MutationVariables) > 0 {
			updates++
		}
	}
	if plan.UpdateCount != updates || plan.VariableAddCount != additions || plan.VariableChangeCount != changes ||
		plan.VariableUnchangedCount != unchanged || plan.VariableUntouchedCount != untouched {
		return d.ProcessInstanceVariableUpdatePlan{}, processInstanceExecutionPlanValidationError("plan aggregate counts are inconsistent")
	}
	return copyProcessInstanceVariableUpdatePlan(plan), nil
}

// validateProcessInstanceVariableClassifications enforces complete disjoint
// requested classifications and exact mutation and verification maps.
func validateProcessInstanceVariableClassifications(instance d.ProcessInstanceVariablePlan, requested map[string]any) error {
	seen := make(map[string]struct{}, len(requested)+len(instance.Untouched))
	expectedMutation := make(map[string]any, len(instance.Additions)+len(instance.Changes))
	for _, value := range instance.Additions {
		if err := validateProcessInstancePlannedName(instance.ProcessInstanceKey, value.Name, seen); err != nil {
			return err
		}
		expected, exists := requested[value.Name]
		if !exists || !normalizedRequestedProcessInstanceValuesEqual(expected, value.Value) {
			return processInstanceExecutionPlanValidationError("process-instance %s addition %q differs from requested values", instance.ProcessInstanceKey, value.Name)
		}
		expectedMutation[value.Name] = value.Value
	}
	for _, value := range instance.Changes {
		if err := validateProcessInstancePlannedName(instance.ProcessInstanceKey, value.Name, seen); err != nil {
			return err
		}
		expected, exists := requested[value.Name]
		if !exists || !normalizedRequestedProcessInstanceValuesEqual(expected, value.After) {
			return processInstanceExecutionPlanValidationError("process-instance %s change %q differs from requested values", instance.ProcessInstanceKey, value.Name)
		}
		expectedMutation[value.Name] = value.After
	}
	for _, value := range instance.UnchangedRequested {
		if err := validateProcessInstancePlannedName(instance.ProcessInstanceKey, value.Name, seen); err != nil {
			return err
		}
		expected, exists := requested[value.Name]
		if !exists || value.APITruncated || !normalizedRequestedProcessInstanceValuesEqual(expected, value.Value) {
			return processInstanceExecutionPlanValidationError("process-instance %s unchanged variable %q has inconsistent evidence", instance.ProcessInstanceKey, value.Name)
		}
	}
	for _, value := range instance.Untouched {
		if err := validateProcessInstancePlannedName(instance.ProcessInstanceKey, value.Name, seen); err != nil {
			return err
		}
		if _, exists := requested[value.Name]; exists {
			return processInstanceExecutionPlanValidationError("process-instance %s untouched variable %q was requested", instance.ProcessInstanceKey, value.Name)
		}
	}
	for name := range requested {
		if _, exists := seen[name]; !exists {
			return processInstanceExecutionPlanValidationError("process-instance %s does not classify requested variable %q", instance.ProcessInstanceKey, name)
		}
	}
	if !equalProcessInstanceVariableMaps(expectedMutation, instance.MutationVariables) {
		return processInstanceExecutionPlanValidationError("process-instance %s mutation payload differs from classifications", instance.ProcessInstanceKey)
	}
	if !equalProcessInstanceVariableMaps(requested, instance.VerificationValues) {
		return processInstanceExecutionPlanValidationError("process-instance %s verification values differ from requested values", instance.ProcessInstanceKey)
	}
	return nil
}

// validateProcessInstancePlannedName enforces one category per nonblank name.
func validateProcessInstancePlannedName(key, name string, seen map[string]struct{}) error {
	if strings.TrimSpace(name) == "" {
		return processInstanceExecutionPlanValidationError("process-instance %s has a blank planned variable name", key)
	}
	if _, exists := seen[name]; exists {
		return processInstanceExecutionPlanValidationError("process-instance %s repeats planned variable %q", key, name)
	}
	seen[name] = struct{}{}
	return nil
}

// equalProcessInstanceVariableMaps compares exact JSON-compatible name/value sets.
func equalProcessInstanceVariableMaps(expected, actual map[string]any) bool {
	if len(expected) != len(actual) {
		return false
	}
	for name, value := range expected {
		other, exists := actual[name]
		if !exists || !normalizedRequestedProcessInstanceValuesEqual(value, other) {
			return false
		}
	}
	return true
}

// normalizedRequestedProcessInstanceValuesEqual compares two requested JSON values.
func normalizedRequestedProcessInstanceValuesEqual(a, b any) bool {
	normalizedA, okA := normalizeProcessInstanceRequestedValue(a)
	normalizedB, okB := normalizeProcessInstanceRequestedValue(b)
	return okA && okB && reflect.DeepEqual(normalizedA, normalizedB)
}

// copyProcessInstanceVariableUpdatePlan freezes every mutable collection before workers start.
func copyProcessInstanceVariableUpdatePlan(plan d.ProcessInstanceVariableUpdatePlan) d.ProcessInstanceVariableUpdatePlan {
	out := plan
	out.RequestedKeys = append([]string(nil), plan.RequestedKeys...)
	out.RequestedValues = copyProcessInstanceVariableMap(plan.RequestedValues)
	out.Instances = append([]d.ProcessInstanceVariablePlan(nil), plan.Instances...)
	for i := range out.Instances {
		out.Instances[i].Additions = copyProcessInstancePlannedValues(plan.Instances[i].Additions)
		out.Instances[i].Changes = append([]d.ProcessInstanceVariablePlannedChange(nil), plan.Instances[i].Changes...)
		for j := range out.Instances[i].Changes {
			out.Instances[i].Changes[j].Before = copyProcessInstanceVariableValue(plan.Instances[i].Changes[j].Before)
			out.Instances[i].Changes[j].After = copyProcessInstanceVariableValue(plan.Instances[i].Changes[j].After)
		}
		out.Instances[i].UnchangedRequested = copyProcessInstancePlannedValues(plan.Instances[i].UnchangedRequested)
		out.Instances[i].Untouched = copyProcessInstancePlannedValues(plan.Instances[i].Untouched)
		out.Instances[i].MutationVariables = copyProcessInstanceVariableMap(plan.Instances[i].MutationVariables)
		out.Instances[i].VerificationValues = copyProcessInstanceVariableMap(plan.Instances[i].VerificationValues)
	}
	return out
}

// copyProcessInstancePlannedValues deep-copies planned values.
func copyProcessInstancePlannedValues(values []d.ProcessInstanceVariablePlannedValue) []d.ProcessInstanceVariablePlannedValue {
	out := append([]d.ProcessInstanceVariablePlannedValue(nil), values...)
	for i := range out {
		out[i].Value = copyProcessInstanceVariableValue(values[i].Value)
	}
	return out
}

// processInstanceExecutionPlanValidationError classifies unsafe caller-supplied plans.
func processInstanceExecutionPlanValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: invalid process-instance variable update plan: %s", d.ErrValidation, fmt.Sprintf(format, args...))
}
