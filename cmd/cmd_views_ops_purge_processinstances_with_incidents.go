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
	renderOpsPurgeProcessInstancesWithIncidentsReportFile(cmd, result)
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
	renderOpsPurgeProcessInstancesWithIncidentsDiscoveryStatus(cmd, result.Discovery.DiscoveryScopeStatus)
	if len(result.Discovery.DuplicateCandidateProcessInstanceKeys) > 0 {
		renderHumanLine(cmd, "duplicate candidate process instances: %d", len(result.Discovery.DuplicateCandidateProcessInstanceKeys))
	}
	if len(result.Discovery.SkippedIncidents) > 0 {
		renderHumanLine(cmd, "skipped incidents: %d", len(result.Discovery.SkippedIncidents))
	}
	renderOpsPurgeProcessInstancesWithIncidentsNotices(cmd, result.Discovery.Notices)
	if flagVerbose {
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "incident keys", result.Discovery.IncidentKeys)
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "candidate process-instance keys", result.Discovery.CandidateProcessInstanceKeys)
		renderOpsPurgeProcessInstancesWithIncidentsKeys(cmd, "duplicate candidate process-instance keys", result.Discovery.DuplicateCandidateProcessInstanceKeys)
		renderOpsPurgeProcessInstancesWithIncidentsSkipped(cmd, result.Discovery.SkippedIncidents)
	}
}

func renderOpsPurgeProcessInstancesWithIncidentsDiscoveryStatus(cmd *cobra.Command, status ops.DiscoveryScopeStatus) {
	renderOpsDiscoveryStatus(cmd, status)
}

func renderOpsPurgeProcessInstancesWithIncidentsNotices(cmd *cobra.Command, notices []ops.IncidentPurgeWorkflowNotice) {
	for _, notice := range notices {
		if notice.Code != "bounded_search_scope" || notice.Message == "" {
			continue
		}
		renderHumanLine(cmd, "%s", notice.Message)
	}
}

// renderOpsPurgeProcessInstancesWithIncidentsPlan prints the current delete-plan step status.
func renderOpsPurgeProcessInstancesWithIncidentsPlan(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.DeletePlan.Status == "" {
		return
	}
	if result.Request.DryRun {
		renderOpsPurgeProcessInstancesWithIncidentsDryRunDeletePreview(cmd, result)
		return
	}
	if result.DeletePlan.Status == ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "delete plan: skipped")
		return
	}
	renderHumanLine(cmd, "delete plan: %s; %d candidate incident(s), %d candidate process instance(s), %d affected process instance(s) across %d root(s) will be deleted",
		result.DeletePlan.Status,
		result.Discovery.IncidentCount,
		len(result.DeletePlan.CandidateProcessInstanceKeys),
		len(result.DeletePlan.AffectedKeys),
		len(result.DeletePlan.ResolvedRootKeys),
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

func renderOpsPurgeProcessInstancesWithIncidentsDryRunDeletePreview(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.DeletePlan.Status == ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "delete preview: skipped (no incident process-instance targets)")
		return
	}
	renderHumanLine(cmd, "delete preview: %d candidate incident(s), %d candidate process instance(s), %d affected process instance(s) across %d root(s) would be deleted",
		result.Discovery.IncidentCount,
		len(result.DeletePlan.CandidateProcessInstanceKeys),
		len(result.DeletePlan.AffectedKeys),
		len(result.DeletePlan.ResolvedRootKeys),
	)
	renderOpsProcessInstanceDependencyExpansion(cmd, len(result.DeletePlan.CandidateProcessInstanceKeys), len(result.DeletePlan.AffectedKeys))
	if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
		renderHumanLine(cmd, "non-final affected process instances: %d (use --force to cancel before delete)", len(result.DeletePlan.NonFinalAffectedItems))
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
	renderHumanLine(cmd, "deletion: %s", opsWorkflowDeletionSummary(string(result.Deletion.Status), len(result.Deletion.Items), "process-instance tree", "process-instance trees", result.Deletion.NoWait))
}

// renderOpsPurgeProcessInstancesWithIncidentsOutcome prints the final workflow outcome with hidden-key guidance.
func renderOpsPurgeProcessInstancesWithIncidentsOutcome(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.Outcome == "" {
		return
	}
	elapsed := opsWorkflowElapsedSuffix(result.Report.Duration)
	if !result.Deletion.Submitted && result.Outcome == ops.IncidentPurgeOutcomePlanned {
		line := fmt.Sprintf("outcome: %s; no changes applied", result.Outcome)
		if !flagVerbose && incidentPurgeHasHiddenKeys(result) {
			line += "; use --verbose to list process-instance keys"
		}
		line += elapsed
		renderHumanLine(cmd, "%s", line)
		return
	}
	renderHumanLine(cmd, "outcome: %s%s", result.Outcome, elapsed)
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

// renderOpsPurgeProcessInstancesWithIncidentsReportFile prints the compact audit report location.
func renderOpsPurgeProcessInstancesWithIncidentsReportFile(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.Request.ReportFile == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", result.Request.ReportFile)
}
