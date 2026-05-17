// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	ExecuteSmokeTest(ctx context.Context, request SmokeTestRequest, opts ...options.FacadeOption) (SmokeTestResult, error)
	PurgeOrphanProcessInstances(ctx context.Context, request OrphanPurgeRequest, opts ...options.FacadeOption) (OrphanPurgeResult, error)
	ExecuteRetentionPolicy(ctx context.Context, request RetentionPolicyRequest, opts ...options.FacadeOption) (RetentionPolicyResult, error)
	PurgeProcessInstancesWithIncidents(ctx context.Context, request IncidentPurgeRequest, opts ...options.FacadeOption) (IncidentPurgeResult, error)
	PurgeAllProcessDefinitions(ctx context.Context, request AllProcessDefinitionsPurgeRequest, opts ...options.FacadeOption) (AllProcessDefinitionsPurgeResult, error)
	RepairIncidents(ctx context.Context, request RepairRequest, opts ...options.FacadeOption) (RepairResult, error)
	RepairProcessInstances(ctx context.Context, request RepairRequest, opts ...options.FacadeOption) (RepairResult, error)
}

var _ API = (*client)(nil)
