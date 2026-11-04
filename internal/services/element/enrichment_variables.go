// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"fmt"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// scopeVariableSearcher reads a complete local scope using independent identities.
type scopeVariableSearcher interface {
	SearchScopeVariables(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error)
}

// EnrichmentAPI composes enrichment with existing element selection and traversal.
type EnrichmentAPI interface {
	GetElementWithEnrichment(context.Context, string, d.ElementEnrichment, ...services.CallOption) (d.Element, error)
	SearchElementsWithEnrichment(context.Context, d.ElementSearchQuery, d.ElementEnrichment, ...services.CallOption) (d.ElementSearchResult, error)
	SearchElementsPagesWithEnrichment(context.Context, d.ElementSearchQuery, d.ElementEnrichment, d.ElementSearchPageVisitor, ...services.CallOption) (d.ElementSearchPagesResult, error)
}

// enrichmentService keeps local reads and listener orchestration below the facade.
type enrichmentService struct {
	elements  API
	jobs      jobSearcher
	variables scopeVariableSearcher
}

// NewEnrichment composes existing shared services without creating clients or limiters.
func NewEnrichment(elements API, jobs jobSearcher, variables scopeVariableSearcher) EnrichmentAPI {
	return &enrichmentService{elements: elements, jobs: jobs, variables: variables}
}

// requireDependencies rejects unwired requested capabilities before any discovery.
func (s *enrichmentService) requireDependencies(e d.ElementEnrichment) error {
	if s.elements == nil {
		return fmt.Errorf("%w: element enrichment requires an element service", d.ErrPrecondition)
	}
	if e.Variables && s.variables == nil {
		return fmt.Errorf("%w: element variable enrichment requires a variable service", d.ErrPrecondition)
	}
	if e.Listeners && s.jobs == nil {
		return fmt.Errorf("%w: element listener enrichment requires a job service", d.ErrPrecondition)
	}
	return nil
}

// GetElementWithEnrichment selects once and binds enrichment to the observed identity.
func (s *enrichmentService) GetElementWithEnrichment(ctx context.Context, key string, e d.ElementEnrichment, opts ...services.CallOption) (d.Element, error) {
	if err := s.requireDependencies(e); err != nil {
		return d.Element{}, err
	}
	if !e.Variables && !e.Listeners {
		return s.elements.GetElement(ctx, key, opts...)
	}
	// Explicit keys bypass the discovery tenant for every dependent read.
	opts = append(append([]services.CallOption(nil), opts...), services.WithIgnoreTenant())
	item, err := s.elements.GetElement(ctx, key, opts...)
	if err != nil {
		return d.Element{}, err
	}
	if e.Variables && item.ElementInstanceKey != key {
		return d.Element{}, fmt.Errorf("%w: selected element identity differs from requested key", d.ErrMalformedResponse)
	}
	items, err := s.enrichItems(ctx, []d.Element{item}, e, opts...)
	if err != nil {
		return d.Element{}, err
	}
	return items[0], nil
}

// SearchElementsWithEnrichment enriches a bounded selection without repeating discovery.
func (s *enrichmentService) SearchElementsWithEnrichment(ctx context.Context, q d.ElementSearchQuery, e d.ElementEnrichment, opts ...services.CallOption) (d.ElementSearchResult, error) {
	if err := s.requireDependencies(e); err != nil {
		return d.ElementSearchResult{}, err
	}
	result, err := s.elements.SearchElements(ctx, q, opts...)
	if err != nil {
		return result, err
	}
	result.Items, err = s.enrichItems(ctx, result.Items, e, opts...)
	if err != nil {
		return d.ElementSearchResult{}, err
	}
	return result, nil
}

// SearchElementsPagesWithEnrichment retains enriched copies because base traversal
// accumulates element structs before invoking its visitor. Metadata and actions pass through.
func (s *enrichmentService) SearchElementsPagesWithEnrichment(ctx context.Context, q d.ElementSearchQuery, e d.ElementEnrichment, visitor d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error) {
	if err := s.requireDependencies(e); err != nil {
		return d.ElementSearchPagesResult{}, err
	}
	if !e.Variables && !e.Listeners {
		return s.elements.SearchElementsPages(ctx, q, visitor, opts...)
	}
	items := make([]d.Element, 0)
	result, err := s.elements.SearchElementsPages(ctx, q, func(step d.ElementSearchPageStep) (d.ElementSearchPageAction, error) {
		enriched, err := s.enrichItems(ctx, step.Page.Items, e, opts...)
		if err != nil {
			return d.ElementSearchPageActionStop, err
		}
		items = append(items, enriched...)
		step.Page.Items = enriched
		if visitor != nil {
			return visitor(step)
		}
		return d.ElementSearchPageActionContinue, nil
	}, opts...)
	if err != nil {
		return d.ElementSearchPagesResult{}, err
	}
	result.Items = items
	return result, nil
}

// enrichItems publishes only complete local reads and leaves the selected input untouched.
func (s *enrichmentService) enrichItems(ctx context.Context, items []d.Element, e d.ElementEnrichment, opts ...services.CallOption) ([]d.Element, error) {
	if !e.Variables && !e.Listeners {
		return items, nil
	}
	enriched := append([]d.Element{}, items...)
	if e.Variables {
		for i, item := range enriched {
			query := d.ScopeVariableQuery{ProcessInstanceKey: item.ProcessInstanceKey, ScopeKey: item.ElementInstanceKey}
			if err := query.Validate(); err != nil {
				return nil, fmt.Errorf("%w: selected element requires owner and scope identities", d.ErrMalformedResponse)
			}
			if strings.TrimSpace(item.TenantId) != "" {
				tenant := item.TenantId
				query.ExpectedTenantId = &tenant
			}
			values, err := s.variables.SearchScopeVariables(ctx, query, opts...)
			if err != nil {
				return nil, err
			}
			// Allocate even for a nil successful result: requested empty scopes serialize as [].
			local := append([]d.ProcessInstanceVariable{}, values...)
			enriched[i].Variables = &local
		}
	}
	if e.Listeners {
		return enrichElementItemsWithListeners(ctx, s.jobs, enriched, opts...)
	}
	return enriched, nil
}
