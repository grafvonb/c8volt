// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

// renderOpsRepairIncidentResult renders explicit incident repair through the shared machine contract or compact human output.
func renderOpsRepairIncidentResult(cmd *cobra.Command, result ops.RepairResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: repair incidents")
	} else {
		renderHumanLine(cmd, "repair incidents")
	}
	if result.Request.DiscoveryMode == ops.RepairDiscoveryModeSearch {
		renderHumanLine(cmd, "discovery: search filters %s", result.FrozenSet.IncidentFilters.String())
	}
	renderHumanLine(cmd, "frozen incidents: %d", len(result.FrozenSet.IncidentKeys))
	if len(result.FrozenSet.JobKeys) > 0 || len(result.Plan) > 0 {
		renderHumanLine(cmd, "related jobs: %d applicable, %d not applicable", len(result.FrozenSet.JobKeys), countOpsRepairIncidentJobNotApplicable(result.Plan))
	}
	if len(result.FrozenSet.VariableScopes) > 0 || len(result.VariableUpdates) > 0 {
		renderHumanLine(cmd, "variable scopes: %d", countOpsRepairVariableScopes(result))
	}
	renderOpsRepairReportFile(cmd, result.Request.ReportFile)
	if flagVerbose {
		renderOpsRepairKeys(cmd, "incident keys", result.FrozenSet.IncidentKeys)
		renderOpsRepairKeys(cmd, "process-instance keys", result.FrozenSet.ProcessInstanceKeys)
		renderOpsRepairKeys(cmd, "job keys", result.FrozenSet.JobKeys)
		renderOpsRepairVariableUpdates(cmd, result.VariableUpdates)
		for _, item := range result.Plan {
			renderHumanLine(cmd, "incident %s: vars=%s retry=%s timeout=%s resolution=%s confirmation=%s",
				item.IncidentKey,
				item.VariableUpdateStatus,
				item.RetryUpdateStatus,
				item.TimeoutUpdateStatus,
				item.ResolutionStatus,
				item.ConfirmationStatus,
			)
		}
	}
	if result.Outcome != "" {
		line := fmt.Sprintf("outcome: %s", result.Outcome)
		if result.Request.DryRun || result.Outcome == ops.RepairOutcomePlanned {
			line += "; no changes applied"
		}
		if !flagVerbose && opsRepairIncidentHasHiddenKeys(result) {
			line += "; use --verbose to list keys"
		}
		line += opsWorkflowElapsedSuffix(result.Report.Duration)
		renderHumanLine(cmd, "%s", line)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

// renderOpsRepairProcessInstanceResult renders process-instance selected repair through shared machine or human output.
func renderOpsRepairProcessInstanceResult(cmd *cobra.Command, result ops.RepairResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: repair process-instance incidents")
	} else {
		renderHumanLine(cmd, "repair process-instance incidents")
	}
	if result.Request.DiscoveryMode == ops.RepairDiscoveryModeSearch {
		renderHumanLine(cmd, "discovery: process filters %s", result.FrozenSet.ProcessFilters.String())
	}
	renderHumanLine(cmd, "frozen process instances: %d", len(result.FrozenSet.ProcessInstanceKeys))
	renderHumanLine(cmd, "frozen incidents: %d deduped", len(result.FrozenSet.IncidentKeys))
	if len(result.FrozenSet.JobKeys) > 0 || len(result.Plan) > 0 {
		renderHumanLine(cmd, "related jobs: %d applicable, %d not applicable", len(result.FrozenSet.JobKeys), countOpsRepairIncidentJobNotApplicable(result.Plan))
	}
	if len(result.FrozenSet.VariableScopes) > 0 || len(result.VariableUpdates) > 0 {
		renderHumanLine(cmd, "variable scopes: %d", countOpsRepairVariableScopes(result))
	}
	renderOpsRepairReportFile(cmd, result.Request.ReportFile)
	if flagVerbose {
		renderOpsRepairKeys(cmd, "process-instance keys", result.FrozenSet.ProcessInstanceKeys)
		renderOpsRepairKeys(cmd, "incident keys", result.FrozenSet.IncidentKeys)
		renderOpsRepairKeys(cmd, "job keys", result.FrozenSet.JobKeys)
		renderOpsRepairVariableUpdates(cmd, result.VariableUpdates)
		for _, item := range result.Plan {
			renderHumanLine(cmd, "process-instance %s incident %s: vars=%s retry=%s timeout=%s resolution=%s confirmation=%s",
				item.ProcessInstanceKey,
				item.IncidentKey,
				item.VariableUpdateStatus,
				item.RetryUpdateStatus,
				item.TimeoutUpdateStatus,
				item.ResolutionStatus,
				item.ConfirmationStatus,
			)
		}
	}
	if result.Outcome != "" {
		line := fmt.Sprintf("outcome: %s", result.Outcome)
		if result.Request.DryRun || result.Outcome == ops.RepairOutcomePlanned {
			line += "; no changes applied"
		}
		if !flagVerbose && opsRepairProcessInstanceHasHiddenKeys(result) {
			line += "; use --verbose to list keys"
		}
		line += opsWorkflowElapsedSuffix(result.Report.Duration)
		renderHumanLine(cmd, "%s", line)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func countOpsRepairIncidentJobNotApplicable(items []ops.RepairPlanItem) int {
	count := 0
	for _, item := range items {
		if item.RetryUpdateStatus == ops.WorkflowStepStatusNotApplicable || item.TimeoutUpdateStatus == ops.WorkflowStepStatusNotApplicable {
			count++
		}
	}
	return count
}

// countOpsRepairVariableScopes reports deduped variable scopes from either frozen discovery or executed scope updates.
func countOpsRepairVariableScopes(result ops.RepairResult) int {
	if len(result.VariableUpdates) > 0 {
		return len(result.VariableUpdates)
	}
	return len(result.FrozenSet.VariableScopes)
}

// opsRepairProcessInstanceHasHiddenKeys reports whether compact output omitted target details.
func opsRepairProcessInstanceHasHiddenKeys(result ops.RepairResult) bool {
	return len(result.FrozenSet.ProcessInstanceKeys) > 0 ||
		len(result.FrozenSet.IncidentKeys) > 0 ||
		len(result.FrozenSet.JobKeys) > 0 ||
		len(result.FrozenSet.VariableScopes) > 0
}

// opsRepairIncidentHasHiddenKeys reports whether compact incident output omitted target details.
func opsRepairIncidentHasHiddenKeys(result ops.RepairResult) bool {
	return len(result.FrozenSet.IncidentKeys) > 0 ||
		len(result.FrozenSet.ProcessInstanceKeys) > 0 ||
		len(result.FrozenSet.JobKeys) > 0 ||
		len(result.FrozenSet.VariableScopes) > 0
}

func renderOpsRepairKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(keys, ", "))
}

