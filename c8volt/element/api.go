// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/typex"
)

// API exposes runtime element lookup, search, enrichment and local variable planning.
type API interface {
	// PlanElementVariableUpdates freezes complete local classifications for all selected keys.
	PlanElementVariableUpdates(context.Context, typex.Keys, map[string]any, ...options.FacadeOption) (ElementVariableUpdatePlan, error)
	// GetElementWithEnrichment reads optional local variables and listeners for one element.
	GetElementWithEnrichment(ctx context.Context, key string, enrichment Enrichment, opts ...options.FacadeOption) (Element, error)
	// SearchElementsWithEnrichment enriches the selected collection without rediscovery.
	SearchElementsWithEnrichment(ctx context.Context, request SearchRequest, enrichment Enrichment, opts ...options.FacadeOption) (SearchResult, error)
	// SearchElementsPagesWithEnrichment preserves page visitation and enriched aggregate items.
	SearchElementsPagesWithEnrichment(ctx context.Context, request SearchRequest, enrichment Enrichment, visitor SearchPageVisitor, opts ...options.FacadeOption) (SearchPagesResult, error)
	GetElement(ctx context.Context, key string, opts ...options.FacadeOption) (Element, error)
	GetElementWithListeners(ctx context.Context, key string, opts ...options.FacadeOption) (Element, error)
	SearchElements(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (SearchResult, error)
	SearchElementsWithListeners(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (SearchResult, error)
	SearchElementsPages(ctx context.Context, request SearchRequest, visitor SearchPageVisitor, opts ...options.FacadeOption) (SearchPagesResult, error)
	SearchElementsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...options.FacadeOption) (Page, error)
	SearchElementsTotal(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (int64, error)
}

var _ API = (*client)(nil)
