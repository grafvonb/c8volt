package resource

import "github.com/grafvonb/c8volt/c8volt/process"

type ProcessDefinitionDeployment struct {
	Key               string `json:"key"`
	DefinitionId      string `json:"processDefinitionId,omitempty"`
	DefinitionKey     string `json:"processDefinitionKey,omitempty"`
	DefinitionVersion int32  `json:"processDefinitionVersion,omitempty"`
	ResourceName      string `json:"resourceName,omitempty"`
	TenantId          string `json:"tenantId,omitempty"`
}

type DeploymentUnitData struct {
	Name        string // filename for multipart
	ContentType string // e.g. application/xml
	Data        []byte
}

type DeleteReport = process.Reporter

type DeleteReports struct {
	Items []DeleteReport `json:"items,omitempty"`
}

func (c DeleteReports) Totals() (total int, oks int, noks int) {
	return process.TotalsOf(c.Items)
}
