// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"fmt"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceVariableUpdates delegates complete collection planning to
// the composed service and isolates all mutable values at the public boundary.
func (c *client) PlanProcessInstanceVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...options.FacadeOption) (ProcessInstanceVariableUpdatePlan, error) {
	if c.updateApi == nil {
		return ProcessInstanceVariableUpdatePlan{}, ferr.FromDomain(fmt.Errorf("%w: process-instance variable planning requires a variable update service", d.ErrPrecondition))
	}
	plan, err := c.updateApi.PlanProcessInstanceVariableUpdates(
		ctx, append(typex.Keys(nil), keys...), copyProcessInstancePlanVariableMap(variables), options.MapFacadeOptionsToCallOptions(opts)...,
	)
	if err != nil {
		return ProcessInstanceVariableUpdatePlan{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstanceVariableUpdatePlan(plan), nil
}

// ExecuteProcessInstanceVariableUpdates delegates a frozen public plan to the
// composed service and preserves any partial ordered results alongside errors.
func (c *client) ExecuteProcessInstanceVariableUpdates(ctx context.Context, plan ProcessInstanceVariableUpdatePlan, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceVariableUpdateResults, error) {
	if c.updateApi == nil {
		return ProcessInstanceVariableUpdateResults{}, ferr.FromDomain(fmt.Errorf("%w: process-instance variable execution requires a variable update service", d.ErrPrecondition))
	}
	results, err := c.updateApi.ExecuteProcessInstanceVariableUpdates(
		ctx, toDomainProcessInstanceVariableUpdatePlan(plan), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...,
	)
	return fromDomainProcessInstanceVariableUpdateResults(results), ferr.FromDomain(err)
}
