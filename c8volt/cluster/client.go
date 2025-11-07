package cluster

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/options"
	csvc "github.com/grafvonb/c8volt/internal/services/cluster"
)

type client struct {
	api csvc.API
	log *slog.Logger
}

func New(api csvc.API, log *slog.Logger) API { return &client{api: api, log: log} }

func (c *client) GetClusterTopology(ctx context.Context, opts ...options.FacadeOption) (Topology, error) {
	t, err := c.api.GetClusterTopology(ctx, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Topology{}, ferrors.FromDomain(err)
	}
	return fromDomainTopology(t), nil
}
