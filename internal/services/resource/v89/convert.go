// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDeploymentResult(r camundav89.DeploymentResult) d.Deployment {
	return d.Deployment{
		Key:      r.DeploymentKey,
		Units:    toolx.MapSlice(r.Deployments, fromDeploymentUnit),
		TenantId: r.TenantId,
	}
}

func fromDeploymentUnit(b camundav89.DeploymentMetadataResult) d.DeploymentUnit {
	return d.DeploymentUnit{
		ProcessDefinition: fromDeploymentProcessResult(*b.ProcessDefinition),
	}
}

func fromDeploymentProcessResult(p camundav89.DeploymentProcessResult) d.ProcessDefinitionDeployment {
	return d.ProcessDefinitionDeployment{
		TenantId:                 p.TenantId,
		ProcessDefinitionKey:     p.ProcessDefinitionKey,
		ProcessDefinitionId:      p.ProcessDefinitionId,
		ProcessDefinitionVersion: p.ProcessDefinitionVersion,
		ResourceName:             p.ResourceName,
	}
}

func fromResourceResult(r camundav89.ResourceResult) d.Resource {
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
