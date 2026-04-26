// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package traversal

import (
	"context"
	"errors"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type Outcome string

const (
	OutcomeComplete   Outcome = "complete"
	OutcomePartial    Outcome = "partial"
	OutcomeUnresolved Outcome = "unresolved"
)

type Mode string

const (
	ModeAncestry    Mode = "ancestry"
	ModeDescendants Mode = "descendants"
	ModeFamily      Mode = "family"
)

type MissingAncestor struct {
	Key      string
	StartKey string
}

type Result struct {
	Mode             Mode
	StartKey         string
	RootKey          string
	Keys             []string
	Edges            map[string][]string
	Chain            map[string]d.ProcessInstance
	MissingAncestors []MissingAncestor
	Warning          string
	Outcome          Outcome
}

func (r Result) HasActionableResults() bool {
	return len(r.Keys) > 0 || len(r.Chain) > 0
}

type LegacyAPI interface {
	Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
}

func BuildAncestryResult(ctx context.Context, api LegacyAPI, startKey string, opts ...services.CallOption) (Result, error) {
	rootOrBoundaryKey, path, chain, err := api.Ancestry(ctx, startKey, opts...)
	result := Result{
		Mode:     ModeAncestry,
		StartKey: startKey,
		Keys:     append([]string(nil), path...),
		Chain:    chain,
	}
	if len(path) > 0 {
		result.RootKey = path[len(path)-1]
	} else if err == nil {
		result.RootKey = rootOrBoundaryKey
	}
	if err == nil {
		result.Outcome = OutcomeComplete
		return result, nil
	}
	if errors.Is(err, services.ErrOrphanedInstance) {
		result.MissingAncestors = []MissingAncestor{{Key: rootOrBoundaryKey, StartKey: startKey}}
		result.Warning = "one or more parent process instances were not found"
		if result.HasActionableResults() {
			result.Outcome = OutcomePartial
		} else {
			result.Outcome = OutcomeUnresolved
		}
		return result, nil
	}
	if result.HasActionableResults() {
		result.Outcome = OutcomeUnresolved
	}
	return result, err
}

func BuildDescendantsResult(ctx context.Context, api LegacyAPI, rootKey string, opts ...services.CallOption) (Result, error) {
	keys, edges, chain, err := api.Descendants(ctx, rootKey, opts...)
	result := Result{
		Mode:     ModeDescendants,
		StartKey: rootKey,
		RootKey:  rootKey,
		Keys:     append([]string(nil), keys...),
		Edges:    edges,
		Chain:    chain,
	}
	if err == nil {
		result.Outcome = OutcomeComplete
		return result, nil
	}
	if result.HasActionableResults() {
		result.Outcome = OutcomeUnresolved
	}
	return result, err
}

func BuildFamilyResult(ctx context.Context, api LegacyAPI, startKey string, opts ...services.CallOption) (Result, error) {
	ancestry, err := BuildAncestryResult(ctx, api, startKey, opts...)
	if err != nil {
		return Result{}, err
	}
	if ancestry.RootKey == "" {
		return ancestry, nil
	}

	descendants, err := BuildDescendantsResult(ctx, api, ancestry.RootKey, opts...)
	if err != nil {
		return Result{}, err
	}

	result := descendants
	result.Mode = ModeFamily
	result.StartKey = startKey
	result.MissingAncestors = ancestry.MissingAncestors
	result.Warning = ancestry.Warning
	result.Outcome = descendants.Outcome
	if len(result.MissingAncestors) > 0 {
		result.Outcome = OutcomePartial
	}
	return result, nil
}
