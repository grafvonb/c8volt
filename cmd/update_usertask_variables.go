// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// validateUpdateUserTaskJSONConfirmation keeps machine output prompt-free and stable.
func validateUpdateUserTaskJSONConfirmation(cmd *cobra.Command) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update user-task")
	}
	if flagDryRun || pickMode() != RenderModeJSON || shouldImplicitlyConfirm(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json update user-task requires --dry-run, --auto-confirm, or --automation")
}

// planUpdateUserTaskVariables delegates complete scope discovery and freezes the returned plan.
func planUpdateUserTaskVariables(cmd *cobra.Command, cli task.API, keys types.Keys, variables map[string]any) (task.UserTaskVariableUpdatePlan, error) {
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("preparing update variable plan for %d user task(s)", len(keys)))
	defer stopActivity()

	plan, err := cli.PlanUserTaskVariableUpdates(cmd.Context(), keys, variables, collectExplicitAdminInputOptions()...)
	if err != nil {
		return task.UserTaskVariableUpdatePlan{}, err
	}
	attachTenantContext(cmd, userTaskUpdateTenantContext(plan.TenantContext))
	return plan, nil
}

// userTaskVariablePlanHasChanges reports whether the frozen plan has physical scope writes.
func userTaskVariablePlanHasChanges(plan task.UserTaskVariableUpdatePlan) bool {
	return len(plan.Targets) > 0
}

// userTaskUpdateTenantContext converts the facade progress schema to the shared command schema.
func userTaskUpdateTenantContext(source foptions.TenantContext) tenant.Context {
	warnings := make([]tenant.ContextWarning, len(source.Warnings))
	for i, warning := range source.Warnings {
		warnings[i] = tenant.ContextWarning{Code: tenant.ContextWarningCode(warning.Code), Message: warning.Message}
	}
	return tenant.Context{
		Mode:               tenant.ContextMode(source.Mode),
		Filter:             tenant.ContextFilter(source.Filter),
		ConfiguredTenantID: source.ConfiguredTenantID,
		TargetTenantID:     source.TargetTenantID,
		ResolvedTenantIDs:  append([]string(nil), source.ResolvedTenantIDs...),
		UnknownTargetCount: source.UnknownTargetCount,
		CrossTenant:        source.CrossTenant,
		Warnings:           warnings,
	}
}
