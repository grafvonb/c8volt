// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	esvc "github.com/grafvonb/c8volt/internal/services/element"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/typex"
)

// NewWithVariableUpdates adds collection planning and execution to the existing element facade.
// Legacy constructors remain valid and report preconditions for unwired update operations.
func NewWithVariableUpdates(api esvc.API, jobs jsvc.API, enrichment esvc.EnrichmentAPI, updates esvc.VariableUpdateAPI, log *slog.Logger) API {
	if enrichment == nil {
		enrichment = esvc.NewEnrichment(api, jobs, nil)
	}
	return &client{api: api, jobAPI: jobs, enrichmentAPI: enrichment, updateAPI: updates, log: log}
}

// PlanElementVariableUpdates maps and isolates the public inputs, delegates all
// planning to the service, and publishes only a complete converted plan.
func (c *client) PlanElementVariableUpdates(ctx context.Context, keys typex.Keys, variables map[string]any, opts ...foptions.FacadeOption) (ElementVariableUpdatePlan, error) {
	if c.updateAPI == nil {
		return ElementVariableUpdatePlan{}, ferrors.FromDomain(fmt.Errorf("%w: element variable planning requires a variable update service", d.ErrPrecondition))
	}
	// Validate representability before recursive copying, including cyclic maps.
	if _, err := json.Marshal(variables); err != nil {
		return ElementVariableUpdatePlan{}, ferrors.FromDomain(fmt.Errorf("%w: requested variables are not JSON-compatible: %v", d.ErrValidation, err))
	}
	plan, err := c.updateAPI.PlanElementVariableUpdates(ctx, append(typex.Keys(nil), keys...), copyElementPlanVariableMap(variables), foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ElementVariableUpdatePlan{}, ferrors.FromDomain(err)
	}
	return fromDomainElementVariableUpdatePlan(plan), nil
}

// ExecuteElementVariableUpdates only snapshots/maps inputs, delegates execution,
// and converts results alongside the original failure classification.
func (c *client) ExecuteElementVariableUpdates(ctx context.Context, plan ElementVariableUpdatePlan, wantedWorkers int, opts ...foptions.FacadeOption) (ElementVariableUpdateResults, error) {
	if c.updateAPI == nil {
		return ElementVariableUpdateResults{}, ferrors.FromDomain(fmt.Errorf("%w: element variable execution requires a variable update service", d.ErrPrecondition))
	}
	if _, err := json.Marshal(plan); err != nil {
		return ElementVariableUpdateResults{}, ferrors.FromDomain(fmt.Errorf("%w: element variable plan is not JSON-compatible: %v", d.ErrValidation, err))
	}
	result, err := c.updateAPI.ExecuteElementVariableUpdates(ctx, toDomainElementVariableUpdatePlan(plan), wantedWorkers, foptions.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainElementVariableUpdateResults(result), ferrors.FromDomain(err)
}
