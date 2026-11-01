// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// copyRepairVariables snapshots nested JSON containers; invalid input is retained for authoritative service validation.
func copyRepairVariables(x map[string]any) map[string]any {
	copy, err := toolx.CopyJSONMap(x)
	if err != nil {
		return toolx.CopyMap(x)
	}
	return copy
}

// toDomainRepairFrozenPlan maps copied invocation-local evidence without discovering or authorizing targets.
func toDomainRepairFrozenPlan(x RepairFrozenPlan) d.OpsRepairFrozenPlan {
	return d.OpsRepairFrozenPlan{Selection: toDomainRepairSelection(x.Selection), Tenant: x.Tenant,
		Incidents: d.ProcessInstancesIncidentPlan{Keys: append([]string(nil), x.Incidents.Keys...), CompletedKeys: append([]string(nil), x.Incidents.CompletedKeys...), Incidents: toolx.MapSlice(x.Incidents.Incidents, toDomainRepairIncident)},
		Variables: copyRepairVariables(x.Variables), RequestedRetries: toolx.CopyPtr(x.RequestedRetries), RequestedJobTimeout: x.RequestedJobTimeout, DirectIncidentsOnly: x.DirectIncidentsOnly}
}

// fromDomainRepairFrozenPlan preserves the reviewed criteria and exact action values for confirmation handoff.
func fromDomainRepairFrozenPlan(x d.OpsRepairFrozenPlan) RepairFrozenPlan {
	return RepairFrozenPlan{Selection: fromDomainRepairSelection(x.Selection), Tenant: x.Tenant,
		Incidents: incident.ProcessInstancesIncidentPlan{Keys: append([]string(nil), x.Incidents.Keys...), CompletedKeys: append([]string(nil), x.Incidents.CompletedKeys...), Incidents: toolx.MapSlice(x.Incidents.Incidents, fromDomainIncidentDetail)},
		Variables: copyRepairVariables(x.Variables), RequestedRetries: toolx.CopyPtr(x.RequestedRetries), RequestedJobTimeout: x.RequestedJobTimeout, DirectIncidentsOnly: x.DirectIncidentsOnly}
}

// toDomainRepairSelection copies selection evidence without inferring completion or changing filters.
func toDomainRepairSelection(x process.ProcessInstanceSelection) d.ProcessInstanceSelection {
	r := x.Request.SearchRequest
	return d.ProcessInstanceSelection{Request: d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCardinality(x.Request.Cardinality), SearchRequest: d.ProcessInstanceSearchRequest{
		Filter: toDomainProcessInstanceFilter(r.Filter), Page: d.ProcessInstancePageRequest(r.Page), Limit: r.Limit, LocalFilters: d.ProcessInstanceSearchLocalFilters(r.LocalFilters), DirectIncidentIndex: r.DirectIncidentIndex, ReportedTotalAllowed: r.ReportedTotalAllowed,
		DirectIncidentFilter: d.IncidentFilter{State: r.DirectIncidentFilter.State, ErrorType: r.DirectIncidentFilter.ErrorType, ErrorMessage: r.DirectIncidentFilter.ErrorMessage, ProcessDefinitionKey: r.DirectIncidentFilter.ProcessDefinitionKey, ProcessDefinitionId: r.DirectIncidentFilter.ProcessDefinitionId},
	}}, Origin: x.Origin, Items: toolx.MapSlice(x.Items, toDomainRepairProcessInstance), Keys: append([]string(nil), x.Keys...), Completion: d.ProcessInstanceSearchCompletion(x.Completion), Pages: x.Pages, IgnoreTenant: x.IgnoreTenant,
		TenantEvidence: d.TenantEvidence{ResolvedTenantIDs: append([]string(nil), x.TenantEvidence.ResolvedTenantIDs...), UnknownTargetCount: x.TenantEvidence.UnknownTargetCount, TargetCount: x.TenantEvidence.TargetCount, Targets: toolx.MapSlice(x.TenantEvidence.Targets, func(t process.TenantEvidenceTarget) d.TenantEvidenceTarget {
			return d.TenantEvidenceTarget{Key: t.Key, TenantID: t.TenantID}
		})}}
}

// fromDomainRepairSelection retains all search controls and each resource's own observed identity.
func fromDomainRepairSelection(x d.ProcessInstanceSelection) process.ProcessInstanceSelection {
	r := x.Request.SearchRequest
	return process.ProcessInstanceSelection{Request: process.ProcessInstanceSelectionRequest{Cardinality: process.ProcessInstanceSelectionCardinality(x.Request.Cardinality), SearchRequest: process.ProcessInstanceSearchRequest{
		Filter: fromDomainProcessInstanceFilter(r.Filter), Page: process.ProcessInstancePageRequest(r.Page), Limit: r.Limit, LocalFilters: process.ProcessInstanceSearchLocalFilters(r.LocalFilters), DirectIncidentIndex: r.DirectIncidentIndex, ReportedTotalAllowed: r.ReportedTotalAllowed,
		DirectIncidentFilter: process.ProcessInstanceIncidentSearchFilter{State: r.DirectIncidentFilter.State, ErrorType: r.DirectIncidentFilter.ErrorType, ErrorMessage: r.DirectIncidentFilter.ErrorMessage, ProcessDefinitionKey: r.DirectIncidentFilter.ProcessDefinitionKey, ProcessDefinitionId: r.DirectIncidentFilter.ProcessDefinitionId},
	}}, Origin: x.Origin, Items: toolx.MapSlice(x.Items, fromDomainProcessInstance), Keys: append([]string(nil), x.Keys...), Completion: process.ProcessInstanceSearchCompletion(x.Completion), Pages: x.Pages, IgnoreTenant: x.IgnoreTenant, TenantEvidence: fromDomainOpsTenantEvidence(x.TenantEvidence)}
}

// toDomainRepairProcessInstance copies observations without replacing their IDs with criteria.
func toDomainRepairProcessInstance(x process.ProcessInstance) d.ProcessInstance {
	return d.ProcessInstance{BusinessId: toolx.CopyPtr(x.BusinessId), BpmnProcessId: x.BpmnProcessId, EndDate: x.EndDate, Incident: x.Incident, Key: x.Key, ParentElementInstanceKey: x.ParentElementInstanceKey, ParentKey: x.ParentKey, ProcessDefinitionKey: x.ProcessDefinitionKey, RootProcessInstanceKey: x.RootProcessInstanceKey, ProcessVersion: x.ProcessVersion, ProcessVersionTag: x.ProcessVersionTag, StartDate: x.StartDate, SuspendedDate: x.SuspendedDate, State: d.State(x.State), TenantId: x.TenantId, Variables: copyRepairVariables(x.Variables)}
}

// toDomainRepairIncident copies the reviewed job and variable-owner identities.
func toDomainRepairIncident(x incident.ProcessInstanceIncidentDetail) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{IncidentKey: x.IncidentKey, CreationTime: x.CreationTime, ProcessInstanceKey: x.ProcessInstanceKey, TenantId: x.TenantId, State: x.State, ErrorType: x.ErrorType, ErrorMessage: x.ErrorMessage, ElementId: x.ElementId, ElementInstanceKey: x.ElementInstanceKey, JobKey: x.JobKey, RootProcessInstanceKey: x.RootProcessInstanceKey, ProcessDefinitionKey: x.ProcessDefinitionKey, ProcessDefinitionId: x.ProcessDefinitionId}
}
