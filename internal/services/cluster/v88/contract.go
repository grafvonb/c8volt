package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error)
	GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error)
}

type GenClusterClient interface {
	GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetTopologyResponse, error)
	GetLicenseWithResponse(ctx context.Context, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetLicenseResponse, error)
}

var _ API = (*Service)(nil)
var _ GenClusterClient = (*camundav88.ClientWithResponses)(nil)
