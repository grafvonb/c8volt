// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
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

func renderOpsPurgeProcessInstancesWithIncidentsDryRunDeletePreview(cmd *cobra.Command, result ops.IncidentPurgeResult) {
	if result.DeletePlan.Status == ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "delete preview: skipped (no incident process-instance targets)")
		return
	}
	renderHumanLine(cmd, "delete preview: %d incident(s), %d process-instance candidate(s), %d process-instance tree(s), %d process instance(s) would be deleted",
		result.Discovery.IncidentCount,
		len(result.DeletePlan.CandidateProcessInstanceKeys),
		len(result.DeletePlan.ResolvedRootKeys),
		len(result.DeletePlan.AffectedKeys),
	)
	if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
		renderHumanLine(cmd, "non-final process instances in scope: %d", len(result.DeletePlan.NonFinalAffectedItems))
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

// renderOpsPurgeProcessInstancesWithIncidentsJSONReport encodes the complete audit report deterministically.
func renderOpsPurgeProcessInstancesWithIncidentsJSONReport(report ops.IncidentPurgeReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderOpsPurgeProcessInstancesWithIncidentsMarkdownReport renders a readable incident-purge audit report.
func renderOpsPurgeProcessInstancesWithIncidentsMarkdownReport(report ops.IncidentPurgeReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Purge Process Instances With Incidents Audit Report\n\n")
	writeMarkdownReportField(&out, "Schema Version", report.SchemaVersion)
	writeMarkdownReportField(&out, "Command", report.CommandName)
	writeMarkdownReportField(&out, "Started", formatOpsPurgeReportTime(report.StartedAt, cfg))
	writeMarkdownReportField(&out, "Finished", formatOpsPurgeReportTime(report.FinishedAt, cfg))
	writeMarkdownReportField(&out, "Duration", report.Duration)
	writeMarkdownReportField(&out, "Dry Run", fmt.Sprintf("%t", report.DryRun))
	writeMarkdownReportField(&out, "C8volt Version", report.C8voltVersion)
	writeMarkdownReportField(&out, "Camunda Version", report.CamundaVersion)
	writeMarkdownReportField(&out, "Profile", report.ProfileIdentity)
	writeMarkdownReportField(&out, "Tenant", report.TenantID)
	writeMarkdownReportField(&out, "Auto Confirm", fmt.Sprintf("%t", report.AutoConfirm))
	writeMarkdownReportField(&out, "Automation", fmt.Sprintf("%t", report.Automation))
	writeMarkdownReportField(&out, "No Wait", fmt.Sprintf("%t", report.NoWait))
	writeMarkdownReportField(&out, "Force", fmt.Sprintf("%t", report.Force))
	writeMarkdownReportField(&out, "Fail Fast", fmt.Sprintf("%t", report.FailFast))
	writeMarkdownReportField(&out, "No Worker Limit", fmt.Sprintf("%t", report.NoWorkerLimit))
	writeMarkdownReportField(&out, "Outcome", string(report.Outcome))

	out.WriteString("\n## Selection\n\n")
	writeMarkdownReportField(&out, "Filters", report.SelectionFilters.String())

	out.WriteString("\n## Discovery\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Discovery.Status))
	writeMarkdownReportField(&out, "Candidate Incidents", fmt.Sprintf("%d", report.Discovery.IncidentCount))
	writeMarkdownReportField(&out, "Candidate Process Instances", fmt.Sprintf("%d", report.Discovery.CandidateProcessInstanceCount))
	writeMarkdownReportList(&out, "Incident Keys", report.Discovery.IncidentKeys)
	writeMarkdownReportList(&out, "Candidate Process-Instance Keys", report.Discovery.CandidateProcessInstanceKeys)
	writeMarkdownReportList(&out, "Duplicate Candidate Process-Instance Keys", report.Discovery.DuplicateCandidateProcessInstanceKeys)
	writeMarkdownReportList(&out, "Skipped Incidents", incidentPurgeSkippedIncidentItems(report.Discovery.SkippedIncidents))
	writeMarkdownReportList(&out, "Notices", incidentPurgeNoticeItems(report.Discovery.Notices))
	writeMarkdownReportList(&out, "Errors", report.Discovery.Errors)

	out.WriteString("\n## Delete Plan\n\n")
	writeMarkdownReportField(&out, "Status", string(report.DeletePlan.Status))
	writeMarkdownReportField(&out, "Requires Confirmation", fmt.Sprintf("%t", report.DeletePlan.RequiresConfirmation))
	writeMarkdownReportList(&out, "Candidate Process-Instance Keys", report.DeletePlan.CandidateProcessInstanceKeys)
	writeMarkdownReportList(&out, "Resolved Root Keys", report.DeletePlan.ResolvedRootKeys)
	writeMarkdownReportList(&out, "Affected Process-Instance Keys", report.DeletePlan.AffectedKeys)
	writeMarkdownReportList(&out, "Duplicate Candidate Process-Instance Keys", report.DeletePlan.DuplicateCandidateProcessInstanceKeys)
	writeMarkdownReportList(&out, "Duplicate Resolved Root Keys", report.DeletePlan.DuplicateResolvedRootKeys)
	writeMarkdownReportList(&out, "Final State Items", retentionProcessInstanceItems(report.DeletePlan.FinalStateItems))
	writeMarkdownReportList(&out, "Non-Final Affected Items", retentionProcessInstanceItems(report.DeletePlan.NonFinalAffectedItems))
	writeMarkdownReportList(&out, "Missing Ancestors", retentionMissingAncestorItems(report.DeletePlan.MissingAncestors))
	writeMarkdownReportList(&out, "Traversal Warnings", report.DeletePlan.TraversalWarnings)
	writeMarkdownReportList(&out, "Errors", report.DeletePlan.Errors)

	out.WriteString("\n## Deletion\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Deletion.Status))
	writeMarkdownReportField(&out, "Submitted", fmt.Sprintf("%t", report.Deletion.Submitted))
	writeMarkdownReportField(&out, "Confirmed", fmt.Sprintf("%t", report.Deletion.Confirmed))
	writeMarkdownReportField(&out, "No Wait", fmt.Sprintf("%t", report.Deletion.NoWait))
	writeMarkdownReportList(&out, "Submitted Root Keys", report.Deletion.SubmittedRootKeys)
	if len(report.Deletion.Items) > 0 {
		out.WriteString("- Items:\n")
		for _, item := range report.Deletion.Items {
			out.WriteString(fmt.Sprintf("  - key=%s ok=%t status=%s statusCode=%d\n", item.Key, item.Ok, item.Status, item.StatusCode))
		}
	}
	writeMarkdownReportList(&out, "Errors", report.Deletion.Errors)
	writeMarkdownReportList(&out, "Run Notices", incidentPurgeNoticeItems(report.Notices))
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}

// incidentPurgeSkippedIncidentItems formats skipped incidents for Markdown reports.
func incidentPurgeSkippedIncidentItems(items []ops.IncidentPurgeSkippedIncident) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := item.Incident.IncidentKey
		if key == "" {
			key = "<unknown>"
		}
		if item.Reason != "" {
			key += " reason=" + item.Reason
		}
		out = append(out, key)
	}
	return out
}

// incidentPurgeNoticeItems formats structured notices without dropping report-only details.
func incidentPurgeNoticeItems(items []ops.IncidentPurgeWorkflowNotice) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := item.Code
		if item.Severity != "" {
			text += " severity=" + item.Severity
		}
		if item.Message != "" {
			text += " message=" + item.Message
		}
		out = append(out, text)
	}
	return out
}
