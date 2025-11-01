package v87

import (
	"context"
	"io"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
)

type GenResourceClientCamunda interface {
	PostDeploymentsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error)
	PostResourcesResourceKeyDeletionWithResponse(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error)
}

var _ GenResourceClientCamunda = (*camundav87.ClientWithResponses)(nil)
