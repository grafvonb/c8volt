// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

// PlanSelectedProcessInstanceVariableUpdates copies selection and values into the service-owned planner.
func (c *client) PlanSelectedProcessInstanceVariableUpdates(ctx context.Context, request ProcessInstanceSelectionRequest, variables map[string]any, opts ...options.FacadeOption) (ProcessInstanceVariableUpdatePlan, error) {
	plan, err := pisvc.PlanSelectedProcessInstanceVariableUpdates(ctx, c.piApi, c.incApi, c.updateApi, toDomainSelectionRequest(request), copyProcessInstancePlanVariableMap(variables), options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceVariableUpdatePlan(plan), ferrors.FromDomain(err)
}

// PlanSelectedProcessInstanceSuspension maps service-owned exact or descendant planning.
func (c *client) PlanSelectedProcessInstanceSuspension(ctx context.Context, request ProcessInstanceSelectionRequest, tree bool, opts ...options.FacadeOption) (ProcessInstanceSuspensionPlan, error) {
	plan, err := pisvc.PlanSelectedProcessInstanceSuspension(ctx, c.piApi, c.incApi, toDomainSelectionRequest(request), tree, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceSuspensionPlan(plan), ferrors.FromDomain(err)
}

// ExecuteProcessInstanceSuspensionPlan copies reviewed targets and preserves partial outcomes.
func (c *client) ExecuteProcessInstanceSuspensionPlan(ctx context.Context, plan ProcessInstanceSuspensionPlan, workers int, opts ...options.FacadeOption) (SuspensionResults, error) {
	results, err := pisvc.ExecuteProcessInstanceSuspensionPlan(ctx, c.piApi, c.log, toDomainProcessInstanceSuspensionPlan(plan), workers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSuspensionResults(results), ferrors.FromDomain(err)
}

// PlanSelectedProcessInstanceResumption maps service-owned exact or descendant planning.
func (c *client) PlanSelectedProcessInstanceResumption(ctx context.Context, request ProcessInstanceSelectionRequest, tree bool, opts ...options.FacadeOption) (ProcessInstanceResumptionPlan, error) {
	plan, err := pisvc.PlanSelectedProcessInstanceResumption(ctx, c.piApi, c.incApi, toDomainSelectionRequest(request), tree, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceResumptionPlan(plan), ferrors.FromDomain(err)
}

// ExecuteProcessInstanceResumptionPlan copies reviewed targets and preserves partial outcomes.
func (c *client) ExecuteProcessInstanceResumptionPlan(ctx context.Context, plan ProcessInstanceResumptionPlan, workers int, opts ...options.FacadeOption) (ResumptionResults, error) {
	results, err := pisvc.ExecuteProcessInstanceResumptionPlan(ctx, c.piApi, c.log, toDomainProcessInstanceResumptionPlan(plan), workers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainResumptionResults(results), ferrors.FromDomain(err)
}
