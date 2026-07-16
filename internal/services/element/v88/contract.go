// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API describes the runtime element operations expected from the v8.8 adapter.
type API interface {
	GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error)
	SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error)
	SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error)
}

// GenElementClient contains the generated Camunda methods needed for runtime element operations.
type GenElementClient interface {
	GetElementInstanceWithResponse(ctx context.Context, elementInstanceKey camundav88.ElementInstanceKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetElementInstanceResponse, error)
	SearchElementInstancesWithResponse(ctx context.Context, body camundav88.SearchElementInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchElementInstancesResponse, error)
}

var _ API = (*Service)(nil)
var _ GenElementClient = (*camundav88.ClientWithResponses)(nil)
