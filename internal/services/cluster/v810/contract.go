// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API is the version-neutral cluster contract implemented by V810.
type API interface {
	GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error)
	GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error)
}

// GenClusterClient is the V810 unified generated-client subset used by this adapter.
type GenClusterClient interface {
	GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetTopologyResponse, error)
	GetLicenseWithResponse(ctx context.Context, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetLicenseResponse, error)
}

var _ GenClusterClient = (*camundav810.ClientWithResponses)(nil)
