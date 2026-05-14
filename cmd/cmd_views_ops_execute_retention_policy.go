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

func renderOpsExecuteRetentionPolicyResult(cmd *cobra.Command, result ops.RetentionPolicyResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	cmd.Printf("retention policy: %s\n", result.Outcome)
	cmd.Printf("retention days: %d\n", result.Request.RetentionDays)
	if result.Request.DerivedEndDateBoundary != "" {
		cmd.Printf("retention boundary: endDate <= %s\n", result.Request.DerivedEndDateBoundary)
	}
	if filters := result.Discovery.Filters.String(); filters != "" {
		cmd.Printf("selection filters: %s\n", filters)
	}
	if result.Discovery.Status != "" {
		cmd.Printf("retention discovery: %s\n", result.Discovery.Status)
		cmd.Printf("retention seeds: %d\n", result.Discovery.Count)
		if result.Discovery.Count == 0 {
			cmd.Printf("no retention cleanup targets found\n")
		}
	}
	if result.DeletePlan.Status != "" {
		cmd.Printf("delete plan: %s (seeds: %d, roots: %d, affected: %d)\n",
			result.DeletePlan.Status,
			len(result.DeletePlan.SeedKeys),
			len(result.DeletePlan.ResolvedRootKeys),
			len(result.DeletePlan.AffectedKeys),
		)
		if len(result.DeletePlan.DuplicateKeys) > 0 {
			cmd.Printf("duplicate roots: %d\n", len(result.DeletePlan.DuplicateKeys))
		}
		if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
			cmd.Printf("non-final descendants in final-root scope: %d (use --force to cancel before delete)\n", len(result.DeletePlan.NonFinalAffectedItems))
		}
		if len(result.DeletePlan.SkippedSeedKeys) > 0 {
			cmd.Printf("skipped retention seeds with non-final roots: %d\n", len(result.DeletePlan.SkippedSeedKeys))
		}
		if len(result.DeletePlan.MissingAncestors) > 0 {
			cmd.Printf("missing ancestors: %d\n", len(result.DeletePlan.MissingAncestors))
		}
		for _, warning := range result.DeletePlan.TraversalWarnings {
			if warning != "" {
				cmd.Printf("traversal warning: %s\n", warning)
			}
		}
		cmd.Printf("confirmation required: %t\n", result.DeletePlan.RequiresConfirmation)
		if flagVerbose {
			printOpsExecuteRetentionPolicyKeys(cmd, "retention seed keys", result.DeletePlan.SeedKeys)
			printOpsExecuteRetentionPolicyKeys(cmd, "skipped retention seed keys", result.DeletePlan.SkippedSeedKeys)
			printOpsExecuteRetentionPolicyItems(cmd, "skipped non-final roots", result.DeletePlan.SkippedNonFinalRoots)
			printOpsExecuteRetentionPolicyKeys(cmd, "resolved root keys", result.DeletePlan.ResolvedRootKeys)
			printOpsExecuteRetentionPolicyKeys(cmd, "affected process-instance keys", result.DeletePlan.AffectedKeys)
		}
	}
	if result.Deletion.Status != "" {
		if !result.Deletion.Submitted {
			cmd.Printf("deletion: %s; no deletion request submitted\n", result.Deletion.Status)
		} else {
			cmd.Printf("deletion: %s (requests: %d)\n", result.Deletion.Status, len(result.Deletion.Items))
			if result.Deletion.NoWait {
				cmd.Printf("deletion confirmation: skipped (--no-wait)\n")
			} else {
				cmd.Printf("deletion confirmation: %t\n", result.Deletion.Confirmed)
			}
		}
	}
	if result.Outcome != "" {
		if !result.Deletion.Submitted && result.Outcome == ops.RetentionPolicyOutcomePlanned {
			cmd.Printf("outcome: %s; no changes applied\n", result.Outcome)
		} else {
			cmd.Printf("outcome: %s\n", result.Outcome)
		}
	}
	renderOpsExecuteRetentionPolicyReportFile(cmd, result)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func printOpsExecuteRetentionPolicyKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		cmd.Printf("%s: none\n", label)
		return
	}
	cmd.Printf("%s: %s\n", label, strings.Join(keys, ", "))
}

