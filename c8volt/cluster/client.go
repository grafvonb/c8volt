// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cluster

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	csvc "github.com/grafvonb/c8volt/internal/services/cluster"
)

type client struct {
	api csvc.API
	log *slog.Logger
}

func New(api csvc.API, log *slog.Logger) API { return &client{api: api, log: log} }

func (c *client) GetClusterTopology(ctx context.Context, opts ...foptions.FacadeOption) (Topology, error) {
	t, err := c.api.GetClusterTopology(ctx, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Topology{}, ferrors.FromDomain(err)
	}
	return fromDomainTopology(t), nil
}

func (c *client) GetClusterLicense(ctx context.Context, opts ...foptions.FacadeOption) (License, error) {
	l, err := c.api.GetClusterLicense(ctx, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return License{}, ferrors.FromDomain(err)
	}
	return fromDomainLicense(l), nil
}
