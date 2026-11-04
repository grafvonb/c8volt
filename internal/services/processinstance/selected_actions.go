// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// validateActionSelection checks receiving-adapter capability before empty or prepared-plan shortcuts.
// Like other invocation-local plans, this proves consistency, not snapshot authenticity.
func validateActionSelection(api API, selection *d.ProcessInstanceSelection, tenant string, keys typex.Keys, opts ...services.CallOption) error {
	invalid := func() error {
		return fmt.Errorf("%w: action selection is incomplete or inconsistent", d.ErrPrecondition)
	}
	if selection == nil {
		return invalid()
	}
	if err := api.ValidateProcessInstanceSearchFilter(selection.Request.SearchRequest.Filter); err != nil {
		return err
	}
	request := selection.Request
	if request.Cardinality != d.ProcessInstanceSelectionCollection || selection.Origin != "search" || selection.Pages < 1 ||
		tenant != api.ProcessInstanceSearchTenant(opts...) || selection.IgnoreTenant != services.ApplyCallOptions(opts).IgnoreTenant ||
		request.SearchRequest.Limit < 0 || request.SearchRequest.Page.From != 0 || request.SearchRequest.Page.After != "" ||
		!slices.Equal(keys, selection.Keys) || !distinctNonemptyKeys(keys) || len(keys) != len(selection.Items) {
		return invalid()
	}
	switch selection.Completion {
	case d.ProcessInstanceSearchCompletionExhausted:
	case d.ProcessInstanceSearchCompletionLimitReached:
		if request.SearchRequest.Limit <= 0 || len(keys) != int(request.SearchRequest.Limit) {
			return invalid()
		}
	default:
		return invalid()
	}
	if request.SearchRequest.Limit > 0 && len(keys) > int(request.SearchRequest.Limit) {
		return invalid()
	}
	for i, item := range selection.Items {
		if item.Key != keys[i] || (tenant != "" && item.TenantId != tenant) {
			return invalid()
		}
	}
	return nil
}

// PlanSelectedProcessInstanceVariableUpdates selects once, then freezes existing exact-root payloads.
func PlanSelectedProcessInstanceVariableUpdates(ctx context.Context, api API, incAPI SearchProcessInstanceIncidentAPI, updates VariableUpdateAPI, request d.ProcessInstanceSelectionRequest, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdatePlan, error) {
	selection, err := ResolveProcessInstanceSelection(ctx, api, incAPI, request, opts...)
	tenant := api.ProcessInstanceSearchTenant(opts...)
	plan := d.ProcessInstanceVariableUpdatePlan{Selection: &selection, SelectionTenant: tenant}
	if err != nil {
		return plan, err
	}
	if err := validateActionSelection(api, &selection, tenant, selection.Keys, opts...); err != nil {
		return plan, err
	}
	if updates == nil {
		return plan, fmt.Errorf("%w: variable update service is required", d.ErrPrecondition)
	}
	planned, err := updates.PlanProcessInstanceVariableUpdates(ctx, selection.Keys, copyProcessInstanceVariableMap(variables), append(slices.Clone(opts), services.WithIgnoreTenant())...)
	planned.Selection, planned.SelectionTenant = &selection, tenant
	return planned, err
}

// PlanSelectedProcessInstanceSuspension resolves a collection before exact or descendant-only planning.
func PlanSelectedProcessInstanceSuspension(ctx context.Context, api API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSelectionRequest, tree bool, opts ...services.CallOption) (d.ProcessInstanceSuspensionPlan, error) {
	selection, err := ResolveProcessInstanceSelection(ctx, api, incAPI, request, opts...)
	tenant := api.ProcessInstanceSearchTenant(opts...)
	plan := d.ProcessInstanceSuspensionPlan{Selection: &selection, SelectionTenant: tenant, Exact: !tree, RequestedKeys: slices.Clone(selection.Keys), TargetKeys: make(typex.Keys, 0), Instances: make([]d.ProcessInstanceSuspensionPlanItem, 0)}
	if err != nil {
		return plan, err
	}
	if err := validateActionSelection(api, &selection, tenant, selection.Keys, opts...); err != nil {
		return plan, err
	}
	if err := api.ValidateProcessInstanceSuspension(); err != nil {
		return plan, err
	}
	if len(selection.Keys) == 0 {
		return plan, nil
	}
	keyed := append(slices.Clone(opts), services.WithIgnoreTenant())
	if tree {
		planned, err := PlanProcessInstanceSuspension(ctx, api, selection.Keys, keyed...)
		planned.Selection, planned.SelectionTenant = &selection, tenant
		return planned, err
	}
	// Preserve exact-key semantics: a local preview validates timing but never filters state.
	if _, err := api.SuspendProcessInstance(ctx, selection.Keys[0], append(keyed, services.WithDryRun())...); err != nil {
		return plan, err
	}
	plan.TargetKeys = slices.Clone(selection.Keys)
	return plan, nil
}

