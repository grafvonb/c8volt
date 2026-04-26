// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"io"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error)
	Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error
	Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error)
}

type GenResourceClientCamunda interface {
	CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error)
	DeleteResourceOpWithResponse(ctx context.Context, resourceKey camundav89.ResourceKey, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error)
	GetResourceWithResponse(ctx context.Context, resourceKey camundav89.ResourceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error)
}

type GenProcessDefinitionClientCamunda interface {
	GetProcessDefinitionWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error)
}

var _ GenResourceClientCamunda = (*camundav89.ClientWithResponses)(nil)
var _ GenProcessDefinitionClientCamunda = (*camundav89.ClientWithResponses)(nil)
