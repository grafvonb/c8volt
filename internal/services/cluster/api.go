package cluster

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/cluster/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/cluster/v88"
)

type API interface {
	GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error)
	GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
