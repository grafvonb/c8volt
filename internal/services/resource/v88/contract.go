package v88

import (
	"context"
	"io"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error)
	Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error
}

type GenResourceClientCamunda interface {
	CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error)
	DeleteResourceWithResponse(ctx context.Context, resourceKey camundav88.ResourceKey, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error)
}

var _ API = (*Service)(nil)
var _ GenResourceClientCamunda = (*camundav88.ClientWithResponses)(nil)
