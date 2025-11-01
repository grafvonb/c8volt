package v87

import (
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromProcessInstanceResponse(r operatev87.ProcessInstance) d.ProcessInstance {
	return d.ProcessInstance{
		BpmnProcessId:             toolx.Deref(r.BpmnProcessId, ""),
		EndDate:                   toolx.Deref(r.EndDate, ""),
		Incident:                  toolx.Deref(r.Incident, false),
		Key:                       toolx.Int64PtrToString(r.Key),
		ParentFlowNodeInstanceKey: toolx.Int64PtrToString(r.ParentFlowNodeInstanceKey),
		ParentKey:                 toolx.Int64PtrToString(r.ParentKey),
		ProcessDefinitionKey:      toolx.Int64PtrToString(r.ProcessDefinitionKey),
		ProcessVersion:            toolx.Deref(r.ProcessVersion, int32(0)),
		ProcessVersionTag:         toolx.Deref(r.ProcessVersionTag, ""),
		StartDate:                 toolx.Deref(r.StartDate, ""),
		State:                     d.State(*r.State),
		TenantId:                  toolx.Deref(r.TenantId, ""),
	}
}
