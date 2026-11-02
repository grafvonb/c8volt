// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
)

// PlanSelectedProcessInstanceResolution freezes the complete selected family union before incident planning.
func PlanSelectedProcessInstanceResolution(ctx context.Context, api API, incidents incsvc.API, request d.ProcessInstanceSelectionRequest, workers int, opts ...services.CallOption) (d.SelectedProcessInstanceResolutionPlan, error) {
	scope, err := PlanProcessInstanceSelectedMutation(ctx, api, incidents, d.ProcessInstanceSelectedMutationRequest{Operation: "resolve", Selection: request, Workers: workers}, opts...)
	plan := d.SelectedProcessInstanceResolutionPlan{Scope: scope}
	if err != nil {
		return plan, err
	}
	plan.Incidents, err = incsvc.PlanProcessInstancesIncidents(ctx, incidents, scope.Expansion.Collected, append(slices.Clone(opts), services.WithIgnoreTenant())...)
	return plan, err
}

// ExecuteSelectedProcessInstanceResolution validates capability, tenant, family and incident evidence before any shortcut or write.
func ExecuteSelectedProcessInstanceResolution(ctx context.Context, api API, incidents incsvc.API, plan d.SelectedProcessInstanceResolutionPlan, opts ...services.CallOption) (d.ProcessInstanceResolutionResults, error) {
	if err := validateSelectedMutationPlan(api, plan.Scope, "resolve", opts...); err != nil {
		return d.ProcessInstanceResolutionResults{}, err
	}
	if !slices.Equal(plan.Scope.Expansion.Collected, plan.Incidents.Keys) {
		return d.ProcessInstanceResolutionResults{}, fmt.Errorf("%w: incident work differs from reviewed family", d.ErrPrecondition)
	}
	return incsvc.ExecuteProcessInstancesIncidentPlan(ctx, incidents, plan.Incidents, plan.Scope.Request.Workers, append(slices.Clone(opts), services.WithIgnoreTenant())...)
}
