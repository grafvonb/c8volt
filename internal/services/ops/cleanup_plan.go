// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// cleanupSelectionRequest normalizes only the page control used by existing cleanup discovery.
func cleanupSelectionRequest(filter d.ProcessInstanceFilter, size, limit int32) d.ProcessInstanceSelectionRequest {
	if size <= 0 || size > consts.MaxPISearchSize {
		size = consts.MaxPISearchSize
	}
	return d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: filter, Page: d.ProcessInstancePageRequest{Size: size}, Limit: limit}}
}

// copyCleanupScope isolates all nested observations and criteria at the service boundary.
func copyCleanupScope(scope d.CleanupFrozenScope) d.CleanupFrozenScope {
	scope.Selection = pisvc.CopyProcessInstanceSelection(scope.Selection)
	scope.Seeds = slices.Clone(scope.Seeds)
	scope.Ancestries = copyCleanupFamilies(scope.Ancestries)
	scope.Families = copyCleanupFamilies(scope.Families)
	return scope
}

// copyCleanupFamilies preserves observed parent relationships and nested variables without new reads.
func copyCleanupFamilies(families []d.ProcessInstanceMutationFamily) []d.ProcessInstanceMutationFamily {
	families = slices.Clone(families)
	for i := range families {
		families[i].Keys = slices.Clone(families[i].Keys)
		families[i].Items = pisvc.CopyProcessInstanceSelection(d.ProcessInstanceSelection{Items: families[i].Items}).Items
	}
	return families
}

// cleanupFamily converts completed traversal observations into ordered frozen membership.
func cleanupFamily(result traversal.Result) (d.ProcessInstanceMutationFamily, error) {
	family := d.ProcessInstanceMutationFamily{RootKey: result.RootKey, Keys: slices.Clone(result.Keys)}
	for _, key := range result.Keys {
		item, ok := result.Chain[key]
		if !ok {
			return family, fmt.Errorf("%w: cleanup traversal omitted an observation", d.ErrMalformedResponse)
		}
		family.Items = append(family.Items, item)
	}
	return copyCleanupFamilies([]d.ProcessInstanceMutationFamily{family})[0], nil
}

// planCleanupFamilies expands each eligible root once, keeping retention's skipped nonfinal roots separate.
func (s *Service) planCleanupFamilies(ctx context.Context, scope d.CleanupFrozenScope, retention bool, workers int, opts ...services.CallOption) (d.CleanupFrozenScope, error) {
	scope.ExpansionOutcome = d.TraversalOutcomeUnresolved
	expansionOpts := append(slices.Clone(opts), services.WithIgnoreTenant())
	cfg := services.ApplyCallOptions(opts)
	roots := slices.Clone(scope.Seeds)
	if retention {
		paths, err := pool.ExecuteSlice(ctx, scope.Seeds, toolx.DetermineNoOfWorkers(len(scope.Seeds), workers, cfg.NoWorkerLimit), cfg.FailFast, func(ctx context.Context, key string, _ int) (traversal.Result, error) {
			return s.piAPI.AncestryResult(ctx, key, expansionOpts...)
		})
		roots = nil
		for _, path := range paths {
			if path.Outcome != traversal.OutcomeComplete || len(path.MissingAncestors) > 0 || path.Warning != "" || len(path.Keys) == 0 {
				if err == nil {
					err = fmt.Errorf("%w: incomplete retention ancestry", d.ErrPrecondition)
				}
				continue
			}
			// Ancestry keys run seed-to-root; frozen families always run parent-before-child.
			path.Keys = slices.Clone(path.Keys)
			slices.Reverse(path.Keys)
			family, mapErr := cleanupFamily(path)
			if mapErr != nil {
				return scope, mapErr
			}
			scope.Ancestries = append(scope.Ancestries, family)
			if family.Items[0].State.IsTerminal() {
				roots = append(roots, family.RootKey)
			}
		}
		if err != nil {
			return scope, err
		}
	}
	roots = typex.Keys(roots).Unique()
	results, err := pool.ExecuteSlice(ctx, roots, toolx.DetermineNoOfWorkers(len(roots), workers, cfg.NoWorkerLimit), cfg.FailFast, func(ctx context.Context, key string, _ int) (traversal.Result, error) {
		return s.piAPI.DescendantsResult(ctx, key, expansionOpts...)
	})
	for _, result := range results {
		if result.Outcome != traversal.OutcomeComplete || len(result.MissingAncestors) > 0 || result.Warning != "" {
			if err == nil {
				err = fmt.Errorf("%w: incomplete cleanup descendants", d.ErrPrecondition)
			}
		}
		family, mapErr := cleanupFamily(result)
		if mapErr != nil {
			return scope, mapErr
		}
		scope.Families = append(scope.Families, family)
	}
	if err != nil {
		return scope, err
	}
	scope.ExpansionOutcome = d.TraversalOutcomeComplete
	return scope, nil
}

