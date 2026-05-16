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

func fromResourceDeleteResponses(keys []string, responses []d.ResourceDeleteResponse) DeleteReports {
	items := make([]DeleteReport, 0, len(responses))
	for i, resp := range responses {
		key := ""
		if i < len(keys) {
			key = keys[i]
		}
		items = append(items, fromResourceDeleteResponse(key, resp, resp.Ok))
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

func fromDomainDeleteProcessDefinitionPlanItem(item d.DeleteProcessDefinitionPlanItem) DeleteProcessDefinitionPlanItem {
	return DeleteProcessDefinitionPlanItem{
		Key:                        item.Key,
		BpmnProcessId:              item.BpmnProcessId,
		ProcessVersion:             item.ProcessVersion,
		ProcessVersionTag:          item.ProcessVersionTag,
		ActiveProcessInstanceCount: item.ActiveProcessInstanceCount,
		ActiveProcessInstanceKeys:  append([]string(nil), item.ActiveProcessInstanceKeys...),
		CancellationPlan:           fromDomainDryRunPIKeyExpansion(item.CancellationPlan),
		Warnings:                   append([]string(nil), item.Warnings...),
	}
}

func fromDomainDryRunPIKeyExpansion(x d.DryRunPIKeyExpansion) process.DryRunPIKeyExpansion {
	return process.DryRunPIKeyExpansion{
		Roots:                      append([]string(nil), x.Roots...),
		Collected:                  append([]string(nil), x.Collected...),
		SelectedFinalState:         toolx.MapSlice(x.SelectedFinalState, fromDomainProcessInstance),
		RequiresCancelBeforeDelete: toolx.MapSlice(x.RequiresCancelBeforeDelete, fromDomainProcessInstance),
		MissingAncestors: toolx.MapSlice(x.MissingAncestors, func(item d.MissingAncestor) process.MissingAncestor {
			return process.MissingAncestor{Key: item.Key, StartKey: item.StartKey}
		}),
		Warning: x.Warning,
		Outcome: process.TraversalOutcome(x.Outcome),
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
