package process

import (
	d "github.com/grafvonb/c8volt/internal/domain"
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

func fromDomainProcessInstancePage(x d.ProcessInstancePage) ProcessInstancePage {
	return ProcessInstancePage{
		Request: ProcessInstancePageRequest{
			From: x.Request.From,
			Size: x.Request.Size,
		},
		OverflowState: ProcessInstanceOverflowState(x.OverflowState),
		Items:         toolx.MapSlice(x.Items, fromDomainProcessInstance),
	}
}

func fromDomainProcessInstanceMap(xs map[string]d.ProcessInstance) map[string]ProcessInstance {
	return toolx.MapMap(xs, fromDomainProcessInstance)
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
		ProcessVersion:            x.ProcessVersion,
		ProcessVersionTag:         x.ProcessVersionTag,
		StartDate:                 x.StartDate,
		State:                     d.State(x.State),
		TenantId:                  x.TenantId,
		Variables:                 toolx.CopyMap(x.Variables),
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

func toDomainProcessInstancePageRequest(x ProcessInstancePageRequest) d.ProcessInstancePageRequest {
	return d.ProcessInstancePageRequest{
		From: x.From,
		Size: x.Size,
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
