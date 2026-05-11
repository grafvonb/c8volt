// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
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
		Errors:           append([]string(nil), x.Errors...),
		Outcome:          OrphanPurgeOutcome(x.Outcome),
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
