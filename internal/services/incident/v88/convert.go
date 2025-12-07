package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromIncidentResult(r camundav88.IncidentResult) d.Incident {
	return d.Incident{
		Key:                  toolx.Deref(r.IncidentKey, ""),
		CreationTime:         r.CreationTime.String(),
		ElementId:            toolx.Deref(r.ElementId, ""),
		ElementInstanceKey:   toolx.Deref(r.ElementInstanceKey, ""),
		ErrorMessage:         toolx.Deref(r.ErrorMessage, ""),
		ErrorType:            d.IncidentErrorType(*r.ErrorType),
		ProcessDefinitionId:  toolx.Deref(r.ProcessDefinitionId, ""),
		ProcessDefinitionKey: toolx.Deref(r.ProcessDefinitionKey, ""),
		ProcessInstanceKey:   toolx.Deref(r.ProcessInstanceKey, ""),
		State:                d.State(*r.State),
		TenantId:             toolx.Deref(r.TenantId, ""),
	}
}
