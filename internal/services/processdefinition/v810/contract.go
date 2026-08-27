// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"io"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API describes process-definition operations implemented by the native v8.10 adapter.
type API interface {
	SearchProcessDefinitionsPage(ctx context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, opts ...services.CallOption) (d.ProcessDefinitionPage, error)
	SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
	GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error)
}

// GenProcessDefinitionClientCamunda captures the generated Camunda calls used by the v8.10 process-definition service.
type GenProcessDefinitionClientCamunda interface {
	GetProcessDefinitionWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessDefinitionResponse, error)
	GetProcessDefinitionXMLWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessDefinitionXMLResponse, error)
	SearchProcessDefinitionsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessDefinitionsResponse, error)
	SearchProcessInstancesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error)
}

var _ API = (*Service)(nil)
var _ GenProcessDefinitionClientCamunda = (*camundav810.ClientWithResponses)(nil)
