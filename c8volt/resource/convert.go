// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromProcessDefinitionDeployment(dep d.Deployment) []ProcessDefinitionDeployment {
	if len(dep.Units) == 0 {
		return []ProcessDefinitionDeployment{{
			Key:      dep.Key,
			TenantId: dep.TenantId,
		}}
	}
	out := make([]ProcessDefinitionDeployment, 0, len(dep.Units))
	for _, u := range dep.Units {
		pd := u.ProcessDefinition
		out = append(out, ProcessDefinitionDeployment{
			Key:               dep.Key,
			DefinitionId:      pd.ProcessDefinitionId,
			DefinitionVersion: pd.ProcessDefinitionVersion,
			DefinitionKey:     pd.ProcessDefinitionKey,
			ResourceName:      pd.ResourceName,
			TenantId:          dep.TenantId,
		})
	}
	return out
}

func fromResource(resource d.Resource) Resource {
	return Resource{
		ID:         resource.ID,
		Key:        resource.Key,
		Name:       resource.Name,
		TenantId:   resource.TenantId,
		Version:    resource.Version,
		VersionTag: resource.VersionTag,
	}
}

func fromResourceDeleteResponse(key string, resp d.ResourceDeleteResponse, ok bool) DeleteReport {
	return DeleteReport{
		Key:               key,
		Ok:                ok,
		StatusCode:        resp.StatusCode,
		Status:            resp.Status,
		DeleteHistory:     resp.DeleteHistory,
		BatchOperationKey: resp.BatchOperationKey,
		BatchState:        resp.BatchState,
	}
}

// fromResourceDeleteResponses preserves the identity selected by the service, including unattempted slots.
func fromResourceDeleteResponses(responses []d.ResourceDeleteResponse) DeleteReports {
	items := make([]DeleteReport, 0, len(responses))
	for _, resp := range responses {
		items = append(items, fromResourceDeleteResponse(resp.Key, resp, resp.Ok))
	}
	return DeleteReports{Items: items}
}

func fromDomainDeleteProcessDefinitionPlan(plan d.DeleteProcessDefinitionPlan) DeleteProcessDefinitionPlan {
	return DeleteProcessDefinitionPlan{
		Items:                 toolx.MapSlice(plan.Items, fromDomainDeleteProcessDefinitionPlanItem),
		StateCheckSkipped:     plan.StateCheckSkipped,
		ProcessDefinitionKeys: append([]string(nil), plan.ProcessDefinitionKeys...),
		Warnings:              append([]string(nil), plan.Warnings...),
	}
}

// fromDomainDeleteProcessDefinitionPlanItem copies observed cleanup evidence across the public boundary.
func fromDomainDeleteProcessDefinitionPlanItem(item d.DeleteProcessDefinitionPlanItem) DeleteProcessDefinitionPlanItem {
	return DeleteProcessDefinitionPlanItem{
		Key:                            item.Key,
		BpmnProcessId:                  item.BpmnProcessId,
		ProcessVersion:                 item.ProcessVersion,
		ProcessVersionTag:              item.ProcessVersionTag,
		TenantId:                       item.TenantId,
		ActiveProcessInstanceCount:     item.ActiveProcessInstanceCount,
		ActiveProcessInstanceKeys:      append([]string(nil), item.ActiveProcessInstanceKeys...),
		SuspendedProcessInstanceCount:  item.SuspendedProcessInstanceCount,
		SuspendedCountSupported:        item.SuspendedCountSupported,
		UnfinishedProcessInstanceCount: item.UnfinishedProcessInstanceCount,
		UnfinishedProcessInstanceKeys:  append([]string(nil), item.UnfinishedProcessInstanceKeys...),
		CancellationPlan:               fromDomainDryRunPIKeyExpansion(item.CancellationPlan),
		Warnings:                       append([]string(nil), item.Warnings...),
	}
}

