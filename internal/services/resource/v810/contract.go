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

type API interface {
	Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error)
	Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.ResourceDeleteResponse, error)
	Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error)
}

type GenResourceClientCamunda interface {
	CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateDeploymentResponse, error)
	DeleteResourceOpWithResponse(ctx context.Context, resourceKey camundav810.ResourceKey, body camundav810.DeleteResourceOpJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteResourceOpResponse, error)
	GetBatchOperationWithResponse(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error)
	GetResourceWithResponse(ctx context.Context, resourceKey camundav810.ResourceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetResourceResponse, error)
}

type GenProcessDefinitionClientCamunda interface {
	GetProcessDefinitionWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessDefinitionResponse, error)
}

var _ GenResourceClientCamunda = (*camundav810.ClientWithResponses)(nil)
var _ GenProcessDefinitionClientCamunda = (*camundav810.ClientWithResponses)(nil)
