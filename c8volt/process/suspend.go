// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/typex"
)

// SuspendProcessInstance suspends only key on Camunda 8.10. DryRun previews without
// I/O; NoWait reports acceptance without reads. Otherwise confirmation is bounded
// by configured timing and ctx. Evidence is returned alongside any converted error.
func (c *client) SuspendProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (SuspensionResult, error) {
	result, err := c.piApi.SuspendProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSuspensionResult(result), ferr.FromDomain(err)
}

// SuspendProcessInstances delegates normalization, workers and waiting to services.
// Parent and child keys stay independent; partial results retain every selected key.
func (c *client) SuspendProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (SuspensionResults, error) {
	result, err := pisvc.SuspendProcessInstances(ctx, c.piApi, c.log, append(typex.Keys(nil), keys...), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainSuspensionResults(result), ferr.FromDomain(err)
}
