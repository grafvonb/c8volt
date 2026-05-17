// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDomainProcessDefinition(x d.ProcessDefinition) ProcessDefinition {
	return ProcessDefinition{
		BpmnProcessId:     x.BpmnProcessId,
		Key:               x.Key,
		Name:              x.Name,
		TenantId:          x.TenantId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
		Statistics:        toolx.MapPtr(x.Statistics, fromProcessDefinitionStatistics),
	}
}

func fromProcessDefinitionStatistics(r d.ProcessDefinitionStatistics) ProcessDefinitionStatistics {
	return ProcessDefinitionStatistics{
		Active:                 r.Active,
		Canceled:               r.Canceled,
		Completed:              r.Completed,
		Incidents:              r.Incidents,
		IncidentCountSupported: r.IncidentCountSupported,
	}
}

func fromDomainProcessDefinitions(xs []d.ProcessDefinition) ProcessDefinitions {
	items := toolx.MapSlice(xs, fromDomainProcessDefinition)
	return ProcessDefinitions{
		Total: int32(len(items)),
		Items: items,
	}
}

func fromDomainProcessInstance(x d.ProcessInstance) ProcessInstance {
	return ProcessInstance{
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
		State:                     State(x.State),
		TenantId:                  x.TenantId,
		Variables:                 toolx.CopyMap(x.Variables),
	}
}

func fromDomainProcessInstanceCreation(x d.ProcessInstanceCreation) ProcessInstance {
	return ProcessInstance{
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		ProcessVersion:       x.ProcessDefinitionVersion,
		Variables:            toolx.CopyMap(x.Variables),
		TenantId:             x.TenantId,
		StartDate:            x.StartDate,
	}
}

func fromDomainProcessInstances(xs []d.ProcessInstance) ProcessInstances {
	items := toolx.MapSlice(xs, fromDomainProcessInstance)
	return ProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}
}

