// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

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

func fromDomainDeleteReport(x d.Reporter) process.DeleteReport {
	return process.DeleteReport{
		Key:        x.Key,
		Ok:         x.Ok,
		StatusCode: x.StatusCode,
		Status:     x.Status,
	}
}
