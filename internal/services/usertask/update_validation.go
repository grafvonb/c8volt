// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"fmt"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// validateAndCopyUserTaskVariableUpdatePlan rejects plans whose aggregate,
// task, target, or association facts no longer describe one frozen workflow.
func validateAndCopyUserTaskVariableUpdatePlan(plan d.UserTaskVariableUpdatePlan) (d.UserTaskVariableUpdatePlan, error) {
	if plan.MutationSubmitted {
		return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("plan already records mutation submission")
	}
	if plan.RequestedCount != len(plan.RequestedKeys) || len(plan.UserTasks) != len(plan.RequestedKeys) {
		return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("requested task counts are inconsistent")
	}
	seenTasks := make(map[string]struct{}, len(plan.RequestedKeys))
	taskTenantByKey := make(map[string]string, len(plan.RequestedKeys))
	for i, key := range plan.RequestedKeys {
		if strings.TrimSpace(key) == "" {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("requested task key must not be blank")
		}
		if _, exists := seenTasks[key]; exists {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("requested task key %s is duplicated", key)
		}
		seenTasks[key] = struct{}{}
		if plan.UserTasks[i].UserTaskKey != key {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("task order does not match requested key %s", key)
		}
		taskTenantByKey[key] = plan.UserTasks[i].TenantId
	}

	expectedTargets := make(map[scopeVariableIdentity]any)
	expectedAssociations := make(map[taskScopeVariableIdentity]struct{})
	additions, changes, unchanged, untouched, updates := 0, 0, 0, 0, 0
	for _, task := range plan.UserTasks {
		if strings.TrimSpace(task.ElementInstanceKey) == "" {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("user task %s has no element-instance scope", task.UserTaskKey)
		}
		seenNames := make(map[string]struct{})
		additions += len(task.Additions)
		changes += len(task.Changes)
		unchanged += len(task.UnchangedRequested)
		untouched += len(task.Untouched)
		if len(task.Additions)+len(task.Changes) > 0 {
			updates++
		}
		for _, value := range task.Additions {
			if err := validatePlannedNameAndScope(task.UserTaskKey, value.Name, value.ScopeKey, seenNames); err != nil {
				return d.UserTaskVariableUpdatePlan{}, err
			}
			if value.ScopeKey != task.ElementInstanceKey || value.Inherited {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("user task %s addition %q is not local", task.UserTaskKey, value.Name)
			}
			if err := recordExpectedTarget(expectedTargets, value.ScopeKey, value.Name, value.Value); err != nil {
				return d.UserTaskVariableUpdatePlan{}, err
			}
			expectedAssociations[taskScopeVariableIdentity{taskKey: task.UserTaskKey, scopeKey: value.ScopeKey, name: value.Name}] = struct{}{}
		}
		for _, value := range task.Changes {
			if err := validatePlannedNameAndScope(task.UserTaskKey, value.Name, value.ScopeKey, seenNames); err != nil {
				return d.UserTaskVariableUpdatePlan{}, err
			}
			if value.Inherited != (value.ScopeKey != task.ElementInstanceKey) {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("user task %s change %q has inconsistent scope classification", task.UserTaskKey, value.Name)
			}
			if err := recordExpectedTarget(expectedTargets, value.ScopeKey, value.Name, value.After); err != nil {
				return d.UserTaskVariableUpdatePlan{}, err
			}
			expectedAssociations[taskScopeVariableIdentity{taskKey: task.UserTaskKey, scopeKey: value.ScopeKey, name: value.Name}] = struct{}{}
		}
		for _, value := range task.UnchangedRequested {
			if err := validatePlannedNameAndScope(task.UserTaskKey, value.Name, value.ScopeKey, seenNames); err != nil {
				return d.UserTaskVariableUpdatePlan{}, err
			}
			if value.APITruncated || value.Inherited != (value.ScopeKey != task.ElementInstanceKey) {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("user task %s unchanged variable %q has inconsistent evidence", task.UserTaskKey, value.Name)
			}
		}
		for _, value := range task.Untouched {
			if err := validatePlannedNameAndScope(task.UserTaskKey, value.Name, value.ScopeKey, seenNames); err != nil {
				return d.UserTaskVariableUpdatePlan{}, err
			}
			if value.Inherited != (value.ScopeKey != task.ElementInstanceKey) {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("user task %s untouched variable %q has inconsistent scope classification", task.UserTaskKey, value.Name)
			}
		}
	}
	if plan.UpdateCount != updates || plan.VariableAddCount != additions || plan.VariableChangeCount != changes ||
		plan.VariableUnchangedCount != unchanged || plan.VariableUntouchedCount != untouched {
		return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("plan aggregate counts are inconsistent")
	}

	seenScopes := make(map[string]struct{}, len(plan.Targets))
	seenTargetValues := make(map[scopeVariableIdentity]any, len(expectedTargets))
	for _, target := range plan.Targets {
		if strings.TrimSpace(target.ScopeKey) == "" || len(target.Variables) == 0 {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope targets require a key and variables")
		}
		if _, exists := seenScopes[target.ScopeKey]; exists {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope target %s is duplicated", target.ScopeKey)
		}
		seenScopes[target.ScopeKey] = struct{}{}
		associationNames := make(map[string]map[string]struct{}, len(target.Associations))
		for _, association := range target.Associations {
			if _, exists := seenTasks[association.UserTaskKey]; !exists || len(association.Names) == 0 {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s has an invalid task association", target.ScopeKey)
			}
			if _, exists := associationNames[association.UserTaskKey]; exists {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s repeats task association %s", target.ScopeKey, association.UserTaskKey)
			}
			if knownStringsConflict(target.TenantId, taskTenantByKey[association.UserTaskKey]) {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s has conflicting tenant evidence for task %s", target.ScopeKey, association.UserTaskKey)
			}
			associationNames[association.UserTaskKey] = make(map[string]struct{}, len(association.Names))
			for _, name := range association.Names {
				if _, exists := associationNames[association.UserTaskKey][name]; exists {
					return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s repeats association name %q", target.ScopeKey, name)
				}
				associationNames[association.UserTaskKey][name] = struct{}{}
				if _, exists := target.Variables[name]; !exists {
					return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s association references absent variable %q", target.ScopeKey, name)
				}
				identity := taskScopeVariableIdentity{taskKey: association.UserTaskKey, scopeKey: target.ScopeKey, name: name}
				if _, exists := expectedAssociations[identity]; !exists {
					return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s has unexpected association for task %s variable %q", target.ScopeKey, association.UserTaskKey, name)
				}
				delete(expectedAssociations, identity)
			}
		}
		for name, value := range target.Variables {
			identity := scopeVariableIdentity{scopeKey: target.ScopeKey, name: name}
			expected, exists := expectedTargets[identity]
			if !exists || !normalizedRequestedValuesEqual(expected, value) {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s variable %q differs from task plans", target.ScopeKey, name)
			}
			seenTargetValues[identity] = value
		}
	}
	if len(seenTargetValues) != len(expectedTargets) {
		return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope targets do not cover every planned change")
	}
	if len(expectedAssociations) != 0 {
		return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope targets do not associate every planned task variable")
	}
	for _, task := range plan.UserTasks {
		expectedScopes := make([]string, 0, len(task.TargetScopeKeys))
		for _, target := range plan.Targets {
			if targetAssociatesTask(target, task.UserTaskKey) {
				expectedScopes = append(expectedScopes, target.ScopeKey)
			}
		}
		if !sameUniqueStrings(task.TargetScopeKeys, expectedScopes) {
			return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("user task %s target scopes are inconsistent", task.UserTaskKey)
		}
		for _, scopeKey := range task.TargetScopeKeys {
			target := findScopeVariableTarget(plan.Targets, scopeKey)
			if target == nil || !targetAssociatesTask(*target, task.UserTaskKey) {
				return d.UserTaskVariableUpdatePlan{}, executionPlanValidationError("scope %s does not associate user task %s", scopeKey, task.UserTaskKey)
			}
		}
	}
	return copyUserTaskVariableUpdatePlan(plan), nil
}

