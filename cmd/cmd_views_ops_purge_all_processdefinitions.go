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
	renderOpsPurgeAllProcessDefinitionsReportFile(cmd, result)
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
		renderOpsPurgeAllProcessDefinitionsDefinitions(cmd, result.Discovery.CandidateProcessDefinitions)
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
		renderOpsPurgeAllProcessDefinitionsKeys(cmd, "affected process-instance keys", allProcessDefinitionsPurgeAffectedProcessInstanceKeys(result.DeletePlan))
		renderOpsPurgeAllProcessDefinitionsKeys(cmd, "blocked process-instance keys", allProcessDefinitionsPurgeBlockedProcessInstanceKeys(result.DeletePlan))
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

// renderOpsPurgeAllProcessDefinitionsReportFile prints the compact audit report location.
func renderOpsPurgeAllProcessDefinitionsReportFile(cmd *cobra.Command, result ops.AllProcessDefinitionsPurgeResult) {
	if result.Request.ReportFile == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", result.Request.ReportFile)
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

// renderOpsPurgeAllProcessDefinitionsDefinitions prints verbose candidate metadata when discovery has it.
func renderOpsPurgeAllProcessDefinitionsDefinitions(cmd *cobra.Command, definitions []process.ProcessDefinition) {
	if len(definitions) == 0 {
		renderHumanLine(cmd, "candidate process-definition details: none")
		return
	}
	items := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		item := definition.Key
		if item == "" {
			item = "<unknown>"
		}
		parts := make([]string, 0, 3)
		if definition.BpmnProcessId != "" {
			parts = append(parts, "bpmnProcessId="+definition.BpmnProcessId)
		}
		if definition.ProcessVersion > 0 {
			parts = append(parts, fmt.Sprintf("version=%d", definition.ProcessVersion))
		}
		if definition.ProcessVersionTag != "" {
			parts = append(parts, "versionTag="+definition.ProcessVersionTag)
		}
		if len(parts) > 0 {
			item += " (" + strings.Join(parts, ", ") + ")"
		}
		items = append(items, item)
	}
	renderHumanLine(cmd, "candidate process-definition details: %s", strings.Join(items, ", "))
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
	out.WriteString("# All Process Definitions Purge Audit Report\n\n")
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
