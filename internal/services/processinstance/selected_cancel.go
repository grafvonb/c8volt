// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceSelectedMutation finishes bounded discovery before expanding each distinct family once.
// Only the selection uses tenant filtering; observed technical keys authorize subsequent family reads.
func PlanProcessInstanceSelectedMutation(ctx context.Context, api API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSelectedMutationRequest, opts ...services.CallOption) (d.ProcessInstanceSelectedMutationPlan, error) {
	request.Selection = copyProcessInstanceSelectionRequest(request.Selection)
	plan := d.ProcessInstanceSelectedMutationPlan{Request: request, Expansion: d.DryRunPIKeyExpansion{Outcome: d.TraversalOutcomeUnresolved}}
	if err := api.ValidateProcessInstanceSearchFilter(request.Selection.SearchRequest.Filter); err != nil {
		return plan, err
	}
	if (request.Operation != "cancel" && request.Operation != "delete" && request.Operation != "resolve") || request.Selection.Cardinality != d.ProcessInstanceSelectionCollection {
		return plan, fmt.Errorf("%w: selected mutation requires a cancel, delete or resolve collection", d.ErrValidation)
	}
	plan.Tenant = api.ProcessInstanceSearchTenant(opts...)
	selection, err := ResolveProcessInstanceSelection(ctx, api, incAPI, request.Selection, opts...)
	plan.Selection = selection
	if err != nil {
		return plan, err
	}
	if len(selection.Keys) == 0 {
		plan.Expansion.Outcome = d.TraversalOutcomeComplete
		return plan, nil
	}
	expansionOpts := append(slices.Clone(opts), services.WithIgnoreTenant())
	expansion, families, err := dryRunCancelOrDeletePlan(ctx, api, selection.Keys, request.Workers, expansionOpts...)
	plan.Expansion = expansion
	for _, family := range families {
		frozen := d.ProcessInstanceMutationFamily{RootKey: family.RootKey, Keys: slices.Clone(family.Keys)}
		for _, key := range family.Keys {
			item := family.Chain[key]
			item.BusinessId = toolx.CopyPtr(item.BusinessId)
			item.Variables = copyProcessInstanceVariableMap(item.Variables)
			frozen.Items = append(frozen.Items, item)
		}
		plan.Families = append(plan.Families, frozen)
	}
	if err != nil {
		return plan, err
	}
	if err := validateSelectedMutationPlan(api, plan, request.Operation, opts...); err != nil {
		plan.Expansion.Outcome = d.TraversalOutcomeUnresolved
		return plan, err
	}
	return plan, nil
}

// ExecuteProcessInstanceCancellationPlan validates received evidence before even an empty return,
// then delegates only frozen roots to existing cancellation, state checks and wait mechanics.
func ExecuteProcessInstanceCancellationPlan(ctx context.Context, api API, log *slog.Logger, plan d.ProcessInstanceSelectedMutationPlan, opts ...services.CallOption) ([]d.Reporter, error) {
	if err := validateSelectedMutationPlan(api, plan, "cancel", opts...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(plan.Selection.Keys) == 0 || services.ApplyCallOptions(opts).DryRun {
		return nil, nil
	}
	executionOpts := append(slices.Clone(opts), services.WithIgnoreTenant())
	reports, err := CancelProcessInstances(ctx, api, log, slices.Clone(plan.Expansion.Roots), plan.Request.Workers, len(plan.Expansion.Collected), executionOpts...)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		for _, report := range reports {
			if !report.Ok {
				err = fmt.Errorf("%w: one or more planned roots could not be canceled", d.ErrPrecondition)
				break
			}
		}
	}
	return reports, err
}

