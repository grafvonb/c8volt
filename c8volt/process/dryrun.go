package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type legacyDryRunTraversalOnly interface {
	LegacyDryRunTraversalOnly() bool
}

func (c *client) DryRunCancelOrDeleteGetPIKeys(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (roots types.Keys, collected types.Keys, err error) {
	plan, err := c.DryRunCancelOrDeletePlan(ctx, keys, opts...)
	if err != nil {
		return nil, nil, err
	}
	return plan.Roots, plan.Collected, nil
}

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

	return DryRunPIKeyExpansion{
		Roots:            roots,
		Collected:        collected.Unique(),
		MissingAncestors: uniqueMissingAncestors(append(ancestryMissing, descendantsMissing...)),
		Warning:          warning,
		Outcome:          outcome,
	}, nil
}

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
