package process

import (
	"context"
	"fmt"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/internal/services"
	types "github.com/grafvonb/c8volt/typex"
)

type legacyDryRunTraversalOnly interface {
	LegacyDryRunTraversalOnly() bool
}

// DryRunCancelOrDeleteGetPIKeys returns the root keys and all collected descendant keys that would be affected.
// keys are the user-selected process-instance keys; opts controls traversal verbosity and behavior through facade options.
func (c *client) DryRunCancelOrDeleteGetPIKeys(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (roots types.Keys, collected types.Keys, err error) {
	plan, err := c.DryRunCancelOrDeletePlan(ctx, keys, opts...)
	if err != nil {
		return nil, nil, err
	}
	return plan.Roots, plan.Collected, nil
}

// DryRunCancelOrDeletePlan expands selected process-instance keys into the cancellation/deletion dependency plan.
// keys may contain children; the returned plan reports unique roots, descendants, partial traversal warnings, and missing ancestors.
func (c *client) DryRunCancelOrDeletePlan(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (DryRunPIKeyExpansion, error) {
	if legacyOnly, ok := c.piApi.(legacyDryRunTraversalOnly); ok && legacyOnly.LegacyDryRunTraversalOnly() {
		return c.dryRunCancelOrDeletePlanLegacy(ctx, keys, opts...)
	}

	var roots types.Keys
	var collected types.Keys
	var ancestryResults []TraversalResult
	for _, key := range keys {
		result, err := c.AncestryResult(ctx, key, opts...)
		if err != nil {
			return DryRunPIKeyExpansion{}, ferr.FromDomain(err)
		}
		ancestryResults = append(ancestryResults, result)
		if result.RootKey != "" {
			roots = append(roots, result.RootKey)
		}
	}
	roots = roots.Unique()

	var descendantResults []TraversalResult
	for _, root := range roots {
		result, err := c.DescendantsResult(ctx, root, opts...)
		if err != nil {
			return DryRunPIKeyExpansion{}, ferr.FromDomain(err)
		}
		descendantResults = append(descendantResults, result)
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

	plan := DryRunPIKeyExpansion{
		Roots:            roots,
		Collected:        collected.Unique(),
		MissingAncestors: uniqueMissingAncestors(append(ancestryMissing, descendantsMissing...)),
		Warning:          warning,
		Outcome:          outcome,
	}

	return plan, validateDryRunPIKeyExpansion(plan)
}

// dryRunCancelOrDeletePlanLegacy preserves the older traversal contract for services that cannot report structured partial results.
// It treats successful ancestry and descendant calls as a complete plan and leaves missing-ancestor details unavailable.
func (c *client) dryRunCancelOrDeletePlanLegacy(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (DryRunPIKeyExpansion, error) {
	var roots types.Keys
	var collected types.Keys

	for _, key := range keys {
		rootKey, _, _, err := c.Ancestry(ctx, key, opts...)
		if err != nil {
			return DryRunPIKeyExpansion{}, err
		}
		roots = append(roots, rootKey)
	}
	roots = roots.Unique()

	for _, root := range roots {
		desc, _, _, err := c.Descendants(ctx, root, opts...)
		if err != nil {
			return DryRunPIKeyExpansion{}, err
		}
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