func fromDomainProcessInstanceIncidentDetail(x d.ProcessInstanceIncidentDetail) ProcessInstanceIncidentDetail {
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

func fromDomainProcessInstanceIncidentDetails(xs []d.ProcessInstanceIncidentDetail) []ProcessInstanceIncidentDetail {
	return toolx.MapSlice(xs, fromDomainProcessInstanceIncidentDetail)
}

func fromDomainProcessInstanceVariable(x d.ProcessInstanceVariable) ProcessInstanceVariable {
	return ProcessInstanceVariable{
		Name:               x.Name,
		Value:              x.Value,
		VariableKey:        x.VariableKey,
		ProcessInstanceKey: x.ProcessInstanceKey,
		ScopeKey:           x.ScopeKey,
		TenantId:           x.TenantId,
		APITruncated:       x.APITruncated,
	}
}

func fromDomainProcessInstanceVariables(xs []d.ProcessInstanceVariable) []ProcessInstanceVariable {
	return toolx.MapSlice(xs, fromDomainProcessInstanceVariable)
}

// fromDomainIncidentEnrichedProcessInstance maps one service-enriched process instance into the public facade model.
func fromDomainIncidentEnrichedProcessInstance(x d.IncidentEnrichedProcessInstance) IncidentEnrichedProcessInstance {
	return IncidentEnrichedProcessInstance{
		Item:      fromDomainProcessInstance(x.Item),
		Incidents: fromDomainProcessInstanceIncidentDetails(x.Incidents),
	}
}

// fromDomainIncidentEnrichedProcessInstances maps service-enriched process instances into the public facade model.
func fromDomainIncidentEnrichedProcessInstances(x d.IncidentEnrichedProcessInstances) IncidentEnrichedProcessInstances {
	return IncidentEnrichedProcessInstances{
		Total: x.Total,
		Items: toolx.MapSlice(x.Items, fromDomainIncidentEnrichedProcessInstance),
	}
}

// fromDomainVariableEnrichedProcessInstance maps one service-enriched process instance and its variables into the public facade model.
func fromDomainVariableEnrichedProcessInstance(x d.VariableEnrichedProcessInstance) VariableEnrichedProcessInstance {
	return VariableEnrichedProcessInstance{
		Item:      fromDomainProcessInstance(x.Item),
		Variables: fromDomainProcessInstanceVariables(x.Variables),
	}
}

// fromDomainVariableEnrichedProcessInstances maps service-enriched variables into the public facade model.
func fromDomainVariableEnrichedProcessInstances(x d.VariableEnrichedProcessInstances) VariableEnrichedProcessInstances {
	return VariableEnrichedProcessInstances{
		Total: x.Total,
		Items: toolx.MapSlice(x.Items, fromDomainVariableEnrichedProcessInstance),
	}
}

// fromDomainIncidentEnrichedTraversalItem maps one service-enriched traversal item into the public facade model.
func fromDomainIncidentEnrichedTraversalItem(x d.IncidentEnrichedTraversalItem) IncidentEnrichedTraversalItem {
	return IncidentEnrichedTraversalItem{
		Item:      fromDomainProcessInstance(x.Item),
		Incidents: fromDomainProcessInstanceIncidentDetails(x.Incidents),
	}
}

// fromDomainIncidentEnrichedTraversalResult maps service-enriched traversal output into the public facade model.
func fromDomainIncidentEnrichedTraversalResult(x d.IncidentEnrichedTraversalResult) IncidentEnrichedTraversalResult {
	return IncidentEnrichedTraversalResult{
		Mode:             TraversalMode(x.Mode),
		Outcome:          TraversalOutcome(x.Outcome),
		StartKey:         x.StartKey,
		RootKey:          x.RootKey,
		Keys:             append([]string(nil), x.Keys...),
		Edges:            x.Edges,
		Items:            toolx.MapSlice(x.Items, fromDomainIncidentEnrichedTraversalItem),
		MissingAncestors: toolx.MapSlice(x.MissingAncestors, fromDomainMissingAncestor),
		Warning:          x.Warning,
	}
}

// fromDomainMissingAncestor maps one domain missing-ancestor marker into the public facade model.
func fromDomainMissingAncestor(item d.MissingAncestor) MissingAncestor {
	return MissingAncestor{Key: item.Key, StartKey: item.StartKey}
}

func fromDomainProcessInstanceVariableUpdateResult(x d.ProcessInstanceVariableUpdateResult) ProcessInstanceVariableUpdateResult {
	return ProcessInstanceVariableUpdateResult{
		Key:                x.Key,
		Status:             ProcessInstanceVariableUpdateStatus(x.Status),
		MutationAccepted:   x.MutationAccepted,
		ConfirmationStatus: x.ConfirmationStatus,
		StatusCode:         x.StatusCode,
		Message:            x.Message,
		Error:              x.Error,
		Variables:          toolx.CopyMap(x.Variables),
	}
}

func fromDomainProcessInstanceVariableUpdateResults(x d.ProcessInstanceVariableUpdateResults) ProcessInstanceVariableUpdateResults {
	return ProcessInstanceVariableUpdateResults{
		Items: toolx.MapSlice(x.Items, fromDomainProcessInstanceVariableUpdateResult),
	}
}

func fromDomainProcessInstanceVariableUpdateResponse(x d.ProcessInstanceVariableUpdateResponse, variables map[string]any) ProcessInstanceVariableUpdateResult {
	status := ProcessInstanceVariableUpdateStatusSubmitted
	if !x.Ok {
		status = ProcessInstanceVariableUpdateStatusMutationFailed
	}
	return ProcessInstanceVariableUpdateResult{
		Key:              x.Key,
		Status:           status,
		MutationAccepted: x.Ok,
		StatusCode:       x.StatusCode,
		Message:          x.Status,
		Variables:        toolx.CopyMap(variables),
	}
}

func toDomainProcessInstanceVariableUpdateRequest(x ProcessInstanceVariableUpdateRequest) d.ProcessInstanceVariableUpdateRequest {
	return d.ProcessInstanceVariableUpdateRequest{
		Key:       x.Key,
		Variables: toolx.CopyMap(x.Variables),
	}
}

func fromDomainReporter(x d.Reporter) Reporter {
	return Reporter{
		Key:        x.Key,
		Ok:         x.Ok,
		StatusCode: x.StatusCode,
		Status:     x.Status,
	}
}

func fromDomainCancelReports(xs []d.Reporter) CancelReports {
	return CancelReports{Items: toolx.MapSlice(xs, func(x d.Reporter) CancelReport { return fromDomainReporter(x) })}
}

func fromDomainDeleteReports(xs []d.Reporter) DeleteReports {
	return DeleteReports{Items: toolx.MapSlice(xs, func(x d.Reporter) DeleteReport { return fromDomainReporter(x) })}
}

func fromDomainDryRunPIKeyExpansion(x d.DryRunPIKeyExpansion) DryRunPIKeyExpansion {
	return DryRunPIKeyExpansion{
		Roots:                      append([]string(nil), x.Roots...),
		Collected:                  append([]string(nil), x.Collected...),
		SelectedFinalState:         toolx.MapSlice(x.SelectedFinalState, fromDomainProcessInstance),
		RequiresCancelBeforeDelete: toolx.MapSlice(x.RequiresCancelBeforeDelete, fromDomainProcessInstance),
		MissingAncestors: toolx.MapSlice(x.MissingAncestors, func(item d.MissingAncestor) MissingAncestor {
			return MissingAncestor{Key: item.Key, StartKey: item.StartKey}
		}),
		Warning: x.Warning,
		Outcome: TraversalOutcome(x.Outcome),
	}
}

func fromDomainProcessInstancePage(x d.ProcessInstancePage) ProcessInstancePage {
	return ProcessInstancePage{
		Request: ProcessInstancePageRequest{
			From:  x.Request.From,
			Size:  x.Request.Size,
			After: x.Request.After,
		},
		OverflowState: ProcessInstanceOverflowState(x.OverflowState),
		ReportedTotal: toolx.MapPtr(x.ReportedTotal, fromDomainProcessInstanceReportedTotal),
		EndCursor:     x.EndCursor,
		Items:         toolx.MapSlice(x.Items, fromDomainProcessInstance),
	}
}

func fromDomainOrphanDiscovery(x pisvc.OrphanDiscovery) OrphanDiscovery {
	return OrphanDiscovery{
		Filter: fromDomainProcessInstanceFilter(x.Filter),
		Items:  toolx.MapSlice(x.Items, fromDomainProcessInstance),
		Keys:   append([]string(nil), x.Keys...),
	}
}

func fromDomainProcessInstanceReportedTotal(x d.ProcessInstanceReportedTotal) ProcessInstanceReportedTotal {
	return ProcessInstanceReportedTotal{
		Count: x.Count,
		Kind:  ProcessInstanceReportedTotalKind(x.Kind),
	}
}

func fromDomainProcessInstanceMap(xs map[string]d.ProcessInstance) map[string]ProcessInstance {
	return toolx.MapMap(xs, fromDomainProcessInstance)
}

// toDomainProcessInstanceMap maps public process-instance maps into domain values for service workflows.
func toDomainProcessInstanceMap(xs map[string]ProcessInstance) map[string]d.ProcessInstance {
	return toolx.MapMap(xs, toDomainProcessInstance)
}

// toServiceTraversalResult maps public traversal output back into service-layer traversal input for enrichment.
func toServiceTraversalResult(in TraversalResult) pitraversal.Result {
	return pitraversal.Result{
		Mode:             pitraversal.Mode(in.Mode),
		StartKey:         in.StartKey,
		RootKey:          in.RootKey,
		Keys:             append([]string(nil), in.Keys...),
		Edges:            in.Edges,
		Chain:            toDomainProcessInstanceMap(in.Chain),
		MissingAncestors: toServiceMissingAncestors(in.MissingAncestors),
		Warning:          in.Warning,
		Outcome:          pitraversal.Outcome(in.Outcome),
	}
}

// toServiceMissingAncestors maps public missing-ancestor markers into traversal package values.
func toServiceMissingAncestors(in []MissingAncestor) []pitraversal.MissingAncestor {
	if len(in) == 0 {
		return nil
	}
	out := make([]pitraversal.MissingAncestor, len(in))
	for i, item := range in {
		out[i] = pitraversal.MissingAncestor{
			Key:      item.Key,
			StartKey: item.StartKey,
		}
	}
	return out
}

func fromDomainProcessInstanceExpectationResponse(x d.ProcessInstanceExpectationResponse) ProcessInstanceExpectationReport {
	return ProcessInstanceExpectationReport{
		Key:      x.Key,
		Ok:       x.Ok,
		State:    State(x.State),
		Incident: fromDomainIncidentExpectation(x.Incident),
		Status:   x.Status,
	}
}

func fromDomainProcessInstanceExpectationResponses(xs d.ProcessInstanceExpectationResponses) ProcessInstanceExpectationReports {
	return ProcessInstanceExpectationReports{
		Items: toolx.MapSlice(xs.Items, fromDomainProcessInstanceExpectationResponse),
	}
}

func fromDomainIncidentExpectation(x *bool) *IncidentExpectation {
	if x == nil {
		return nil
	}
	out := IncidentExpectation(*x)
	return &out
}

func toDomainProcessInstance(x ProcessInstance) d.ProcessInstance {
	return d.ProcessInstance{
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
		State:                     d.State(x.State),
		TenantId:                  x.TenantId,
		Variables:                 toolx.CopyMap(x.Variables),
	}
}

func toDomainProcessInstanceExpectationRequest(x ProcessInstanceExpectationRequest) d.ProcessInstanceExpectationRequest {
	return d.ProcessInstanceExpectationRequest{
		States:   toolx.MapSlice(x.States, func(s State) d.State { return d.State(s) }),
		Incident: toDomainIncidentExpectation(x.Incident),
	}
}

func toDomainIncidentExpectation(x *IncidentExpectation) *bool {
	if x == nil {
		return nil
	}
	out := x.Bool()
	return &out
}

func toDomainProcessInstanceIncidentDetail(x ProcessInstanceIncidentDetail) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
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

func toDomainProcessDefinitionFilter(x ProcessDefinitionFilter) d.ProcessDefinitionFilter {
	return d.ProcessDefinitionFilter{
		Key:               x.Key,
		BpmnProcessId:     x.BpmnProcessId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
	}
}

func toDomainProcessInstanceFilter(x ProcessInstanceFilter) d.ProcessInstanceFilter {
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

func fromDomainProcessInstanceFilter(x d.ProcessInstanceFilter) ProcessInstanceFilter {
	return ProcessInstanceFilter{
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessVersion:       x.ProcessVersion,
		ProcessVersionTag:    x.ProcessVersionTag,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		StartDateAfter:       x.StartDateAfter,
		StartDateBefore:      x.StartDateBefore,
		EndDateAfter:         x.EndDateAfter,
		EndDateBefore:        x.EndDateBefore,
		State:                State(x.State),
		ParentKey:            x.ParentKey,
		HasParent:            x.HasParent,
		HasIncident:          x.HasIncident,
	}
}

func toDomainProcessInstancePageRequest(x ProcessInstancePageRequest) d.ProcessInstancePageRequest {
	return d.ProcessInstancePageRequest{
		From:  x.From,
		Size:  x.Size,
		After: x.After,
	}
}

func toProcessInstanceData(x ProcessInstanceData) d.ProcessInstanceData {
	return d.ProcessInstanceData{
		BpmnProcessId:               x.BpmnProcessId,
		ProcessDefinitionSpecificId: x.ProcessDefinitionSpecificId,
		ProcessDefinitionVersion:    x.ProcessDefinitionVersion,
		Variables:                   toolx.CopyMap(x.Variables),
		TenantId:                    x.TenantId,
	}
}
