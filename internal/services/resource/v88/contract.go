package v88

import (
	"context"
	"io"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenResourceClientCamunda interface {
	CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error)
	DeleteResourceWithResponse(ctx context.Context, resourceKey camundav88.ResourceKey, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error)
}

var _ GenResourceClientCamunda = (*camundav88.ClientWithResponses)(nil)
