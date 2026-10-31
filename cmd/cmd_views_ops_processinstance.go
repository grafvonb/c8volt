// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// renderOpsProcessInstancePurgeFailure keeps partial observations in the existing
// envelope and reports one bounded aggregate without exposing transport chains.
func renderOpsProcessInstancePurgeFailure(cmd *cobra.Command, log *slog.Logger, payload any, workflow string, selected int, items []process.DeleteReport, err, reportErr error) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		evidence := err
		if reportErr != nil {
			evidence = fmt.Errorf("%w; write audit report: %v", err, reportErr)
		}
		envelope := resultEnvelopeForError(cmd, evidence)
		envelope.Payload = payload
		return renderResultEnvelope(cmd, envelope)
	}
	accepted := 0
	for _, item := range items {
		if item.Ok {
			accepted++
		}
	}
	suffix := ""
	if reportErr != nil {
		suffix = "; audit report could not be written"
	}
	if blocked := opsProcessInstancePurgeBlockedMessage(payload); blocked != "" {
		log.Error(blocked + suffix)
		return nil
	}
	log.Error(fmt.Sprintf("%s: selected %d, accepted %d; outcome unconfirmed; %s%s", workflow, selected, accepted, boundedCommandFailureReason(err), suffix))
	return nil
}

// opsProcessInstancePurgeBlockedMessage derives actionable local blockers from
// structured plan evidence, never from an arbitrary wrapped error string.
func opsProcessInstancePurgeBlockedMessage(payload any) string {
	switch result := payload.(type) {
	case ops.RetentionPolicyResult:
		if result.Deletion.Status == ops.WorkflowStepStatusBlocked && len(result.DeletePlan.NonFinalAffectedItems) > 0 {
			return fmt.Sprintf("refusing to delete retention process-instance scope: %s; no delete request was submitted; use --force to cancel the non-final affected scope before delete", formatOpsExecuteRetentionPolicyNonFinalScope(result.DeletePlan))
		}
	case ops.IncidentPurgeResult:
		if result.Deletion.Status == ops.WorkflowStepStatusBlocked && len(result.DeletePlan.NonFinalAffectedItems) > 0 {
			return fmt.Sprintf("refusing to delete incident purge process-instance scope: %d non-final affected process instance(s); no delete request was submitted; use --force to cancel before delete", len(result.DeletePlan.NonFinalAffectedItems))
		}
	case ops.OrphanPurgeResult:
		if result.Deletion.Status == ops.WorkflowStepStatusBlocked && len(result.DeletionPlan.DryRunPreview.RequiresCancelBeforeDelete) > 0 {
			return fmt.Sprintf("refusing to delete orphan process-instance scope: %d affected process instance(s) are not in a final state; no delete request was submitted; use --force to cancel the entire affected scope before delete", len(result.DeletionPlan.DryRunPreview.RequiresCancelBeforeDelete))
		}
	}
	return ""
}

// renderOpsObservedProcessInstances logs only existing observations; key/status
// placeholders never trigger enrichment or inherit identity from another item.
func renderOpsObservedProcessInstances(cmd *cobra.Command, label string, items []process.ProcessInstance) {
	for _, row := range retentionProcessInstanceItems(items) {
		renderHumanLine(cmd, "%s: %s", label, row)
	}
}
