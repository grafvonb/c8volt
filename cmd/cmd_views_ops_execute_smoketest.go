// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

func renderOpsExecuteSmokeTestResult(cmd *cobra.Command, result ops.SmokeTestResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: execute smoke test")
	} else {
		renderHumanLine(cmd, "execute smoke test")
	}
	renderAttachedTenantContext(cmd)
	renderOpsExecuteSmokeTestPlan(cmd, result)
	renderOpsExecuteSmokeTestDeployment(cmd, result)
	renderOpsExecuteSmokeTestRun(cmd, result)
	renderOpsExecuteSmokeTestWalk(cmd, result)
	renderOpsExecuteSmokeTestCleanup(cmd, result)
	renderOpsExecuteSmokeTestReportFile(cmd, result)
	renderOpsExecuteSmokeTestOutcome(cmd, result)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func renderOpsExecuteSmokeTestPlan(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Plan.Fixture.File != "" {
		renderHumanLine(cmd, "fixture: %s", result.Plan.Fixture.File)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "smoke-test preview: %s", smokeTestDryRunWorkflow(result.Request.Count))
		renderHumanLine(cmd, "cleanup: %s", smokeTestCleanupIntent(result.Plan.CleanupRequested))
	}
	if !flagVerbose {
		return
	}
	if result.Plan.CamundaVersion != "" {
		renderHumanLine(cmd, "camunda version: %s", result.Plan.CamundaVersion)
	}
	if result.Plan.Fixture.File != "" {
		renderHumanLine(cmd, "fixture availability: %s", availabilityLabel(result.Plan.Fixture.Available))
	}
	for _, step := range result.Plan.PlannedSteps {
		if step.Name == "" || step.Status == "" {
			continue
		}
		if step.Message == "" {
			renderHumanLine(cmd, "%s: %s", step.Name, step.Status)
		} else {
			renderHumanLine(cmd, "%s: %s - %s", step.Name, step.Status, step.Message)
		}
	}
}

// renderOpsExecuteSmokeTestDeployment keeps default output compact and exposes the full returned scope in verbose mode.
func renderOpsExecuteSmokeTestDeployment(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Deployment.Status == "" || result.Deployment.Status == ops.WorkflowStepStatusPlanned || result.Deployment.Status == ops.WorkflowStepStatusSkipped {
		return
	}
	renderHumanLine(cmd, "deployment: %s", result.Deployment.Status)
	if !flagVerbose {
		return
	}
	if len(result.Deployment.ProcessDefinitions) > 0 {
		for _, definition := range result.Deployment.ProcessDefinitions {
			renderHumanLine(cmd, "deployed process definition: %s", smokeTestDefinitionIdentity(definition))
		}
		return
	}
	if result.Deployment.ProcessDefinitionKey != "" {
		version := ""
		if result.Deployment.ProcessDefinitionVersion > 0 {
			version = fmt.Sprintf(", version %d", result.Deployment.ProcessDefinitionVersion)
		}
		renderHumanLine(cmd, "process definition: %s (%s%s)", result.Deployment.ProcessDefinitionKey, result.Deployment.BpmnProcessID, version)
	} else if result.Deployment.BpmnProcessID != "" {
		renderHumanLine(cmd, "process definition: %s", result.Deployment.BpmnProcessID)
	}
	if result.Deployment.TenantID != "" {
		renderHumanLine(cmd, "tenant: %s", result.Deployment.TenantID)
	}
}

func renderOpsExecuteSmokeTestRun(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Run.Status == "" || result.Run.Status == ops.WorkflowStepStatusPlanned || result.Run.Status == ops.WorkflowStepStatusSkipped {
		return
	}
	renderHumanLine(cmd, "created process instances: %d/%d", result.Run.CreatedCount, result.Run.RequestedCount)
	if flagVerbose && len(result.Run.ProcessInstanceKeys) > 0 {
		renderHumanLine(cmd, "created keys: %s", strings.Join(result.Run.ProcessInstanceKeys, ", "))
	}
}

