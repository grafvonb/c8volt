// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

type legacyDryRunTraversalOnly interface {
	LegacyDryRunTraversalOnly() bool
}

func DryRunCancelOrDeletePlan(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.DryRunPIKeyExpansion, error) {
	if legacyOnly, ok := api.(legacyDryRunTraversalOnly); ok && legacyOnly.LegacyDryRunTraversalOnly() {
		return dryRunCancelOrDeletePlanLegacy(ctx, api, keys, wantedWorkers, opts...)
	}

	var roots typex.Keys
	var collected typex.Keys
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	ancestryWorkers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	ancestryResults, err := pool.ExecuteSlice[string, pitraversal.Result](ctx, ukeys, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (pitraversal.Result, error) {
		return api.AncestryResult(ctx, key, opts...)
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, result := range ancestryResults {
		if result.RootKey != "" {
			roots = append(roots, result.RootKey)
		}
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantResults, err := pool.ExecuteSlice[string, pitraversal.Result](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (pitraversal.Result, error) {
		return api.DescendantsResult(ctx, root, opts...)
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
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
	if outcome == d.TraversalOutcomeComplete {
		outcome = descendantsOutcome
	} else if descendantsOutcome == d.TraversalOutcomePartial {
		outcome = d.TraversalOutcomePartial
	}

	collected = collected.Unique()
	plan := d.DryRunPIKeyExpansion{
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

func dryRunCancelOrDeletePlanLegacy(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.DryRunPIKeyExpansion, error) {
	var roots typex.Keys
	var collected typex.Keys
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	ancestryWorkers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	legacyRoots, err := pool.ExecuteSlice[string, string](ctx, ukeys, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (string, error) {
		rootKey, _, _, err := api.Ancestry(ctx, key, opts...)
		return rootKey, err
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, rootKey := range legacyRoots {
		if rootKey != "" {
			roots = append(roots, rootKey)
		}
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantLists, err := pool.ExecuteSlice[string, typex.Keys](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (typex.Keys, error) {
		desc, _, _, err := api.Descendants(ctx, root, opts...)
		return desc, err
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, desc := range descendantLists {
		collected = append(collected, desc...)
	}
	return d.DryRunPIKeyExpansion{
		Roots:     roots,
		Collected: collected.Unique(),
		Outcome:   d.TraversalOutcomeComplete,
	}, nil
}

func mapDryRunTraversalWarning(results []pitraversal.Result) (warning string, missing []d.MissingAncestor, outcome d.TraversalOutcome) {
	outcome = d.TraversalOutcomeComplete
	for _, result := range results {
		if len(result.MissingAncestors) > 0 {
			missing = append(missing, domainMissingAncestors(result.MissingAncestors)...)
		}
		if result.Warning != "" && warning == "" {
			warning = result.Warning
		}
		switch result.Outcome {
		case pitraversal.OutcomeUnresolved:
			if outcome == d.TraversalOutcomeComplete {
				outcome = d.TraversalOutcomeUnresolved
			}
		case pitraversal.OutcomePartial:
			outcome = d.TraversalOutcomePartial
		}
	}
	if len(missing) > 0 && warning == "" {
		warning = "one or more parent process instances were not found"
	}
	return warning, missing, outcome
}

func domainMissingAncestors(items []pitraversal.MissingAncestor) []d.MissingAncestor {
	out := make([]d.MissingAncestor, 0, len(items))
	for _, item := range items {
		out = append(out, d.MissingAncestor{Key: item.Key, StartKey: item.StartKey})
	}
	return out
}

func uniqueMissingAncestors(items []d.MissingAncestor) []d.MissingAncestor {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]d.MissingAncestor, 0, len(items))
	for _, item := range items {
		key := item.Key + "\x00" + item.StartKey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func nonFinalProcessInstances(keys typex.Keys, results []pitraversal.Result) []d.ProcessInstance {
	if len(keys) == 0 || len(results) == 0 {
		return nil
	}
	byKey := make(map[string]d.ProcessInstance)
	for _, result := range results {
		for key, pi := range result.Chain {
			if _, ok := byKey[key]; !ok {
				byKey[key] = pi
			}
		}
	}
	out := make([]d.ProcessInstance, 0, len(keys))
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

func selectedFinalStateProcessInstances(keys typex.Keys, results []pitraversal.Result) []d.ProcessInstance {
	if len(keys) == 0 || len(results) == 0 {
		return nil
	}
	byStartKey := make(map[string]pitraversal.Result, len(results))
	for _, result := range results {
		byStartKey[result.StartKey] = result
	}
	out := make([]d.ProcessInstance, 0, len(keys))
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

func validateDryRunPIKeyExpansion(plan d.DryRunPIKeyExpansion) error {
	if plan.HasActionableResults() || plan.Outcome != d.TraversalOutcomeUnresolved {
		return nil
	}
	return fmt.Errorf("%w: no process instances resolved during dependency expansion", services.ErrOrphanedInstance)
}
