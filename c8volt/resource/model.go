package resource

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
