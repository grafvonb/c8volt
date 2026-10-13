// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// preparedProcessInstanceVariableUpdate retains the facade plan that supplied
// the command preview so later execution can use the same frozen evidence.
type preparedProcessInstanceVariableUpdate struct {
	plan    process.ProcessInstanceVariableUpdatePlan
	preview processInstanceVariableUpdatePreview
}

// parseUpdateProcessInstanceVariablesFromFlags selects exactly one variable payload source and decodes it.
func parseUpdateProcessInstanceVariablesFromFlags(cmd *cobra.Command, raw string, filePath string) (map[string]any, error) {
	return parseUpdateVariablesFromFlags(cmd, raw, filePath)
}

// parseUpdateProcessInstanceVariables decodes the --vars JSON object used for process-instance updates.
func parseUpdateProcessInstanceVariables(raw string, source string) (map[string]any, error) {
	return parseUpdateVariables(raw, source)
}

// validateUpdateProcessInstanceJSONConfirmation keeps machine-readable mutation output free of prompts and human plans.
func validateUpdateProcessInstanceJSONConfirmation(cmd *cobra.Command) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update process-instance")
	}
	if flagDryRun || pickMode() != RenderModeJSON || shouldImplicitlyConfirm(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json update process-instance requires --dry-run, --auto-confirm, or --automation")
}

// planUpdateProcessInstanceVariables delegates complete collection planning
// and mechanically adapts its frozen classifications to the existing preview.
func planUpdateProcessInstanceVariables(ctx context.Context, cmd *cobra.Command, cli process.API, keys types.Keys, requested map[string]any) (preparedProcessInstanceVariableUpdate, error) {
	uniqueKeys := keys.Unique()
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("preparing update variable plan for %d process instance(s)", len(uniqueKeys)))
	defer stopActivity()

	plan, err := cli.PlanProcessInstanceVariableUpdates(ctx, uniqueKeys, requested, collectExplicitPIAdminInputOptions()...)
	if err != nil {
		return preparedProcessInstanceVariableUpdate{}, err
	}
	attachTenantContext(cmd, userTaskUpdateTenantContext(plan.TenantContext))
	return preparedProcessInstanceVariableUpdate{
		plan:    plan,
		preview: processInstanceVariableUpdatePreviewFromFacadePlan(plan),
	}, nil
}

// processInstanceVariableUpdatePreviewFromFacadePlan preserves the existing
// command payload while discarding execution-only fields from the public plan.
func processInstanceVariableUpdatePreviewFromFacadePlan(plan process.ProcessInstanceVariableUpdatePlan) processInstanceVariableUpdatePreview {
	instances := make([]processInstanceVariableUpdatePlan, len(plan.Instances))
	for i, instance := range plan.Instances {
		instances[i] = processInstanceVariableUpdatePlan{
			ProcessInstanceKey: instance.ProcessInstanceKey,
			Additions:          processInstanceVariablePlannedValuesFromFacade(instance.Additions),
			Changes:            processInstanceVariablePlannedChangesFromFacade(instance.Changes),
			UnchangedRequested: processInstanceVariablePlannedValuesFromFacade(instance.UnchangedRequested),
			Untouched:          processInstanceVariablePlannedValuesFromFacade(instance.Untouched),
		}
	}
	return processInstanceVariableUpdatePreview{
		Operation:              "update",
		RequestedKeys:          append([]string(nil), plan.RequestedKeys...),
		RequestedCount:         plan.RequestedCount,
		UpdateCount:            plan.UpdateCount,
		VariableAddCount:       plan.VariableAddCount,
		VariableChangeCount:    plan.VariableChangeCount,
		VariableUnchangedCount: plan.VariableUnchangedCount,
		VariableUntouchedCount: plan.VariableUntouchedCount,
		ProcessInstances:       instances,
		MutationSubmitted:      false,
	}
}

// processInstanceVariablePlannedValuesFromFacade copies classified values
// into the command-only preview representation without reclassification.
func processInstanceVariablePlannedValuesFromFacade(values []process.ProcessInstanceVariablePlannedValue) []processInstanceVariablePlannedValue {
	if values == nil {
		return nil
	}
	out := make([]processInstanceVariablePlannedValue, len(values))
	for i, value := range values {
		out[i] = processInstanceVariablePlannedValue{Name: value.Name, Value: value.Value, APITruncated: value.APITruncated}
	}
	return out
}

// processInstanceVariablePlannedChangesFromFacade copies classified changes
// into the command-only preview representation without reclassification.
func processInstanceVariablePlannedChangesFromFacade(changes []process.ProcessInstanceVariablePlannedChange) []processInstanceVariablePlannedChange {
	if changes == nil {
		return nil
	}
	out := make([]processInstanceVariablePlannedChange, len(changes))
	for i, change := range changes {
		out[i] = processInstanceVariablePlannedChange{
			Name: change.Name, Before: change.Before, After: change.After, APITruncated: change.APITruncated,
		}
	}
	return out
}
