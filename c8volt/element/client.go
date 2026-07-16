// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	esvc "github.com/grafvonb/c8volt/internal/services/element"
)

// client delegates public element operations to the internal service boundary.
type client struct {
	api esvc.API
	log *slog.Logger
}

// New creates a public element facade over the internal element service.
func New(api esvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

// GetElement fetches one runtime element instance by element instance key.
func (c *client) GetElement(ctx context.Context, key string, opts ...foptions.FacadeOption) (Element, error) {
	result, err := c.api.GetElement(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
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

// SearchElementsPage fetches one runtime element search page.
func (c *client) SearchElementsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...foptions.FacadeOption) (Page, error) {
	result, err := c.api.SearchElementsPage(ctx, toDomainSearchRequest(request), toDomainPageRequest(page), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainPage(result)
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}
