// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

func renderOpsPurgeOrphanProcessInstancesResult(cmd *cobra.Command, result ops.OrphanPurgeResult) error {
	switch pickMode() {
	case RenderModeJSON:
		return renderSucceededResult(cmd, result)
	case RenderModeKeysOnly:
		for _, key := range result.Discovery.Keys {
			renderOutputLine(cmd, "%s", key)
		}
		return nil
	default:
		return renderOpsPurgeOrphanProcessInstancesHuman(cmd, result)
	}
}

func renderOpsPurgeOrphanProcessInstancesHuman(cmd *cobra.Command, result ops.OrphanPurgeResult) error {
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: purge orphan process-instances")
	} else {
		renderHumanLine(cmd, "purge orphan process-instances")
	}
	if result.Discovery.Count == 0 {
		renderHumanLine(cmd, "candidate orphan process instances: 0")
		elapsed := opsWorkflowElapsedSuffix(result.Report.Duration)
		if result.Request.DryRun {
			renderHumanLine(cmd, "delete preview: skipped (no orphan process-instance targets)")
			renderOpsPurgeOrphanProcessInstancesReportFile(cmd, result)
			renderHumanLine(cmd, "outcome: planned; no changes applied%s", elapsed)
		} else {
			renderHumanLine(cmd, "delete plan: skipped")
			renderOpsPurgeOrphanProcessInstancesReportFile(cmd, result)
			renderHumanLine(cmd, "outcome: planned; no targets deleted%s", elapsed)
		}
		return nil
	}
	renderHumanLine(cmd, "selection filters: %s", result.Discovery.Filters.String())
	renderHumanLine(cmd, "candidate orphan process instances: %d", result.Discovery.Count)
	if flagVerbose {
		renderHumanLine(cmd, "candidate keys: %s", strings.Join(result.Discovery.Keys, ", "))
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "delete preview: %d orphan candidate(s), %d process-instance tree(s), %d process instance(s) would be deleted",
			len(result.DeletionPlan.RequestedKeys),
			len(result.DeletionPlan.RootKeys),
			len(result.DeletionPlan.AffectedKeys),
		)
	} else {
		renderHumanLine(cmd, "delete plan: %s (candidate orphan process instances: %d, roots: %d, affected process instances: %d)",
			result.DeletionPlan.Status,
			len(result.DeletionPlan.RequestedKeys),
			len(result.DeletionPlan.RootKeys),
			len(result.DeletionPlan.AffectedKeys),
		)
	}
	renderOpsHumanNotices(cmd, opsPurgeOrphanProcessInstancesHumanNotices(result), opsPurgeOrphanProcessInstancesNoticeFilter(result))
	if result.DeleteRequested {
		renderHumanLine(cmd, "deletion: %s (requests: %d)", result.Deletion.Status, len(result.Deletion.Items))
		if result.Deletion.NoWait {
			renderHumanLine(cmd, "deletion confirmation: skipped (--no-wait)")
		} else {
			renderHumanLine(cmd, "deletion confirmation: %t", result.Deletion.Confirmed)
		}
		renderOpsPurgeOrphanProcessInstancesReportFile(cmd, result)
		renderHumanLine(cmd, "outcome: %s%s", result.Outcome, opsWorkflowElapsedSuffix(result.Report.Duration))
		return nil
	}
	if flagVerbose {
		renderHumanLine(cmd, "deletion: %s; no deletion request submitted", result.Deletion.Status)
	}
	line := fmt.Sprintf("outcome: %s; no changes applied", result.Outcome)
	if !flagVerbose && len(result.DeletionPlan.AffectedKeys) > 0 {
		line += "; use --verbose to list process-instance keys"
	}
	line += opsWorkflowElapsedSuffix(result.Report.Duration)
	renderOpsPurgeOrphanProcessInstancesReportFile(cmd, result)
	renderHumanLine(cmd, "%s", line)
	return nil
}

func renderOpsPurgeOrphanProcessInstancesReportFile(cmd *cobra.Command, result ops.OrphanPurgeResult) {
	if result.Request.ReportFile == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", result.Request.ReportFile)
}

func opsPurgeOrphanProcessInstancesHumanNotices(result ops.OrphanPurgeResult) []opsHumanNotice {
	return []opsHumanNotice{
		{
			Level: opsHumanNoticeLevelWarning,
			Code:  opsHumanNoticeOrphanParentMissing,
			Text:  result.DeletionPlan.DryRunPreview.Warning,
		},
	}
}

func opsPurgeOrphanProcessInstancesNoticeFilter(_ ops.OrphanPurgeResult) opsHumanNoticeFilter {
	return func(notice opsHumanNotice) bool {
		if notice.Code == opsHumanNoticeOrphanParentMissing {
			return false
		}
		return true
	}
}

