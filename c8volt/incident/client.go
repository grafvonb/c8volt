// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	types "github.com/grafvonb/c8volt/typex"
)

type client struct {
	api incsvc.API
	log *slog.Logger
}

func New(api incsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

func (c *client) GetIncident(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceIncidentDetail, error) {
	incident, err := c.api.GetIncident(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstanceIncidentDetail{}, ferr.FromDomain(err)
	}
	return fromDomainIncident(incident), nil
}

func (c *client) GetIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (Incidents, error) {
	incidents, err := incsvc.GetIncidents(ctx, c.api, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Incidents{}, ferr.FromDomain(err)
	}
	return fromDomainIncidents(incidents), nil
}

func (c *client) SearchIncidents(ctx context.Context, filter Filter, size int32, opts ...options.FacadeOption) (Incidents, error) {
	incidents, err := incsvc.SearchIncidents(ctx, c.api, toDomainFilter(filter), size, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Incidents{}, ferr.FromDomain(err)
	}
	return fromDomainIncidents(incidents), nil
}

func (c *client) SearchIncidentsPage(ctx context.Context, filter Filter, page PageRequest, opts ...options.FacadeOption) (Page, error) {
	incidents, err := c.api.SearchIncidentsPage(ctx, toDomainFilter(filter), toDomainPageRequest(page), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Page{}, ferr.FromDomain(err)
	}
	return fromDomainPage(incidents), nil
}

func (c *client) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) ([]ProcessInstanceIncidentDetail, error) {
	incidents, err := c.api.SearchProcessInstanceIncidents(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromDomainIncidentDetails(incidents), nil
}

func (c *client) ResolveIncident(ctx context.Context, key string, opts ...options.FacadeOption) (ResolutionResult, error) {
	result, err := incsvc.ResolveIncidentOperation(ctx, c.api, key, options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainResolutionResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) ResolveProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceResolutionResult, error) {
	result, err := incsvc.ResolveProcessInstanceIncidentsOperation(ctx, c.api, key, options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainProcessInstanceResolutionResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) ResolveIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ResolutionResults, error) {
	results, err := incsvc.ResolveIncidents(ctx, c.api, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainResolutionResults(results), ferr.FromDomain(err)
}

func (c *client) ResolveProcessInstancesIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceResolutionResults, error) {
	results, err := incsvc.ResolveProcessInstancesIncidents(ctx, c.api, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceResolutionResults(results), ferr.FromDomain(err)
}
