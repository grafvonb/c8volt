// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	opsvc "github.com/grafvonb/c8volt/internal/services/ops"
)

type client struct {
	api opsvc.API
	log *slog.Logger
}

func New(api opsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

func (c *client) ExecuteSmokeTest(ctx context.Context, request SmokeTestRequest, opts ...options.FacadeOption) (SmokeTestResult, error) {
	result, err := c.api.ExecuteSmokeTest(ctx, toDomainSmokeTestRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainSmokeTestResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) PurgeOrphanProcessInstances(ctx context.Context, request OrphanPurgeRequest, opts ...options.FacadeOption) (OrphanPurgeResult, error) {
	result, err := c.api.PurgeOrphanProcessInstances(ctx, toDomainOrphanPurgeRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainOrphanPurgeResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) ExecuteRetentionPolicy(ctx context.Context, request RetentionPolicyRequest, opts ...options.FacadeOption) (RetentionPolicyResult, error) {
	result, err := c.api.ExecuteRetentionPolicy(ctx, toDomainRetentionPolicyRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainRetentionPolicyResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) PurgeProcessInstancesWithIncidents(ctx context.Context, request IncidentPurgeRequest, opts ...options.FacadeOption) (IncidentPurgeResult, error) {
	result, err := c.api.PurgeProcessInstancesWithIncidents(ctx, toDomainIncidentPurgeRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainIncidentPurgeResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) PurgeAllProcessDefinitions(ctx context.Context, request AllProcessDefinitionsPurgeRequest, opts ...options.FacadeOption) (AllProcessDefinitionsPurgeResult, error) {
	result, err := c.api.PurgeAllProcessDefinitions(ctx, toDomainAllProcessDefinitionsPurgeRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainAllProcessDefinitionsPurgeResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}