func renderOpsPurgeOrphanProcessInstancesJSONReport(report ops.OrphanPurgeReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderOpsPurgeOrphanProcessInstancesMarkdownReport(report ops.OrphanPurgeReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Orphan Process Instance Purge Audit Report\n\n")
	writeMarkdownReportField(&out, "Schema Version", report.SchemaVersion)
	writeMarkdownReportField(&out, "Command", report.CommandName)
	writeMarkdownReportField(&out, "Started", formatOpsPurgeReportTime(report.StartedAt, cfg))
	writeMarkdownReportField(&out, "Finished", formatOpsPurgeReportTime(report.FinishedAt, cfg))
	writeMarkdownReportField(&out, "Duration", report.Duration)
	writeMarkdownReportField(&out, "Dry Run", fmt.Sprintf("%t", report.DryRun))
	writeMarkdownReportField(&out, "C8volt Version", report.C8voltVersion)
	writeMarkdownReportField(&out, "Camunda Version", report.CamundaVersion)
	writeMarkdownReportField(&out, "Profile", report.ProfileIdentity)
	writeMarkdownReportField(&out, "Auto Confirm", fmt.Sprintf("%t", report.AutoConfirm))
	writeMarkdownReportField(&out, "Automation", fmt.Sprintf("%t", report.Automation))
	writeMarkdownReportField(&out, "No Wait", fmt.Sprintf("%t", report.NoWait))
	writeMarkdownReportField(&out, "Delete Requested", fmt.Sprintf("%t", report.DeleteRequested))
	writeMarkdownReportField(&out, "Outcome", string(report.Outcome))

	out.WriteString("\n## Selection\n\n")
	writeMarkdownReportField(&out, "Filters", report.SelectionFilters.String())

	out.WriteString("\n## Discovery\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Discovery.Status))
	writeMarkdownReportField(&out, "Count", fmt.Sprintf("%d", report.Discovery.Count))
	writeMarkdownReportList(&out, "Candidate Keys", report.Discovery.Keys)
	writeMarkdownReportList(&out, "Errors", report.Discovery.Errors)

	out.WriteString("\n## Deletion Plan\n\n")
	writeMarkdownReportField(&out, "Status", string(report.DeletionPlan.Status))
	writeMarkdownReportField(&out, "Requires Confirmation", fmt.Sprintf("%t", report.DeletionPlan.RequiresConfirmation))
	writeMarkdownReportList(&out, "Candidate Keys", report.DeletionPlan.RequestedKeys)
	writeMarkdownReportList(&out, "Root Keys", report.DeletionPlan.RootKeys)
	writeMarkdownReportList(&out, "Affected Keys", report.DeletionPlan.AffectedKeys)
	writeMarkdownReportList(&out, "Errors", report.DeletionPlan.Errors)

	out.WriteString("\n## Deletion\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Deletion.Status))
	writeMarkdownReportField(&out, "Submitted", fmt.Sprintf("%t", report.Deletion.Submitted))
	writeMarkdownReportField(&out, "Confirmed", fmt.Sprintf("%t", report.Deletion.Confirmed))
	writeMarkdownReportField(&out, "No Wait", fmt.Sprintf("%t", report.Deletion.NoWait))
	if len(report.Deletion.Items) > 0 {
		out.WriteString("- Items:\n")
		for _, item := range report.Deletion.Items {
			out.WriteString(fmt.Sprintf("  - key=%s ok=%t status=%s statusCode=%d\n", item.Key, item.Ok, item.Status, item.StatusCode))
		}
	}
	writeMarkdownReportList(&out, "Errors", report.Deletion.Errors)
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}

func writeMarkdownReportField(out *strings.Builder, name string, value string) {
	if value == "" {
		value = "-"
	}
	out.WriteString(fmt.Sprintf("- %s: %s\n", name, value))
}

func writeMarkdownReportList(out *strings.Builder, name string, values []string) {
	if len(values) == 0 {
		out.WriteString(fmt.Sprintf("- %s: -\n", name))
		return
	}
	out.WriteString(fmt.Sprintf("- %s:\n", name))
	for _, value := range values {
		out.WriteString(fmt.Sprintf("  - %s\n", value))
	}
}

func formatOpsPurgeReportTime(t time.Time, cfg *config.Config) string {
	if t.IsZero() {
		return ""
	}
	showTimezoneOffset := false
	if cfg != nil {
		showTimezoneOffset = cfg.App.ShowTimezoneOffset
	}
	return toolx.FormatTime(t.UTC(), showTimezoneOffset)
}
