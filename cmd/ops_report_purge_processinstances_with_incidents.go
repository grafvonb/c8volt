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
)

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
	writeMarkdownTenantContext(&out, report.TenantContext)
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
	writeMarkdownReportField(&out, "Completeness", incidentPurgeDiscoveryCompletenessText(report.Discovery.DiscoveryScopeStatus))
	writeMarkdownReportField(&out, "Discovery Limit", fmt.Sprintf("%d", report.Discovery.Limit))
	writeMarkdownReportField(&out, "Discovery Batch Size", fmt.Sprintf("%d", report.Discovery.BatchSize))
	writeMarkdownReportField(&out, "Discovery Pages", fmt.Sprintf("%d", report.Discovery.Pages))
	writeMarkdownReportField(&out, "Discovery Candidates Seen", fmt.Sprintf("%d", report.Discovery.CandidatesSeen))
	writeMarkdownReportField(&out, "Discovery Candidates Frozen", fmt.Sprintf("%d", report.Discovery.CandidatesFrozen))
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

func incidentPurgeDiscoveryCompletenessText(status ops.DiscoveryScopeStatus) string {
	if status.Limited {
		return "discovery user-limited"
	}
	if status.Complete {
		return "discovery complete"
	}
	return "unknown"
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
