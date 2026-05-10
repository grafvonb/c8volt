// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

func ResolveIncidents(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.IncidentResolutionResults, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	nw := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice[string, d.IncidentResolutionResult](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.IncidentResolutionResult, error) {
		return ResolveIncidentOperation(ctx, api, key, opts...)
	})
	items = compactIncidentResolutionResults(items)
	return summarizeIncidentResolutionResults(items), err
}

func ResolveProcessInstancesIncidents(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceResolutionResults, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	nw := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice[string, d.ProcessInstanceResolutionResult](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceResolutionResult, error) {
		return ResolveProcessInstanceIncidentsOperation(ctx, api, key, opts...)
	})
	items = compactProcessInstanceResolutionResults(items)
	return summarizeProcessInstanceResolutionResults(items), err
}
