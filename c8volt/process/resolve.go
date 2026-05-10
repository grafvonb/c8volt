// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	types "github.com/grafvonb/c8volt/typex"
)

func (c *client) ResolveIncident(ctx context.Context, key string, opts ...options.FacadeOption) (IncidentResolutionResult, error) {
	result, err := incsvc.ResolveIncidentOperation(ctx, c.incApi, key, options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainIncidentResolutionResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) ResolveProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceResolutionResult, error) {
	result, err := incsvc.ResolveProcessInstanceIncidentsOperation(ctx, c.incApi, key, options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainProcessInstanceResolutionResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) ResolveIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (IncidentResolutionResults, error) {
	results, err := incsvc.ResolveIncidents(ctx, c.incApi, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainIncidentResolutionResults(results), ferr.FromDomain(err)
}

func (c *client) ResolveProcessInstancesIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceResolutionResults, error) {
	results, err := incsvc.ResolveProcessInstancesIncidents(ctx, c.incApi, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceResolutionResults(results), ferr.FromDomain(err)
}