// sameUniqueStrings compares scope references as a set while rejecting duplicates.
func sameUniqueStrings(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	seen := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	for _, value := range expected {
		if _, exists := seen[value]; !exists {
			return false
		}
	}
	return true
}

type taskScopeVariableIdentity struct {
	taskKey  string
	scopeKey string
	name     string
}

// recordExpectedTarget preserves one consistent requested value for a shared identity.
func recordExpectedTarget(targets map[scopeVariableIdentity]any, scopeKey, name string, value any) error {
	identity := scopeVariableIdentity{scopeKey: scopeKey, name: name}
	if previous, exists := targets[identity]; exists && !normalizedRequestedValuesEqual(previous, value) {
		return executionPlanValidationError("scope %s variable %q has conflicting task values", scopeKey, name)
	}
	targets[identity] = value
	return nil
}

// validatePlannedNameAndScope enforces one category per task-variable name.
func validatePlannedNameAndScope(taskKey, name, scopeKey string, seen map[string]struct{}) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(scopeKey) == "" {
		return executionPlanValidationError("user task %s has a blank planned name or scope", taskKey)
	}
	if _, exists := seen[name]; exists {
		return executionPlanValidationError("user task %s repeats planned variable %q", taskKey, name)
	}
	seen[name] = struct{}{}
	return nil
}

