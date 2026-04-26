// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	d "github.com/grafvonb/c8volt/internal/domain"
)

func fromProcessDefinitionDeployment(dep d.Deployment) []ProcessDefinitionDeployment {
	if len(dep.Units) == 0 {
		return []ProcessDefinitionDeployment{{
			Key:      dep.Key,
			TenantId: dep.TenantId,
		}}
	}
	out := make([]ProcessDefinitionDeployment, 0, len(dep.Units))
	for _, u := range dep.Units {
		pd := u.ProcessDefinition
		out = append(out, ProcessDefinitionDeployment{
			Key:               dep.Key,
			DefinitionId:      pd.ProcessDefinitionId,
			DefinitionVersion: pd.ProcessDefinitionVersion,
			DefinitionKey:     pd.ProcessDefinitionKey,
			ResourceName:      pd.ResourceName,
			TenantId:          dep.TenantId,
		})
	}
	return out
}

func fromResource(resource d.Resource) Resource {
	return Resource{
		ID:         resource.ID,
		Key:        resource.Key,
		Name:       resource.Name,
		TenantId:   resource.TenantId,
		Version:    resource.Version,
		VersionTag: resource.VersionTag,
	}
}

func toDeploymentUnitDatas(units []DeploymentUnitData) []d.DeploymentUnitData {
	result := make([]d.DeploymentUnitData, len(units))
	for i, u := range units {
		result[i] = d.DeploymentUnitData{
			Name:        u.Name,
			ContentType: u.ContentType,
			Data:        u.Data,
		}
	}
	return result
}