// renderOpsRepairVariableUpdates prints deduped variable scope status rows in verbose human output.
func renderOpsRepairVariableUpdates(cmd *cobra.Command, updates []ops.RepairVariableScopeUpdate) {
	for _, update := range updates {
		names := "none"
		if len(update.VariableNames) > 0 {
			names = strings.Join(update.VariableNames, ", ")
		}
		renderHumanLine(cmd, "variable scope %s: names=%s status=%s dependents=%s",
			update.ScopeKey,
			names,
			update.Status,
			strings.Join(update.DependentIncidentKeys, ", "),
		)
	}
}

// renderOpsRepairReportFile prints the compact audit report location after a successful write.
func renderOpsRepairReportFile(cmd *cobra.Command, path string) {
	if path == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", path)
}

// writeOpsRepairReport renders and writes the requested repair audit report.
func writeOpsRepairReport(result ops.RepairResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	report := enrichOpsRepairReport(result.Report, cfg)
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsRepairJSONReport(report)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsRepairMarkdownReport(report, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

// enrichOpsRepairReport adds local runtime metadata that is not known to the service layer.
func enrichOpsRepairReport(report ops.RepairAuditReport, cfg *config.Config) ops.RepairAuditReport {
	report.C8voltVersion = CurrentBuildInfo().Version
	if cfg != nil {
		report.CamundaVersion = cfg.App.CamundaVersion.String()
		report.TenantID = cfg.App.ViewTenant()
		if cfg.ActiveProfile != "" {
			report.ProfileIdentity = "profile:" + cfg.ActiveProfile
		} else {
			report.ProfileIdentity = "default"
		}
	}
	return report
}

// renderOpsRepairJSONReport encodes the complete repair audit model deterministically.
func renderOpsRepairJSONReport(report ops.RepairAuditReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderOpsRepairMarkdownReport renders the structured repair audit model as readable Markdown.
func renderOpsRepairMarkdownReport(report ops.RepairAuditReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Repair Audit Report\n\n")
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
	writeMarkdownReportField(&out, "Fail Fast", fmt.Sprintf("%t", report.FailFast))
	writeMarkdownReportField(&out, "No Worker Limit", fmt.Sprintf("%t", report.NoWorkerLimit))
	writeMarkdownReportField(&out, "Outcome", string(report.Outcome))

	out.WriteString("\n## Request\n\n")
	writeMarkdownReportField(&out, "Target", string(report.Request.Target))
	writeMarkdownReportField(&out, "Discovery Mode", string(report.Request.DiscoveryMode))
	writeMarkdownReportField(&out, "Output Mode", report.Request.OutputMode)
	writeMarkdownReportField(&out, "Report File", report.Request.ReportFile)
	writeMarkdownReportField(&out, "Report Format", report.Request.ReportFormat)
	writeMarkdownReportList(&out, "Input Keys", report.Request.InputKeys)
	writeMarkdownReportList(&out, "Requested Variables", opsRepairReportVariableNames(report))
	writeMarkdownReportField(&out, "Requested Retries", opsRepairReportRetries(report.Request.RequestedRetries))
	writeMarkdownReportField(&out, "Requested Job Timeout", report.Request.RequestedJobTimeout.String())

	out.WriteString("\n## Frozen Targets\n\n")
	writeMarkdownReportField(&out, "Status", string(report.FrozenSet.Status))
	writeMarkdownReportField(&out, "Target", string(report.FrozenSet.Target))
	writeMarkdownReportField(&out, "Discovery Mode", string(report.FrozenSet.DiscoveryMode))
	writeMarkdownReportField(&out, "Incident Filters", report.FrozenSet.IncidentFilters.String())
	writeMarkdownReportField(&out, "Process Filters", report.FrozenSet.ProcessFilters.String())
	writeMarkdownReportList(&out, "Input Keys", report.FrozenSet.InputKeys)
	writeMarkdownReportList(&out, "Incident Keys", report.FrozenSet.IncidentKeys)
	writeMarkdownReportList(&out, "Process-Instance Keys", report.FrozenSet.ProcessInstanceKeys)
	writeMarkdownReportList(&out, "Root Process-Instance Keys", report.FrozenSet.RootProcessKeys)
	writeMarkdownReportList(&out, "Job Keys", report.FrozenSet.JobKeys)
	writeMarkdownReportList(&out, "Variable Scopes", report.FrozenSet.VariableScopes)
	writeMarkdownReportList(&out, "Discovery Errors", report.FrozenSet.Errors)

	out.WriteString("\n## Variable Updates\n\n")
	writeMarkdownReportList(&out, "Scopes", opsRepairVariableUpdateReportItems(report.VariableUpdates))

	out.WriteString("\n## Job Applicability\n\n")
	writeMarkdownReportList(&out, "Jobs", opsRepairJobApplicabilityReportItems(report.JobApplicability))

	out.WriteString("\n## Incident Steps\n\n")
	writeMarkdownReportList(&out, "Incidents", opsRepairPlanReportItems(report.Plan))

	out.WriteString("\n## Remaining Incidents\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Remaining.Status))
	writeMarkdownReportField(&out, "Checked", fmt.Sprintf("%t", report.Remaining.Checked))
	writeMarkdownReportList(&out, "Active Keys", report.Remaining.ActiveKeys)
	writeMarkdownReportList(&out, "Errors", report.Remaining.Errors)

	writeMarkdownReportList(&out, "Notices", opsRepairNoticeReportItems(report.Notices))
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}

// opsRepairReportVariableNames returns sorted requested variable names for report summaries.
func opsRepairReportVariableNames(report ops.RepairAuditReport) []string {
	if len(report.Request.Variables) == 0 {
		return nil
	}
	names := make([]string, 0, len(report.Request.Variables))
	for name := range report.Request.Variables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// opsRepairReportRetries formats optional retry requests without inventing a default.
func opsRepairReportRetries(retries *int32) string {
	if retries == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *retries)
}

// opsRepairVariableUpdateReportItems flattens variable scope updates for Markdown lists.
func opsRepairVariableUpdateReportItems(items []ops.RepairVariableScopeUpdate) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("scope=%s status=%s names=%s dependents=%s",
			item.ScopeKey,
			item.Status,
			strings.Join(item.VariableNames, ","),
			strings.Join(item.DependentIncidentKeys, ","),
		))
	}
	return out
}

// opsRepairJobApplicabilityReportItems flattens per-incident job applicability for Markdown.
func opsRepairJobApplicabilityReportItems(items []ops.RepairJobApplicability) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := fmt.Sprintf("incident=%s job=%s retry=%s timeout=%s",
			item.IncidentKey,
			item.JobKey,
			item.RetryStatus,
			item.TimeoutStatus,
		)
		if item.Reason != "" {
			text += " reason=" + item.Reason
		}
		out = append(out, text)
	}
	return out
}

// opsRepairPlanReportItems summarizes incident-local repair step statuses.
func opsRepairPlanReportItems(items []ops.RepairPlanItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("incident=%s processInstance=%s job=%s vars=%s retry=%s timeout=%s resolution=%s confirmation=%s",
			item.IncidentKey,
			item.ProcessInstanceKey,
			item.JobKey,
			item.VariableUpdateStatus,
			item.RetryUpdateStatus,
			item.TimeoutUpdateStatus,
			item.ResolutionStatus,
			item.ConfirmationStatus,
		))
	}
	return out
}

// opsRepairNoticeReportItems preserves structured notice code, severity, and message text.
func opsRepairNoticeReportItems(items []ops.RepairNotice) []string {
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
