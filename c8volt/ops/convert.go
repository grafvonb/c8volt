// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

func toDomainSmokeTestRequest(x SmokeTestRequest) d.SmokeTestRequest {
	return d.SmokeTestRequest{
		CommandName:   x.CommandName,
		DryRun:        x.DryRun,
		Count:         x.Count,
		Workers:       x.Workers,
		FailFast:      x.FailFast,
		NoWorkerLimit: x.NoWorkerLimit,
		NoCleanup:     x.NoCleanup,
		AutoConfirm:   x.AutoConfirm,
		Automation:    x.Automation,
		NoWait:        x.NoWait,
		OutputMode:    x.OutputMode,
		ReportFile:    x.ReportFile,
		ReportFormat:  x.ReportFormat,
		StartedAt:     x.StartedAt,
	}
}

func fromDomainSmokeTestResult(x d.SmokeTestResult) SmokeTestResult {
	return SmokeTestResult{
		Request:    fromDomainSmokeTestRequest(x.Request),
		Plan:       fromDomainSmokeTestPlan(x.Plan),
		Fixture:    fromDomainEmbeddedSmokeTestFixture(x.Fixture),
		Deployment: fromDomainSmokeTestDeploymentResult(x.Deployment),
		Run:        fromDomainSmokeTestRunResult(x.Run),
		Walk:       fromDomainSmokeTestWalkResult(x.Walk),
		Cleanup:    fromDomainSmokeTestCleanupResult(x.Cleanup),
		Report:     fromDomainSmokeTestAuditReport(x.Report),
		Outcome:    SmokeTestOutcome(x.Outcome),
		Errors:     append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestRequest(x d.SmokeTestRequest) SmokeTestRequest {
	return SmokeTestRequest{
		CommandName:   x.CommandName,
		DryRun:        x.DryRun,
		Count:         x.Count,
		Workers:       x.Workers,
		FailFast:      x.FailFast,
		NoWorkerLimit: x.NoWorkerLimit,
		NoCleanup:     x.NoCleanup,
		AutoConfirm:   x.AutoConfirm,
		Automation:    x.Automation,
		NoWait:        x.NoWait,
		OutputMode:    x.OutputMode,
		ReportFile:    x.ReportFile,
		ReportFormat:  x.ReportFormat,
		StartedAt:     x.StartedAt,
	}
}

func fromDomainWorkflowStepResult(x d.WorkflowStepResult) WorkflowStepResult {
	return WorkflowStepResult{
		Name:    x.Name,
		Status:  WorkflowStepStatus(x.Status),
		Message: x.Message,
		Errors:  append([]string(nil), x.Errors...),
	}
}

func fromDomainEmbeddedSmokeTestFixture(x d.EmbeddedSmokeTestFixture) EmbeddedSmokeTestFixture {
	return EmbeddedSmokeTestFixture{
		CamundaVersion: x.CamundaVersion,
		File:           x.File,
		BpmnProcessID:  x.BpmnProcessID,
		Available:      x.Available,
	}
}

func fromDomainSmokeTestPlan(x d.SmokeTestPlan) SmokeTestPlan {
	return SmokeTestPlan{
		Status:           WorkflowStepStatus(x.Status),
		CamundaVersion:   x.CamundaVersion,
		Fixture:          fromDomainEmbeddedSmokeTestFixture(x.Fixture),
		CleanupRequested: x.CleanupRequested,
		PlannedSteps:     toolx.MapSlice(x.PlannedSteps, fromDomainWorkflowStepResult),
		Errors:           append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestDeploymentResult(x d.SmokeTestDeploymentResult) SmokeTestDeploymentResult {
	return SmokeTestDeploymentResult{
		Status:                   WorkflowStepStatus(x.Status),
		FixtureFile:              x.FixtureFile,
		BpmnProcessID:            x.BpmnProcessID,
		ProcessDefinitionKey:     x.ProcessDefinitionKey,
		ProcessDefinitionVersion: x.ProcessDefinitionVersion,
		TenantID:                 x.TenantID,
		Errors:                   append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestRunItem(x d.SmokeTestRunItem) SmokeTestRunItem {
	return SmokeTestRunItem{
		ProcessInstanceKey: x.ProcessInstanceKey,
		Status:             WorkflowStepStatus(x.Status),
		Error:              x.Error,
	}
}

func fromDomainSmokeTestRunResult(x d.SmokeTestRunResult) SmokeTestRunResult {
	return SmokeTestRunResult{
		Status:              WorkflowStepStatus(x.Status),
		RequestedCount:      x.RequestedCount,
		CreatedCount:        x.CreatedCount,
		ProcessInstanceKeys: append(typex.Keys(nil), x.ProcessInstanceKeys...),
		Items:               toolx.MapSlice(x.Items, fromDomainSmokeTestRunItem),
		Errors:              append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestTraversalSummary(x d.SmokeTestTraversalSummary) SmokeTestTraversalSummary {
	return SmokeTestTraversalSummary{
		ProcessInstanceKey:     x.ProcessInstanceKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		FamilyKeys:             append(typex.Keys(nil), x.FamilyKeys...),
		MissingAncestors:       toolx.MapSlice(x.MissingAncestors, fromDomainMissingAncestor),
		Warning:                x.Warning,
		Outcome:                process.TraversalOutcome(x.Outcome),
	}
}

func fromDomainSmokeTestWalkItem(x d.SmokeTestWalkItem) SmokeTestWalkItem {
	return SmokeTestWalkItem{
		ProcessInstanceKey: x.ProcessInstanceKey,
		Status:             WorkflowStepStatus(x.Status),
		Summary:            fromDomainSmokeTestTraversalSummary(x.Summary),
		Error:              x.Error,
	}
}

func fromDomainSmokeTestWalkResult(x d.SmokeTestWalkResult) SmokeTestWalkResult {
	return SmokeTestWalkResult{
		Status: WorkflowStepStatus(x.Status),
		Items:  toolx.MapSlice(x.Items, fromDomainSmokeTestWalkItem),
		Errors: append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestCleanupEligibility(x d.SmokeTestCleanupEligibility) SmokeTestCleanupEligibility {
	return SmokeTestCleanupEligibility{
		Status:   WorkflowStepStatus(x.Status),
		Eligible: x.Eligible,
		Blockers: append([]string(nil), x.Blockers...),
		Errors:   append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestProcessInstanceCleanupResult(x d.SmokeTestProcessInstanceCleanupResult) SmokeTestProcessInstanceCleanupResult {
	return SmokeTestProcessInstanceCleanupResult{
		Status:        WorkflowStepStatus(x.Status),
		SubmittedKeys: append(typex.Keys(nil), x.SubmittedKeys...),
		Items:         toolx.MapSlice(x.Items, fromDomainDeleteReport),
		Submitted:     x.Submitted,
		Confirmed:     x.Confirmed,
		NoWait:        x.NoWait,
		Errors:        append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestProcessDefinitionCleanupResult(x d.SmokeTestProcessDefinitionCleanupResult) SmokeTestProcessDefinitionCleanupResult {
	return SmokeTestProcessDefinitionCleanupResult{
		Status:                        WorkflowStepStatus(x.Status),
		SubmittedProcessDefinitionKey: x.SubmittedProcessDefinitionKey,
		Items:                         toolx.MapSlice(x.Items, fromDomainResourceDeleteReport),
		Submitted:                     x.Submitted,
		Confirmed:                     x.Confirmed,
		NoWait:                        x.NoWait,
		Errors:                        append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestCleanupResult(x d.SmokeTestCleanupResult) SmokeTestCleanupResult {
	return SmokeTestCleanupResult{
		ProcessInstanceCleanup:       fromDomainSmokeTestProcessInstanceCleanupResult(x.ProcessInstanceCleanup),
		ProcessDefinitionEligibility: fromDomainSmokeTestCleanupEligibility(x.ProcessDefinitionEligibility),
		ProcessDefinitionCleanup:     fromDomainSmokeTestProcessDefinitionCleanupResult(x.ProcessDefinitionCleanup),
		NoCleanup:                    x.NoCleanup,
		RetainedProcessInstanceKeys:  append(typex.Keys(nil), x.RetainedProcessInstanceKeys...),
		RetainedProcessDefinitionKey: x.RetainedProcessDefinitionKey,
		RetainedBpmnProcessID:        x.RetainedBpmnProcessID,
		RetainedTenantID:             x.RetainedTenantID,
		Errors:                       append([]string(nil), x.Errors...),
	}
}

func fromDomainSmokeTestAuditReport(x d.SmokeTestAuditReport) SmokeTestAuditReport {
	return SmokeTestAuditReport{
		SchemaVersion:    x.SchemaVersion,
		CommandName:      x.CommandName,
		StartedAt:        x.StartedAt,
		FinishedAt:       x.FinishedAt,
		Duration:         x.Duration,
		DryRun:           x.DryRun,
		C8voltVersion:    x.C8voltVersion,
		CamundaVersion:   x.CamundaVersion,
		ProfileIdentity:  x.ProfileIdentity,
		TenantID:         x.TenantID,
		Fixture:          fromDomainEmbeddedSmokeTestFixture(x.Fixture),
		Plan:             fromDomainSmokeTestPlan(x.Plan),
		Deployment:       fromDomainSmokeTestDeploymentResult(x.Deployment),
		Run:              fromDomainSmokeTestRunResult(x.Run),
		Walk:             fromDomainSmokeTestWalkResult(x.Walk),
		CleanupRequested: x.CleanupRequested,
		NoCleanup:        x.NoCleanup,
		Cleanup:          fromDomainSmokeTestCleanupResult(x.Cleanup),
		AutoConfirm:      x.AutoConfirm,
		Automation:       x.Automation,
		NoWait:           x.NoWait,
		Errors:           append([]string(nil), x.Errors...),
		Outcome:          SmokeTestOutcome(x.Outcome),
	}
}

func toDomainOrphanPurgeRequest(x OrphanPurgeRequest) d.OrphanPurgeRequest {
	out := d.OrphanPurgeRequest{
		CommandName:  x.CommandName,
		DryRun:       x.DryRun,
		AutoConfirm:  x.AutoConfirm,
		Automation:   x.Automation,
		NoWait:       x.NoWait,
		OutputMode:   x.OutputMode,
		Selection:    toDomainProcessInstanceFilter(x.Selection),
		BatchSize:    x.BatchSize,
		Limit:        x.Limit,
		Workers:      x.Workers,
		ReportFile:   x.ReportFile,
		ReportFormat: x.ReportFormat,
		StartedAt:    x.StartedAt,
	}
	if x.DiscoveredKeys != nil {
		out.DiscoveredKeys = append(typex.Keys{}, x.DiscoveredKeys...)
	}
	return out
}

func fromDomainOrphanPurgeResult(x d.OrphanPurgeResult) OrphanPurgeResult {
	return OrphanPurgeResult{
		Request:         fromDomainOrphanPurgeRequest(x.Request),
		Discovery:       fromDomainOrphanDiscoveryResult(x.Discovery),
		DeletionPlan:    fromDomainDeletionPlan(x.DeletionPlan),
		Deletion:        fromDomainDeletionResult(x.Deletion),
		Report:          fromDomainOrphanPurgeReport(x.Report),
		DeleteRequested: x.DeleteRequested,
		Outcome:         OrphanPurgeOutcome(x.Outcome),
		Errors:          append([]string(nil), x.Errors...),
	}
}

func fromDomainOrphanPurgeRequest(x d.OrphanPurgeRequest) OrphanPurgeRequest {
	out := OrphanPurgeRequest{
		CommandName:  x.CommandName,
		DryRun:       x.DryRun,
		AutoConfirm:  x.AutoConfirm,
		Automation:   x.Automation,
		NoWait:       x.NoWait,
		OutputMode:   x.OutputMode,
		Selection:    fromDomainProcessInstanceFilter(x.Selection),
		BatchSize:    x.BatchSize,
		Limit:        x.Limit,
		Workers:      x.Workers,
		ReportFile:   x.ReportFile,
		ReportFormat: x.ReportFormat,
		StartedAt:    x.StartedAt,
	}
	if x.DiscoveredKeys != nil {
		out.DiscoveredKeys = append(typex.Keys{}, x.DiscoveredKeys...)
	}
	return out
}

func fromDomainOrphanDiscoveryResult(x d.OrphanDiscoveryResult) OrphanDiscoveryResult {
	return OrphanDiscoveryResult{
		Status:  WorkflowStepStatus(x.Status),
		Filters: fromDomainProcessInstanceFilter(x.Filters),
		Keys:    append([]string(nil), x.Keys...),
		Count:   x.Count,
		Errors:  append([]string(nil), x.Errors...),
	}
}

func fromDomainDeletionPlan(x d.DeletionPlan) DeletionPlan {
	return DeletionPlan{
		Status:               WorkflowStepStatus(x.Status),
		RequestedKeys:        append([]string(nil), x.RequestedKeys...),
		AffectedKeys:         append([]string(nil), x.AffectedKeys...),
		RootKeys:             append([]string(nil), x.RootKeys...),
		RequiresConfirmation: x.RequiresConfirmation,
		DryRunPreview:        fromDomainDryRunPIKeyExpansion(x.DryRunPreview),
		Errors:               append([]string(nil), x.Errors...),
	}
}

func fromDomainDeletionResult(x d.DeletionResult) DeletionResult {
	return DeletionResult{
		Status:    WorkflowStepStatus(x.Status),
		Items:     toolx.MapSlice(x.Items, fromDomainDeleteReport),
		Errors:    append([]string(nil), x.Errors...),
		Submitted: x.Submitted,
		Confirmed: x.Confirmed,
		NoWait:    x.NoWait,
	}
}

func fromDomainOrphanPurgeReport(x d.OrphanPurgeReport) OrphanPurgeReport {
	return OrphanPurgeReport{
		SchemaVersion:    x.SchemaVersion,
		CommandName:      x.CommandName,
		StartedAt:        x.StartedAt,
		FinishedAt:       x.FinishedAt,
		Duration:         x.Duration,
		DryRun:           x.DryRun,
		C8voltVersion:    x.C8voltVersion,
		CamundaVersion:   x.CamundaVersion,
		ProfileIdentity:  x.ProfileIdentity,
		SelectionFilters: fromDomainProcessInstanceFilter(x.SelectionFilters),
		Discovery:        fromDomainOrphanDiscoveryResult(x.Discovery),
		DeletionPlan:     fromDomainDeletionPlan(x.DeletionPlan),
		Deletion:         fromDomainDeletionResult(x.Deletion),
		DeleteRequested:  x.DeleteRequested,
		AutoConfirm:      x.AutoConfirm,
		Automation:       x.Automation,
		NoWait:           x.NoWait,
		Errors:           append([]string(nil), x.Errors...),
		Outcome:          OrphanPurgeOutcome(x.Outcome),
	}
}

func toDomainRetentionPolicyRequest(x RetentionPolicyRequest) d.RetentionPolicyRequest {
	out := d.RetentionPolicyRequest{
		CommandName:            x.CommandName,
		RetentionDays:          x.RetentionDays,
		DerivedEndDateBoundary: x.DerivedEndDateBoundary,
		DryRun:                 x.DryRun,
		AutoConfirm:            x.AutoConfirm,
		Automation:             x.Automation,
		OutputMode:             x.OutputMode,
		Selection:              toDomainProcessInstanceFilter(x.Selection),
		BatchSize:              x.BatchSize,
		Limit:                  x.Limit,
		Workers:                x.Workers,
		NoWait:                 x.NoWait,
		NoStateCheck:           x.NoStateCheck,
		Force:                  x.Force,
		FailFast:               x.FailFast,
		NoWorkerLimit:          x.NoWorkerLimit,
		ReportFile:             x.ReportFile,
		ReportFormat:           x.ReportFormat,
		StartedAt:              x.StartedAt,
	}
	if x.DiscoveredKeys != nil {
		out.DiscoveredKeys = append(typex.Keys{}, x.DiscoveredKeys...)
	}
	return out
}

func fromDomainRetentionPolicyResult(x d.RetentionPolicyResult) RetentionPolicyResult {
	return RetentionPolicyResult{
		Request:    fromDomainRetentionPolicyRequest(x.Request),
		Discovery:  fromDomainRetentionDiscoveryResult(x.Discovery),
		DeletePlan: fromDomainRetentionDeletePlan(x.DeletePlan),
		Deletion:   fromDomainRetentionDeletionResult(x.Deletion),
		Report:     fromDomainRetentionAuditReport(x.Report),
		Outcome:    RetentionPolicyOutcome(x.Outcome),
		Errors:     append([]string(nil), x.Errors...),
	}
}

func fromDomainRetentionPolicyRequest(x d.RetentionPolicyRequest) RetentionPolicyRequest {
	out := RetentionPolicyRequest{
		CommandName:            x.CommandName,
		RetentionDays:          x.RetentionDays,
		DerivedEndDateBoundary: x.DerivedEndDateBoundary,
		DryRun:                 x.DryRun,
		AutoConfirm:            x.AutoConfirm,
		Automation:             x.Automation,
		OutputMode:             x.OutputMode,
		Selection:              fromDomainProcessInstanceFilter(x.Selection),
		BatchSize:              x.BatchSize,
		Limit:                  x.Limit,
		Workers:                x.Workers,
		NoWait:                 x.NoWait,
		NoStateCheck:           x.NoStateCheck,
		Force:                  x.Force,
		FailFast:               x.FailFast,
		NoWorkerLimit:          x.NoWorkerLimit,
		ReportFile:             x.ReportFile,
		ReportFormat:           x.ReportFormat,
		StartedAt:              x.StartedAt,
	}
	if x.DiscoveredKeys != nil {
		out.DiscoveredKeys = append(typex.Keys{}, x.DiscoveredKeys...)
	}
	return out
}

func fromDomainRetentionDiscoveryResult(x d.RetentionDiscoveryResult) RetentionDiscoveryResult {
	return RetentionDiscoveryResult{
		Status:                 WorkflowStepStatus(x.Status),
		RetentionDays:          x.RetentionDays,
		DerivedEndDateBoundary: x.DerivedEndDateBoundary,
		Filters:                fromDomainProcessInstanceFilter(x.Filters),
		SeedKeys:               append([]string(nil), x.SeedKeys...),
		Count:                  x.Count,
		Notices:                toolx.MapSlice(x.Notices, fromDomainRetentionWorkflowNotice),
		Errors:                 append([]string(nil), x.Errors...),
	}
}

func fromDomainRetentionDeletePlan(x d.RetentionDeletePlan) RetentionDeletePlan {
	return RetentionDeletePlan{
		Status:                WorkflowStepStatus(x.Status),
		SeedKeys:              append([]string(nil), x.SeedKeys...),
		ResolvedRootKeys:      append([]string(nil), x.ResolvedRootKeys...),
		AffectedKeys:          append([]string(nil), x.AffectedKeys...),
		DuplicateKeys:         append([]string(nil), x.DuplicateKeys...),
		FinalStateItems:       toolx.MapSlice(x.FinalStateItems, fromDomainProcessInstance),
		NonFinalAffectedItems: toolx.MapSlice(x.NonFinalAffectedItems, fromDomainProcessInstance),
		SkippedSeedKeys:       append([]string(nil), x.SkippedSeedKeys...),
		SkippedNonFinalRoots:  toolx.MapSlice(x.SkippedNonFinalRoots, fromDomainProcessInstance),
		MissingAncestors:      toolx.MapSlice(x.MissingAncestors, fromDomainMissingAncestor),
		TraversalWarnings:     append([]string(nil), x.TraversalWarnings...),
		RequiresConfirmation:  x.RequiresConfirmation,
		Errors:                append([]string(nil), x.Errors...),
	}
}

func fromDomainRetentionDeletionResult(x d.RetentionDeletionResult) RetentionDeletionResult {
	return RetentionDeletionResult{
		Status:            WorkflowStepStatus(x.Status),
		SubmittedRootKeys: append([]string(nil), x.SubmittedRootKeys...),
		Items:             toolx.MapSlice(x.Items, fromDomainDeleteReport),
		Submitted:         x.Submitted,
		Confirmed:         x.Confirmed,
		NoWait:            x.NoWait,
		Errors:            append([]string(nil), x.Errors...),
	}
}

func fromDomainRetentionAuditReport(x d.RetentionAuditReport) RetentionAuditReport {
	return RetentionAuditReport{
		SchemaVersion:          x.SchemaVersion,
		CommandName:            x.CommandName,
		StartedAt:              x.StartedAt,
		FinishedAt:             x.FinishedAt,
		Duration:               x.Duration,
		DryRun:                 x.DryRun,
		C8voltVersion:          x.C8voltVersion,
		CamundaVersion:         x.CamundaVersion,
		ProfileIdentity:        x.ProfileIdentity,
		TenantID:               x.TenantID,
		RetentionDays:          x.RetentionDays,
		DerivedEndDateBoundary: x.DerivedEndDateBoundary,
		SelectionFilters:       fromDomainProcessInstanceFilter(x.SelectionFilters),
		Discovery:              fromDomainRetentionDiscoveryResult(x.Discovery),
		DeletePlan:             fromDomainRetentionDeletePlan(x.DeletePlan),
		Deletion:               fromDomainRetentionDeletionResult(x.Deletion),
		AutoConfirm:            x.AutoConfirm,
		Automation:             x.Automation,
		NoWait:                 x.NoWait,
		NoStateCheck:           x.NoStateCheck,
		Force:                  x.Force,
		FailFast:               x.FailFast,
		NoWorkerLimit:          x.NoWorkerLimit,
		Errors:                 append([]string(nil), x.Errors...),
		Outcome:                RetentionPolicyOutcome(x.Outcome),
	}
}

func fromDomainRetentionWorkflowNotice(x d.RetentionWorkflowNotice) RetentionWorkflowNotice {
	return RetentionWorkflowNotice{
		Code:     x.Code,
		Severity: x.Severity,
		Message:  x.Message,
		Details:  toolx.CopyMap(x.Details),
	}
}

// toDomainIncidentPurgeRequest maps the public purge request to the service contract.
func toDomainIncidentPurgeRequest(x IncidentPurgeRequest) d.IncidentPurgeRequest {
	out := d.IncidentPurgeRequest{
		CommandName:   x.CommandName,
		DryRun:        x.DryRun,
		AutoConfirm:   x.AutoConfirm,
		Automation:    x.Automation,
		OutputMode:    x.OutputMode,
		Selection:     toDomainIncidentFilter(x.Selection),
		BatchSize:     x.BatchSize,
		Limit:         x.Limit,
		Workers:       x.Workers,
		FailFast:      x.FailFast,
		NoWorkerLimit: x.NoWorkerLimit,
		NoWait:        x.NoWait,
		Force:         x.Force,
		ReportFile:    x.ReportFile,
		ReportFormat:  x.ReportFormat,
		StartedAt:     x.StartedAt,
	}
	if x.DiscoveredCandidateProcessInstanceKeys != nil {
		out.DiscoveredCandidateProcessInstanceKeys = append(typex.Keys{}, x.DiscoveredCandidateProcessInstanceKeys...)
	}
	return out
}

// fromDomainIncidentPurgeResult maps the service result to the public facade model.
func fromDomainIncidentPurgeResult(x d.IncidentPurgeResult) IncidentPurgeResult {
	return IncidentPurgeResult{
		Request:    fromDomainIncidentPurgeRequest(x.Request),
		Discovery:  fromDomainIncidentDiscoveryResult(x.Discovery),
		DeletePlan: fromDomainIncidentPurgeDeletePlan(x.DeletePlan),
		Deletion:   fromDomainIncidentPurgeDeletionResult(x.Deletion),
		Report:     fromDomainIncidentPurgeReport(x.Report),
		Outcome:    IncidentPurgeOutcome(x.Outcome),
		Errors:     append([]string(nil), x.Errors...),
		Notices:    toolx.MapSlice(x.Notices, fromDomainIncidentPurgeWorkflowNotice),
	}
}

// fromDomainIncidentPurgeRequest maps a service request back to public output.
func fromDomainIncidentPurgeRequest(x d.IncidentPurgeRequest) IncidentPurgeRequest {
	out := IncidentPurgeRequest{
		CommandName:   x.CommandName,
		DryRun:        x.DryRun,
		AutoConfirm:   x.AutoConfirm,
		Automation:    x.Automation,
		OutputMode:    x.OutputMode,
		Selection:     fromDomainIncidentFilter(x.Selection),
		BatchSize:     x.BatchSize,
		Limit:         x.Limit,
		Workers:       x.Workers,
		FailFast:      x.FailFast,
		NoWorkerLimit: x.NoWorkerLimit,
		NoWait:        x.NoWait,
		Force:         x.Force,
		ReportFile:    x.ReportFile,
		ReportFormat:  x.ReportFormat,
		StartedAt:     x.StartedAt,
	}
	if x.DiscoveredCandidateProcessInstanceKeys != nil {
		out.DiscoveredCandidateProcessInstanceKeys = append(typex.Keys{}, x.DiscoveredCandidateProcessInstanceKeys...)
	}
	return out
}

// fromDomainIncidentDiscoveryResult maps discovery details to the public model.
func fromDomainIncidentDiscoveryResult(x d.IncidentDiscoveryResult) IncidentDiscoveryResult {
	return IncidentDiscoveryResult{
		Status:                                WorkflowStepStatus(x.Status),
		Filters:                               fromDomainIncidentFilter(x.Filters),
		CandidateIncidents:                    toolx.MapSlice(x.CandidateIncidents, fromDomainIncidentDetail),
		IncidentKeys:                          append(typex.Keys{}, x.IncidentKeys...),
		CandidateProcessInstanceKeys:          append(typex.Keys{}, x.CandidateProcessInstanceKeys...),
		DuplicateCandidateProcessInstanceKeys: append(typex.Keys{}, x.DuplicateCandidateProcessInstanceKeys...),
		SkippedIncidents:                      toolx.MapSlice(x.SkippedIncidents, fromDomainIncidentPurgeSkippedIncident),
		IncidentCount:                         x.IncidentCount,
		CandidateProcessInstanceCount:         x.CandidateProcessInstanceCount,
		Notices:                               toolx.MapSlice(x.Notices, fromDomainIncidentPurgeWorkflowNotice),
		Errors:                                append([]string(nil), x.Errors...),
	}
}

// fromDomainIncidentPurgeSkippedIncident maps skipped incident metadata to public output.
func fromDomainIncidentPurgeSkippedIncident(x d.IncidentPurgeSkippedIncident) IncidentPurgeSkippedIncident {
	return IncidentPurgeSkippedIncident{
		Incident: fromDomainIncidentDetail(x.Incident),
		Reason:   x.Reason,
	}
}

// fromDomainIncidentPurgeDeletePlan maps the service delete plan to the public model.
func fromDomainIncidentPurgeDeletePlan(x d.IncidentPurgeDeletePlan) IncidentPurgeDeletePlan {
	return IncidentPurgeDeletePlan{
		Status:                                WorkflowStepStatus(x.Status),
		CandidateProcessInstanceKeys:          append(typex.Keys{}, x.CandidateProcessInstanceKeys...),
		ResolvedRootKeys:                      append(typex.Keys{}, x.ResolvedRootKeys...),
		AffectedKeys:                          append(typex.Keys{}, x.AffectedKeys...),
		DuplicateCandidateProcessInstanceKeys: append(typex.Keys{}, x.DuplicateCandidateProcessInstanceKeys...),
		DuplicateResolvedRootKeys:             append(typex.Keys{}, x.DuplicateResolvedRootKeys...),
		FinalStateItems:                       toolx.MapSlice(x.FinalStateItems, fromDomainProcessInstance),
		NonFinalAffectedItems:                 toolx.MapSlice(x.NonFinalAffectedItems, fromDomainProcessInstance),
		MissingAncestors:                      toolx.MapSlice(x.MissingAncestors, fromDomainMissingAncestor),
		TraversalWarnings:                     append([]string(nil), x.TraversalWarnings...),
		RequiresConfirmation:                  x.RequiresConfirmation,
		Errors:                                append([]string(nil), x.Errors...),
	}
}

// fromDomainIncidentPurgeDeletionResult maps deletion submission results to the public model.
func fromDomainIncidentPurgeDeletionResult(x d.IncidentPurgeDeletionResult) IncidentPurgeDeletionResult {
	return IncidentPurgeDeletionResult{
		Status:            WorkflowStepStatus(x.Status),
		SubmittedRootKeys: append(typex.Keys{}, x.SubmittedRootKeys...),
		Items:             toolx.MapSlice(x.Items, fromDomainDeleteReport),
		Submitted:         x.Submitted,
		Confirmed:         x.Confirmed,
		NoWait:            x.NoWait,
		Errors:            append([]string(nil), x.Errors...),
	}
}

// fromDomainIncidentPurgeReport maps the service audit model to public output.
func fromDomainIncidentPurgeReport(x d.IncidentPurgeReport) IncidentPurgeReport {
	return IncidentPurgeReport{
		SchemaVersion:    x.SchemaVersion,
		CommandName:      x.CommandName,
		StartedAt:        x.StartedAt,
		FinishedAt:       x.FinishedAt,
		Duration:         x.Duration,
		DryRun:           x.DryRun,
		C8voltVersion:    x.C8voltVersion,
		CamundaVersion:   x.CamundaVersion,
		ProfileIdentity:  x.ProfileIdentity,
		TenantID:         x.TenantID,
		SelectionFilters: fromDomainIncidentFilter(x.SelectionFilters),
		Discovery:        fromDomainIncidentDiscoveryResult(x.Discovery),
		DeletePlan:       fromDomainIncidentPurgeDeletePlan(x.DeletePlan),
		Deletion:         fromDomainIncidentPurgeDeletionResult(x.Deletion),
		AutoConfirm:      x.AutoConfirm,
		Automation:       x.Automation,
		NoWait:           x.NoWait,
		Force:            x.Force,
		FailFast:         x.FailFast,
		NoWorkerLimit:    x.NoWorkerLimit,
		Errors:           append([]string(nil), x.Errors...),
		Notices:          toolx.MapSlice(x.Notices, fromDomainIncidentPurgeWorkflowNotice),
		Outcome:          IncidentPurgeOutcome(x.Outcome),
	}
}

// fromDomainIncidentPurgeWorkflowNotice maps structured workflow notices to public output.
func fromDomainIncidentPurgeWorkflowNotice(x d.IncidentPurgeWorkflowNotice) IncidentPurgeWorkflowNotice {
	return IncidentPurgeWorkflowNotice{
		Code:     x.Code,
		Severity: x.Severity,
		Message:  x.Message,
		Details:  toolx.CopyMap(x.Details),
	}
}

// toDomainRepairRequest maps public repair controls to the service request model.
func toDomainRepairRequest(x RepairRequest) d.OpsRepairRequest {
	return d.OpsRepairRequest{
		CommandName:              x.CommandName,
		Target:                   d.OpsRepairTarget(x.Target),
		DiscoveryMode:            d.OpsRepairDiscoveryMode(x.DiscoveryMode),
		InputKeys:                append(typex.Keys{}, x.InputKeys...),
		IncidentSelection:        toDomainIncidentFilter(x.IncidentSelection),
		ProcessInstanceSelection: toDomainProcessInstanceFilter(x.ProcessInstanceSelection),
		DirectIncidentsOnly:      x.DirectIncidentsOnly,
		BatchSize:                x.BatchSize,
		Limit:                    x.Limit,
		Workers:                  x.Workers,
		FailFast:                 x.FailFast,
		NoWorkerLimit:            x.NoWorkerLimit,
		DryRun:                   x.DryRun,
		AutoConfirm:              x.AutoConfirm,
		Automation:               x.Automation,
		NoWait:                   x.NoWait,
		OutputMode:               x.OutputMode,
		Variables:                toolx.CopyMap(x.Variables),
		VariablesFile:            x.VariablesFile,
		RequestedRetries:         toolx.CopyPtr(x.RequestedRetries),
		RequestedJobTimeout:      x.RequestedJobTimeout,
		ReportFile:               x.ReportFile,
		ReportFormat:             x.ReportFormat,
		StartedAt:                x.StartedAt,
	}
}

// fromDomainRepairResult maps the service repair result to the public facade model.
func fromDomainRepairResult(x d.OpsRepairResult) RepairResult {
	return RepairResult{
		Request:          fromDomainRepairRequest(x.Request),
		FrozenSet:        fromDomainRepairFrozenSet(x.FrozenSet),
		Plan:             toolx.MapSlice(x.Plan, fromDomainRepairPlanItem),
		VariableUpdates:  toolx.MapSlice(x.VariableUpdates, fromDomainRepairVariableScopeUpdate),
		JobApplicability: toolx.MapSlice(x.JobApplicability, fromDomainRepairJobApplicability),
		Remaining:        fromDomainRepairRemainingIncidentSummary(x.Remaining),
		Report:           fromDomainRepairAuditReport(x.Report),
		Outcome:          RepairOutcome(x.Outcome),
		Errors:           append([]string(nil), x.Errors...),
		Notices:          toolx.MapSlice(x.Notices, fromDomainRepairNotice),
	}
}

// fromDomainRepairRequest maps a service repair request back to public output.
func fromDomainRepairRequest(x d.OpsRepairRequest) RepairRequest {
	return RepairRequest{
		CommandName:              x.CommandName,
		Target:                   RepairTarget(x.Target),
		DiscoveryMode:            RepairDiscoveryMode(x.DiscoveryMode),
		InputKeys:                append(typex.Keys{}, x.InputKeys...),
		IncidentSelection:        fromDomainIncidentFilter(x.IncidentSelection),
		ProcessInstanceSelection: fromDomainProcessInstanceFilter(x.ProcessInstanceSelection),
		DirectIncidentsOnly:      x.DirectIncidentsOnly,
		BatchSize:                x.BatchSize,
		Limit:                    x.Limit,
		Workers:                  x.Workers,
		FailFast:                 x.FailFast,
		NoWorkerLimit:            x.NoWorkerLimit,
		DryRun:                   x.DryRun,
		AutoConfirm:              x.AutoConfirm,
		Automation:               x.Automation,
		NoWait:                   x.NoWait,
		OutputMode:               x.OutputMode,
		Variables:                toolx.CopyMap(x.Variables),
		VariablesFile:            x.VariablesFile,
		RequestedRetries:         toolx.CopyPtr(x.RequestedRetries),
		RequestedJobTimeout:      x.RequestedJobTimeout,
		ReportFile:               x.ReportFile,
		ReportFormat:             x.ReportFormat,
		StartedAt:                x.StartedAt,
	}
}

// fromDomainRepairFrozenSet maps frozen repair targets to public output.
func fromDomainRepairFrozenSet(x d.OpsRepairFrozenSet) RepairFrozenSet {
	return RepairFrozenSet{
		Status:              WorkflowStepStatus(x.Status),
		Target:              RepairTarget(x.Target),
		DiscoveryMode:       RepairDiscoveryMode(x.DiscoveryMode),
		InputKeys:           append(typex.Keys{}, x.InputKeys...),
		IncidentKeys:        append(typex.Keys{}, x.IncidentKeys...),
		ProcessInstanceKeys: append(typex.Keys{}, x.ProcessInstanceKeys...),
		RootProcessKeys:     append(typex.Keys{}, x.RootProcessKeys...),
		JobKeys:             append(typex.Keys{}, x.JobKeys...),
		VariableScopes:      append(typex.Keys{}, x.VariableScopes...),
		OriginalIncidents:   toolx.MapSlice(x.OriginalIncidents, fromDomainIncidentDetail),
		IncidentFilters:     fromDomainIncidentFilter(x.IncidentFilters),
		ProcessFilters:      fromDomainProcessInstanceFilter(x.ProcessFilters),
		Errors:              append([]string(nil), x.Errors...),
	}
}

// fromDomainRepairPlanItem maps one service repair plan row to public output.
func fromDomainRepairPlanItem(x d.OpsRepairPlanItem) RepairPlanItem {
	return RepairPlanItem{
		IncidentKey:            x.IncidentKey,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		JobKey:                 x.JobKey,
		VariableScopeKey:       x.VariableScopeKey,
		RequestedVariableNames: append([]string(nil), x.RequestedVariableNames...),
		RequestedRetries:       toolx.CopyPtr(x.RequestedRetries),
		RequestedTimeout:       x.RequestedTimeout,
		VariableUpdateStatus:   WorkflowStepStatus(x.VariableUpdateStatus),
		RetryUpdateStatus:      WorkflowStepStatus(x.RetryUpdateStatus),
		TimeoutUpdateStatus:    WorkflowStepStatus(x.TimeoutUpdateStatus),
		ResolutionStatus:       WorkflowStepStatus(x.ResolutionStatus),
		ConfirmationStatus:     WorkflowStepStatus(x.ConfirmationStatus),
		Notices:                toolx.MapSlice(x.Notices, fromDomainRepairNotice),
		Errors:                 append([]string(nil), x.Errors...),
	}
}

// fromDomainRepairVariableScopeUpdate maps one variable scope update result to public output.
func fromDomainRepairVariableScopeUpdate(x d.OpsRepairVariableScopeUpdate) RepairVariableScopeUpdate {
	return RepairVariableScopeUpdate{
		ScopeKey:              x.ScopeKey,
		VariableNames:         append([]string(nil), x.VariableNames...),
		Payload:               toolx.CopyMap(x.Payload),
		DependentIncidentKeys: append(typex.Keys{}, x.DependentIncidentKeys...),
		Status:                WorkflowStepStatus(x.Status),
		Errors:                append([]string(nil), x.Errors...),
	}
}

// fromDomainRepairJobApplicability maps one job repair applicability decision to public output.
func fromDomainRepairJobApplicability(x d.OpsRepairJobApplicability) RepairJobApplicability {
	return RepairJobApplicability{
		IncidentKey:        x.IncidentKey,
		JobKey:             x.JobKey,
		RetryStatus:        WorkflowStepStatus(x.RetryStatus),
		TimeoutStatus:      WorkflowStepStatus(x.TimeoutStatus),
		Reason:             x.Reason,
		RequestedRetries:   toolx.CopyPtr(x.RequestedRetries),
		RequestedTimeout:   x.RequestedTimeout,
		UnsupportedVersion: x.UnsupportedVersion,
	}
}

// fromDomainRepairRemainingIncidentSummary maps post-repair incident context to public output.
func fromDomainRepairRemainingIncidentSummary(x d.OpsRepairRemainingIncidentSummary) RepairRemainingIncidentSummary {
	return RepairRemainingIncidentSummary{
		Status:     WorkflowStepStatus(x.Status),
		ActiveKeys: append(typex.Keys{}, x.ActiveKeys...),
		Incidents:  toolx.MapSlice(x.Incidents, fromDomainIncidentDetail),
		Checked:    x.Checked,
		Errors:     append([]string(nil), x.Errors...),
	}
}

// fromDomainRepairAuditReport maps the service repair audit model to public output.
func fromDomainRepairAuditReport(x d.OpsRepairAuditReport) RepairAuditReport {
	return RepairAuditReport{
		SchemaVersion:    x.SchemaVersion,
		CommandName:      x.CommandName,
		StartedAt:        x.StartedAt,
		FinishedAt:       x.FinishedAt,
		Duration:         x.Duration,
		DryRun:           x.DryRun,
		C8voltVersion:    x.C8voltVersion,
		CamundaVersion:   x.CamundaVersion,
		ProfileIdentity:  x.ProfileIdentity,
		TenantID:         x.TenantID,
		Request:          fromDomainRepairRequest(x.Request),
		FrozenSet:        fromDomainRepairFrozenSet(x.FrozenSet),
		Plan:             toolx.MapSlice(x.Plan, fromDomainRepairPlanItem),
		VariableUpdates:  toolx.MapSlice(x.VariableUpdates, fromDomainRepairVariableScopeUpdate),
		JobApplicability: toolx.MapSlice(x.JobApplicability, fromDomainRepairJobApplicability),
		Remaining:        fromDomainRepairRemainingIncidentSummary(x.Remaining),
		AutoConfirm:      x.AutoConfirm,
		Automation:       x.Automation,
		NoWait:           x.NoWait,
		FailFast:         x.FailFast,
		NoWorkerLimit:    x.NoWorkerLimit,
		Errors:           append([]string(nil), x.Errors...),
		Notices:          toolx.MapSlice(x.Notices, fromDomainRepairNotice),
		Outcome:          RepairOutcome(x.Outcome),
	}
}

// fromDomainRepairNotice maps structured repair notices to public output.
func fromDomainRepairNotice(x d.OpsRepairWorkflowNotice) RepairNotice {
	return RepairNotice{
		Code:     x.Code,
		Severity: x.Severity,
		Message:  x.Message,
		Details:  toolx.CopyMap(x.Details),
	}
}

// toDomainAllProcessDefinitionsPurgeRequest maps the public purge request to the service contract.
func toDomainAllProcessDefinitionsPurgeRequest(x AllProcessDefinitionsPurgeRequest) d.AllProcessDefinitionsPurgeRequest {
	out := d.AllProcessDefinitionsPurgeRequest{
		CommandName:   x.CommandName,
		DryRun:        x.DryRun,
		AutoConfirm:   x.AutoConfirm,
		Automation:    x.Automation,
		OutputMode:    x.OutputMode,
		Selection:     toDomainProcessDefinitionSelection(x.Selection),
		Workers:       x.Workers,
		FailFast:      x.FailFast,
		NoWorkerLimit: x.NoWorkerLimit,
		NoWait:        x.NoWait,
		Force:         x.Force,
		ReportFile:    x.ReportFile,
		ReportFormat:  x.ReportFormat,
		StartedAt:     x.StartedAt,
	}
	if x.DiscoveredCandidateProcessDefinitionKeys != nil {
		out.DiscoveredCandidateProcessDefinitionKeys = append(typex.Keys{}, x.DiscoveredCandidateProcessDefinitionKeys...)
	}
	return out
}

// fromDomainAllProcessDefinitionsPurgeResult maps the service result to the public facade model.
func fromDomainAllProcessDefinitionsPurgeResult(x d.AllProcessDefinitionsPurgeResult) AllProcessDefinitionsPurgeResult {
	return AllProcessDefinitionsPurgeResult{
		Request:    fromDomainAllProcessDefinitionsPurgeRequest(x.Request),
		Discovery:  fromDomainProcessDefinitionDiscoveryResult(x.Discovery),
		DeletePlan: fromDomainAllProcessDefinitionsPurgeDeletePlan(x.DeletePlan),
		Deletion:   fromDomainAllProcessDefinitionsPurgeDeletionResult(x.Deletion),
		Report:     fromDomainAllProcessDefinitionsPurgeReport(x.Report),
		Outcome:    AllProcessDefinitionsPurgeOutcome(x.Outcome),
		Errors:     append([]string(nil), x.Errors...),
		Notices:    toolx.MapSlice(x.Notices, fromDomainAllProcessDefinitionsPurgeNotice),
	}
}

// fromDomainAllProcessDefinitionsPurgeRequest maps a service request back to public output.
func fromDomainAllProcessDefinitionsPurgeRequest(x d.AllProcessDefinitionsPurgeRequest) AllProcessDefinitionsPurgeRequest {
	out := AllProcessDefinitionsPurgeRequest{
		CommandName:   x.CommandName,
		DryRun:        x.DryRun,
		AutoConfirm:   x.AutoConfirm,
		Automation:    x.Automation,
		OutputMode:    x.OutputMode,
		Selection:     fromDomainProcessDefinitionSelection(x.Selection),
		Workers:       x.Workers,
		FailFast:      x.FailFast,
		NoWorkerLimit: x.NoWorkerLimit,
		NoWait:        x.NoWait,
		Force:         x.Force,
		ReportFile:    x.ReportFile,
		ReportFormat:  x.ReportFormat,
		StartedAt:     x.StartedAt,
	}
	if x.DiscoveredCandidateProcessDefinitionKeys != nil {
		out.DiscoveredCandidateProcessDefinitionKeys = append(typex.Keys{}, x.DiscoveredCandidateProcessDefinitionKeys...)
	}
	return out
}

// fromDomainProcessDefinitionDiscoveryResult maps discovery details to the public model.
func fromDomainProcessDefinitionDiscoveryResult(x d.ProcessDefinitionDiscoveryResult) ProcessDefinitionDiscoveryResult {
	return ProcessDefinitionDiscoveryResult{
		Status:                                  WorkflowStepStatus(x.Status),
		Filters:                                 fromDomainProcessDefinitionSelection(x.Filters),
		CandidateProcessDefinitionKeys:          append(typex.Keys{}, x.CandidateProcessDefinitionKeys...),
		CandidateProcessDefinitions:             toolx.MapSlice(x.CandidateProcessDefinitions, fromDomainProcessDefinition),
		DuplicateCandidateProcessDefinitionKeys: append(typex.Keys{}, x.DuplicateCandidateProcessDefinitionKeys...),
		CandidateProcessDefinitionCount:         x.CandidateProcessDefinitionCount,
		LatestOnly:                              x.LatestOnly,
		Notices:                                 toolx.MapSlice(x.Notices, fromDomainAllProcessDefinitionsPurgeNotice),
		Errors:                                  append([]string(nil), x.Errors...),
	}
}

// fromDomainAllProcessDefinitionsPurgeDeletePlan maps the service delete plan to the public model.
func fromDomainAllProcessDefinitionsPurgeDeletePlan(x d.AllProcessDefinitionsPurgeDeletePlan) AllProcessDefinitionsPurgeDeletePlan {
	return AllProcessDefinitionsPurgeDeletePlan{
		Status:                                  WorkflowStepStatus(x.Status),
		CandidateProcessDefinitionKeys:          append(typex.Keys{}, x.CandidateProcessDefinitionKeys...),
		Items:                                   toolx.MapSlice(x.Items, fromDomainDeleteProcessDefinitionPlanItem),
		DuplicateCandidateProcessDefinitionKeys: append(typex.Keys{}, x.DuplicateCandidateProcessDefinitionKeys...),
		AffectedProcessInstanceCount:            x.AffectedProcessInstanceCount,
		ActiveProcessInstanceCount:              x.ActiveProcessInstanceCount,
		RequiresConfirmation:                    x.RequiresConfirmation,
		RequiresForce:                           x.RequiresForce,
		Errors:                                  append([]string(nil), x.Errors...),
	}
}

// fromDomainAllProcessDefinitionsPurgeDeletionResult maps deletion submission results to the public model.
func fromDomainAllProcessDefinitionsPurgeDeletionResult(x d.AllProcessDefinitionsPurgeDeletionResult) AllProcessDefinitionsPurgeDeletionResult {
	return AllProcessDefinitionsPurgeDeletionResult{
		Status:                         WorkflowStepStatus(x.Status),
		SubmittedProcessDefinitionKeys: append(typex.Keys{}, x.SubmittedProcessDefinitionKeys...),
		Items:                          toolx.MapSlice(x.Items, fromDomainResourceDeleteReport),
		Submitted:                      x.Submitted,
		Confirmed:                      x.Confirmed,
		NoWait:                         x.NoWait,
		Errors:                         append([]string(nil), x.Errors...),
	}
}

// fromDomainAllProcessDefinitionsPurgeReport maps the service audit model to public output.
func fromDomainAllProcessDefinitionsPurgeReport(x d.AllProcessDefinitionsPurgeReport) AllProcessDefinitionsPurgeReport {
	return AllProcessDefinitionsPurgeReport{
		SchemaVersion:    x.SchemaVersion,
		CommandName:      x.CommandName,
		StartedAt:        x.StartedAt,
		FinishedAt:       x.FinishedAt,
		Duration:         x.Duration,
		DryRun:           x.DryRun,
		C8voltVersion:    x.C8voltVersion,
		CamundaVersion:   x.CamundaVersion,
		ProfileIdentity:  x.ProfileIdentity,
		TenantID:         x.TenantID,
		SelectionFilters: fromDomainProcessDefinitionSelection(x.SelectionFilters),
		Discovery:        fromDomainProcessDefinitionDiscoveryResult(x.Discovery),
		DeletePlan:       fromDomainAllProcessDefinitionsPurgeDeletePlan(x.DeletePlan),
		Deletion:         fromDomainAllProcessDefinitionsPurgeDeletionResult(x.Deletion),
		AutoConfirm:      x.AutoConfirm,
		Automation:       x.Automation,
		NoWait:           x.NoWait,
		Force:            x.Force,
		FailFast:         x.FailFast,
		NoWorkerLimit:    x.NoWorkerLimit,
		Errors:           append([]string(nil), x.Errors...),
		Notices:          toolx.MapSlice(x.Notices, fromDomainAllProcessDefinitionsPurgeNotice),
		Outcome:          AllProcessDefinitionsPurgeOutcome(x.Outcome),
	}
}

// fromDomainAllProcessDefinitionsPurgeNotice maps structured workflow notices to public output.
func fromDomainAllProcessDefinitionsPurgeNotice(x d.AllProcessDefinitionsPurgeWorkflowNotice) AllProcessDefinitionsPurgeNotice {
	return AllProcessDefinitionsPurgeNotice{
		Code:     x.Code,
		Severity: x.Severity,
		Message:  x.Message,
		Details:  toolx.CopyMap(x.Details),
	}
}

// toDomainIncidentFilter maps public incident selection flags to the service filter.
func toDomainIncidentFilter(x incident.Filter) d.IncidentFilter {
	return d.IncidentFilter{
		Keys:                   append([]string(nil), x.Keys...),
		State:                  x.State,
		ErrorType:              x.ErrorType,
		ErrorMessage:           x.ErrorMessage,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
		FlowNodeId:             x.FlowNodeId,
		FlowNodeInstanceKey:    x.FlowNodeInstanceKey,
		CreationTimeAfter:      x.CreationTimeAfter,
		CreationTimeBefore:     x.CreationTimeBefore,
	}
}

// fromDomainIncidentFilter maps service incident filters to the public model.
func fromDomainIncidentFilter(x d.IncidentFilter) incident.Filter {
	return incident.Filter{
		Keys:                   append([]string(nil), x.Keys...),
		State:                  x.State,
		ErrorType:              x.ErrorType,
		ErrorMessage:           x.ErrorMessage,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
		FlowNodeId:             x.FlowNodeId,
		FlowNodeInstanceKey:    x.FlowNodeInstanceKey,
		CreationTimeAfter:      x.CreationTimeAfter,
		CreationTimeBefore:     x.CreationTimeBefore,
	}
}

// fromDomainIncidentDetail maps one service incident detail to public output.
func fromDomainIncidentDetail(x d.ProcessInstanceIncidentDetail) incident.ProcessInstanceIncidentDetail {
	return incident.ProcessInstanceIncidentDetail{
		IncidentKey:            x.IncidentKey,
		CreationTime:           x.CreationTime,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		TenantId:               x.TenantId,
		State:                  x.State,
		ErrorType:              x.ErrorType,
		ErrorMessage:           x.ErrorMessage,
		FlowNodeId:             x.FlowNodeId,
		FlowNodeInstanceKey:    x.FlowNodeInstanceKey,
		JobKey:                 x.JobKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
	}
}

func toDomainProcessInstanceFilter(x process.ProcessInstanceFilter) d.ProcessInstanceFilter {
	return d.ProcessInstanceFilter{
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessVersion:       x.ProcessVersion,
		ProcessVersionTag:    x.ProcessVersionTag,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		StartDateAfter:       x.StartDateAfter,
		StartDateBefore:      x.StartDateBefore,
		EndDateAfter:         x.EndDateAfter,
		EndDateBefore:        x.EndDateBefore,
		State:                d.State(x.State),
		ParentKey:            x.ParentKey,
		HasParent:            x.HasParent,
		HasIncident:          x.HasIncident,
	}
}

func fromDomainProcessInstanceFilter(x d.ProcessInstanceFilter) process.ProcessInstanceFilter {
	return process.ProcessInstanceFilter{
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessVersion:       x.ProcessVersion,
		ProcessVersionTag:    x.ProcessVersionTag,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		StartDateAfter:       x.StartDateAfter,
		StartDateBefore:      x.StartDateBefore,
		EndDateAfter:         x.EndDateAfter,
		EndDateBefore:        x.EndDateBefore,
		State:                process.State(x.State),
		ParentKey:            x.ParentKey,
		HasParent:            x.HasParent,
		HasIncident:          x.HasIncident,
	}
}

func fromDomainDryRunPIKeyExpansion(x d.DryRunPIKeyExpansion) process.DryRunPIKeyExpansion {
	return process.DryRunPIKeyExpansion{
		Roots:                      append([]string(nil), x.Roots...),
		Collected:                  append([]string(nil), x.Collected...),
		SelectedFinalState:         toolx.MapSlice(x.SelectedFinalState, fromDomainProcessInstance),
		RequiresCancelBeforeDelete: toolx.MapSlice(x.RequiresCancelBeforeDelete, fromDomainProcessInstance),
		MissingAncestors:           toolx.MapSlice(x.MissingAncestors, fromDomainMissingAncestor),
		Warning:                    x.Warning,
		Outcome:                    process.TraversalOutcome(x.Outcome),
	}
}

func fromDomainProcessInstance(x d.ProcessInstance) process.ProcessInstance {
	return process.ProcessInstance{
		BpmnProcessId:             x.BpmnProcessId,
		EndDate:                   x.EndDate,
		Incident:                  x.Incident,
		Key:                       x.Key,
		ParentFlowNodeInstanceKey: x.ParentFlowNodeInstanceKey,
		ParentKey:                 x.ParentKey,
		ProcessDefinitionKey:      x.ProcessDefinitionKey,
		RootProcessInstanceKey:    x.RootProcessInstanceKey,
		ProcessVersion:            x.ProcessVersion,
		ProcessVersionTag:         x.ProcessVersionTag,
		StartDate:                 x.StartDate,
		State:                     process.State(x.State),
		TenantId:                  x.TenantId,
		Variables:                 toolx.CopyMap(x.Variables),
	}
}

func fromDomainMissingAncestor(x d.MissingAncestor) process.MissingAncestor {
	return process.MissingAncestor{Key: x.Key, StartKey: x.StartKey}
}

func toDomainProcessDefinitionSelection(x ProcessDefinitionSelection) d.ProcessDefinitionFilter {
	return d.ProcessDefinitionFilter{
		Key:               x.Key,
		BpmnProcessId:     x.BpmnProcessId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
		IsLatestVersion:   x.LatestOnly,
	}
}

func fromDomainProcessDefinitionSelection(x d.ProcessDefinitionFilter) ProcessDefinitionSelection {
	return ProcessDefinitionSelection{
		Key:               x.Key,
		BpmnProcessId:     x.BpmnProcessId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
		LatestOnly:        x.IsLatestVersion,
	}
}

func fromDomainProcessDefinition(x d.ProcessDefinition) process.ProcessDefinition {
	return process.ProcessDefinition{
		BpmnProcessId:     x.BpmnProcessId,
		Key:               x.Key,
		Name:              x.Name,
		TenantId:          x.TenantId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
		Statistics:        fromDomainProcessDefinitionStatistics(x.Statistics),
	}
}

func fromDomainProcessDefinitionStatistics(x *d.ProcessDefinitionStatistics) *process.ProcessDefinitionStatistics {
	if x == nil {
		return nil
	}
	return &process.ProcessDefinitionStatistics{
		Active:                 x.Active,
		Canceled:               x.Canceled,
		Completed:              x.Completed,
		Incidents:              x.Incidents,
		IncidentCountSupported: x.IncidentCountSupported,
	}
}

func fromDomainDeleteProcessDefinitionPlanItem(x d.DeleteProcessDefinitionPlanItem) resource.DeleteProcessDefinitionPlanItem {
	return resource.DeleteProcessDefinitionPlanItem{
		Key:                        x.Key,
		BpmnProcessId:              x.BpmnProcessId,
		ProcessVersion:             x.ProcessVersion,
		ProcessVersionTag:          x.ProcessVersionTag,
		TenantId:                   x.TenantId,
		ActiveProcessInstanceCount: x.ActiveProcessInstanceCount,
		ActiveProcessInstanceKeys:  append([]string(nil), x.ActiveProcessInstanceKeys...),
		CancellationPlan:           fromDomainDryRunPIKeyExpansion(x.CancellationPlan),
		Warnings:                   append([]string(nil), x.Warnings...),
	}
}

func fromDomainResourceDeleteReport(x d.ResourceDeleteResponse) resource.DeleteReport {
	return resource.DeleteReport{
		Key:               x.Key,
		Ok:                x.Ok,
		StatusCode:        x.StatusCode,
		Status:            x.Status,
		BatchOperationKey: x.BatchOperationKey,
		BatchState:        x.BatchState,
		DeleteHistory:     x.DeleteHistory,
	}
}

func fromDomainDeleteReport(x d.Reporter) process.DeleteReport {
	return process.DeleteReport{
		Key:        x.Key,
		Ok:         x.Ok,
		StatusCode: x.StatusCode,
		Status:     x.Status,
	}
}
