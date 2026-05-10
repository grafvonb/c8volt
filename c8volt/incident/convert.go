// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDomainIncident(x d.ProcessInstanceIncidentDetail) ProcessInstanceIncidentDetail {
	return ProcessInstanceIncidentDetail{
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

func fromDomainIncidents(xs []d.ProcessInstanceIncidentDetail) Incidents {
	items := toolx.MapSlice(xs, fromDomainIncident)
	return Incidents{Total: int32(len(items)), Items: items}
}

func fromDomainIncidentDetails(xs []d.ProcessInstanceIncidentDetail) []ProcessInstanceIncidentDetail {
	return toolx.MapSlice(xs, fromDomainIncident)
}

func toDomainFilter(x Filter) d.IncidentFilter {
	return d.IncidentFilter{
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

func toDomainPageRequest(x PageRequest) d.IncidentPageRequest {
	return d.IncidentPageRequest{From: x.From, Size: x.Size, After: x.After}
}

func fromDomainPage(x d.IncidentPage) Page {
	return Page{
		Request: PageRequest{
			From:  x.Request.From,
			Size:  x.Request.Size,
			After: x.Request.After,
		},
		OverflowState: OverflowState(x.OverflowState),
		ReportedTotal: toolx.MapPtr(x.ReportedTotal, func(t d.IncidentReportedTotal) ReportedTotal {
			return ReportedTotal{Count: t.Count, Kind: ReportedTotalKind(t.Kind)}
		}),
		EndCursor: x.EndCursor,
		Items:     fromDomainIncidentDetails(x.Items),
	}
}

func fromDomainResolutionResult(x d.IncidentResolutionResult) ResolutionResult {
	return ResolutionResult{
		IncidentKey:        x.IncidentKey,
		ProcessInstanceKey: x.ProcessInstanceKey,
		MutationAccepted:   x.MutationAccepted,
		Status:             ResolutionStatus(x.Status),
		ConfirmationStatus: x.ConfirmationStatus,
		StatusCode:         x.StatusCode,
		Message:            x.Message,
		Error:              x.Error,
		DryRun:             x.DryRun,
		MutationSubmitted:  x.MutationSubmitted,
		WouldResolve:       x.WouldResolve,
		IncidentState:      x.IncidentState,
		Incident:           toolx.MapPtr(x.Incident, fromDomainIncident),
	}
}

func fromDomainResolutionResults(x d.IncidentResolutionResults) ResolutionResults {
	return ResolutionResults{
		Operation:         ResolutionOperation(x.Operation),
		Items:             toolx.MapSlice(x.Items, fromDomainResolutionResult),
		Total:             x.Total,
		Submitted:         x.Submitted,
		Confirmed:         x.Confirmed,
		Skipped:           x.Skipped,
		Failed:            x.Failed,
		DryRun:            x.DryRun,
		MutationSubmitted: x.MutationSubmitted,
	}
}

func fromDomainProcessInstanceResolutionResult(x d.ProcessInstanceResolutionResult) ProcessInstanceResolutionResult {
	return ProcessInstanceResolutionResult{
		ProcessInstanceKey:    x.ProcessInstanceKey,
		AttemptedIncidentKeys: append([]string(nil), x.AttemptedIncidentKeys...),
		ResolvedIncidentKeys:  append([]string(nil), x.ResolvedIncidentKeys...),
		SkippedIncidentKeys:   append([]string(nil), x.SkippedIncidentKeys...),
		FailedIncidentKeys:    append([]string(nil), x.FailedIncidentKeys...),
		ConfirmationStatus:    x.ConfirmationStatus,
		Status:                ProcessInstanceResolutionStatus(x.Status),
		Error:                 x.Error,
		DryRun:                x.DryRun,
		MutationSubmitted:     x.MutationSubmitted,
		Incidents:             fromDomainIncidentDetails(x.Incidents),
	}
}

func fromDomainProcessInstanceResolutionResults(x d.ProcessInstanceResolutionResults) ProcessInstanceResolutionResults {
	return ProcessInstanceResolutionResults{
		Operation:         ResolutionOperation(x.Operation),
		Items:             toolx.MapSlice(x.Items, fromDomainProcessInstanceResolutionResult),
		Total:             x.Total,
		Submitted:         x.Submitted,
		Confirmed:         x.Confirmed,
		Skipped:           x.Skipped,
		Failed:            x.Failed,
		DryRun:            x.DryRun,
		MutationSubmitted: x.MutationSubmitted,
	}
}
