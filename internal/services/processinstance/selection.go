// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// ResolveProcessInstanceSelection proves bounded collection or unique-start cardinality using shared discovery.
// Search authorization options are preserved; callers must not supply explicit-key tenant bypasses by accident.
func ResolveProcessInstanceSelection(ctx context.Context, api API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSelectionRequest, opts ...services.CallOption) (d.ProcessInstanceSelection, error) {
	request = copyProcessInstanceSelectionRequest(request)
	result := d.ProcessInstanceSelection{Request: request, Origin: "search", Completion: d.ProcessInstanceSearchCompletionIncomplete, IgnoreTenant: services.ApplyCallOptions(opts).IgnoreTenant}
	if err := api.ValidateProcessInstanceSearchFilter(request.SearchRequest.Filter); err != nil {
		return result, err
	}
	switch request.Cardinality {
	case d.ProcessInstanceSelectionCollection, d.ProcessInstanceSelectionUniqueStart:
	default:
		return result, fmt.Errorf("%w: unknown process selection cardinality", d.ErrValidation)
	}
	if request.SearchRequest.Limit < 0 || request.SearchRequest.Page.From != 0 || request.SearchRequest.Page.After != "" {
		return result, fmt.Errorf("%w: selection requires a nonnegative limit and discovery from the beginning", d.ErrValidation)
	}
	var visitor d.ProcessInstanceSearchPageVisitor
	if request.Cardinality == d.ProcessInstanceSelectionUniqueStart {
		visitor = func(step d.ProcessInstanceSearchPageStep) (d.ProcessInstanceSearchPageAction, error) {
			if step.CumulativeCount > 1 {
				return d.ProcessInstanceSearchPageActionStop, fmt.Errorf("%w: business-ID selection is ambiguous; use --key to choose a process instance", d.ErrPrecondition)
			}
			return d.ProcessInstanceSearchPageActionContinue, nil
		}
	}
	found, err := SearchProcessInstancesPages(ctx, api, incAPI, request.SearchRequest, visitor, opts...)
	result.Completion = found.Completion
	result.Pages = found.Pages
	for _, item := range found.Items {
		item.BusinessId = toolx.CopyPtr(item.BusinessId)
		item.Variables = copyProcessInstanceVariableMap(item.Variables)
		result.Items = append(result.Items, item)
		result.Keys = append(result.Keys, item.Key)
	}
	result.TenantEvidence = selectionTenantEvidence(result.Items)
	if err != nil {
		return result, err
	}
	for _, key := range result.Keys {
		if key == "" {
			result.Completion = d.ProcessInstanceSearchCompletionIncomplete
			return result, fmt.Errorf("%w: selected process instance has no key", d.ErrMalformedResponse)
		}
	}
	if found.Completion == d.ProcessInstanceSearchCompletionExhausted {
		return result, nil
	}
	if request.Cardinality == d.ProcessInstanceSelectionCollection && found.Completion == d.ProcessInstanceSearchCompletionLimitReached && request.SearchRequest.Limit > 0 {
		return result, nil
	}
	return result, fmt.Errorf("%w: process selection completeness could not be established", d.ErrPrecondition)
}

// copyProcessInstanceSelectionRequest isolates every mutable search criterion before discovery.
func copyProcessInstanceSelectionRequest(request d.ProcessInstanceSelectionRequest) d.ProcessInstanceSelectionRequest {
	f := &request.SearchRequest.Filter
	f.BusinessId = toolx.CopyPtr(f.BusinessId)
	f.HasParent = toolx.CopyPtr(f.HasParent)
	f.HasIncident = toolx.CopyPtr(f.HasIncident)
	f.States = slices.Clone(f.States)
	f.VariableFilters.Clauses = slices.Clone(f.VariableFilters.Clauses)
	for i := range f.VariableFilters.Clauses {
		f.VariableFilters.Clauses[i].Exists = toolx.CopyPtr(f.VariableFilters.Clauses[i].Exists)
	}
	return request
}

// ValidateProcessInstanceSelection checks receiving capability, completed collection evidence and original tenant context without I/O.
func ValidateProcessInstanceSelection(api API, selection d.ProcessInstanceSelection, tenant string, opts ...services.CallOption) error {
	return validateActionSelection(api, &selection, tenant, selection.Keys, opts...)
}

// CopyProcessInstanceSelection isolates nested selection criteria and observed resource data.
func CopyProcessInstanceSelection(selection d.ProcessInstanceSelection) d.ProcessInstanceSelection {
	return copyActionSelection(selection)
}

// selectionTenantEvidence summarizes actual selected observations without borrowing expanded-family tenants.
func selectionTenantEvidence(items []d.ProcessInstance) d.TenantEvidence {
	accumulator := common.NewTenantEvidenceAccumulator()
	for _, item := range items {
		accumulator.Add(item.Key, item.TenantId)
	}
	evidence := domainTenantEvidence(accumulator.Snapshot())
	for _, item := range items {
		evidence.Targets = append(evidence.Targets, d.TenantEvidenceTarget{Key: item.Key, TenantID: item.TenantId})
	}
	return evidence
}
