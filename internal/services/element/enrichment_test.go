// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type stubElementAPI struct {
	get    func(context.Context, string, ...services.CallOption) (d.Element, error)
	search func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error)
}

func (s stubElementAPI) GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error) {
	if s.get == nil {
		return d.Element{}, errors.New("unexpected element get")
	}
	return s.get(ctx, key, opts...)
}

func (s stubElementAPI) SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	if s.search == nil {
		return d.ElementSearchResult{}, errors.New("unexpected element search")
	}
	return s.search(ctx, query, opts...)
}

func (stubElementAPI) SearchElementsPages(context.Context, d.ElementSearchQuery, d.ElementSearchPageVisitor, ...services.CallOption) (d.ElementSearchPagesResult, error) {
	return d.ElementSearchPagesResult{}, errors.New("unexpected element page search")
}

func (stubElementAPI) SearchElementsPage(context.Context, d.ElementSearchQuery, d.ElementPageRequest, ...services.CallOption) (d.ElementSearchPage, error) {
	return d.ElementSearchPage{}, errors.New("unexpected element page")
}

func (stubElementAPI) SearchElementsTotal(context.Context, d.ElementSearchQuery, ...services.CallOption) (int64, error) {
	return 0, errors.New("unexpected element total")
}

type stubJobAPI struct {
	search func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error)
}

func (s stubJobAPI) SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
	if s.search == nil {
		return d.JobSearchResult{}, errors.New("unexpected job search")
	}
	return s.search(ctx, query, opts...)
}

func TestEnrichSearchElementsWithListenersEmitsProgressForUniqueProcessInstances(t *testing.T) {
	var events []d.OpsProgressEvent
	got, err := EnrichSearchElementsWithListeners(context.Background(), stubElementAPI{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			require.Equal(t, d.ElementSearchQuery{ElementId: "review"}, query)
			return d.ElementSearchResult{Items: []d.Element{
				{ElementInstanceKey: "el-1", ProcessInstanceKey: "pi-1"},
				{ElementInstanceKey: "el-2", ProcessInstanceKey: "pi-1"},
				{ElementInstanceKey: "el-3", ProcessInstanceKey: "pi-2"},
			}}, nil
		},
	}, stubJobAPI{
		search: func(_ context.Context, _ d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			return d.JobSearchResult{}, nil
		},
	}, d.ElementSearchQuery{ElementId: "review"}, services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, err)
	require.Len(t, got.Items, 3)
	require.Len(t, events, 3)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 0, Total: 2}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 1, Total: 2}, *events[1].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[2].FrozenScope)
}

func TestEnrichElementWithListenersEmitsProgressForKeyedElement(t *testing.T) {
	var events []d.OpsProgressEvent
	got, err := EnrichElementWithListeners(context.Background(), stubElementAPI{
		get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
			require.Equal(t, "el-1", key)
			return d.Element{ElementInstanceKey: "el-1", ProcessInstanceKey: "pi-1"}, nil
		},
	}, stubJobAPI{
		search: func(_ context.Context, _ d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			return d.JobSearchResult{}, nil
		},
	}, "el-1", services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, err)
	require.Equal(t, "el-1", got.ElementInstanceKey)
	require.Len(t, events, 2)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 0, Total: 1}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 1, Total: 1}, *events[1].FrozenScope)
}
