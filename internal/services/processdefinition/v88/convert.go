package v88

import (
	operatev88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromProcessDefinitionResponse(r operatev88.ProcessDefinition) d.ProcessDefinition {
	return d.ProcessDefinition{
		BpmnProcessId:     toolx.Deref(r.BpmnProcessId, ""),
		Key:               toolx.Int64PtrToString(r.Key),
		Name:              toolx.Deref(r.Name, ""),
		TenantId:          toolx.Deref(r.TenantId, ""),
		ProcessVersion:    toolx.Deref(r.Version, int32(0)),
		ProcessVersionTag: toolx.Deref(r.VersionTag, ""),
	}
}
