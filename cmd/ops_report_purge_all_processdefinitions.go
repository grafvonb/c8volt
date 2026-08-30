// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
)

// renderOpsPurgeAllProcessDefinitionsJSONReport encodes the complete audit report deterministically.
func renderOpsPurgeAllProcessDefinitionsJSONReport(report ops.AllProcessDefinitionsPurgeReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderOpsPurgeAllProcessDefinitionsMarkdownReport renders a readable all-process-definitions purge audit report.
func renderOpsPurgeAllProcessDefinitionsMarkdownReport(report ops.AllProcessDefinitionsPurgeReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Purge All Process Definitions Audit Report\n\n")
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
	writeMarkdownReportField(&out, "Candidate Process Definitions", fmt.Sprintf("%d", report.Discovery.CandidateProcessDefinitionCount))
	writeMarkdownReportField(&out, "Latest Only", fmt.Sprintf("%t", report.Discovery.LatestOnly))
	writeMarkdownReportList(&out, "Candidate Process-Definition Keys", report.Discovery.CandidateProcessDefinitionKeys)
	writeMarkdownReportList(&out, "Candidate Process Definitions", allProcessDefinitionsPurgeDefinitionItems(report.Discovery.CandidateProcessDefinitions))
	writeMarkdownReportList(&out, "Duplicate Candidate Process-Definition Keys", report.Discovery.DuplicateCandidateProcessDefinitionKeys)
	writeMarkdownReportList(&out, "Notices", allProcessDefinitionsPurgeNoticeItems(report.Discovery.Notices))
	writeMarkdownReportList(&out, "Errors", report.Discovery.Errors)

	out.WriteString("\n## Delete Plan\n\n")
	writeMarkdownReportField(&out, "Status", string(report.DeletePlan.Status))
	writeMarkdownReportField(&out, "Requires Confirmation", fmt.Sprintf("%t", report.DeletePlan.RequiresConfirmation))
	writeMarkdownReportField(&out, "Requires Force", fmt.Sprintf("%t", report.DeletePlan.RequiresForce))
	writeMarkdownReportField(&out, "Affected Process Instances", fmt.Sprintf("%d", report.DeletePlan.AffectedProcessInstanceCount))
	writeMarkdownReportField(&out, "Active Process Instances", fmt.Sprintf("%d", report.DeletePlan.ActiveProcessInstanceCount))
	writeMarkdownReportList(&out, "Candidate Process-Definition Keys", report.DeletePlan.CandidateProcessDefinitionKeys)
	writeMarkdownReportList(&out, "Duplicate Candidate Process-Definition Keys", report.DeletePlan.DuplicateCandidateProcessDefinitionKeys)
	writeMarkdownReportList(&out, "Affected Process-Instance Keys", allProcessDefinitionsPurgeAffectedProcessInstanceKeys(report.DeletePlan))
	writeMarkdownReportList(&out, "Blocked Process-Instance Keys", allProcessDefinitionsPurgeBlockedProcessInstanceKeys(report.DeletePlan))
	if len(report.DeletePlan.Items) > 0 {
		out.WriteString("- Items:\n")
		for _, item := range report.DeletePlan.Items {
			out.WriteString(fmt.Sprintf("  - key=%s activeProcessInstances=%d affectedProcessInstances=%d\n", item.Key, item.ActiveProcessInstances(), len(item.CancellationPlan.Collected)))
		}
	}
	writeMarkdownReportList(&out, "Errors", report.DeletePlan.Errors)

	out.WriteString("\n## Deletion\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Deletion.Status))
	writeMarkdownReportField(&out, "Submitted", fmt.Sprintf("%t", report.Deletion.Submitted))
	writeMarkdownReportField(&out, "Confirmed", fmt.Sprintf("%t", report.Deletion.Confirmed))
	writeMarkdownReportField(&out, "No Wait", fmt.Sprintf("%t", report.Deletion.NoWait))
	writeMarkdownReportList(&out, "Submitted Process-Definition Keys", report.Deletion.SubmittedProcessDefinitionKeys)
	if len(report.Deletion.Items) > 0 {
		out.WriteString("- Items:\n")
		for _, item := range report.Deletion.Items {
			out.WriteString(fmt.Sprintf("  - key=%s ok=%t status=%s statusCode=%d\n", item.Key, item.Ok, item.Status, item.StatusCode))
		}
	}
	writeMarkdownReportList(&out, "Errors", report.Deletion.Errors)
	writeMarkdownReportList(&out, "Run Notices", allProcessDefinitionsPurgeNoticeItems(report.Notices))
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}

// allProcessDefinitionsPurgeAffectedProcessInstanceKeys extracts the affected process-instance key set from delete-plan items.
func allProcessDefinitionsPurgeAffectedProcessInstanceKeys(plan ops.AllProcessDefinitionsPurgeDeletePlan) []string {
	var keys []string
	for _, item := range plan.Items {
		keys = append(keys, item.ActiveProcessInstanceKeys...)
		keys = append(keys, item.CancellationPlan.Collected...)
	}
	return toolx.UniqueSlice(keys)
}

// allProcessDefinitionsPurgeBlockedProcessInstanceKeys extracts active keys that require force before deletion.
func allProcessDefinitionsPurgeBlockedProcessInstanceKeys(plan ops.AllProcessDefinitionsPurgeDeletePlan) []string {
	if !plan.RequiresForce {
		return nil
	}
	var keys []string
	for _, item := range plan.Items {
		keys = append(keys, item.ActiveProcessInstanceKeys...)
		for _, instance := range item.CancellationPlan.RequiresCancelBeforeDelete {
			if instance.Key != "" {
				keys = append(keys, instance.Key)
			}
		}
	}
	return toolx.UniqueSlice(keys)
}

// allProcessDefinitionsPurgeDefinitionItems formats candidate metadata for Markdown reports.
func allProcessDefinitionsPurgeDefinitionItems(definitions []process.ProcessDefinition) []string {
	out := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		item := definition.Key
		if item == "" {
			item = "<unknown>"
		}
		if definition.BpmnProcessId != "" {
			item += " bpmnProcessId=" + definition.BpmnProcessId
		}
		if definition.ProcessVersion > 0 {
			item += fmt.Sprintf(" version=%d", definition.ProcessVersion)
		}
		if definition.ProcessVersionTag != "" {
			item += " versionTag=" + definition.ProcessVersionTag
		}
		out = append(out, item)
	}
	return out
}

// allProcessDefinitionsPurgeNoticeItems formats structured notices without dropping report-only details.
func allProcessDefinitionsPurgeNoticeItems(items []ops.AllProcessDefinitionsPurgeNotice) []string {
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
