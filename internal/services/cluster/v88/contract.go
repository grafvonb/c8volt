package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenClusterClient interface {
	GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetTopologyResponse, error)
}

var _ GenClusterClient = (*camundav88.ClientWithResponses)(nil)
