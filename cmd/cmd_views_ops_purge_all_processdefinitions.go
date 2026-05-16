// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderOpsPurgeAllProcessDefinitionsResult renders all-process-definitions purge output through the shared contract.
func renderOpsPurgeAllProcessDefinitionsResult(cmd *cobra.Command, result ops.AllProcessDefinitionsPurgeResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: purge all process definitions")
	} else {
		renderHumanLine(cmd, "purge all process definitions")
	}
	renderOpsPurgeAllProcessDefinitionsDiscovery(cmd, result)
	renderOpsPurgeAllProcessDefinitionsPlan(cmd, result)
	renderOpsPurgeAllProcessDefinitionsDeletion(cmd, result)
	renderOpsPurgeAllProcessDefinitionsOutcome(cmd, result)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

// renderOpsPurgeAllProcessDefinitionsDiscovery prints candidate discovery counts and verbose key details.
func renderOpsPurgeAllProcessDefinitionsDiscovery(cmd *cobra.Command, result ops.AllProcessDefinitionsPurgeResult) {
	if filters := result.Discovery.Filters.String(); filters != "" {
		renderHumanLine(cmd, "selection filters: %s", filters)
	}
	if result.Discovery.Status == "" {
		return
	}
	renderHumanLine(cmd, "candidate process definitions: %d", result.Discovery.CandidateProcessDefinitionCount)
	if result.Discovery.LatestOnly {
		renderHumanLine(cmd, "candidate scope: latest matching process definitions")
	}
	if len(result.Discovery.DuplicateCandidateProcessDefinitionKeys) > 0 {
		renderHumanLine(cmd, "duplicate candidate process definitions: %d", len(result.Discovery.DuplicateCandidateProcessDefinitionKeys))
	}
	if flagVerbose {
		renderOpsPurgeAllProcessDefinitionsKeys(cmd, "candidate process-definition keys", result.Discovery.CandidateProcessDefinitionKeys)
		renderOpsPurgeAllProcessDefinitionsKeys(cmd, "duplicate candidate process-definition keys", result.Discovery.DuplicateCandidateProcessDefinitionKeys)
	}
}

// renderOpsPurgeAllProcessDefinitionsPlan prints the current delete-plan step status.
func renderOpsPurgeAllProcessDefinitionsPlan(cmd *cobra.Command, result ops.AllProcessDefinitionsPurgeResult) {
	if result.DeletePlan.Status == "" {
		return
	}
	if result.DeletePlan.Status == ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "delete plan: skipped")
		return
	}
	renderHumanLine(cmd, "delete plan: %s (candidate process definitions: %d, affected process instances: %d)",
		result.DeletePlan.Status,
		len(result.DeletePlan.CandidateProcessDefinitionKeys),
		result.DeletePlan.AffectedProcessInstanceCount,
	)
	if result.DeletePlan.RequiresForce {
		renderHumanLine(cmd, "active-instance blocker: %d active process instances require --force before deletion", result.DeletePlan.ActiveProcessInstanceCount)
	}
	if flagVerbose {
		renderOpsPurgeAllProcessDefinitionsKeys(cmd, "planned candidate process-definition keys", result.DeletePlan.CandidateProcessDefinitionKeys)
	}
}

// renderOpsPurgeAllProcessDefinitionsDeletion prints deletion status when the workflow reaches mutation.
func renderOpsPurgeAllProcessDefinitionsDeletion(cmd *cobra.Command, result ops.AllProcessDefinitionsPurgeResult) {
	if result.Deletion.Status == "" || (!result.Deletion.Submitted && !flagVerbose) {
		return
	}
	if !result.Deletion.Submitted {
		renderHumanLine(cmd, "deletion: %s; no deletion request submitted", result.Deletion.Status)
		return
	}
	renderHumanLine(cmd, "deletion: %s (submitted process-definition deletes: %d)", result.Deletion.Status, len(result.Deletion.Items))
	if result.Deletion.NoWait {
		renderHumanLine(cmd, "deletion confirmation: skipped (--no-wait)")
	} else {
		renderHumanLine(cmd, "deletion confirmation: %t", result.Deletion.Confirmed)
	}
}

// renderOpsPurgeAllProcessDefinitionsOutcome prints the final workflow outcome with hidden-key guidance.
func renderOpsPurgeAllProcessDefinitionsOutcome(cmd *cobra.Command, result ops.AllProcessDefinitionsPurgeResult) {
	if result.Outcome == "" {
		return
	}
	if !result.Deletion.Submitted && result.Outcome == ops.AllProcessDefinitionsPurgeOutcomePlanned {
		line := fmt.Sprintf("outcome: %s; no changes applied", result.Outcome)
		if !flagVerbose && allProcessDefinitionsPurgeHasHiddenKeys(result) {
			line += "; use --verbose to list process-definition keys"
		}
		renderHumanLine(cmd, "%s", line)
		return
	}
	renderHumanLine(cmd, "outcome: %s", result.Outcome)
}

// allProcessDefinitionsPurgeHasHiddenKeys reports whether compact output suppressed verbose key details.
func allProcessDefinitionsPurgeHasHiddenKeys(result ops.AllProcessDefinitionsPurgeResult) bool {
	return len(result.Discovery.CandidateProcessDefinitionKeys) > 0 ||
		len(result.Discovery.DuplicateCandidateProcessDefinitionKeys) > 0 ||
		len(result.DeletePlan.CandidateProcessDefinitionKeys) > 0 ||
		len(result.Deletion.SubmittedProcessDefinitionKeys) > 0
}

// renderOpsPurgeAllProcessDefinitionsKeys prints a comma-separated key list for verbose output.
func renderOpsPurgeAllProcessDefinitionsKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(keys, ", "))
}
