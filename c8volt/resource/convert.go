package resource

import (
	d "github.com/grafvonb/c8volt/internal/domain"
)

func fromProcessDefinitionDeployment(d d.Deployment) ProcessDefinitionDeployment {
	return ProcessDefinitionDeployment{
		Key: d.Key,
		//DefinitionId:      d.Units[0].ProcessDefinition.ProcessDefinitionId,
		//DefinitionVersion: d.Units[0].ProcessDefinition.ProcessDefinitionVersion,
		//DefinitionKey:     d.Units[0].ProcessDefinition.ProcessDefinitionKey,
		//ResourceName:      d.Units[0].ProcessDefinition.ResourceName,
		TenantId: d.TenantId,
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