// ExecuteProcessInstanceSuspensionPlan revalidates provenance then submits only frozen target keys.
func ExecuteProcessInstanceSuspensionPlan(ctx context.Context, api API, log *slog.Logger, plan d.ProcessInstanceSuspensionPlan, workers int, opts ...services.CallOption) (d.ProcessInstanceSuspensionResults, error) {
	empty := d.ProcessInstanceSuspensionResults{Selection: toolx.MapPtr(plan.Selection, copyActionSelection), Items: make([]d.ProcessInstanceSuspensionResult, 0)}
	if err := validateActionSelection(api, plan.Selection, plan.SelectionTenant, plan.RequestedKeys, opts...); err != nil {
		return empty, err
	}
	if err := api.ValidateProcessInstanceSuspension(); err != nil {
		return empty, err
	}
	invalid := func() (d.ProcessInstanceSuspensionResults, error) {
		return empty, fmt.Errorf("%w: suspension scope is inconsistent", d.ErrPrecondition)
	}
	if !distinctNonemptyKeys(plan.TargetKeys) {
		return invalid()
	}
	if plan.Exact {
		if !slices.Equal(plan.TargetKeys, plan.RequestedKeys) || len(plan.Instances) > 0 {
			return invalid()
		}
	} else {
		seen := map[string]bool{}
		targets := typex.Keys{}
		for _, item := range plan.Instances {
			if item.Key == "" || seen[item.Key] {
				return invalid()
			}
			seen[item.Key] = true
			classified := classifySuspensionPlanItem(d.ProcessInstance{Key: item.Key, State: item.ObservedState})
			if classified.Classification != item.Classification {
				return invalid()
			}
			if item.Classification == d.ProcessInstanceSuspensionPlanEligible {
				targets = append(targets, item.Key)
			}
		}
		for _, key := range plan.RequestedKeys {
			if !seen[key] {
				return invalid()
			}
		}
		if len(plan.RequestedKeys) == 0 && len(plan.Instances) > 0 {
			return invalid()
		}
		if !slices.Equal(targets, plan.TargetKeys) {
			return invalid()
		}
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if len(plan.TargetKeys) == 0 {
		return empty, nil
	}
	results, err := SuspendProcessInstances(ctx, api, log, slices.Clone(plan.TargetKeys), workers, append(slices.Clone(opts), services.WithIgnoreTenant())...)
	results.Selection = toolx.MapPtr(plan.Selection, copyActionSelection)
	return results, err
}

// PlanSelectedProcessInstanceResumption resolves a collection before exact or descendant-only planning.
func PlanSelectedProcessInstanceResumption(ctx context.Context, api API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSelectionRequest, tree bool, opts ...services.CallOption) (d.ProcessInstanceResumptionPlan, error) {
	selection, err := ResolveProcessInstanceSelection(ctx, api, incAPI, request, opts...)
	tenant := api.ProcessInstanceSearchTenant(opts...)
	plan := d.ProcessInstanceResumptionPlan{Selection: &selection, SelectionTenant: tenant, Exact: !tree, RequestedKeys: slices.Clone(selection.Keys), TargetKeys: make(typex.Keys, 0), Instances: make([]d.ProcessInstanceResumptionPlanItem, 0)}
	if err != nil {
		return plan, err
	}
	if err := validateActionSelection(api, &selection, tenant, selection.Keys, opts...); err != nil {
		return plan, err
	}
	if err := api.ValidateProcessInstanceResumption(); err != nil {
		return plan, err
	}
	if len(selection.Keys) == 0 {
		return plan, nil
	}
	keyed := append(slices.Clone(opts), services.WithIgnoreTenant())
	if tree {
		planned, err := PlanProcessInstanceResumption(ctx, api, selection.Keys, keyed...)
		planned.Selection, planned.SelectionTenant = &selection, tenant
		return planned, err
	}
	// Preserve exact-key semantics: a local preview validates timing but never filters state.
	if _, err := api.ResumeProcessInstance(ctx, selection.Keys[0], append(keyed, services.WithDryRun())...); err != nil {
		return plan, err
	}
	plan.TargetKeys = slices.Clone(selection.Keys)
	return plan, nil
}

// ExecuteProcessInstanceResumptionPlan revalidates provenance then submits only frozen target keys.
func ExecuteProcessInstanceResumptionPlan(ctx context.Context, api API, log *slog.Logger, plan d.ProcessInstanceResumptionPlan, workers int, opts ...services.CallOption) (d.ProcessInstanceResumptionResults, error) {
	empty := d.ProcessInstanceResumptionResults{Selection: toolx.MapPtr(plan.Selection, copyActionSelection), Items: make([]d.ProcessInstanceResumptionResult, 0)}
	if err := validateActionSelection(api, plan.Selection, plan.SelectionTenant, plan.RequestedKeys, opts...); err != nil {
		return empty, err
	}
	if err := api.ValidateProcessInstanceResumption(); err != nil {
		return empty, err
	}
	invalid := func() (d.ProcessInstanceResumptionResults, error) {
		return empty, fmt.Errorf("%w: resumption scope is inconsistent", d.ErrPrecondition)
	}
	if !distinctNonemptyKeys(plan.TargetKeys) {
		return invalid()
	}
	if plan.Exact {
		if !slices.Equal(plan.TargetKeys, plan.RequestedKeys) || len(plan.Instances) > 0 {
			return invalid()
		}
	} else {
		seen := map[string]bool{}
		targets := typex.Keys{}
		for _, item := range plan.Instances {
			if item.Key == "" || seen[item.Key] {
				return invalid()
			}
			seen[item.Key] = true
			classified := classifyResumptionPlanItem(d.ProcessInstance{Key: item.Key, State: item.ObservedState})
			if classified.Classification != item.Classification {
				return invalid()
			}
			if item.Classification == d.ProcessInstanceResumptionPlanEligible {
				targets = append(targets, item.Key)
			}
		}
		for _, key := range plan.RequestedKeys {
			if !seen[key] {
				return invalid()
			}
		}
		if len(plan.RequestedKeys) == 0 && len(plan.Instances) > 0 {
			return invalid()
		}
		if !slices.Equal(targets, plan.TargetKeys) {
			return invalid()
		}
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if len(plan.TargetKeys) == 0 {
		return empty, nil
	}
	results, err := ResumeProcessInstances(ctx, api, log, slices.Clone(plan.TargetKeys), workers, append(slices.Clone(opts), services.WithIgnoreTenant())...)
	results.Selection = toolx.MapPtr(plan.Selection, copyActionSelection)
	return results, err
}

// copyActionSelection freezes nested criteria, observations and tenant evidence for prepared actions.
func copyActionSelection(selection d.ProcessInstanceSelection) d.ProcessInstanceSelection {
	out := selection
	out.Request = copyProcessInstanceSelectionRequest(selection.Request)
	out.Keys = slices.Clone(selection.Keys)
	out.Items = slices.Clone(selection.Items)
	for i := range out.Items {
		out.Items[i].BusinessId = toolx.CopyPtr(selection.Items[i].BusinessId)
		out.Items[i].Variables = copyProcessInstanceVariableMap(selection.Items[i].Variables)
	}
	out.TenantEvidence.ResolvedTenantIDs = slices.Clone(selection.TenantEvidence.ResolvedTenantIDs)
	out.TenantEvidence.Targets = slices.Clone(selection.TenantEvidence.Targets)
	return out
}
