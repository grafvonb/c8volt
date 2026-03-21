package v87

import (
	"context"
	"io"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error)
	Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error
}

type GenResourceClientCamunda interface {
	PostDeploymentsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error)
	PostResourcesResourceKeyDeletionWithResponse(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error)
}

var _ API = (*Service)(nil)
var _ GenResourceClientCamunda = (*camundav87.ClientWithResponses)(nil)