func renderOpsExecuteSmokeTestWalk(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Walk.Status == "" || result.Walk.Status == ops.WorkflowStepStatusPlanned || result.Walk.Status == ops.WorkflowStepStatusSkipped {
		return
	}
	renderHumanLine(cmd, "walk: %s (process instances: %d)", result.Walk.Status, len(result.Walk.Items))
	if !flagVerbose {
		return
	}
	for _, item := range result.Walk.Items {
		if item.ProcessInstanceKey == "" {
			continue
		}
		familyCount := len(item.Summary.FamilyKeys)
		if item.Summary.RootProcessInstanceKey != "" {
			renderHumanLine(cmd, "walk %s: %s, root %s, family %d", item.ProcessInstanceKey, item.Status, item.Summary.RootProcessInstanceKey, familyCount)
			continue
		}
		renderHumanLine(cmd, "walk %s: %s, family %d", item.ProcessInstanceKey, item.Status, familyCount)
	}
}

// renderOpsExecuteSmokeTestCleanup renders aggregate collection evidence without performing discovery.
func renderOpsExecuteSmokeTestCleanup(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Request.DryRun {
		return
	}
	cleanup := result.Cleanup
	if cleanup.NoCleanup {
		if cleanup.DefinitionSummary != nil {
			renderHumanLine(cmd, "cleanup: %s (--no-cleanup); process definitions: %s", cleanup.DefinitionSummary.Status, smokeTestDefinitionDispositionSummary(*cleanup.DefinitionSummary))
		} else {
			renderHumanLine(cmd, "cleanup: skipped (--no-cleanup)")
		}
		if flagVerbose && len(cleanup.RetainedProcessInstanceKeys) > 0 {
			renderHumanLine(cmd, "retained process instances: %s", strings.Join(cleanup.RetainedProcessInstanceKeys, ", "))
		}
		if flagVerbose && len(cleanup.DefinitionResults) > 0 {
			renderOpsExecuteSmokeTestDefinitionResults(cmd, cleanup.DefinitionResults)
		} else if flagVerbose && cleanup.RetainedProcessDefinitionKey != "" {
			if cleanup.RetainedBpmnProcessID != "" {
				renderHumanLine(cmd, "retained process definition: %s (%s)", cleanup.RetainedProcessDefinitionKey, cleanup.RetainedBpmnProcessID)
			} else {
				renderHumanLine(cmd, "retained process definition: %s", cleanup.RetainedProcessDefinitionKey)
			}
		} else if flagVerbose && cleanup.RetainedBpmnProcessID != "" {
			renderHumanLine(cmd, "retained process definition: %s", cleanup.RetainedBpmnProcessID)
		}
		return
	}
	status := smokeTestCleanupStatus(cleanup)
	if status != "" {
		renderHumanLine(cmd, "cleanup: %s", smokeTestCleanupSummary(cleanup, status))
	}
	if !flagVerbose {
		if len(cleanup.ProcessDefinitionEligibility.Blockers) > 0 {
			renderHumanLine(cmd, "process-definition cleanup blockers: %d", len(cleanup.ProcessDefinitionEligibility.Blockers))
		}
		return
	}
	if cleanup.ProcessInstanceCleanup.Status != "" && cleanup.ProcessInstanceCleanup.Status != ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "process-instance cleanup: %s", cleanup.ProcessInstanceCleanup.Status)
		if len(cleanup.ProcessInstanceCleanup.SubmittedKeys) > 0 {
			renderHumanLine(cmd, "cleanup roots: %s", strings.Join(cleanup.ProcessInstanceCleanup.SubmittedKeys, ", "))
		}
	}
	if cleanup.ProcessDefinitionEligibility.Status != "" && cleanup.ProcessDefinitionEligibility.Status != ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "process-definition cleanup eligibility: %s", cleanup.ProcessDefinitionEligibility.Status)
		if len(cleanup.ProcessDefinitionEligibility.Blockers) > 0 {
			renderHumanLine(cmd, "process-definition cleanup blockers: %s", strings.Join(cleanup.ProcessDefinitionEligibility.Blockers, ", "))
		}
	}
	if cleanup.ProcessDefinitionCleanup.Status != "" && cleanup.ProcessDefinitionCleanup.Status != ops.WorkflowStepStatusSkipped {
		renderHumanLine(cmd, "process-definition cleanup: %s", cleanup.ProcessDefinitionCleanup.Status)
		if cleanup.ProcessDefinitionCleanup.SubmittedProcessDefinitionKey != "" {
			renderHumanLine(cmd, "cleanup process definition: %s", cleanup.ProcessDefinitionCleanup.SubmittedProcessDefinitionKey)
		}
	}
	if len(cleanup.DefinitionResults) > 0 {
		renderOpsExecuteSmokeTestDefinitionResults(cmd, cleanup.DefinitionResults)
	}
}

