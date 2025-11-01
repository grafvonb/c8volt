package cluster

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/options"
	csvc "github.com/grafvonb/c8volt/internal/services/cluster"
)

type API interface {
	GetClusterTopology(ctx context.Context, opts ...options.FacadeOption) (Topology, error)
}

type client struct{ api csvc.API }

func New(api csvc.API) API { return &client{api: api} }

func (c *client) GetClusterTopology(ctx context.Context, opts ...options.FacadeOption) (Topology, error) {
	t, err := c.api.GetClusterTopology(ctx, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Topology{}, ferrors.FromDomain(err)
	}
	return fromDomainTopology(t), nil
}
