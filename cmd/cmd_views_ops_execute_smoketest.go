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

func renderOpsExecuteSmokeTestResult(cmd *cobra.Command, result ops.SmokeTestResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: execute smoke test")
	} else {
		renderHumanLine(cmd, "execute smoke test")
	}
	renderOpsExecuteSmokeTestPlan(cmd, result)
	renderOpsExecuteSmokeTestDeployment(cmd, result)
	renderOpsExecuteSmokeTestRun(cmd, result)
	renderOpsExecuteSmokeTestWalk(cmd, result)
	renderOpsExecuteSmokeTestCleanup(cmd, result)
	renderOpsExecuteSmokeTestOutcome(cmd, result)
	renderOpsExecuteSmokeTestReportFile(cmd, result)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func renderOpsExecuteSmokeTestPlan(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Plan.CamundaVersion != "" {
		renderHumanLine(cmd, "camunda version: %s", result.Plan.CamundaVersion)
	}
	if result.Plan.Fixture.File != "" {
		renderHumanLine(cmd, "fixture: %s (%s)", result.Plan.Fixture.File, availabilityLabel(result.Plan.Fixture.Available))
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

func renderOpsExecuteSmokeTestDeployment(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Deployment.Status == "" || result.Deployment.Status == ops.WorkflowStepStatusPlanned || result.Deployment.Status == ops.WorkflowStepStatusSkipped {
		return
	}
	renderHumanLine(cmd, "deployment result: %s", result.Deployment.Status)
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
	renderHumanLine(cmd, "run result: %s", result.Run.Status)
	renderHumanLine(cmd, "created process instances: %d/%d", result.Run.CreatedCount, result.Run.RequestedCount)
	if len(result.Run.ProcessInstanceKeys) > 0 {
		renderHumanLine(cmd, "created keys: %s", strings.Join(result.Run.ProcessInstanceKeys, ", "))
	}
}

func renderOpsExecuteSmokeTestWalk(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Walk.Status == "" || result.Walk.Status == ops.WorkflowStepStatusPlanned || result.Walk.Status == ops.WorkflowStepStatusSkipped {
		return
	}
	renderHumanLine(cmd, "walk result: %s", result.Walk.Status)
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

func renderOpsExecuteSmokeTestCleanup(cmd *cobra.Command, result ops.SmokeTestResult) {
	cleanup := result.Cleanup
	if cleanup.NoCleanup {
		renderHumanLine(cmd, "cleanup: skipped (--no-cleanup)")
		if len(cleanup.RetainedProcessInstanceKeys) > 0 {
			renderHumanLine(cmd, "retained process instances: %s", strings.Join(cleanup.RetainedProcessInstanceKeys, ", "))
		}
		if cleanup.RetainedProcessDefinitionKey != "" {
			if cleanup.RetainedBpmnProcessID != "" {
				renderHumanLine(cmd, "retained process definition: %s (%s)", cleanup.RetainedProcessDefinitionKey, cleanup.RetainedBpmnProcessID)
			} else {
				renderHumanLine(cmd, "retained process definition: %s", cleanup.RetainedProcessDefinitionKey)
			}
		} else if cleanup.RetainedBpmnProcessID != "" {
			renderHumanLine(cmd, "retained process definition: %s", cleanup.RetainedBpmnProcessID)
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
}

func renderOpsExecuteSmokeTestOutcome(cmd *cobra.Command, result ops.SmokeTestResult) {
	if result.Outcome == "" {
		return
	}
	elapsed := opsWorkflowElapsedSuffix(result.Report.Duration)
	if result.Outcome == ops.SmokeTestOutcomePlanned {
		renderHumanLine(cmd, "outcome: %s; no changes applied%s", result.Outcome, elapsed)
		return
	}
	renderHumanLine(cmd, "outcome: %s%s", result.Outcome, elapsed)
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

func renderOpsExecuteSmokeTestJSONReport(report ops.SmokeTestAuditReport) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderOpsExecuteSmokeTestMarkdownReport(report ops.SmokeTestAuditReport, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Smoke Test Audit Report\n\n")
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
	writeMarkdownReportList(&out, "Run Errors", report.Errors)

	return []byte(out.String()), nil
}
