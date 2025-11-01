package v87

import (
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromProcessDefinitionResponse(r operatev87.ProcessDefinition) d.ProcessDefinition {
	return d.ProcessDefinition{
		BpmnProcessId: toolx.Deref(r.BpmnProcessId, ""),
		Key:           toolx.Int64PtrToString(r.Key),
		Name:          toolx.Deref(r.Name, ""),
		TenantId:      toolx.Deref(r.TenantId, ""),
		Version:       toolx.Deref(r.Version, int32(0)),
		VersionTag:    toolx.Deref(r.VersionTag, ""),
	}
}
