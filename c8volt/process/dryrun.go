// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"fmt"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

type legacyDryRunTraversalOnly interface {
	LegacyDryRunTraversalOnly() bool
}

// DryRunCancelOrDeleteGetPIKeys returns the root keys and all collected descendant keys that would be affected.
// keys are the user-selected process-instance keys; opts controls traversal verbosity and behavior through facade options.
func (c *client) DryRunCancelOrDeleteGetPIKeys(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (roots types.Keys, collected types.Keys, err error) {
	plan, err := c.DryRunCancelOrDeletePlan(ctx, keys, wantedWorkers, opts...)
	if err != nil {
		return nil, nil, err
	}
	return plan.Roots, plan.Collected, nil
}

// DryRunCancelOrDeletePlan expands selected process-instance keys into the cancellation/deletion dependency plan.
// keys may contain children; the returned plan reports unique roots, descendants, partial traversal warnings, and missing ancestors.
func (c *client) DryRunCancelOrDeletePlan(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DryRunPIKeyExpansion, error) {
	if legacyOnly, ok := c.piApi.(legacyDryRunTraversalOnly); ok && legacyOnly.LegacyDryRunTraversalOnly() {
		return c.dryRunCancelOrDeletePlanLegacy(ctx, keys, wantedWorkers, opts...)
	}

	var roots types.Keys
	var collected types.Keys
	cfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	ancestryWorkers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	ancestryResults, err := pool.ExecuteSlice[string, TraversalResult](ctx, ukeys, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (TraversalResult, error) {
		result, err := c.AncestryResult(ctx, key, opts...)
		if err != nil {
			return TraversalResult{}, ferr.FromDomain(err)
		}
		return result, nil
	})
	if err != nil {
		return DryRunPIKeyExpansion{}, err
	}

	for _, result := range ancestryResults {
		if result.RootKey != "" {
			roots = append(roots, result.RootKey)
		}
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantResults, err := pool.ExecuteSlice[string, TraversalResult](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (TraversalResult, error) {
		result, err := c.DescendantsResult(ctx, root, opts...)
		if err != nil {
			return TraversalResult{}, ferr.FromDomain(err)
		}
		return result, nil
	})
	if err != nil {
		return DryRunPIKeyExpansion{}, err
	}
	for _, result := range descendantResults {
		collected = append(collected, result.Keys...)
	}

	ancestryWarning, ancestryMissing, ancestryOutcome := mapDryRunTraversalWarning(ancestryResults)
	descendantsWarning, descendantsMissing, descendantsOutcome := mapDryRunTraversalWarning(descendantResults)

	warning := ancestryWarning
	if warning == "" {
		warning = descendantsWarning
	}
	outcome := ancestryOutcome
	if outcome == TraversalOutcomeComplete {
		outcome = descendantsOutcome
	} else if descendantsOutcome == TraversalOutcomePartial {
		outcome = TraversalOutcomePartial
	}

	collected = collected.Unique()
	plan := DryRunPIKeyExpansion{
		Roots:                      roots,
		Collected:                  collected,
		SelectedFinalState:         selectedFinalStateProcessInstances(keys, ancestryResults),
		RequiresCancelBeforeDelete: nonFinalProcessInstances(collected, descendantResults),
		MissingAncestors:           uniqueMissingAncestors(append(ancestryMissing, descendantsMissing...)),
		Warning:                    warning,
		Outcome:                    outcome,
	}

	return plan, validateDryRunPIKeyExpansion(plan)
}

// nonFinalProcessInstances returns in-scope instances that still require a terminal state before deletion.
func nonFinalProcessInstances(keys types.Keys, results []TraversalResult) []ProcessInstance {
	if len(keys) == 0 || len(results) == 0 {
		return nil
	}

	byKey := make(map[string]ProcessInstance)
	for _, result := range results {
		for key, pi := range result.Chain {
			if _, ok := byKey[key]; !ok {
				byKey[key] = pi
			}
		}
	}

	out := make([]ProcessInstance, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		pi, ok := byKey[key]
		if !ok || pi.State == "" || pi.State.IsTerminal() {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pi)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// selectedFinalStateProcessInstances returns selected instances already unaffected by cancellation.
func selectedFinalStateProcessInstances(keys types.Keys, results []TraversalResult) []ProcessInstance {
	if len(keys) == 0 || len(results) == 0 {
		return nil
	}
	byStartKey := make(map[string]TraversalResult, len(results))
	for _, result := range results {
		byStartKey[result.StartKey] = result
	}

	out := make([]ProcessInstance, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		result, ok := byStartKey[key]
		if !ok || result.Chain == nil {
			continue
		}
		pi, ok := result.Chain[key]
		if !ok || !pi.State.IsTerminal() {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pi)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// dryRunCancelOrDeletePlanLegacy preserves the older traversal contract for services that cannot report structured partial results.
// It treats successful ancestry and descendant calls as a complete plan and leaves missing-ancestor details unavailable.
func (c *client) dryRunCancelOrDeletePlanLegacy(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DryRunPIKeyExpansion, error) {
	var roots types.Keys
	var collected types.Keys
	cfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()

	ancestryWorkers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	legacyRoots, err := pool.ExecuteSlice[string, string](ctx, ukeys, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (string, error) {
		rootKey, _, _, err := c.Ancestry(ctx, key, opts...)
		if err != nil {
			return "", err
		}
		return rootKey, nil
	})
	if err != nil {
		return DryRunPIKeyExpansion{}, err
	}
	for _, rootKey := range legacyRoots {
		if rootKey != "" {
			roots = append(roots, rootKey)
		}
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantLists, err := pool.ExecuteSlice[string, types.Keys](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (types.Keys, error) {
		desc, _, _, err := c.Descendants(ctx, root, opts...)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})
	if err != nil {
		return DryRunPIKeyExpansion{}, err
	}
	for _, desc := range descendantLists {
		collected = append(collected, desc...)
	}

	return DryRunPIKeyExpansion{
		Roots:     roots,
		Collected: collected.Unique(),
		Outcome:   TraversalOutcomeComplete,
	}, nil
}

// validateDryRunPIKeyExpansion rejects unresolved dry-run plans that found no actionable process instances.
func validateDryRunPIKeyExpansion(plan DryRunPIKeyExpansion) error {
	if plan.HasActionableResults() {
		return nil
	}
	if plan.Outcome != TraversalOutcomeUnresolved {
		return nil
	}

	return fmt.Errorf("%w: no process instances resolved during dependency expansion", services.ErrOrphanedInstance)
}
