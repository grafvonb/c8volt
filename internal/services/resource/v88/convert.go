// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

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

func fromResourceResult(r camundav88.ResourceResult) d.Resource {
	return d.Resource{
		ID:         r.ResourceId,
		Key:        r.ResourceKey,
		Name:       r.ResourceName,
		TenantId:   r.TenantId,
		Version:    r.Version,
		VersionTag: valueOrEmpty(r.VersionTag),
	}
}

func valueOrEmpty[T ~string](v *T) T {
	if v == nil {
		return ""
	}
	return *v
}
