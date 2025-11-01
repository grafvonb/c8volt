package v88

import (
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	operatev88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

//nolint:unused
func fromProcessInstanceResponse(r operatev88.ProcessInstance) d.ProcessInstance {
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

func fromProcessInstanceResult(r camundav88.ProcessInstanceResult) d.ProcessInstance {
	return d.ProcessInstance{
		BpmnProcessId:             r.ProcessDefinitionId,
		EndDate:                   formatTimePtr(r.EndDate),
		Incident:                  r.HasIncident,
		Key:                       r.ProcessInstanceKey,
		ParentFlowNodeInstanceKey: toolx.Deref(r.ParentElementInstanceKey, ""),
		ParentKey:                 toolx.Deref(r.ParentProcessInstanceKey, ""),
		ProcessDefinitionKey:      r.ProcessDefinitionKey,
		ProcessVersion:            r.ProcessDefinitionVersion,
		ProcessVersionTag:         toolx.Deref(r.ProcessDefinitionVersionTag, ""),
		StartDate:                 formatTime(r.StartDate),
		State:                     d.State(r.State),
		TenantId:                  r.TenantId,
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func formatTimePtr(p *time.Time) string {
	if p == nil {
		return ""
	}
	return formatTime(*p)
}
