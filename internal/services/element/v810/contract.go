// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API describes the runtime element operations expected from the v8.10 adapter.
type API interface {
	GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error)
	SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error)
	SearchElementsPages(ctx context.Context, query d.ElementSearchQuery, visitor d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error)
	SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error)
	SearchElementsTotal(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (int64, error)
}

// GenElementClient contains the generated Camunda methods needed for runtime element operations.
type GenElementClient interface {
	GetElementInstanceWithResponse(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetElementInstanceResponse, error)
	SearchElementInstancesWithResponse(ctx context.Context, body camundav810.SearchElementInstancesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchElementInstancesResponse, error)
}

var _ API = (*Service)(nil)
var _ GenElementClient = (*camundav810.ClientWithResponses)(nil)