func renderOpsExecuteSmokeTestOutcome(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Outcome == "" {
		return
	}
	elapsed := opsWorkflowElapsedSuffix(result.Report.Duration)
	if result.Outcome == ops.SmokeTestOutcomePlanned {
		line := fmt.Sprintf("outcome: %s; no changes applied", result.Outcome)
		if !flagVerbose && smokeTestHasHiddenKeys(result) {
			line += "; use --verbose to list process-instance keys"
		}
		line += elapsed
		renderHumanLine(cmd, "%s", line)
		return
	}
	if result.Outcome == ops.SmokeTestOutcomePassedCleanupSkipped && !flagVerbose && smokeTestHasHiddenRetainedResources(result) {
		renderHumanLine(cmd, "outcome: %s; use --verbose to list retained resources%s", result.Outcome, elapsed)
		return
	}
	renderHumanLine(cmd, "outcome: %s%s", result.Outcome, elapsed)
}

// smokeTestCleanupStatus prefers the authoritative definition aggregate while retaining legacy fallback behavior.
func smokeTestCleanupStatus(cleanup ops.SmokeTestCleanupResult) ops.WorkflowStepStatus {
	if cleanup.DefinitionSummary != nil {
		if cleanup.ProcessInstanceCleanup.Status == ops.WorkflowStepStatusFailed || cleanup.ProcessInstanceCleanup.Status == ops.WorkflowStepStatusBlocked {
			return cleanup.ProcessInstanceCleanup.Status
		}
		if cleanup.DefinitionSummary.Status == ops.WorkflowStepStatusFailed || cleanup.DefinitionSummary.Status == ops.WorkflowStepStatusBlocked || cleanup.DefinitionSummary.Status == ops.WorkflowStepStatusSkipped {
			return cleanup.DefinitionSummary.Status
		}
		if cleanup.ProcessInstanceCleanup.Status == ops.WorkflowStepStatusSubmitted || cleanup.DefinitionSummary.Status == ops.WorkflowStepStatusSubmitted {
			return ops.WorkflowStepStatusSubmitted
		}
		return cleanup.DefinitionSummary.Status
	}
	for _, status := range []ops.WorkflowStepStatus{
		cleanup.ProcessInstanceCleanup.Status,
		cleanup.ProcessDefinitionEligibility.Status,
		cleanup.ProcessDefinitionCleanup.Status,
	} {
		if status == ops.WorkflowStepStatusFailed || status == ops.WorkflowStepStatusBlocked {
			return status
		}
	}
	for _, status := range []ops.WorkflowStepStatus{
		cleanup.ProcessInstanceCleanup.Status,
		cleanup.ProcessDefinitionCleanup.Status,
	} {
		if status == ops.WorkflowStepStatusSubmitted {
			return status
		}
	}
	if cleanup.ProcessInstanceCleanup.Status == ops.WorkflowStepStatusConfirmed ||
		cleanup.ProcessDefinitionCleanup.Status == ops.WorkflowStepStatusConfirmed {
		return ops.WorkflowStepStatusConfirmed
	}
	return cleanup.ProcessInstanceCleanup.Status
}

func smokeTestProcessDefinitionCleanupStatus(cleanup ops.SmokeTestCleanupResult) ops.WorkflowStepStatus {
	if cleanup.ProcessDefinitionEligibility.Status == ops.WorkflowStepStatusBlocked ||
		cleanup.ProcessDefinitionEligibility.Status == ops.WorkflowStepStatusFailed {
		return cleanup.ProcessDefinitionEligibility.Status
	}
	return cleanup.ProcessDefinitionCleanup.Status
}

