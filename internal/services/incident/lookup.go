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

func GetIncidents(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	workers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	return pool.ExecuteSlice[string, d.ProcessInstanceIncidentDetail](ctx, ukeys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceIncidentDetail, error) {
		return api.GetIncident(ctx, key, opts...)
	})
}

func SearchIncidents(ctx context.Context, api API, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if incidentSearchNeedsPagedLocalFiltering(filter) {
		return searchIncidentPagesUntilLimit(ctx, api, filter, size, opts...)
	}
	return api.SearchIncidents(ctx, filter, size, opts...)
}

func searchIncidentPagesUntilLimit(ctx context.Context, api API, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if size <= 0 {
		return nil, nil
	}
	req := d.IncidentPageRequest{Size: size}
	out := make([]d.ProcessInstanceIncidentDetail, 0, size)
	for {
		page, err := api.SearchIncidentsPage(ctx, filter, req, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if int32(len(out)) >= size {
				return out, nil
			}
			out = append(out, item)
		}
		if page.OverflowState == d.ProcessInstanceOverflowStateNoMore {
			return out, nil
		}
		req = nextIncidentPageRequest(req, page)
	}
}

func incidentSearchNeedsPagedLocalFiltering(filter d.IncidentFilter) bool {
	return filter.ErrorMessage != "" ||
		filter.CreationTimeAfter != "" ||
		filter.CreationTimeBefore != ""
}

func nextIncidentPageRequest(current d.IncidentPageRequest, page d.IncidentPage) d.IncidentPageRequest {
	if page.EndCursor != "" {
		return d.IncidentPageRequest{Size: current.Size, After: page.EndCursor}
	}
	return d.IncidentPageRequest{From: current.From + current.Size, Size: current.Size}
}
