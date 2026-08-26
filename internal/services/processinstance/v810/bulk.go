// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// GetProcessInstances fetches unique process-instance keys with bounded worker execution.
func (s *Service) GetProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	cCfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("getting pi: requested %d, workers %d", lk, nw), s.log, cCfg.Verbose)
	rs, err := pool.ExecuteSlice[string, d.ProcessInstance](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstance, error) {
		return s.GetProcessInstance(ctx, key, opts...)
	})
	return rs, err
}

// WaitForProcessInstancesState waits for multiple process instances to reach a desired state.
func (s *Service) WaitForProcessInstancesState(ctx context.Context, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error) {
	return waiter.WaitForProcessInstancesState(ctx, s, s.cfg, s.log, keys, desired, wantedWorkers, opts...)
}
