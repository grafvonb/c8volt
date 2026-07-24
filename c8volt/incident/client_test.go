// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"
	"encoding/json"
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
			return d.ProcessInstanceIncidentDetail{IncidentKey: key, ProcessInstanceKey: "pi-a", TenantId: "tenant-a", ElementId: "task-a", ElementInstanceKey: "ei-a"}, nil
		},
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			assert.Equal(t, d.IncidentFilter{
				State:                  "active",
				ErrorType:              "IO_MAPPING_ERROR",
				ProcessInstanceKey:     "pi-a",
				RootProcessInstanceKey: "root-a",
				ProcessDefinitionKey:   "pd-a",
				ProcessDefinitionId:    "bpmn-a",
				ElementId:              "task-a",
				ElementInstanceKey:     "fni-a",
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
		ElementId:              "task-a",
		ElementInstanceKey:     "fni-a",
	}, 5, options.WithVerbose())

	require.Equal(t, "incident-a", gotIncident.IncidentKey)
	require.Equal(t, "task-a", gotIncident.ElementId)
	require.Equal(t, "ei-a", gotIncident.ElementInstanceKey)
	require.Equal(t, int32(1), gotSearch.Total)
	require.Equal(t, "incident-b", gotSearch.Items[0].IncidentKey)
}

func TestProcessInstanceIncidentDetailJSONUsesCanonicalElementFields(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(ProcessInstanceIncidentDetail{
		IncidentKey:        "incident-a",
		ProcessInstanceKey: "pi-a",
		ErrorMessage:       "no retries left",
		ElementId:          "task-a",
		ElementInstanceKey: "ei-a",
	})

	require.NoError(t, err)
	require.Contains(t, string(raw), `"elementId":"task-a"`)
	require.Contains(t, string(raw), `"elementInstanceKey":"ei-a"`)
	require.NotContains(t, string(raw), "flowNode")
}

func TestClient_SearchIncidentsPagesMapsVisitorStepAndAction(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		searchIncidentsPages: func(_ context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, limit int32, visitor d.IncidentSearchPageVisitor, opts ...services.CallOption) (d.IncidentSearchPagesResult, error) {
			assert.Equal(t, d.IncidentFilter{State: "active", ErrorMessage: "intentional"}, filter)
			assert.Equal(t, d.IncidentPageRequest{Size: 2}, page)
			assert.Equal(t, int32(3), limit)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			action, err := visitor(d.IncidentSearchPageStep{
				Page: d.IncidentPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					EndCursor:     "cursor-a",
					Items:         []d.ProcessInstanceIncidentDetail{{IncidentKey: "match-a"}},
				},
				CumulativeCount: 1,
			})
			require.NoError(t, err)
			require.Equal(t, d.IncidentSearchPageActionStop, action)
			return d.IncidentSearchPagesResult{
				Items: []d.ProcessInstanceIncidentDetail{{IncidentKey: "match-a"}},
				Limit: limit,
				Pages: 1,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).SearchIncidentsPages(context.Background(), Filter{State: "active", ErrorMessage: "intentional"}, PageRequest{Size: 2}, 3, func(step SearchPageStep) (SearchPageAction, error) {
		require.Equal(t, int32(1), step.CumulativeCount)
		require.Equal(t, PageRequest{Size: 2}, step.Page.Request)
		require.Equal(t, OverflowStateHasMore, step.Page.OverflowState)
		require.Equal(t, "cursor-a", step.Page.EndCursor)
		require.Equal(t, "match-a", step.Page.Items[0].IncidentKey)
		return SearchPageActionStop, nil
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, int32(3), got.Limit)
	require.Equal(t, int32(1), got.Pages)
	require.Equal(t, "match-a", got.Items[0].IncidentKey)
}

func TestClient_SearchIncidentsTotalMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		searchIncidentsTotal: func(_ context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (int64, error) {
			assert.Equal(t, d.IncidentFilter{ErrorMessage: "intentional"}, filter)
			assert.Equal(t, d.IncidentPageRequest{Size: 5}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return 42, nil
		},
	}

	got, err := New(api, slog.Default()).SearchIncidentsTotal(context.Background(), Filter{ErrorMessage: "intentional"}, PageRequest{Size: 5}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, int64(42), got)
}

// TestClient_SearchIncidentsDelegatesServiceOwnedCollection verifies the facade
// no longer owns local-filter page traversal for SearchIncidents.
func TestClient_SearchIncidentsDelegatesServiceOwnedCollection(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			assert.Equal(t, d.IncidentFilter{ErrorMessage: "intentional"}, filter)
			assert.Equal(t, int32(1), size)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstanceIncidentDetail{{IncidentKey: "match"}}, nil
		},
	}

	got, err := New(api, slog.Default()).SearchIncidents(context.Background(), Filter{ErrorMessage: "intentional"}, 1, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, int32(1), got.Total)
	require.Equal(t, "match", got.Items[0].IncidentKey)
}

