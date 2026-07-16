// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API describes the runtime element operations expected from the v8.9 adapter.
type API interface {
	GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error)
	SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error)
	SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error)
}

// GenElementClient contains the generated Camunda methods needed for runtime element operations.
type GenElementClient interface {
	GetElementInstanceWithResponse(ctx context.Context, elementInstanceKey camundav89.ElementInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error)
	SearchElementInstancesWithResponse(ctx context.Context, body camundav89.SearchElementInstancesJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error)
}

var _ API = (*Service)(nil)
var _ GenElementClient = (*camundav89.ClientWithResponses)(nil)
