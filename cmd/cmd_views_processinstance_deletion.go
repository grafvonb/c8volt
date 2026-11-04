// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

// businessIDDeletionResult preserves root items while exposing attempted members and the reviewed scope.
type businessIDDeletionResult struct {
	process.ProcessInstanceDeletionResult
	Scope processInstanceDryRunPreview `json:"scope"`
}

// renderBusinessIDDeletion writes one result, retaining partial member evidence without reading backend state.
func renderBusinessIDDeletion(cmd *cobra.Command, plan process.ProcessInstanceSelectedMutationPlan, reports process.ProcessInstanceDeletionResult, previewOnly bool, operationErr error) error {
	preview := selectedMutationPreview(plan)
	submitted := reports.MutationSubmitted || len(domain.ProcessInstanceMutationFailures(operationErr)) > 0
	for _, member := range reports.Members {
		if member.StatusCode >= 200 && member.StatusCode < 300 {
			submitted = true
		}
	}
	preview.MutationSubmitted = !previewOnly && submitted
	payload := any(businessIDDeletionResult{ProcessInstanceDeletionResult: reports, Scope: preview})
	if previewOnly {
		payload = preview
	}
	if pickMode() == RenderModeJSON {
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = payload
			return renderResultEnvelope(cmd, envelope)
		}
		if previewOnly || !preview.MutationSubmitted {
			return renderSucceededResult(cmd, payload)
		}
		return renderCommandResult(cmd, payload)
	}
	if previewOnly {
		if operationErr != nil {
			return nil
		}
		return renderProcessInstanceDryRunPreview(cmd, preview)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, report := range reports.Items {
			if report.Ok {
				renderOutputLine(cmd, "%s", report.Key)
			}
		}
		return nil
	}
	renderBusinessIDSelectionScope(cmd, plan.Selection)
	renderHumanLine(cmd, "selected process instances: %d; roots: %d; affected process instances: %d", len(plan.Selection.Keys), len(plan.Expansion.Roots), len(plan.Expansion.Collected))
	renderHumanLine(cmd, "dependency expansion: %d additional process instance(s) due to dependencies", max(0, len(plan.Expansion.Collected)-len(plan.Selection.Keys)))
	if operationErr == nil {
		renderProcessInstanceMutationResultSummary(cmd, "delete", toolx.MapSlice(reports.Items, func(r process.DeleteReport) process.Reporter { return process.Reporter(r) }), processInstancePageImpact{Requested: len(plan.Selection.Keys), Roots: len(plan.Expansion.Roots), Affected: len(plan.Expansion.Collected)})
	}
	if flagVerbose {
		printProcessInstanceDryRunKeys(cmd, "selected process-instance keys", preview.RequestedKeys)
		printProcessInstanceDryRunKeys(cmd, "root process-instance tree keys", preview.ResolvedRoots)
		printProcessInstanceDryRunKeys(cmd, "in-scope process-instance keys", preview.AffectedFamilyKeys)
		for _, member := range reports.Members {
			renderHumanLine(cmd, "deletion member %s: successful=%t", member.Key, member.Ok)
		}
	}
	return nil
}

// formatBusinessIDDeletionFailure reports bounded counts and reasons instead of raw transport chains.
func formatBusinessIDDeletionFailure(plan process.ProcessInstanceSelectedMutationPlan, reports process.ProcessInstanceDeletionResult, err error) string {
	_, ok, _ := reports.Totals()
	members, unconfirmed := 0, 0
	for _, member := range reports.Members {
		if member.Ok {
			members++
		} else if member.StatusCode >= 200 && member.StatusCode < 300 {
			unconfirmed++
		}
	}
	memberOutcome := "deleted"
	if flagNoWait {
		memberOutcome = "submitted"
	}
	message := fmt.Sprintf("delete process instances: selected %d, roots %d, successful %d, %s members %d; %s", len(plan.Selection.Keys), len(plan.Expansion.Roots), ok, memberOutcome, members, boundedCommandFailureReason(err))
	if unconfirmed > 0 {
		message += fmt.Sprintf("; deletion submitted for %d member(s), outcome unconfirmed", unconfirmed)
	}
	if len(plan.Expansion.RequiresCancelBeforeDelete) > 0 && !flagForce {
		message += "; nonterminal affected instances require --force"
	}
	if n := len(domain.ProcessInstanceMutationFailures(err)); n > 0 {
		message += fmt.Sprintf("; cancellation submitted for %d root(s), outcome unconfirmed", n)
	}
	return message
}