// validateSelectedMutationPlan checks capability first, then independent completion and graph evidence.
// Plans are copied run artifacts; consistency checks cannot authenticate a fabricated external snapshot.
func validateSelectedMutationPlan(api API, plan d.ProcessInstanceSelectedMutationPlan, operation string, opts ...services.CallOption) error {
	if err := api.ValidateProcessInstanceSearchFilter(plan.Request.Selection.SearchRequest.Filter); err != nil {
		return err
	}
	if err := api.ValidateProcessInstanceSearchFilter(plan.Selection.Request.SearchRequest.Filter); err != nil {
		return err
	}
	invalid := func() error {
		return fmt.Errorf("%w: selected mutation plan is incomplete or inconsistent", d.ErrPrecondition)
	}
	selection := plan.Selection
	request := plan.Request.Selection
	if plan.Request.Operation != operation || request.Cardinality != d.ProcessInstanceSelectionCollection || !reflect.DeepEqual(request, selection.Request) || selection.Origin != "search" {
		return invalid()
	}
	if plan.Tenant != api.ProcessInstanceSearchTenant(opts...) || selection.IgnoreTenant != services.ApplyCallOptions(opts).IgnoreTenant {
		return invalid()
	}
	if request.SearchRequest.Limit < 0 || request.SearchRequest.Page.From != 0 || request.SearchRequest.Page.After != "" || selection.Pages < 1 {
		return invalid()
	}
	switch selection.Completion {
	case d.ProcessInstanceSearchCompletionExhausted:
	case d.ProcessInstanceSearchCompletionLimitReached:
		if request.SearchRequest.Limit <= 0 || len(selection.Keys) != int(request.SearchRequest.Limit) {
			return invalid()
		}
	default:
		return invalid()
	}
	if request.SearchRequest.Limit > 0 && len(selection.Keys) > int(request.SearchRequest.Limit) {
		return invalid()
	}
	expansion := plan.Expansion
	if expansion.Outcome != d.TraversalOutcomeComplete || len(expansion.MissingAncestors) > 0 || expansion.Warning != "" {
		return invalid()
	}
	if !distinctNonemptyKeys(selection.Keys) || !distinctNonemptyKeys(expansion.Roots) || !distinctNonemptyKeys(expansion.Collected) || len(selection.Keys) != len(selection.Items) || len(expansion.Roots) != len(plan.Families) {
		return invalid()
	}
	selectedKeys := make(map[string]bool, len(selection.Keys))
	for _, key := range selection.Keys {
		selectedKeys[key] = true
	}
	members := map[string]d.ProcessInstance{}
	var union typex.Keys
	for i, family := range plan.Families {
		if family.RootKey != expansion.Roots[i] || len(family.Keys) == 0 || family.Keys[0] != family.RootKey || len(family.Keys) != len(family.Items) || !distinctNonemptyKeys(family.Keys) {
			return invalid()
		}
		parents := map[string]bool{}
		selectedFamily := false
		for j, item := range family.Items {
			if item.Key != family.Keys[j] {
				return invalid()
			}
			if j == 0 {
				if item.ParentKey != "" {
					return invalid()
				}
			} else if !parents[item.ParentKey] {
				return invalid()
			}
			if _, exists := members[item.Key]; exists {
				return invalid()
			}
			parents[item.Key] = true
			selectedFamily = selectedFamily || selectedKeys[item.Key]
			members[item.Key] = item
			union = append(union, item.Key)
		}
		if !selectedFamily {
			return invalid()
		}
	}
	if !slices.Equal(union, expansion.Collected) {
		return invalid()
	}
	for i, item := range selection.Items {
		if item.Key != selection.Keys[i] {
			return invalid()
		}
		member, ok := members[item.Key]
		if !ok || member.ParentKey != item.ParentKey || member.TenantId != item.TenantId {
			return invalid()
		}
		if plan.Tenant != "" && item.TenantId != plan.Tenant {
			return invalid()
		}
	}
	if len(selection.Keys) == 0 && (len(union) > 0 || len(plan.Families) > 0) {
		return invalid()
	}
	return nil
}

// distinctNonemptyKeys rejects malformed or repeated executable identities without changing order.
func distinctNonemptyKeys(keys typex.Keys) bool {
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if key == "" || seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}