// smokeTestCleanupSummary describes collection-wide cleanup without claiming unconfirmed removal.
func smokeTestCleanupSummary(cleanup ops.SmokeTestCleanupResult, status ops.WorkflowStepStatus) string {
	resources := smokeTestCleanupResourceSummary(cleanup)
	switch status {
	case ops.WorkflowStepStatusConfirmed:
		return "removed " + resources
	case ops.WorkflowStepStatusSubmitted:
		if cleanup.ProcessInstanceCleanup.NoWait || cleanup.ProcessDefinitionCleanup.NoWait {
			return "submitted " + resources + " (--no-wait)"
		}
		return "submitted " + resources
	case ops.WorkflowStepStatusBlocked, ops.WorkflowStepStatusFailed:
		if cleanup.DefinitionSummary != nil {
			return fmt.Sprintf("%s (%s; process definitions: %s)", status, smokeTestProcessInstanceCountLabel(len(cleanup.ProcessInstanceCleanup.SubmittedKeys)), smokeTestDefinitionDispositionSummary(*cleanup.DefinitionSummary))
		}
		pdStatus := smokeTestProcessDefinitionCleanupStatus(cleanup)
		if pdStatus != "" && pdStatus != ops.WorkflowStepStatusSkipped {
			return fmt.Sprintf("%s (%s, process definition: %s)", status, smokeTestProcessInstanceCountLabel(len(cleanup.ProcessInstanceCleanup.SubmittedKeys)), pdStatus)
		}
	}
	if cleanup.DefinitionSummary != nil {
		return fmt.Sprintf("%s (%s; process definitions: %s)", status, smokeTestProcessInstanceCountLabel(len(cleanup.ProcessInstanceCleanup.SubmittedKeys)), smokeTestDefinitionDispositionSummary(*cleanup.DefinitionSummary))
	}
	return fmt.Sprintf("%s (%s)", status, resources)
}

// smokeTestCleanupResourceSummary uses collection totals when present and singular legacy evidence otherwise.
func smokeTestCleanupResourceSummary(cleanup ops.SmokeTestCleanupResult) string {
	pi := smokeTestProcessInstanceCountLabel(len(cleanup.ProcessInstanceCleanup.SubmittedKeys))
	if cleanup.DefinitionSummary != nil {
		return pi + " and " + smokeTestProcessDefinitionCountLabel(cleanup.DefinitionSummary.Total)
	}
	pdStatus := smokeTestProcessDefinitionCleanupStatus(cleanup)
	switch pdStatus {
	case ops.WorkflowStepStatusConfirmed, ops.WorkflowStepStatusSubmitted:
		return pi + " and fixture process definition"
	case "", ops.WorkflowStepStatusSkipped:
		return pi
	default:
		return fmt.Sprintf("%s, fixture process definition: %s", pi, pdStatus)
	}
}

// renderOpsExecuteSmokeTestDefinitionResults exposes deployment-wide cleanup evidence only in verbose human output.
func renderOpsExecuteSmokeTestDefinitionResults(cmd *cobra.Command, results []ops.SmokeTestDefinitionCleanupItem) {
	for _, item := range results {
		line := fmt.Sprintf("process definition %s: %s", smokeTestDefinitionIdentity(item.Definition), item.Disposition)
		if item.Reason != "" {
			line += " - " + item.Reason
		}
		renderHumanLine(cmd, "%s", line)
	}
}

// smokeTestDefinitionIdentity formats the available exact-key identity without inventing missing evidence.
func smokeTestDefinitionIdentity(definition ops.SmokeTestDeployedDefinition) string {
	identity := definition.ProcessDefinitionKey
	if identity == "" {
		identity = "unknown key"
	}
	var details []string
	if definition.BpmnProcessID != "" {
		details = append(details, definition.BpmnProcessID)
	}
	if definition.ProcessDefinitionVersion > 0 {
		details = append(details, fmt.Sprintf("version %d", definition.ProcessDefinitionVersion))
	}
	if definition.TenantID != "" {
		details = append(details, "tenant "+definition.TenantID)
	}
	if len(details) == 0 {
		return identity
	}
	return fmt.Sprintf("%s (%s)", identity, strings.Join(details, ", "))
}

// smokeTestDefinitionDispositionSummary reports only nonzero mutually exclusive collection counts.
func smokeTestDefinitionDispositionSummary(summary ops.SmokeTestDefinitionCleanupSummary) string {
	counts := []struct {
		value int
		label string
	}{
		{summary.Confirmed, "confirmed"},
		{summary.Submitted, "submitted"},
		{summary.Retained, "retained"},
		{summary.Failed, "failed"},
		{summary.Unattempted, "unattempted"},
	}
	parts := make([]string, 0, len(counts))
	for _, count := range counts {
		if count.value > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count.value, count.label))
		}
	}
	if len(parts) == 0 {
		return "0 accounted"
	}
	return strings.Join(parts, ", ")
}

