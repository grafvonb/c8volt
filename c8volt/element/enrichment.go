// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	esvc "github.com/grafvonb/c8volt/internal/services/element"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
)

// NewWithEnrichment adds composed local-variable and listener reads while retaining
// all ordinary operations. An absent enrichment service retains legacy capabilities.
func NewWithEnrichment(api esvc.API, jobAPI jsvc.API, enrichmentAPI esvc.EnrichmentAPI, log *slog.Logger) API {
	if enrichmentAPI == nil {
		enrichmentAPI = esvc.NewEnrichment(api, jobAPI, nil)
	}
	return &client{api: api, jobAPI: jobAPI, enrichmentAPI: enrichmentAPI, log: log}
}

// GetElementWithEnrichment maps one completely enriched element into public observations.
func (c *client) GetElementWithEnrichment(ctx context.Context, key string, e Enrichment, opts ...foptions.FacadeOption) (Element, error) {
	result, err := c.enrichmentAPI.GetElementWithEnrichment(ctx, key, toDomainEnrichment(e), foptions.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainElement(result), ferrors.FromDomain(err)
}

// SearchElementsWithEnrichment delegates selection and enrichment to the composed service.
func (c *client) SearchElementsWithEnrichment(ctx context.Context, request SearchRequest, e Enrichment, opts ...foptions.FacadeOption) (SearchResult, error) {
	result, err := c.enrichmentAPI.SearchElementsWithEnrichment(ctx, toDomainSearchRequest(request), toDomainEnrichment(e), foptions.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSearchResult(result), ferrors.FromDomain(err)
}

// SearchElementsPagesWithEnrichment maps visitor facts/actions without owning traversal.
func (c *client) SearchElementsPagesWithEnrichment(ctx context.Context, request SearchRequest, e Enrichment, visitor SearchPageVisitor, opts ...foptions.FacadeOption) (SearchPagesResult, error) {
	result, err := c.enrichmentAPI.SearchElementsPagesWithEnrichment(ctx, toDomainSearchRequest(request), toDomainEnrichment(e), toDomainSearchPageVisitor(visitor), foptions.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSearchPagesResult(result), ferrors.FromDomain(err)
}
