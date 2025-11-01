package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDeploymentResult(r camundav88.DeploymentResult) d.Deployment {
	return d.Deployment{
		Key:      r.DeploymentKey,
		Units:    toolx.MapSlice(r.Deployments, fromDeploymentUnit),
		TenantId: r.TenantId,
	}
}

func fromDeploymentUnit(b camundav88.DeploymentMetadataResult) d.DeploymentUnit {
	return d.DeploymentUnit{
		ProcessDefinition: fromDeploymentProcessResult(*b.ProcessDefinition),
	}
}

func fromDeploymentProcessResult(p camundav88.DeploymentProcessResult) d.ProcessDefinitionDeployment {
	return d.ProcessDefinitionDeployment{
		TenantId:                 p.TenantId,
		ProcessDefinitionKey:     p.ProcessDefinitionKey,
		ProcessDefinitionId:      p.ProcessDefinitionId,
		ProcessDefinitionVersion: p.ProcessDefinitionVersion,
		ResourceName:             p.ResourceName,
	}
}
