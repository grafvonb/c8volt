// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API describes the runtime element operations expected from the v8.7 adapter.
type API interface {
	GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error)
	SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error)
	SearchElementsPages(ctx context.Context, query d.ElementSearchQuery, visitor d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error)
	SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error)
	SearchElementsTotal(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (int64, error)
}

var _ API = (*Service)(nil)
