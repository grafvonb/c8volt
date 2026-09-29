// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
)

type userTaskVariableUpdatePreview struct {
	Operation string `json:"operation"`
	task.UserTaskVariableUpdatePlan
}

// renderUpdateUserTaskVariablePreview renders a frozen plan without claiming submission.
func renderUpdateUserTaskVariablePreview(cmd *cobra.Command, plan task.UserTaskVariableUpdatePlan) error {
	if pickMode() == RenderModeJSON {
		return renderSucceededResult(cmd, userTaskVariableUpdatePreview{Operation: "update", UserTaskVariableUpdatePlan: plan})
	}
	if pickMode() == RenderModeKeysOnly {
		for _, item := range plan.UserTasks {
			if len(item.TargetScopeKeys) > 0 {
				renderOutputLine(cmd, "%s", item.UserTaskKey)
			}
		}
		return nil
	}
	renderUserTaskVariablePlanHuman(cmd, plan, "dry run")
	return nil
}

// renderUpdateUserTaskVariablePlan renders the confirmation or successful no-op plan.
func renderUpdateUserTaskVariablePlan(cmd *cobra.Command, plan task.UserTaskVariableUpdatePlan) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, userTaskVariableUpdatePreview{Operation: "update", UserTaskVariableUpdatePlan: plan})
	}
	if pickMode() == RenderModeKeysOnly {
		return nil
	}
	renderUserTaskVariablePlanHuman(cmd, plan, "plan")
	return nil
}

// renderUserTaskVariablePlanHuman keeps the PI summary rhythm while exposing inherited scope targets.
func renderUserTaskVariablePlanHuman(cmd *cobra.Command, plan task.UserTaskVariableUpdatePlan, label string) {
	renderAttachedTenantContext(cmd)
	if !userTaskVariablePlanHasChanges(plan) {
		status := "no confirmation required"
		if label == "dry run" {
			status = "no changes applied"
		}
		renderHumanLine(cmd, "%s: update user-task variables: nothing to update (%d requested value(s) already match visible variables); %s", label, plan.VariableUnchangedCount, status)
		return
	}
	summary := fmt.Sprintf("%s: update user-task variables: %d user task(s), %d change(s), %d addition(s), %d unchanged, %d untouched", label, plan.UpdateCount, plan.VariableChangeCount, plan.VariableAddCount, plan.VariableUnchangedCount, plan.VariableUntouchedCount)
	if label == "dry run" {
		summary += "; no changes applied"
	}
	renderHumanLine(cmd, "%s", summary)
	if len(plan.UserTasks) > 1 {
		if scopes := inheritedUserTaskVariableScopes(plan); len(scopes) > 0 {
			renderHumanLine(cmd, "inherited target scopes: %s", strings.Join(scopes, ", "))
		}
	}
	for _, item := range plan.UserTasks {
		if len(item.TargetScopeKeys) == 0 && !flagVerbose {
			continue
		}
		if flagVerbose || len(plan.UserTasks) == 1 {
			renderHumanLine(cmd, "%s: %s", item.UserTaskKey, formatUserTaskVariablePlan(item))
		}
	}
}

// inheritedUserTaskVariableScopes returns stable unique scope evidence for compact bulk plans.
func inheritedUserTaskVariableScopes(plan task.UserTaskVariableUpdatePlan) []string {
	seen := make(map[string]struct{})
	var scopes []string
	for _, item := range plan.UserTasks {
		for _, change := range item.Changes {
			if !change.Inherited {
				continue
			}
			if _, ok := seen[change.ScopeKey]; ok {
				continue
			}
			seen[change.ScopeKey] = struct{}{}
			scopes = append(scopes, change.ScopeKey)
		}
	}
	return scopes
}

// renderUpdateUserTaskVariableResults renders successful task outcomes without backend reads.
func renderUpdateUserTaskVariableResults(cmd *cobra.Command, results task.UserTaskVariableUpdateResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	return renderUpdateUserTaskVariableResultsHumanOrKeys(cmd, results)
}

