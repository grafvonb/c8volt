// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/incident"
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// PlanSelectedProcessInstanceResolution delegates all discovery to the service and copies its evidence.
func (c *client) PlanSelectedProcessInstanceResolution(ctx context.Context, request ProcessInstanceSelectionRequest, workers int, opts ...options.FacadeOption) (SelectedProcessInstanceResolutionPlan, error) {
	plan, err := pisvc.PlanSelectedProcessInstanceResolution(ctx, c.piApi, c.incApi, toDomainSelectionRequest(request), workers, options.MapFacadeOptionsToCallOptions(opts)...)
	return SelectedProcessInstanceResolutionPlan{Scope: fromDomainSelectedMutationPlan(plan.Scope), Incidents: incident.ProcessInstancesIncidentPlan{Keys: append([]string(nil), plan.Incidents.Keys...), CompletedKeys: append([]string(nil), plan.Incidents.CompletedKeys...), Incidents: fromDomainProcessInstanceIncidentDetails(plan.Incidents.Incidents)}}, ferrors.FromDomain(err)
}

// ExecuteSelectedProcessInstanceResolution maps reviewed evidence into the service-owned executor.
func (c *client) ExecuteSelectedProcessInstanceResolution(ctx context.Context, plan SelectedProcessInstanceResolutionPlan, opts ...options.FacadeOption) (incident.ProcessInstanceResolutionResults, error) {
	result, err := pisvc.ExecuteSelectedProcessInstanceResolution(ctx, c.piApi, c.incApi, d.SelectedProcessInstanceResolutionPlan{Scope: toDomainSelectedMutationPlan(plan.Scope), Incidents: d.ProcessInstancesIncidentPlan{Keys: append(typex.Keys(nil), plan.Incidents.Keys...), CompletedKeys: append(typex.Keys(nil), plan.Incidents.CompletedKeys...), Incidents: toolx.MapSlice(plan.Incidents.Incidents, toDomainProcessInstanceIncidentDetail)}}, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSelectedResolutionResults(result), ferrors.FromDomain(err)
}

// fromDomainSelectedResolutionResult copies incident outcomes into the existing public result contract.
func fromDomainSelectedResolutionResult(x d.ProcessInstanceResolutionResult) incident.ProcessInstanceResolutionResult {
	return incident.ProcessInstanceResolutionResult{
		ProcessInstanceKey:    x.ProcessInstanceKey,
		AttemptedIncidentKeys: append([]string(nil), x.AttemptedIncidentKeys...),
		ResolvedIncidentKeys:  append([]string(nil), x.ResolvedIncidentKeys...),
		SkippedIncidentKeys:   append([]string(nil), x.SkippedIncidentKeys...),
		FailedIncidentKeys:    append([]string(nil), x.FailedIncidentKeys...),
		ConfirmationStatus:    x.ConfirmationStatus,
		Status:                incident.ProcessInstanceResolutionStatus(x.Status),
		Error:                 x.Error,
		DryRun:                x.DryRun,
		MutationSubmitted:     x.MutationSubmitted,
		Incidents:             fromDomainProcessInstanceIncidentDetails(x.Incidents),
	}
}

// fromDomainSelectedResolutionResults copies incident outcomes into the existing public result contract.
func fromDomainSelectedResolutionResults(x d.ProcessInstanceResolutionResults) incident.ProcessInstanceResolutionResults {
	return incident.ProcessInstanceResolutionResults{
		Operation:         incident.ResolutionOperation(x.Operation),
		Items:             toolx.MapSlice(x.Items, fromDomainSelectedResolutionResult),
		Total:             x.Total,
		Submitted:         x.Submitted,
		Confirmed:         x.Confirmed,
		Skipped:           x.Skipped,
		Failed:            x.Failed,
		DryRun:            x.DryRun,
		MutationSubmitted: x.MutationSubmitted,
	}
}
