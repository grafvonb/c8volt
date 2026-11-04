// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

// ResolveProcessInstanceSelection mechanically maps service selection and preserves partial evidence with converted errors.
func (c *client) ResolveProcessInstanceSelection(ctx context.Context, request ProcessInstanceSelectionRequest, opts ...options.FacadeOption) (ProcessInstanceSelection, error) {
	result, err := pisvc.ResolveProcessInstanceSelection(ctx, c.piApi, c.incApi, d.ProcessInstanceSelectionRequest{SearchRequest: toDomainProcessInstanceSearchRequest(request.SearchRequest), Cardinality: d.ProcessInstanceSelectionCardinality(request.Cardinality)}, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceSelection(result), ferrors.FromDomain(err)
}

// PlanProcessInstanceSelectedMutation maps the collection request and preserves partial planning evidence.
func (c *client) PlanProcessInstanceSelectedMutation(ctx context.Context, request ProcessInstanceSelectedMutationRequest, opts ...options.FacadeOption) (ProcessInstanceSelectedMutationPlan, error) {
	result, err := pisvc.PlanProcessInstanceSelectedMutation(ctx, c.piApi, c.incApi, toDomainSelectedMutationRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSelectedMutationPlan(result), ferrors.FromDomain(err)
}

// ExecuteProcessInstanceCancellationPlan copies reviewed evidence and maps partial bulk outcomes.
func (c *client) ExecuteProcessInstanceCancellationPlan(ctx context.Context, plan ProcessInstanceSelectedMutationPlan, opts ...options.FacadeOption) (CancelReports, error) {
	result, err := pisvc.ExecuteProcessInstanceCancellationPlan(ctx, c.piApi, c.log, toDomainSelectedMutationPlan(plan), options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainCancelReports(result), ferrors.FromDomain(err)
}

// ExecuteProcessInstanceDeletionPlan copies reviewed scope and preserves partial member and root outcomes.
func (c *client) ExecuteProcessInstanceDeletionPlan(ctx context.Context, plan ProcessInstanceSelectedMutationPlan, opts ...options.FacadeOption) (ProcessInstanceDeletionResult, error) {
	result, err := pisvc.ExecuteProcessInstanceDeletionPlan(ctx, c.piApi, toDomainSelectedMutationPlan(plan), options.MapFacadeOptionsToCallOptions(opts)...)
	return ProcessInstanceDeletionResult{MutationSubmitted: result.MutationSubmitted, DeleteReports: fromDomainDeleteReports(result.Roots), Members: fromDomainDeleteReports(result.Members).Items}, ferrors.FromDomain(err)
}
