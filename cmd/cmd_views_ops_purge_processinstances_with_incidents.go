// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderOpsPurgeProcessInstancesWithIncidentsResult renders the incident purge workflow through the shared machine contract or compact human output.
func renderOpsPurgeProcessInstancesWithIncidentsResult(cmd *cobra.Command, result ops.IncidentPurgeResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: purge process-instances with incidents")
	} else {
		renderHumanLine(cmd, "purge process-instances with incidents")
	}
	renderOpsPurgeProcessInstancesWithIncidentsDiscovery(cmd, result)
	renderOpsPurgeProcessInstancesWithIncidentsPlan(cmd, result)
	renderOpsPurgeProcessInstancesWithIncidentsDeletion(cmd, result)
	renderOpsPurgeProcessInstancesWithIncidentsOutcome(cmd, result)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

// renderOpsPurgeProcessInstancesWithIncidentsDiscovery prints candidate discovery counts and verbose key details.
func renderOpsPurgeProcessInstancesWithIncidentsDiscovery(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if filters := result.Discovery.Filters.String(); filters != "" {
		renderHumanLine(cmd, "selection filters: %s", filters)
	}
	if result.Discovery.Status == "" {
		return
	}
	renderHumanLine(cmd, "candidate incidents: %d", result.Discovery.IncidentCount)
	renderHumanLine(cmd, "candidate process instances: %d", result.Discovery.CandidateProcessInstanceCount)
	if len(result.Discovery.DuplicateCandidateProcessInstanceKeys) > 0 {
		renderHumanLine(cmd, "duplicate candidate process instances: %d", len(result.Discovery.DuplicateCandidateProcessInstanceKeys))
	}
	if len(result.Discovery.SkippedIncidents) > 0 {
		renderHumanLine(cmd, "skipped incidents: %d", len(result.Discovery.SkippedIncidents))
	}
	if flagVerbose {
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "incident keys", result.Discovery.IncidentKeys)
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "candidate process-instance keys", result.Discovery.CandidateProcessInstanceKeys)
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "duplicate candidate process-instance keys", result.Discovery.DuplicateCandidateProcessInstanceKeys)
		renderOpsPurgeProcessInstancesWithIncidentsSkipped(cmd, result.Discovery.SkippedIncidents)
	}
}

// renderOpsPurgeProcessInstancesWithIncidentsPlan prints the current delete-plan step status.
func renderOpsPurgeProcessInstancesWithIncidentsPlan(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.DeletePlan.Status == "" {
		return
	}
	if result.DeletePlan.Status == ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "delete plan: skipped")
		return
	}
	renderHumanLine(cmd, "delete plan: %s (candidate process instances: %d, roots: %d, affected process instances: %d)",
		result.DeletePlan.Status,
		len(result.DeletePlan.CandidateProcessInstanceKeys),
		len(result.DeletePlan.ResolvedRootKeys),
		len(result.DeletePlan.AffectedKeys),
	)
	if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
		renderHumanLine(cmd, "non-final affected process instances: %d", len(result.DeletePlan.NonFinalAffectedItems))
	}
	if flagVerbose {
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "resolved root keys", result.DeletePlan.ResolvedRootKeys)
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "affected process-instance keys", result.DeletePlan.AffectedKeys)
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "duplicate resolved root keys", result.DeletePlan.DuplicateResolvedRootKeys)
	}
}

// renderOpsPurgeProcessInstancesWithIncidentsDeletion prints deletion status when the workflow reaches mutation.
func renderOpsPurgeProcessInstancesWithIncidentsDeletion(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.Deletion.Status == "" || (!result.Deletion.Submitted && !flagVerbose) {
		return
	}
	if !result.Deletion.Submitted {
		renderHumanLine(cmd, "deletion: %s; no deletion request submitted", result.Deletion.Status)
		return
	}
	renderHumanLine(cmd, "deletion: %s (requests: %d)", result.Deletion.Status, len(result.Deletion.Items))
	if result.Deletion.NoWait {
		renderHumanLine(cmd, "deletion confirmation: skipped (--no-wait)")
	} else {
		renderHumanLine(cmd, "deletion confirmation: %t", result.Deletion.Confirmed)
	}
}

// renderOpsPurgeProcessInstancesWithIncidentsOutcome prints the final workflow outcome with hidden-key guidance.
func renderOpsPurgeProcessInstancesWithIncidentsOutcome(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.Outcome == "" {
		return
	}
	if !result.Deletion.Submitted && result.Outcome == ops.IncidentPurgeOutcomePlanned {
		line := fmt.Sprintf("outcome: %s; no changes applied", result.Outcome)
		if !flagVerbose && incidentPurgeHasHiddenKeys(result) {
			line += "; use --verbose to list process-instance keys"
		}
		renderHumanLine(cmd, "%s", line)
		return
	}
	renderHumanLine(cmd, "outcome: %s", result.Outcome)
}

// incidentPurgeHasHiddenKeys reports whether compact output suppressed verbose key details.
func incidentPurgeHasHiddenKeys(result ops.IncidentPurgeResult) bool {
	return len(result.Discovery.IncidentKeys) > 0 ||
		len(result.Discovery.CandidateProcessInstanceKeys) > 0 ||
		len(result.Discovery.DuplicateCandidateProcessInstanceKeys) > 0 ||
		len(result.DeletePlan.ResolvedRootKeys) > 0 ||
		len(result.DeletePlan.AffectedKeys) > 0
}

// renderOpsPurgeProcessInstancesWithIncidentsKeys prints a comma-separated key list for verbose output.
func renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(keys, ", "))
}

// renderOpsPurgeProcessInstancesWithIncidentsSkipped prints skipped incident details for verbose output.
func renderOpsPurgeProcessInstancesWithIncidentsSkipped(cmd *cobra.Command, skipped []ops.IncidentPurgeSkippedIncident) {
	if len(skipped) == 0 {
		renderHumanLine(cmd, "skipped incident keys: none")
		return
	}
	items := make([]string, 0, len(skipped))
	for _, item := range skipped {
		key := item.Incident.IncidentKey
		if key == "" {
			key = "<unknown>"
		}
		if item.Reason != "" {
			key += " (" + item.Reason + ")"
		}
		items = append(items, key)
	}
	renderHumanLine(cmd, "skipped incident keys: %s", strings.Join(items, ", "))
}