func TestClient_SearchIncidentsPagesMapsErrorsWithPartialResult(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		searchIncidentsPages: func(context.Context, d.IncidentFilter, d.IncidentPageRequest, int32, d.IncidentSearchPageVisitor, ...services.CallOption) (d.IncidentSearchPagesResult, error) {
			return d.IncidentSearchPagesResult{
				Items: []d.ProcessInstanceIncidentDetail{{IncidentKey: "partial"}},
				Limit: 2,
				Pages: 1,
			}, errors.New("boom")
		},
	}

	got, err := New(api, slog.Default()).SearchIncidentsPages(context.Background(), Filter{}, PageRequest{Size: 2}, 2, nil)

	require.Error(t, err)
	require.Equal(t, int32(2), got.Limit)
	require.Equal(t, int32(1), got.Pages)
	require.Equal(t, "partial", got.Items[0].IncidentKey)
}

func TestClient_SearchIncidentsPageMapsBoundary(t *testing.T) {
	t.Parallel()

	api := stubAPI{
		searchIncidentsPage: func(_ context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
			assert.Equal(t, d.IncidentFilter{ErrorMessage: "intentional"}, filter)
			assert.Equal(t, d.IncidentPageRequest{Size: 1}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			total := &d.IncidentReportedTotal{Count: 10, Kind: d.IncidentReportedTotalKindLowerBound}
			return d.IncidentPage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateIndeterminate,
				ReportedTotal: total,
				EndCursor:     "cursor-a",
				Items:         []d.ProcessInstanceIncidentDetail{{IncidentKey: "match"}},
			}, nil
		},
	}

	got, err := New(api, slog.Default()).SearchIncidentsPage(context.Background(), Filter{ErrorMessage: "intentional"}, PageRequest{Size: 1}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, OverflowStateIndeterminate, got.OverflowState)
	require.Equal(t, ReportedTotalKindLowerBound, got.ReportedTotal.Kind)
	require.Equal(t, "cursor-a", got.EndCursor)
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
	searchIncidentsPages           func(context.Context, d.IncidentFilter, d.IncidentPageRequest, int32, d.IncidentSearchPageVisitor, ...services.CallOption) (d.IncidentSearchPagesResult, error)
	searchIncidentsPage            func(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error)
	searchIncidentsTotal           func(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (int64, error)
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

func (s stubAPI) SearchIncidentsPages(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, limit int32, visitor d.IncidentSearchPageVisitor, opts ...services.CallOption) (d.IncidentSearchPagesResult, error) {
	if s.searchIncidentsPages == nil {
		panic("unexpected call")
	}
	return s.searchIncidentsPages(ctx, filter, page, limit, visitor, opts...)
}

func (s stubAPI) SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
	if s.searchIncidentsPage == nil {
		panic("unexpected call")
	}
	return s.searchIncidentsPage(ctx, filter, page, opts...)
}

func (s stubAPI) SearchIncidentsTotal(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (int64, error) {
	if s.searchIncidentsTotal == nil {
		panic("unexpected call")
	}
	return s.searchIncidentsTotal(ctx, filter, page, opts...)
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