// smokeTestProcessDefinitionCountLabel renders stable singular and plural resource wording.
func smokeTestProcessDefinitionCountLabel(count int) string {
	if count == 1 {
		return "1 process definition"
	}
	return fmt.Sprintf("%d process definitions", count)
}

func smokeTestProcessInstanceCountLabel(count int) string {
	if count == 1 {
		return "1 process instance"
	}
	return fmt.Sprintf("%d process instances", count)
}

func smokeTestHasHiddenKeys(result ops.SmokeTestResult) bool {
	return len(result.Run.ProcessInstanceKeys) > 0 ||
		len(result.Cleanup.ProcessInstanceCleanup.SubmittedKeys) > 0 ||
		len(result.Cleanup.ProcessDefinitionEligibility.Blockers) > 0
}

// smokeTestHasHiddenRetainedResources detects evidence intentionally omitted from compact output.
func smokeTestHasHiddenRetainedResources(result ops.SmokeTestResult) bool {
	return len(result.Cleanup.RetainedProcessInstanceKeys) > 0 ||
		len(result.Cleanup.DefinitionResults) > 0 ||
		result.Cleanup.RetainedProcessDefinitionKey != "" ||
		result.Cleanup.RetainedBpmnProcessID != ""
}

// smokeTestCleanupIntent describes the complete fixture deployment scope in dry-run output.
func smokeTestCleanupIntent(cleanupRequested bool) string {
	if cleanupRequested {
		return "included (will remove created process instances and deployed fixture process definitions)"
	}
	return "skipped (--no-cleanup)"
}

func smokeTestDryRunWorkflow(count int) string {
	if count == 1 {
		return "would deploy the fixture, start 1 process instance, and walk its process-instance family"
	}
	return fmt.Sprintf("would deploy the fixture, start %d process instances, and walk their process-instance families", count)
}

func renderOpsExecuteSmokeTestReportFile(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Request.ReportFile == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", result.Request.ReportFile)
}

func availabilityLabel(available bool) string {
	if available {
		return "available"
	}
	return "missing"
}

