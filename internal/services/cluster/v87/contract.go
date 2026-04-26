// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error)
	GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error)
}

type GenClusterClient interface {
	GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetTopologyResponse, error)
	GetLicenseWithResponse(ctx context.Context, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetLicenseResponse, error)
}

var _ API = (*Service)(nil)
var _ GenClusterClient = (*camundav87.ClientWithResponses)(nil)
