// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	esvc "github.com/grafvonb/c8volt/internal/services/element"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
)

// client delegates public element operations to the internal service boundary.
type client struct {
	api    esvc.API
	jobAPI jsvc.API
	log    *slog.Logger
}

// New creates a public element facade over the internal element service.
func New(api esvc.API, log *slog.Logger) API {
	return NewWithListeners(api, nil, log)
}

// NewWithListeners creates a public element facade with listener-job lookup support.
func NewWithListeners(api esvc.API, jobAPI jsvc.API, log *slog.Logger) API {
	return &client{api: api, jobAPI: jobAPI, log: log}
}

// GetElement fetches one runtime element instance by element instance key.
func (c *client) GetElement(ctx context.Context, key string, opts ...foptions.FacadeOption) (Element, error) {
	result, err := c.api.GetElement(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Element{}, ferrors.FromDomain(err)
	}
	return fromDomainElement(result), nil
}

// GetElementWithListeners fetches one runtime element and attaches requested listener jobs.
func (c *client) GetElementWithListeners(ctx context.Context, key string, opts ...foptions.FacadeOption) (Element, error) {
	if c.jobAPI == nil {
		return Element{}, ferrors.FromDomain(fmt.Errorf("%w: element listener enrichment requires a job service", d.ErrUnsupported))
	}
	result, err := esvc.EnrichElementWithListeners(ctx, c.api, c.jobAPI, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Element{}, ferrors.FromDomain(err)
	}
	return fromDomainElement(result), nil
}

// SearchElements collects runtime element instances matching the request filters.
func (c *client) SearchElements(ctx context.Context, request SearchRequest, opts ...foptions.FacadeOption) (SearchResult, error) {
	result, err := c.api.SearchElements(ctx, toDomainSearchRequest(request), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainSearchResult(result)
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}

// SearchElementsWithListeners collects element search results and attaches requested listener jobs.
func (c *client) SearchElementsWithListeners(ctx context.Context, request SearchRequest, opts ...foptions.FacadeOption) (SearchResult, error) {
	if c.jobAPI == nil {
		return SearchResult{}, ferrors.FromDomain(fmt.Errorf("%w: element listener enrichment requires a job service", d.ErrUnsupported))
	}
	result, err := esvc.EnrichSearchElementsWithListeners(ctx, c.api, c.jobAPI, toDomainSearchRequest(request), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainSearchResult(result)
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}

// SearchElementsPage fetches one runtime element search page.
func (c *client) SearchElementsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...foptions.FacadeOption) (Page, error) {
	result, err := c.api.SearchElementsPage(ctx, toDomainSearchRequest(request), toDomainPageRequest(page), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainPage(result)
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}
