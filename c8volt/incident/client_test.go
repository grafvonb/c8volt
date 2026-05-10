// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetIncidentAndSearchIncidentsMapServiceBoundary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubAPI{
		getIncident: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			assert.Equal(t, "incident-a", key)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstanceIncidentDetail{IncidentKey: key, ProcessInstanceKey: "pi-a", TenantId: "tenant-a"}, nil
		},
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			assert.Equal(t, d.IncidentFilter{
				State:                  "active",
				ErrorType:              "IO_MAPPING_ERROR",
				ProcessInstanceKey:     "pi-a",
				RootProcessInstanceKey: "root-a",
				ProcessDefinitionKey:   "pd-a",
				ProcessDefinitionId:    "bpmn-a",
				FlowNodeId:             "task-a",
				FlowNodeInstanceKey:    "fni-a",
			}, filter)
			assert.Equal(t, int32(5), size)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstanceIncidentDetail{{IncidentKey: "incident-b", ProcessInstanceKey: "pi-b"}}, nil
		},
	}

	cli := New(api, slog.Default())
	gotIncident, err := cli.GetIncident(ctx, "incident-a", options.WithVerbose())
	require.NoError(t, err)
	gotSearch, err := cli.SearchIncidents(ctx, Filter{
		State:                  "active",
		ErrorType:              "IO_MAPPING_ERROR",
		ProcessInstanceKey:     "pi-a",
		RootProcessInstanceKey: "root-a",
		ProcessDefinitionKey:   "pd-a",
		ProcessDefinitionId:    "bpmn-a",
		FlowNodeId:             "task-a",
		FlowNodeInstanceKey:    "fni-a",
	}, 5, options.WithVerbose())

	require.Equal(t, "incident-a", gotIncident.IncidentKey)
	require.Equal(t, int32(1), gotSearch.Total)
	require.Equal(t, "incident-b", gotSearch.Items[0].IncidentKey)
}

func TestClient_SearchIncidentsWithMessageFilterPagesUntilEnoughLocalMatches(t *testing.T) {
	t.Parallel()

	var pages []d.IncidentPageRequest
	api := stubAPI{
		searchIncidentsPage: func(_ context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
			assert.Equal(t, d.IncidentFilter{State: "active", ErrorMessage: "intentional"}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			pages = append(pages, page)
			if len(pages) == 1 {
				return d.IncidentPage{Request: page, OverflowState: d.ProcessInstanceOverflowStateHasMore}, nil
			}
			return d.IncidentPage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items:         []d.ProcessInstanceIncidentDetail{{IncidentKey: "match"}},
			}, nil
		},
	}

	got, err := New(api, slog.Default()).SearchIncidents(context.Background(), Filter{State: "active", ErrorMessage: "intentional"}, 1, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, []d.IncidentPageRequest{{Size: 1}, {From: 1, Size: 1}}, pages)
	require.Equal(t, int32(1), got.Total)
	require.Equal(t, "match", got.Items[0].IncidentKey)
}

func TestResolveIncidentWaitsForConfirmation(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		getIncident: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			require.Equal(t, "2251799813685249", key)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstanceIncidentDetail{IncidentKey: key, ProcessInstanceKey: "2251799813685250", State: "ACTIVE"}, nil
		},
		resolveIncident: func(_ context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
			require.Equal(t, "2251799813685249", key)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
		},
		waitForIncidentResolved: func(_ context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
			require.Equal(t, "2251799813685249", key)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.IncidentResolutionResponse{Key: key, Ok: true, Status: "resolved"}, nil
		},
	}

	got, err := New(api, slog.Default()).ResolveIncident(context.Background(), "2251799813685249", options.WithVerbose())

	require.NoError(t, err)
	require.True(t, got.OK())
	require.Equal(t, ResolutionStatusConfirmed, got.Status)
	require.Equal(t, "resolved", got.ConfirmationStatus)
	require.True(t, got.MutationSubmitted)
}

func TestResolveIncidentsBulkFailFastStopsSchedulingAfterFirstFailure(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	api := stubAPI{
		getIncident: func(context.Context, string, ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			return d.ProcessInstanceIncidentDetail{State: "ACTIVE"}, nil
		},
		resolveIncident: func(_ context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
			calls.Add(1)
			require.True(t, services.ApplyCallOptions(opts).FailFast)
			return d.IncidentResolutionResponse{Key: key, Ok: false, StatusCode: 500, Status: "500 Internal Server Error"}, errors.New("mutation rejected")
		},
	}

	got, err := New(api, slog.Default()).ResolveIncidents(context.Background(), typex.Keys{"incident-a", "incident-b", "incident-c"}, 1, options.WithFailFast(), options.WithNoWait())

	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
	require.Equal(t, 1, got.Total)
	require.Equal(t, 1, got.Failed)
}

type stubAPI struct {
	getIncident                    func(context.Context, string, ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	resolveIncident                func(context.Context, string, ...services.CallOption) (d.IncidentResolutionResponse, error)
	searchIncidents                func(context.Context, d.IncidentFilter, int32, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	searchIncidentsPage            func(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error)
	searchProcessInstanceIncidents func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	waitForIncidentResolved        func(context.Context, string, ...services.CallOption) (d.IncidentResolutionResponse, error)
	waitForPIIncidentsResolved     func(context.Context, string, []string, ...services.CallOption) (d.IncidentResolutionResponse, error)
}

func (s stubAPI) GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	if s.getIncident == nil {
		panic("unexpected call")
	}
	return s.getIncident(ctx, key, opts...)
}

func (s stubAPI) ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.resolveIncident == nil {
		panic("unexpected call")
	}
	return s.resolveIncident(ctx, key, opts...)
}

func (s stubAPI) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchIncidents == nil {
		panic("unexpected call")
	}
	return s.searchIncidents(ctx, filter, size, opts...)
}

func (s stubAPI) SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
	if s.searchIncidentsPage == nil {
		panic("unexpected call")
	}
	return s.searchIncidentsPage(ctx, filter, page, opts...)
}

func (s stubAPI) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchProcessInstanceIncidents == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstanceIncidents(ctx, key, opts...)
}

func (s stubAPI) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.waitForIncidentResolved == nil {
		panic("unexpected call")
	}
	return s.waitForIncidentResolved(ctx, key, opts...)
}

func (s stubAPI) WaitForProcessInstanceIncidentsResolved(ctx context.Context, key string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.waitForPIIncidentsResolved == nil {
		panic("unexpected call")
	}
	return s.waitForPIIncidentsResolved(ctx, key, incidentKeys, opts...)
}

var _ incsvc.API = stubAPI{}