func fromDomainDryRunPIKeyExpansion(x d.DryRunPIKeyExpansion) process.DryRunPIKeyExpansion {
	return process.DryRunPIKeyExpansion{
		Roots:     append([]string(nil), x.Roots...),
		Collected: append([]string(nil), x.Collected...),
		TenantEvidence: process.TenantEvidence{
			ResolvedTenantIDs:  append([]string(nil), x.TenantEvidence.ResolvedTenantIDs...),
			UnknownTargetCount: x.TenantEvidence.UnknownTargetCount,
			TargetCount:        x.TenantEvidence.TargetCount,
			Targets: toolx.MapSlice(x.TenantEvidence.Targets, func(target d.TenantEvidenceTarget) process.TenantEvidenceTarget {
				return process.TenantEvidenceTarget{Key: target.Key, TenantID: target.TenantID}
			}),
		},
		SelectedFinalState:         toolx.MapSlice(x.SelectedFinalState, fromDomainProcessInstance),
		RequiresCancelBeforeDelete: toolx.MapSlice(x.RequiresCancelBeforeDelete, fromDomainProcessInstance),
		MissingAncestors: toolx.MapSlice(x.MissingAncestors, func(item d.MissingAncestor) process.MissingAncestor {
			return process.MissingAncestor{Key: item.Key, StartKey: item.StartKey}
		}),
		Warning: x.Warning,
		Outcome: process.TraversalOutcome(x.Outcome),
	}
}

// fromDomainProcessInstance copies observed impact identities across the facade boundary.
func fromDomainProcessInstance(x d.ProcessInstance) process.ProcessInstance {
	return process.ProcessInstance{
		BusinessId:               toolx.CopyPtr(x.BusinessId),
		BpmnProcessId:            x.BpmnProcessId,
		EndDate:                  x.EndDate,
		Incident:                 x.Incident,
		Key:                      x.Key,
		ParentElementInstanceKey: x.ParentElementInstanceKey,
		ParentKey:                x.ParentKey,
		ProcessDefinitionKey:     x.ProcessDefinitionKey,
		RootProcessInstanceKey:   x.RootProcessInstanceKey,
		ProcessVersion:           x.ProcessVersion,
		ProcessVersionTag:        x.ProcessVersionTag,
		StartDate:                x.StartDate,
		State:                    process.State(x.State),
		TenantId:                 x.TenantId,
		Variables:                toolx.CopyMap(x.Variables),
	}
}

func toDeploymentUnitDatas(units []DeploymentUnitData) []d.DeploymentUnitData {
	result := make([]d.DeploymentUnitData, len(units))
	for i, u := range units {
		result[i] = d.DeploymentUnitData{
			Name:        u.Name,
			ContentType: u.ContentType,
			Data:        u.Data,
		}
	}
	return result
}

// toDomainDeletePlanEvidenceItem maps only existing tenant observations; it never discovers missing targets.
func toDomainDeletePlanEvidenceItem(item DeleteProcessDefinitionPlanItem) d.DeleteProcessDefinitionPlanItem {
	e := item.CancellationPlan.TenantEvidence
	return d.DeleteProcessDefinitionPlanItem{Key: item.Key, TenantId: item.TenantId,
		CancellationPlan: d.DryRunPIKeyExpansion{TenantEvidence: d.TenantEvidence{
			ResolvedTenantIDs: append([]string(nil), e.ResolvedTenantIDs...), UnknownTargetCount: e.UnknownTargetCount, TargetCount: e.TargetCount,
			Targets: toolx.MapSlice(e.Targets, func(t process.TenantEvidenceTarget) d.TenantEvidenceTarget {
				return d.TenantEvidenceTarget{Key: t.Key, TenantID: t.TenantID}
			}),
		}},
	}
}

// fromDomainPlanTenantEvidence preserves the public empty-slice conventions and isolates returned observations.
func fromDomainPlanTenantEvidence(e d.TenantEvidence) process.TenantEvidence {
	var targets []process.TenantEvidenceTarget
	if len(e.Targets) > 0 {
		targets = toolx.MapSlice(e.Targets, func(t d.TenantEvidenceTarget) process.TenantEvidenceTarget {
			return process.TenantEvidenceTarget{Key: t.Key, TenantID: t.TenantID}
		})
	}
	return process.TenantEvidence{ResolvedTenantIDs: append([]string{}, e.ResolvedTenantIDs...),
		UnknownTargetCount: e.UnknownTargetCount, TargetCount: e.TargetCount, Targets: targets,
	}
}
