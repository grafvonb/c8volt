// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstancesIncidents mechanically maps the read-only service plan, retaining partial evidence.
func (c *client) PlanProcessInstancesIncidents(ctx context.Context, keys typex.Keys, opts ...options.FacadeOption) (ProcessInstancesIncidentPlan, error) {
	plan, err := incsvc.PlanProcessInstancesIncidents(ctx, c.api, append(typex.Keys(nil), keys...), options.MapFacadeOptionsToCallOptions(opts)...)
	return ProcessInstancesIncidentPlan{Keys: append([]string(nil), plan.Keys...), CompletedKeys: append([]string(nil), plan.CompletedKeys...), Incidents: fromDomainIncidentDetails(plan.Incidents)}, ferrors.FromDomain(err)
}

// ExecuteProcessInstancesIncidentPlan copies the reviewed input and preserves partial results and errors.
func (c *client) ExecuteProcessInstancesIncidentPlan(ctx context.Context, plan ProcessInstancesIncidentPlan, workers int, opts ...options.FacadeOption) (ProcessInstanceResolutionResults, error) {
	result, err := incsvc.ExecuteProcessInstancesIncidentPlan(ctx, c.api, d.ProcessInstancesIncidentPlan{Keys: append(typex.Keys(nil), plan.Keys...), CompletedKeys: append(typex.Keys(nil), plan.CompletedKeys...), Incidents: toolx.MapSlice(plan.Incidents, toDomainIncident)}, workers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceResolutionResults(result), ferrors.FromDomain(err)
}

// toDomainIncident copies resource observations without substituting selection criteria.
func toDomainIncident(x ProcessInstanceIncidentDetail) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
		IncidentKey:            x.IncidentKey,
		CreationTime:           x.CreationTime,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		TenantId:               x.TenantId,
		State:                  x.State,
		ErrorType:              x.ErrorType,
		ErrorMessage:           x.ErrorMessage,
		ElementId:              x.ElementId,
		ElementInstanceKey:     x.ElementInstanceKey,
		JobKey:                 x.JobKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
	}
}
