// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type API interface {
	GetResource(ctx context.Context, key string, opts ...options.FacadeOption) (Resource, error)
	DeployProcessDefinition(ctx context.Context, units []DeploymentUnitData, opts ...options.FacadeOption) ([]ProcessDefinitionDeployment, error)

	DeleteProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error)
	DeleteProcessDefinitions(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error)
}

var _ API = (*client)(nil)
