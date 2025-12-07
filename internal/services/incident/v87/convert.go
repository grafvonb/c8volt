package v87

import (
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromIncidentResult(r camundav87.IncidentResult) d.Incident {
	return d.Incident{
		CreationTime:        r.CreationTime.String(),
		ErrorMessage:        toolx.Deref(r.ErrorMessage, ""),
		ErrorType:           d.IncidentErrorType(*r.ErrorType),
		ElementId:           toolx.Deref(r.FlowNodeId, ""),
		ProcessDefinitionId: toolx.Deref(r.ProcessDefinitionId, ""),
		State:               d.State(*r.State),
		TenantId:            toolx.Deref(r.TenantId, ""),
	}
}