func printOpsExecuteRetentionPolicyItems(cmd *cobra.Command, label string, items []process.ProcessInstance) {
	if len(items) == 0 {
		cmd.Printf("%s: none\n", label)
		return
	}
	cmd.Printf("%s: %s\n", label, strings.Join(retentionProcessInstanceItems(items), ", "))
}

func renderOpsExecuteRetentionPolicyReportFile(cmd *cobra.Command, result ops.RetentionPolicyResult) {
	if result.Request.ReportFile == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", result.Request.ReportFile)
}

func renderOpsExecuteRetentionPolicyJSONReport(report ops.RetentionAuditReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderOpsExecuteRetentionPolicyMarkdownReport(report ops.RetentionAuditReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Retention Policy Audit Report\n\n")
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
	writeMarkdownReportField(&out, "Retention Days", fmt.Sprintf("%d", report.RetentionDays))
	writeMarkdownReportField(&out, "Derived End Date Boundary", report.DerivedEndDateBoundary)
	writeMarkdownReportField(&out, "Auto Confirm", fmt.Sprintf("%t", report.AutoConfirm))
	writeMarkdownReportField(&out, "Automation", fmt.Sprintf("%t", report.Automation))
	writeMarkdownReportField(&out, "No Wait", fmt.Sprintf("%t", report.NoWait))
	writeMarkdownReportField(&out, "No State Check", fmt.Sprintf("%t", report.NoStateCheck))
	writeMarkdownReportField(&out, "Force", fmt.Sprintf("%t", report.Force))
	writeMarkdownReportField(&out, "Fail Fast", fmt.Sprintf("%t", report.FailFast))
	writeMarkdownReportField(&out, "No Worker Limit", fmt.Sprintf("%t", report.NoWorkerLimit))
	writeMarkdownReportField(&out, "Outcome", string(report.Outcome))

	out.WriteString("\n## Selection\n\n")
	writeMarkdownReportField(&out, "Filters", report.SelectionFilters.String())

	out.WriteString("\n## Discovery\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Discovery.Status))
	writeMarkdownReportField(&out, "Retention Days", fmt.Sprintf("%d", report.Discovery.RetentionDays))
	writeMarkdownReportField(&out, "Derived End Date Boundary", report.Discovery.DerivedEndDateBoundary)
	writeMarkdownReportField(&out, "Count", fmt.Sprintf("%d", report.Discovery.Count))
	writeMarkdownReportList(&out, "Seed Keys", report.Discovery.SeedKeys)
	writeMarkdownReportList(&out, "Errors", report.Discovery.Errors)

	out.WriteString("\n## Delete Plan\n\n")
	writeMarkdownReportField(&out, "Status", string(report.DeletePlan.Status))
	writeMarkdownReportField(&out, "Requires Confirmation", fmt.Sprintf("%t", report.DeletePlan.RequiresConfirmation))
	writeMarkdownReportList(&out, "Seed Keys", report.DeletePlan.SeedKeys)
	writeMarkdownReportList(&out, "Resolved Root Keys", report.DeletePlan.ResolvedRootKeys)
	writeMarkdownReportList(&out, "Affected Keys", report.DeletePlan.AffectedKeys)
	writeMarkdownReportList(&out, "Duplicate Keys", report.DeletePlan.DuplicateKeys)
	writeMarkdownReportList(&out, "Final State Items", retentionProcessInstanceItems(report.DeletePlan.FinalStateItems))
	writeMarkdownReportList(&out, "Non-Final Affected Items", retentionProcessInstanceItems(report.DeletePlan.NonFinalAffectedItems))
	writeMarkdownReportList(&out, "Skipped Seed Keys", report.DeletePlan.SkippedSeedKeys)
	writeMarkdownReportList(&out, "Skipped Non-Final Roots", retentionProcessInstanceItems(report.DeletePlan.SkippedNonFinalRoots))
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
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}

func retentionProcessInstanceItems(items []process.ProcessInstance) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("key=%s state=%s", item.Key, item.State))
	}
	return out
}

func retentionMissingAncestorItems(items []process.MissingAncestor) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("key=%s startKey=%s", item.Key, item.StartKey))
	}
	return out
}
