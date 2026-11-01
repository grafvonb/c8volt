// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
)

// ExecuteProcessInstanceDeletionPlan consumes reviewed membership without recursive live discovery.
// Families run independently; a failed member stops its family before any ancestor submission.
func ExecuteProcessInstanceDeletionPlan(ctx context.Context, api API, plan d.ProcessInstanceSelectedMutationPlan, opts ...services.CallOption) (d.ProcessInstanceDeletionResult, error) {
	result := d.ProcessInstanceDeletionResult{}
	if err := validateSelectedMutationPlan(api, plan, "delete", opts...); err != nil {
		return result, err
	}
	return ExecuteProcessInstanceFrozenDeletion(ctx, api, plan.Families, plan.Request.Workers, opts...)
}

// ExecuteProcessInstanceFrozenDeletion shares exact-member execution with operation-owned cleanup plans.
// The caller validates original selection and eligibility; this boundary validates the frozen graph and action states.
func ExecuteProcessInstanceFrozenDeletion(ctx context.Context, api API, families []d.ProcessInstanceMutationFamily, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceDeletionResult, error) {
	result := d.ProcessInstanceDeletionResult{}
	if err := ValidateProcessInstanceDeletionFamilies(families); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	cfg := services.ApplyCallOptions(opts)
	if len(families) == 0 || cfg.DryRun {
		return result, nil
	}
	// Inspect observations themselves; callers cannot bypass force by clearing preview counts.
	if !cfg.Force {
		for _, family := range families {
			for _, item := range family.Items {
				if !item.State.IsTerminal() {
					return result, fmt.Errorf("%w: nonterminal affected process instances require --force", d.ErrPrecondition)
				}
			}
		}
	}
	executionOpts := append(slices.Clone(opts), services.WithIgnoreTenant())
	workers := toolx.DetermineNoOfWorkers(len(families), wantedWorkers, cfg.NoWorkerLimit)
	slots, err := pool.ExecuteSlice[d.ProcessInstanceMutationFamily, *d.ProcessInstanceDeletionResult](ctx, families, workers, cfg.FailFast, func(ctx context.Context, family d.ProcessInstanceMutationFamily, _ int) (*d.ProcessInstanceDeletionResult, error) {
		familyResult := &d.ProcessInstanceDeletionResult{Roots: []d.Reporter{{Key: family.RootKey}}}
		for _, key := range frozenDeletionOrder(family) {
			if err := ctx.Err(); err != nil {
				return familyResult, err
			}
			response, err := api.DeleteProcessInstanceExact(ctx, key, executionOpts...)
			familyResult.MutationSubmitted = familyResult.MutationSubmitted || response.MutationSubmitted || (response.StatusCode >= 200 && response.StatusCode < 300) || len(d.ProcessInstanceMutationFailures(err)) > 0
			report := d.Reporter{Key: key, Ok: response.Ok && err == nil, StatusCode: response.StatusCode, Status: response.Status}
			familyResult.Members = append(familyResult.Members, report)
			if err == nil && !report.Ok {
				err = fmt.Errorf("%w: planned deletion was rejected", d.ErrPrecondition)
			}
			if err != nil {
				familyResult.Roots[0].StatusCode = response.StatusCode
				return familyResult, err
			}
		}
		familyResult.Roots[0].Ok = true
		familyResult.Roots[0].StatusCode = http.StatusNoContent
		return familyResult, nil
	})
	for i, slot := range slots {
		if slot == nil {
			result.Roots = append(result.Roots, d.Reporter{Key: families[i].RootKey, Status: "not attempted"})
			continue
		}
		if slot.MutationSubmitted {
			result.SubmittedRootKeys = append(result.SubmittedRootKeys, families[i].RootKey)
		}
		result.MutationSubmitted = result.MutationSubmitted || slot.MutationSubmitted
		result.Roots = append(result.Roots, slot.Roots...)
		result.Members = append(result.Members, slot.Members...)
	}
	if err == nil {
		err = ctx.Err()
	}
	return result, err
}

// frozenDeletionOrder derives stable postorder from reviewed parents, retaining sibling traversal order.
func frozenDeletionOrder(family d.ProcessInstanceMutationFamily) []string {
	children := make(map[string][]string, len(family.Items))
	for _, item := range family.Items {
		children[item.ParentKey] = append(children[item.ParentKey], item.Key)
	}
	ordered := make([]string, 0, len(family.Keys))
	var visit func(string)
	visit = func(key string) {
		for _, child := range children[key] {
			visit(child)
		}
		ordered = append(ordered, key)
	}
	visit(family.RootKey)
	return ordered
}

// ValidateProcessInstanceDeletionFamilies rejects duplicate, cyclic, foreign-parent or incomplete frozen membership.
// A cleanup root may retain its observed missing parent; descendants must follow an already observed parent.
func ValidateProcessInstanceDeletionFamilies(families []d.ProcessInstanceMutationFamily) error {
	seen := make(map[string]bool)
	for _, family := range families {
		if family.RootKey == "" || len(family.Keys) == 0 || family.Keys[0] != family.RootKey || len(family.Keys) != len(family.Items) {
			return fmt.Errorf("%w: incomplete frozen deletion family", d.ErrPrecondition)
		}
		parents := make(map[string]bool)
		for i, item := range family.Items {
			if item.Key == "" || item.Key != family.Keys[i] || seen[item.Key] || (i > 0 && !parents[item.ParentKey]) || item.ParentKey == item.Key {
				return fmt.Errorf("%w: inconsistent frozen deletion membership", d.ErrPrecondition)
			}
			seen[item.Key], parents[item.Key] = true, true
		}
		if parents[family.Items[0].ParentKey] {
			return fmt.Errorf("%w: cyclic frozen deletion family", d.ErrPrecondition)
		}
	}
	for _, family := range families {
		if seen[family.Items[0].ParentKey] {
			return fmt.Errorf("%w: frozen family root has an in-scope parent", d.ErrPrecondition)
		}
	}
	return nil
}
