// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	opsvc "github.com/grafvonb/c8volt/internal/services/ops"
)

type client struct {
	api opsvc.API
	log *slog.Logger
}

func New(api opsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

func (c *client) PurgeOrphanProcessInstances(ctx context.Context, request OrphanPurgeRequest, opts ...options.FacadeOption) (OrphanPurgeResult, error) {
	result, err := c.api.PurgeOrphanProcessInstances(ctx, toDomainOrphanPurgeRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainOrphanPurgeResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) ExecuteRetentionPolicy(ctx context.Context, request RetentionPolicyRequest, opts ...options.FacadeOption) (RetentionPolicyResult, error) {
	result, err := c.api.ExecuteRetentionPolicy(ctx, toDomainRetentionPolicyRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainRetentionPolicyResult(result)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}