// renderUpdateUserTaskVariableFailure preserves partial task outcomes in the single selected result surface.
func renderUpdateUserTaskVariableFailure(cmd *cobra.Command, results task.UserTaskVariableUpdateResults, operationErr error) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		normalized := ferrors.Normalize(operationErr)
		class := string(ferrors.Classify(normalized))
		return renderResultEnvelope(cmd, ResultEnvelope[task.UserTaskVariableUpdateResults]{
			Outcome: outcomeForError(normalized),
			Class:   class,
			Command: commandPath(cmd),
			Payload: results,
			Detail:  &ResultDetail{Message: strings.TrimSpace(normalized.Error()), Class: class},
		})
	}
	return renderUpdateUserTaskVariableResultsHumanOrKeys(cmd, results)
}

// renderUpdateUserTaskVariableResultsHumanOrKeys keeps keys-only pure and human results compact.
func renderUpdateUserTaskVariableResultsHumanOrKeys(cmd *cobra.Command, results task.UserTaskVariableUpdateResults) error {
	if pickMode() == RenderModeKeysOnly {
		for _, item := range results.Items {
			if item.Status == task.UserTaskVariableUpdateStatusConfirmed || item.Status == task.UserTaskVariableUpdateStatusSubmitted {
				renderOutputLine(cmd, "%s", item.Key)
			}
		}
		return nil
	}
	confirmedOrSubmitted := 0
	failed := 0
	for _, item := range results.Items {
		switch item.Status {
		case task.UserTaskVariableUpdateStatusConfirmed:
			confirmedOrSubmitted++
			renderHumanLine(cmd, "updated user-task %s: confirmed", item.Key)
		case task.UserTaskVariableUpdateStatusSubmitted:
			confirmedOrSubmitted++
			renderHumanLine(cmd, "updated user-task %s: submitted", item.Key)
		case task.UserTaskVariableUpdateStatusUnchanged:
			renderHumanLine(cmd, "updated user-task %s: unchanged", item.Key)
		case task.UserTaskVariableUpdateStatusConfirmationFailed:
			failed++
			renderHumanLine(cmd, "updated user-task %s: confirmation failed: %s", item.Key, item.Error)
		case task.UserTaskVariableUpdateStatusMutationFailed:
			failed++
			renderHumanLine(cmd, "updated user-task %s: mutation failed: %s", item.Key, item.Error)
		default:
			failed++
			renderHumanLine(cmd, "updated user-task %s: %s", item.Key, item.Status)
		}
	}
	renderHumanLine(cmd, "updated: %d (confirmed/submitted: %d, failed/skipped: %d)", len(results.Items), confirmedOrSubmitted, failed)
	return nil
}

// formatUserTaskVariablePlan uses established variable tokens and annotates inherited scope effects.
func formatUserTaskVariablePlan(plan task.UserTaskVariablePlan) string {
	parts := make([]string, 0, len(plan.Additions)+len(plan.Changes)+len(plan.UnchangedRequested)+1)
	for _, item := range plan.Changes {
		parts = append(parts, fmt.Sprintf("~ %s%s: %s -> %s", item.Name, formatUserTaskVariableScope(item.ScopeKey, item.Inherited), formatProcessInstanceVariablePlanValue(item.Before), formatProcessInstanceVariablePlanValue(item.After)))
	}
	for _, item := range plan.Additions {
		parts = append(parts, fmt.Sprintf("+ %s%s: %s", item.Name, formatUserTaskVariableScope(item.ScopeKey, item.Inherited), formatProcessInstanceVariablePlanValue(item.Value)))
	}
	for _, item := range plan.UnchangedRequested {
		parts = append(parts, fmt.Sprintf("~ %s%s: %s (unchanged)", item.Name, formatUserTaskVariableScope(item.ScopeKey, item.Inherited), formatProcessInstanceVariablePlanValue(item.Value)))
	}
	if len(plan.Untouched) > 0 {
		parts = append(parts, fmt.Sprintf("= %d variable(s) left untouched", len(plan.Untouched)))
	}
	if len(parts) == 0 {
		return "no variable changes"
	}
	return strings.Join(parts, "; ")
}

// formatUserTaskVariableScope labels only inherited effects in compact output.
func formatUserTaskVariableScope(scopeKey string, inherited bool) string {
	if !inherited {
		return ""
	}
	return " (inherited scope " + scopeKey + ")"
}
