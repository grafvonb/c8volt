// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/element/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/element/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/element/v89"
)

// API exposes tenant-safe runtime element lookup and search operations.
type API interface {
	GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error)
	SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error)
	SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
