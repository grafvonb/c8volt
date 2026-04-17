package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error)
	GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error)
}

type GenClusterClient interface {
	GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetTopologyResponse, error)
	GetLicenseWithResponse(ctx context.Context, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetLicenseResponse, error)
}

var _ GenClusterClient = (*camundav89.ClientWithResponses)(nil)
