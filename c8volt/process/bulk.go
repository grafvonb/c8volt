// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	types "github.com/grafvonb/c8volt/typex"
)

// CreateNProcessInstances delegates same-data bulk creation to the process-instance service.
func (c *client) CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, wantedWorkers int, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	pics, err := pisvc.CreateNProcessInstances(ctx, c.piApi, c.log, toProcessInstanceData(data), n, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return toolx.MapSlice(pics, fromDomainProcessInstanceCreation), nil
}

// CancelProcessInstances delegates bulk cancellation to the process-instance service.
func (c *client) CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (CancelReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	rs, err := pisvc.CancelProcessInstances(ctx, c.piApi, c.log, keys, wantedWorkers, cCfg.AffectedProcessInstanceCount, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainCancelReports(rs), ferr.FromDomain(err)
}

// DeleteProcessInstances delegates bulk deletion to the process-instance service.
func (c *client) DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	rs, err := pisvc.DeleteProcessInstances(ctx, c.piApi, c.log, keys, wantedWorkers, cCfg.AffectedProcessInstanceCount, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainDeleteReports(rs), ferr.FromDomain(err)
}

// UpdateProcessInstancesVariables delegates bulk variable mutation to the process-instance service.
func (c *client) UpdateProcessInstancesVariables(ctx context.Context, keys types.Keys, variables map[string]any, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceVariableUpdateResults, error) {
	results, err := pisvc.UpdateProcessInstancesVariables(ctx, c.piApi, c.log, keys, variables, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceVariableUpdateResults(results), ferr.FromDomain(err)
}

// WaitForProcessInstancesState maps facade state values and delegates the wait.
func (c *client) WaitForProcessInstancesState(ctx context.Context, keys types.Keys, desired States, wantedWorkers int, opts ...options.FacadeOption) (StateReports, error) {
	got, err := pisvc.WaitForProcessInstancesState(ctx, c.piApi, c.log, keys, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	srs := MapStateResponsesToReports(got)
	if err != nil {
		return srs, ferr.FromDomain(err)
	}
	return srs, nil
}

// WaitForProcessInstancesExpectation maps facade expectation values and delegates the wait.
func (c *client) WaitForProcessInstancesExpectation(ctx context.Context, keys types.Keys, request ProcessInstanceExpectationRequest, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceExpectationReports, error) {
	got, err := pisvc.WaitForProcessInstancesExpectation(ctx, c.piApi, c.log, keys, toDomainProcessInstanceExpectationRequest(request), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	reports := fromDomainProcessInstanceExpectationResponses(got)
	if err != nil {
		return reports, ferr.FromDomain(err)
	}
	return reports, nil
}

// GetProcessInstances delegates bulk lookup to the process-instance service.
func (c *client) GetProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstances, error) {
	pis, err := pisvc.GetProcessInstances(ctx, c.piApi, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstances(pis), nil
}
