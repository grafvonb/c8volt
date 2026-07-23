// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

// API exposes runtime element lookup and search operations.
type API interface {
	GetElement(ctx context.Context, key string, opts ...options.FacadeOption) (Element, error)
	GetElementWithListeners(ctx context.Context, key string, opts ...options.FacadeOption) (Element, error)
	SearchElements(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (SearchResult, error)
	SearchElementsWithListeners(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (SearchResult, error)
	SearchElementsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...options.FacadeOption) (Page, error)
}

var _ API = (*client)(nil)
