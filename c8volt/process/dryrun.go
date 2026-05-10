// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	types "github.com/grafvonb/c8volt/typex"
)

// DryRunCancelOrDeleteGetPIKeys returns the root keys and all collected descendant keys that would be affected.
func (c *client) DryRunCancelOrDeleteGetPIKeys(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (roots types.Keys, collected types.Keys, err error) {
	plan, err := c.DryRunCancelOrDeletePlan(ctx, keys, wantedWorkers, opts...)
	if err != nil {
		return nil, nil, err
	}
	return plan.Roots, plan.Collected, nil
}

// DryRunCancelOrDeletePlan expands selected process-instance keys into the cancellation/deletion dependency plan.
func (c *client) DryRunCancelOrDeletePlan(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DryRunPIKeyExpansion, error) {
	plan, err := pisvc.DryRunCancelOrDeletePlan(ctx, c.piApi, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DryRunPIKeyExpansion{}, ferr.FromDomain(err)
	}
	return fromDomainDryRunPIKeyExpansion(plan), nil
}