func writeOpsExecuteSmokeTestReport(result ops.SmokeTestResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	report := enrichOpsExecuteSmokeTestReport(result.Report, cfg)
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsExecuteSmokeTestJSONReport(report)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsExecuteSmokeTestMarkdownReport(report, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

func enrichOpsExecuteSmokeTestReport(report ops.SmokeTestAuditReport, cfg *config.Config) ops.SmokeTestAuditReport {
	report.C8voltVersion = CurrentBuildInfo().Version
	ctx := opsExecuteSmokeTestReportTenantContext(report, cfg)
	report.TenantContext = cloneTenantContextPtr(ctx)
	report.TenantID = opsLegacyTenantIDForContext(report.TenantContext)
	if cfg != nil {
		report.CamundaVersion = cfg.App.CamundaVersion.String()
		if cfg.ActiveProfile != "" {
			report.ProfileIdentity = "profile:" + cfg.ActiveProfile
		} else {
			report.ProfileIdentity = "default"
		}
	}
	return report
}

// attachOpsExecuteSmokeTestResultTenantContext freezes creation tenant context
// with returned smoke-test resource evidence before rendering.
func attachOpsExecuteSmokeTestResultTenantContext(cmd *cobra.Command, cfg *config.Config, result ops.SmokeTestResult) ops.SmokeTestResult {
	evidence := opsMergedTenantEvidence(result.Deployment.TenantEvidence, result.Run.TenantEvidence)
	ctx := attachOpsCreationTenantContext(cmd, cfg, evidence)
	result.Report.TenantContext = cloneTenantContextPtr(ctx)
	result.Report.TenantID = opsLegacyTenantIDForContext(result.Report.TenantContext)
	return result
}

// opsExecuteSmokeTestReportTenantContext derives report-level creation context
// from smoke-test deployment and run evidence.
func opsExecuteSmokeTestReportTenantContext(report ops.SmokeTestAuditReport, cfg *config.Config) tenant.Context {
	evidence := opsMergedTenantEvidence(report.Deployment.TenantEvidence, report.Run.TenantEvidence)
	return opsTenantContextWithEvidence(newCreationTenantContext(creationTenantID(cfg)), evidence)
}

func renderOpsExecuteSmokeTestJSONReport(report ops.SmokeTestAuditReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderOpsExecuteSmokeTestMarkdownReport writes report-owned evidence without backend lookups.
func renderOpsExecuteSmokeTestMarkdownReport(report ops.SmokeTestAuditReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Execute Smoke Test Audit Report\n\n")
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
	writeMarkdownReportField(&out, "No Cleanup", fmt.Sprintf("%t", report.NoCleanup))
	writeMarkdownReportField(&out, "Outcome", string(report.Outcome))

	out.WriteString("\n## Plan\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Plan.Status))
	writeMarkdownReportField(&out, "Configured Camunda Version", report.Plan.CamundaVersion)
	writeMarkdownReportField(&out, "Cleanup Requested", fmt.Sprintf("%t", report.Plan.CleanupRequested))
	if len(report.Plan.PlannedSteps) > 0 {
		out.WriteString("- Steps:\n")
		for _, step := range report.Plan.PlannedSteps {
			out.WriteString(fmt.Sprintf("  - %s: %s", step.Name, step.Status))
			if step.Message != "" {
				out.WriteString(fmt.Sprintf(" - %s", step.Message))
			}
			out.WriteString("\n")
		}
	}
	writeMarkdownReportList(&out, "Errors", report.Plan.Errors)

	out.WriteString("\n## Fixture\n\n")
	writeMarkdownReportField(&out, "File", report.Fixture.File)
	writeMarkdownReportField(&out, "BPMN Process ID", report.Fixture.BpmnProcessID)
	writeMarkdownReportField(&out, "Available", fmt.Sprintf("%t", report.Fixture.Available))

	out.WriteString("\n## Deployment\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Deployment.Status))
	writeMarkdownReportField(&out, "Fixture File", report.Deployment.FixtureFile)
	writeMarkdownReportField(&out, "BPMN Process ID", report.Deployment.BpmnProcessID)
	writeMarkdownReportField(&out, "Process Definition Key", report.Deployment.ProcessDefinitionKey)
	if report.Deployment.ProcessDefinitionVersion > 0 {
		writeMarkdownReportField(&out, "Process Definition Version", fmt.Sprintf("%d", report.Deployment.ProcessDefinitionVersion))
	}
	writeMarkdownReportField(&out, "Tenant", report.Deployment.TenantID)
	writeMarkdownReportField(&out, "Definition Scope Complete", fmt.Sprintf("%t", report.Deployment.DefinitionScopeComplete))
	writeSmokeTestDeploymentDefinitionsTable(&out, report.Deployment.ProcessDefinitions)

	out.WriteString("\n## Run\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Run.Status))
	writeMarkdownReportField(&out, "Requested Count", fmt.Sprintf("%d", report.Run.RequestedCount))
	writeMarkdownReportField(&out, "Created Count", fmt.Sprintf("%d", report.Run.CreatedCount))
	writeMarkdownReportList(&out, "Created Process Instance Keys", report.Run.ProcessInstanceKeys)

	out.WriteString("\n## Walk\n\n")
	writeMarkdownReportField(&out, "Status", string(report.Walk.Status))
	if len(report.Walk.Items) > 0 {
		out.WriteString("- Traversals:\n")
		for _, item := range report.Walk.Items {
			out.WriteString(fmt.Sprintf("  - %s: %s", item.ProcessInstanceKey, item.Status))
			if item.Summary.RootProcessInstanceKey != "" {
				out.WriteString(fmt.Sprintf(", root %s", item.Summary.RootProcessInstanceKey))
			}
			if len(item.Summary.FamilyKeys) > 0 {
				out.WriteString(fmt.Sprintf(", family keys %s", strings.Join(item.Summary.FamilyKeys, ", ")))
			}
			if item.Summary.Warning != "" {
				out.WriteString(fmt.Sprintf(", warning %s", item.Summary.Warning))
			}
			out.WriteString("\n")
		}
	}

	out.WriteString("\n## Cleanup\n\n")
	writeMarkdownReportField(&out, "No Cleanup", fmt.Sprintf("%t", report.Cleanup.NoCleanup))
	writeMarkdownReportField(&out, "Process Instance Cleanup", string(report.Cleanup.ProcessInstanceCleanup.Status))
	writeMarkdownReportField(&out, "Process Definition Eligibility", string(report.Cleanup.ProcessDefinitionEligibility.Status))
	writeMarkdownReportField(&out, "Process Definition Cleanup", string(report.Cleanup.ProcessDefinitionCleanup.Status))
	writeMarkdownReportList(&out, "Retained Process Instance Keys", report.Cleanup.RetainedProcessInstanceKeys)
	writeMarkdownReportField(&out, "Retained Process Definition Key", report.Cleanup.RetainedProcessDefinitionKey)
	writeMarkdownReportField(&out, "Retained BPMN Process ID", report.Cleanup.RetainedBpmnProcessID)
	writeSmokeTestDefinitionSummary(&out, report.Cleanup.DefinitionSummary)
	writeSmokeTestDefinitionResultsTable(&out, report.Cleanup.DefinitionResults)
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}

// writeSmokeTestDeploymentDefinitionsTable records every returned deployment identity in audit order.
func writeSmokeTestDeploymentDefinitionsTable(out *strings.Builder, definitions []ops.SmokeTestDeployedDefinition) {
	if len(definitions) == 0 {
		return
	}
	out.WriteString("\n### Deployed Process Definitions\n\n")
	out.WriteString("| Key | BPMN Process ID | Version | Tenant | Resource |\n")
	out.WriteString("| --- | --- | ---: | --- | --- |\n")
	for _, definition := range definitions {
		version := ""
		if definition.ProcessDefinitionVersion > 0 {
			version = fmt.Sprintf("%d", definition.ProcessDefinitionVersion)
		}
		out.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			smokeTestMarkdownCell(definition.ProcessDefinitionKey),
			smokeTestMarkdownCell(definition.BpmnProcessID),
			version,
			smokeTestMarkdownCell(definition.TenantID),
			smokeTestMarkdownCell(definition.ResourceName),
		))
	}
}

// writeSmokeTestDefinitionSummary records the authoritative aggregate when collection evidence is available.
func writeSmokeTestDefinitionSummary(out *strings.Builder, summary *ops.SmokeTestDefinitionCleanupSummary) {
	if summary == nil {
		return
	}
	out.WriteString("\n### Process Definition Cleanup Summary\n\n")
	writeMarkdownReportField(out, "Status", string(summary.Status))
	writeMarkdownReportField(out, "Total", fmt.Sprintf("%d", summary.Total))
	writeMarkdownReportField(out, "Confirmed", fmt.Sprintf("%d", summary.Confirmed))
	writeMarkdownReportField(out, "Submitted", fmt.Sprintf("%d", summary.Submitted))
	writeMarkdownReportField(out, "Retained", fmt.Sprintf("%d", summary.Retained))
	writeMarkdownReportField(out, "Failed", fmt.Sprintf("%d", summary.Failed))
	writeMarkdownReportField(out, "Unattempted", fmt.Sprintf("%d", summary.Unattempted))
}

// writeSmokeTestDefinitionResultsTable records each cleanup outcome without rediscovering backend state.
func writeSmokeTestDefinitionResultsTable(out *strings.Builder, results []ops.SmokeTestDefinitionCleanupItem) {
	if len(results) == 0 {
		return
	}
	out.WriteString("\n### Process Definition Cleanup Results\n\n")
	out.WriteString("| Key | BPMN Process ID | Tenant | Disposition | Reason |\n")
	out.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, item := range results {
		out.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			smokeTestMarkdownCell(item.Definition.ProcessDefinitionKey),
			smokeTestMarkdownCell(item.Definition.BpmnProcessID),
			smokeTestMarkdownCell(item.Definition.TenantID),
			smokeTestMarkdownCell(string(item.Disposition)),
			smokeTestMarkdownCell(item.Reason),
		))
	}
}

// smokeTestMarkdownCell keeps dynamic report evidence inside one Markdown table cell.
func smokeTestMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, "\n", " ")
}
