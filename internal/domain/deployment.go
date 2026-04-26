// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type Deployment struct {
	Key      string           `json:"key,omitempty"`
	Units    []DeploymentUnit `json:"units,omitempty"`
	TenantId string           `json:"tenantId,omitempty"`
}

type DeploymentUnit struct {
	ProcessDefinition ProcessDefinitionDeployment `json:"processDefinition,omitempty"`
}

type ProcessDefinitionDeployment struct {
	ProcessDefinitionId      string `json:"processDefinitionId,omitempty"`
	ProcessDefinitionKey     string `json:"processDefinitionKey,omitempty"`
	ProcessDefinitionVersion int32  `json:"processDefinitionVersion,omitempty"`
	ResourceName             string `json:"resourceName,omitempty"`
	TenantId                 string `json:"tenantId,omitempty"`
}

type DeploymentUnitData struct {
	Name        string // filename for multipart
	ContentType string // e.g. application/xml
	Data        []byte
}