// validateCleanupScope checks original capability/context, eligibility and complete frozen graphs before any shortcut.
func (s *Service) validateCleanupScope(scope d.CleanupFrozenScope, expected d.ProcessInstanceSelectionRequest, retention bool, opts ...services.CallOption) error {
	if err := pisvc.ValidateProcessInstanceSelection(s.piAPI, scope.Selection, scope.Tenant, opts...); err != nil {
		return err
	}
	invalid := func() error { return fmt.Errorf("%w: cleanup plan is incomplete or inconsistent", d.ErrPrecondition) }
	if !reflect.DeepEqual(expected, scope.Selection.Request) || scope.ExpansionOutcome != d.TraversalOutcomeComplete {
		return invalid()
	}
	if err := pisvc.ValidateProcessInstanceDeletionFamilies(scope.Families); err != nil {
		return err
	}
	selected := make(map[string]d.ProcessInstance)
	for _, item := range scope.Selection.Items {
		selected[item.Key] = item
	}
	seenSeeds := map[string]bool{}
	for _, key := range scope.Seeds {
		item, ok := selected[key]
		if !ok || seenSeeds[key] || (!retention && item.ParentKey == "") {
			return invalid()
		}
		seenSeeds[key] = true
	}
	var roots typex.Keys
	if retention {
		if !slices.Equal(scope.Seeds, scope.Selection.Keys) || len(scope.Ancestries) != len(scope.Seeds) {
			return invalid()
		}
		for i, path := range scope.Ancestries {
			if err := pisvc.ValidateProcessInstanceDeletionFamilies([]d.ProcessInstanceMutationFamily{path}); err != nil {
				return err
			}
			if path.Items[0].ParentKey != "" || path.Keys[len(path.Keys)-1] != scope.Seeds[i] {
				return invalid()
			}
			item := selected[scope.Seeds[i]]
			last := path.Items[len(path.Items)-1]
			if item.EndDate == "" || item.ParentKey != last.ParentKey || item.TenantId != last.TenantId {
				return invalid()
			}
			if path.Items[0].State.IsTerminal() {
				roots = append(roots, path.RootKey)
			}
		}
	} else {
		if len(scope.Ancestries) > 0 {
			return invalid()
		}
		roots = slices.Clone(scope.Seeds)
	}
	roots = roots.Unique()
	if len(roots) != len(scope.Families) {
		return invalid()
	}
	members := map[string]d.ProcessInstance{}
	for i, family := range scope.Families {
		if family.RootKey != roots[i] {
			return invalid()
		}
		if retention {
			if family.Items[0].ParentKey != "" || !family.Items[0].State.IsTerminal() {
				return invalid()
			}
		} else {
			seed := selected[family.RootKey]
			if seed.ParentKey != family.Items[0].ParentKey || seed.TenantId != family.Items[0].TenantId {
				return invalid()
			}
		}
		for _, item := range family.Items {
			members[item.Key] = item
		}
	}
	for _, key := range scope.Seeds {
		item, ok := members[key]
		if !ok {
			if !retention {
				return invalid()
			}
			continue
		}
		if item.ParentKey != selected[key].ParentKey || item.TenantId != selected[key].TenantId {
			return invalid()
		}
	}
	if retention {
		for _, path := range scope.Ancestries {
			if !path.Items[0].State.IsTerminal() {
				continue
			}
			for _, item := range path.Items {
				member, ok := members[item.Key]
				if !ok || member.ParentKey != item.ParentKey || member.TenantId != item.TenantId {
					return invalid()
				}
			}
		}
	}
	return nil
}

// cleanupExpansion builds existing preview facts solely from reviewed observations.
func cleanupExpansion(scope d.CleanupFrozenScope) d.DryRunPIKeyExpansion {
	out := d.DryRunPIKeyExpansion{Outcome: scope.ExpansionOutcome}
	var observations []d.ProcessInstance
	for _, family := range scope.Families {
		out.Roots = append(out.Roots, family.RootKey)
		out.Collected = append(out.Collected, family.Keys...)
		for _, item := range family.Items {
			observations = append(observations, item)
			if !item.State.IsTerminal() {
				out.RequiresCancelBeforeDelete = append(out.RequiresCancelBeforeDelete, item)
			}
		}
	}
	for _, item := range scope.Selection.Items {
		if slices.Contains(scope.Seeds, item.Key) && item.State.IsTerminal() {
			out.SelectedFinalState = append(out.SelectedFinalState, item)
		}
	}
	out.TenantEvidence = opsTenantEvidenceFromProcessInstances(observations)
	return out
}
