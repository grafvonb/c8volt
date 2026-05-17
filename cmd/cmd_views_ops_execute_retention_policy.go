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
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: execute retention policy")
	} else {
		renderHumanLine(cmd, "execute retention policy")
	}
	renderOpsExecuteRetentionPolicyDiscovery(cmd, result)
	renderOpsExecuteRetentionPolicyDeletePlan(cmd, result)
	renderOpsExecuteRetentionPolicyDeletion(cmd, result)
	renderOpsExecuteRetentionPolicyReportFile(cmd, result)
	renderOpsExecuteRetentionPolicyOutcome(cmd, result)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func renderOpsExecuteRetentionPolicyDiscovery(cmd *cobra.Command, result ops.RetentionPolicyResult) {
	if filters := result.Discovery.Filters.String(); filters != "" {
		renderHumanLine(cmd, "selection filters: %s", filters)
	}
	if result.Discovery.Status != "" {
		renderHumanLine(cmd, "candidate retention process instances: %d", result.Discovery.Count)
		if result.Discovery.Count == 0 {
			renderHumanLine(cmd, "no retention cleanup targets found")
		}
	}
	if flagVerbose {
		renderHumanLine(cmd, "retention days: %d", result.Request.RetentionDays)
		if result.Request.DerivedEndDateBoundary != "" {
			renderHumanLine(cmd, "retention boundary: endDate <= %s", result.Request.DerivedEndDateBoundary)
		}
		if result.Discovery.Status != "" {
			renderHumanLine(cmd, "retention discovery: %s", result.Discovery.Status)
		}
	}
}

func renderOpsExecuteRetentionPolicyDeletePlan(cmd *cobra.Command, result ops.RetentionPolicyResult) {
	if result.DeletePlan.Status != "" {
		if result.Request.DryRun {
			renderOpsExecuteRetentionPolicyDryRunDeletePreview(cmd, result)
			return
		}
		renderHumanLine(cmd, "delete plan: %s (candidate retention process instances: %d, roots: %d, affected process instances: %d)",
			result.DeletePlan.Status,
			len(result.DeletePlan.SeedKeys),
			len(result.DeletePlan.ResolvedRootKeys),
			len(result.DeletePlan.AffectedKeys),
		)
		if flagVerbose && len(result.DeletePlan.DuplicateKeys) > 0 {
			renderHumanLine(cmd, "duplicate roots: %d", len(result.DeletePlan.DuplicateKeys))
		}
		if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
			renderHumanLine(cmd, "non-final descendants in final-root scope: %d (use --force to cancel before delete)", len(result.DeletePlan.NonFinalAffectedItems))
		}
		if len(result.DeletePlan.SkippedSeedKeys) > 0 {
			renderHumanLine(cmd, "skipped candidate retention process instances with non-final roots: %d", len(result.DeletePlan.SkippedSeedKeys))
		}
		if flagVerbose && len(result.DeletePlan.MissingAncestors) > 0 {
			renderHumanLine(cmd, "missing ancestors: %d", len(result.DeletePlan.MissingAncestors))
		}
		if flagVerbose {
			for _, warning := range result.DeletePlan.TraversalWarnings {
				if warning != "" {
					renderHumanLine(cmd, "traversal warning: %s", warning)
				}
			}
			renderHumanLine(cmd, "confirmation required: %t", result.DeletePlan.RequiresConfirmation)
			printOpsExecuteRetentionPolicyKeys(cmd, "candidate keys", result.DeletePlan.SeedKeys)
			printOpsExecuteRetentionPolicyKeys(cmd, "skipped candidate keys", result.DeletePlan.SkippedSeedKeys)
			printOpsExecuteRetentionPolicyItems(cmd, "skipped non-final roots", result.DeletePlan.SkippedNonFinalRoots)
			printOpsExecuteRetentionPolicyKeys(cmd, "resolved root keys", result.DeletePlan.ResolvedRootKeys)
			printOpsExecuteRetentionPolicyKeys(cmd, "affected process-instance keys", result.DeletePlan.AffectedKeys)
		}
	}
}