// findScopeVariableTarget locates a validated target by scope.
func findScopeVariableTarget(targets []d.ScopeVariableUpdateTarget, scopeKey string) *d.ScopeVariableUpdateTarget {
	for i := range targets {
		if targets[i].ScopeKey == scopeKey {
			return &targets[i]
		}
	}
	return nil
}

// targetAssociatesTask reports whether a target carries a task fan-out edge.
func targetAssociatesTask(target d.ScopeVariableUpdateTarget, taskKey string) bool {
	for _, association := range target.Associations {
		if association.UserTaskKey == taskKey {
			return true
		}
	}
	return false
}

// copyUserTaskVariableUpdatePlan freezes all mutable plan collections before workers start.
func copyUserTaskVariableUpdatePlan(plan d.UserTaskVariableUpdatePlan) d.UserTaskVariableUpdatePlan {
	out := plan
	out.RequestedKeys = append([]string(nil), plan.RequestedKeys...)
	out.UserTasks = append([]d.UserTaskVariablePlan(nil), plan.UserTasks...)
	for i := range out.UserTasks {
		out.UserTasks[i].Additions = append([]d.UserTaskVariablePlannedValue(nil), plan.UserTasks[i].Additions...)
		for j := range out.UserTasks[i].Additions {
			out.UserTasks[i].Additions[j].Value = copyVariableValue(plan.UserTasks[i].Additions[j].Value)
		}
		out.UserTasks[i].Changes = append([]d.UserTaskVariablePlannedChange(nil), plan.UserTasks[i].Changes...)
		for j := range out.UserTasks[i].Changes {
			out.UserTasks[i].Changes[j].Before = copyVariableValue(plan.UserTasks[i].Changes[j].Before)
			out.UserTasks[i].Changes[j].After = copyVariableValue(plan.UserTasks[i].Changes[j].After)
		}
		out.UserTasks[i].UnchangedRequested = append([]d.UserTaskVariablePlannedValue(nil), plan.UserTasks[i].UnchangedRequested...)
		for j := range out.UserTasks[i].UnchangedRequested {
			out.UserTasks[i].UnchangedRequested[j].Value = copyVariableValue(plan.UserTasks[i].UnchangedRequested[j].Value)
		}
		out.UserTasks[i].Untouched = append([]d.UserTaskVariablePlannedValue(nil), plan.UserTasks[i].Untouched...)
		for j := range out.UserTasks[i].Untouched {
			out.UserTasks[i].Untouched[j].Value = copyVariableValue(plan.UserTasks[i].Untouched[j].Value)
		}
		out.UserTasks[i].TargetScopeKeys = append([]string(nil), plan.UserTasks[i].TargetScopeKeys...)
	}
	out.Targets = append([]d.ScopeVariableUpdateTarget(nil), plan.Targets...)
	for i := range out.Targets {
		out.Targets[i].Variables = copyVariableMap(plan.Targets[i].Variables)
		out.Targets[i].Associations = append([]d.ScopeVariableUpdateAssociation(nil), plan.Targets[i].Associations...)
		for j := range out.Targets[i].Associations {
			out.Targets[i].Associations[j].Names = append([]string(nil), plan.Targets[i].Associations[j].Names...)
		}
	}
	return out
}

// executionPlanValidationError classifies caller-supplied plan inconsistencies before I/O.
func executionPlanValidationError(format string, args ...any) error {
	return fmt.Errorf("%w: invalid user-task variable update plan: %s", d.ErrValidation, fmt.Sprintf(format, args...))
}
