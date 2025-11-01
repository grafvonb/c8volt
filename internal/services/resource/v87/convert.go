package v87

import (
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDeploymentResult(r camundav87.DeploymentResult) d.Deployment {
	return d.Deployment{
		Key:      "<unknown>",
		TenantId: toolx.Deref(r.TenantId, ""),
	}
}

//nolint:unused
func fromDeploymentUnit(b camundav87.DeploymentMetadataResult) d.DeploymentUnit {
	return d.DeploymentUnit{
		ProcessDefinition: fromDeploymentProcessResult(*b.ProcessDefinition),
	}
}

//nolint:unused
func fromDeploymentProcessResult(p camundav87.DeploymentProcessResult) d.ProcessDefinitionDeployment {
	return d.ProcessDefinitionDeployment{
		TenantId:                 toolx.Deref(p.TenantId, ""),
		ProcessDefinitionId:      toolx.Deref(p.ProcessDefinitionId, ""),
		ProcessDefinitionVersion: toolx.Deref(p.ProcessDefinitionVersion, 0),
		ResourceName:             toolx.Deref(p.ResourceName, ""),
	}
}