func renderOpsExecuteRetentionPolicyDryRunDeletePreview(cmd *cobra.Command, result ops.RetentionPolicyResult) {
	if result.DeletePlan.Status == ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "delete preview: skipped (no retention cleanup targets)")
		return
	}
	renderHumanLine(cmd, "delete preview: %d retention candidate(s), %d process-instance tree(s), %d process instance(s) would be deleted",
		len(result.DeletePlan.SeedKeys),
		len(result.DeletePlan.ResolvedRootKeys),
		len(result.DeletePlan.AffectedKeys),
	)
	if flagVerbose && len(result.DeletePlan.DuplicateKeys) > 0 {
		renderHumanLine(cmd, "duplicate process-instance trees: %d", len(result.DeletePlan.DuplicateKeys))
	}
	if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
		renderHumanLine(cmd, "non-final process instances in scope: %d (use --force to cancel before delete)", len(result.DeletePlan.NonFinalAffectedItems))
	}
	if len(result.DeletePlan.SkippedSeedKeys) > 0 {
		renderHumanLine(cmd, "skipped retention candidates with non-final roots: %d", len(result.DeletePlan.SkippedSeedKeys))
	}
	if flagVerbose && len(result.DeletePlan.MissingAncestors) > 0 {
		renderHumanLine(cmd, "missing ancestors: %d", len(result.DeletePlan.MissingAncestors))
	}
	if flagVerbose {
		for _, warning := range result.DeletePlan.TraversalWarnings {
			if warning != "" {
				renderHumanLine(cmd, "traversal warning: %s", warning)
			}
		}
		renderHumanLine(cmd, "confirmation required: %t", result.DeletePlan.RequiresConfirmation)
		printOpsExecuteRetentionPolicyKeys(cmd, "candidate keys", result.DeletePlan.SeedKeys)
		printOpsExecuteRetentionPolicyKeys(cmd, "skipped candidate keys", result.DeletePlan.SkippedSeedKeys)
		printOpsExecuteRetentionPolicyItems(cmd, "skipped non-final roots", result.DeletePlan.SkippedNonFinalRoots)
		printOpsExecuteRetentionPolicyKeys(cmd, "resolved root keys", result.DeletePlan.ResolvedRootKeys)
		printOpsExecuteRetentionPolicyKeys(cmd, "affected process-instance keys", result.DeletePlan.AffectedKeys)
	}
}

func renderOpsExecuteRetentionPolicyDeletion(cmd *cobra.Command, result ops.RetentionPolicyResult) {
	if result.Deletion.Status != "" {
		if !result.Deletion.Submitted && !flagVerbose {
			return
		}
		if !result.Deletion.Submitted {
			renderHumanLine(cmd, "deletion: %s; no deletion request submitted", result.Deletion.Status)
			return
		}
		renderHumanLine(cmd, "deletion: %s", opsWorkflowDeletionSummary(string(result.Deletion.Status), len(result.Deletion.Items), "process-instance tree", "process-instance trees", result.Deletion.NoWait))
	}
}

func renderOpsExecuteRetentionPolicyOutcome(cmd *cobra.Command, result ops.RetentionPolicyResult) {
	if result.Outcome != "" {
		elapsed := opsWorkflowElapsedSuffix(result.Report.Duration)
		if !result.Deletion.Submitted && result.Outcome == ops.RetentionPolicyOutcomePlanned {
			line := fmt.Sprintf("outcome: %s; no changes applied", result.Outcome)
			if !flagVerbose && retentionPolicyHasHiddenKeys(result) {
				line += "; use --verbose to list process-instance keys"
			}
			line += elapsed
			renderHumanLine(cmd, "%s", line)
		} else {
			renderHumanLine(cmd, "outcome: %s%s", result.Outcome, elapsed)
		}
	}
}

func retentionPolicyHasHiddenKeys(result ops.RetentionPolicyResult) bool {
	return len(result.DeletePlan.SeedKeys) > 0 ||
		len(result.DeletePlan.SkippedSeedKeys) > 0 ||
		len(result.DeletePlan.ResolvedRootKeys) > 0 ||
		len(result.DeletePlan.AffectedKeys) > 0
}

func printOpsExecuteRetentionPolicyKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(keys, ", "))
}

func printOpsExecuteRetentionPolicyItems(cmd *cobra.Command, label string, items []process.ProcessInstance) {
	if len(items) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(retentionProcessInstanceItems(items), ", "))
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
	writeMarkdownReportList(&out, "Candidate Keys", report.Discovery.SeedKeys)
	writeMarkdownReportList(&out, "Errors", report.Discovery.Errors)

	out.WriteString("\n## Delete Plan\n\n")
	writeMarkdownReportField(&out, "Status", string(report.DeletePlan.Status))
	writeMarkdownReportField(&out, "Requires Confirmation", fmt.Sprintf("%t", report.DeletePlan.RequiresConfirmation))
	writeMarkdownReportList(&out, "Candidate Keys", report.DeletePlan.SeedKeys)
	writeMarkdownReportList(&out, "Resolved Root Keys", report.DeletePlan.ResolvedRootKeys)
	writeMarkdownReportList(&out, "Affected Keys", report.DeletePlan.AffectedKeys)
	writeMarkdownReportList(&out, "Duplicate Keys", report.DeletePlan.DuplicateKeys)
	writeMarkdownReportList(&out, "Final State Items", retentionProcessInstanceItems(report.DeletePlan.FinalStateItems))
	writeMarkdownReportList(&out, "Non-Final Affected Items", retentionProcessInstanceItems(report.DeletePlan.NonFinalAffectedItems))
	writeMarkdownReportList(&out, "Skipped Candidate Keys", report.DeletePlan.SkippedSeedKeys)
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
