// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
)

type Walker interface {
	Ancestry(ctx context.Context, startKey string, opts ...options.FacadeOption) (rootKey string, path []string, chain map[string]ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...options.FacadeOption) (desc []string, edges map[string][]string, chain map[string]ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...options.FacadeOption) (fam []string, edges map[string][]string, chain map[string]ProcessInstance, err error)
	AncestryResult(ctx context.Context, startKey string, opts ...options.FacadeOption) (TraversalResult, error)
	DescendantsResult(ctx context.Context, rootKey string, opts ...options.FacadeOption) (TraversalResult, error)
	FamilyResult(ctx context.Context, startKey string, opts ...options.FacadeOption) (TraversalResult, error)
}

func AsWalker(client API) (Walker, bool) {
	w, ok := client.(Walker)
	return w, ok
}

func (c *client) Ancestry(ctx context.Context, startKey string, opts ...options.FacadeOption) (string, []string, map[string]ProcessInstance, error) {
	rootKey, path, dchain, err := c.piApi.Ancestry(ctx, startKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return rootKey, path, fromDomainProcessInstanceMap(dchain), err
}

func (c *client) Descendants(ctx context.Context, rootKey string, opts ...options.FacadeOption) ([]string, map[string][]string, map[string]ProcessInstance, error) {
	desc, edges, dchain, err := c.piApi.Descendants(ctx, rootKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return desc, edges, fromDomainProcessInstanceMap(dchain), err
}

func (c *client) Family(ctx context.Context, startKey string, opts ...options.FacadeOption) ([]string, map[string][]string, map[string]ProcessInstance, error) {
	fam, edges, dchain, err := c.piApi.Family(ctx, startKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return fam, edges, fromDomainProcessInstanceMap(dchain), err
}

func (c *client) AncestryResult(ctx context.Context, startKey string, opts ...options.FacadeOption) (TraversalResult, error) {
	got, err := c.piApi.AncestryResult(ctx, startKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return mapTraversalResult(got), err
}

func (c *client) DescendantsResult(ctx context.Context, rootKey string, opts ...options.FacadeOption) (TraversalResult, error) {
	got, err := c.piApi.DescendantsResult(ctx, rootKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return mapTraversalResult(got), err
}

func (c *client) FamilyResult(ctx context.Context, startKey string, opts ...options.FacadeOption) (TraversalResult, error) {
	got, err := c.piApi.FamilyResult(ctx, startKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return mapTraversalResult(got), err
}

func mapTraversalResult(in pitraversal.Result) TraversalResult {
	return TraversalResult{
		Mode:             TraversalMode(in.Mode),
		StartKey:         in.StartKey,
		RootKey:          in.RootKey,
		Keys:             append([]string(nil), in.Keys...),
		Edges:            in.Edges,
		Chain:            fromDomainProcessInstanceMap(in.Chain),
		MissingAncestors: mapMissingAncestors(in.MissingAncestors),
		Warning:          in.Warning,
		Outcome:          TraversalOutcome(in.Outcome),
	}
}

func mapMissingAncestors(in []pitraversal.MissingAncestor) []MissingAncestor {
	if len(in) == 0 {
		return nil
	}
	out := make([]MissingAncestor, len(in))
	for i, item := range in {
		out[i] = MissingAncestor{
			Key:      item.Key,
			StartKey: item.StartKey,
		}
	}
	return out
}
