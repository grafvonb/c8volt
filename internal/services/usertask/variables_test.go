// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type variableUserTaskAPI struct {
	searchVariables func(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error)
}

// GetUserTask rejects accidental legacy resolver use during variable enrichment.
func (a *variableUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("user-task variable enrichment must not call the legacy resolver")
}

// GetNativeUserTask rejects accidental task re-fetches during variable enrichment.
func (a *variableUserTaskAPI) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("user-task variable enrichment must not re-fetch tasks")
}

// SearchUserTasksPage rejects accidental task discovery during variable enrichment.
func (a *variableUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	panic("user-task variable enrichment must not search tasks")
}

// SearchUserTaskEffectiveVariablesPage delegates to the page behavior configured by each contract case.
func (a *variableUserTaskAPI) SearchUserTaskEffectiveVariablesPage(ctx context.Context, key string, page d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error) {
	if a.searchVariables == nil {
		panic("unexpected effective-variable search")
	}
	return a.searchVariables(ctx, key, page, opts...)
}

// TestSearchUserTaskEffectiveVariablesCompletesExactSparsePages verifies raw counts advance offsets and exact totals survive sparse pages.
func TestSearchUserTaskEffectiveVariablesCompletesExactSparsePages(t *testing.T) {
	t.Parallel()

	requests := make([]d.UserTaskVariablePageRequest, 0, 3)
	api := &variableUserTaskAPI{searchVariables: func(_ context.Context, key string, request d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error) {
		require.Equal(t, "task-a", key)
		require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "zeta", Value: `"z"`}}, 1, 3, d.UserTaskReportedTotalKindExact, false), nil
		case 2:
			return variablePage(request, nil, 0, 3, d.UserTaskReportedTotalKindExact, true), nil
		default:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "beta", Value: `"b"`}, {Name: "alpha", Value: `"a"`}}, 2, 3, d.UserTaskReportedTotalKindExact, false), nil
		}
	}}

	got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a", services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []d.UserTaskVariablePageRequest{
		{Size: consts.MaxPISearchSize},
		{From: 1, Size: consts.MaxPISearchSize},
		{From: 1 + consts.MaxPISearchSize, Size: consts.MaxPISearchSize},
	}, requests)
	require.Equal(t, []string{"alpha", "beta", "zeta"}, variableNames(got))
}

// TestSearchUserTaskEffectiveVariablesRetainsHighestLowerBound verifies capped totals cannot shrink traversal's required raw population.
func TestSearchUserTaskEffectiveVariablesRetainsHighestLowerBound(t *testing.T) {
	t.Parallel()

	requests := make([]d.UserTaskVariablePageRequest, 0, 5)
	api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "a"}}, 1, 3, d.UserTaskReportedTotalKindLowerBound, false), nil
		case 2:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "b"}}, 1, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		case 3:
			return variablePage(request, nil, 0, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		case 4:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "c"}}, 1, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		default:
			return variablePage(request, nil, 0, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		}
	}}

	got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

	require.NoError(t, err)
	require.Equal(t, []d.UserTaskVariablePageRequest{
		{Size: consts.MaxPISearchSize},
		{From: 1, Size: consts.MaxPISearchSize},
		{From: 2, Size: consts.MaxPISearchSize},
		{From: 2 + consts.MaxPISearchSize, Size: consts.MaxPISearchSize},
		{From: 3 + consts.MaxPISearchSize, Size: consts.MaxPISearchSize},
	}, requests)
	require.Equal(t, []string{"a", "b", "c"}, variableNames(got))
}

// TestSearchUserTaskEffectiveVariablesContinuesBeyondSatisfiedCap verifies continuation evidence forces a sparse-page probe beyond a capped total.
func TestSearchUserTaskEffectiveVariablesContinuesBeyondSatisfiedCap(t *testing.T) {
	t.Parallel()

	var requests []d.UserTaskVariablePageRequest
	api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "first"}}, 1, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		case 2:
			return variablePage(request, nil, 0, 1, d.UserTaskReportedTotalKindLowerBound, true), nil
		case 3:
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "later"}}, 1, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		default:
			return variablePage(request, nil, 0, 1, d.UserTaskReportedTotalKindLowerBound, false), nil
		}
	}}

	got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

	require.NoError(t, err)
	require.Equal(t, []string{"first", "later"}, variableNames(got))
	require.Len(t, requests, 4)
}

// TestSearchUserTaskEffectiveVariablesNormalizesDuplicateNames verifies identical repeats collapse while conflicting effective records fail.
func TestSearchUserTaskEffectiveVariablesNormalizesDuplicateNames(t *testing.T) {
	t.Parallel()

	identical := d.ProcessInstanceVariable{Name: "local", Value: `{"ok":true}`, VariableKey: "v-1", ProcessInstanceKey: "pi-1", ScopeKey: "child-scope", TenantId: "tenant-a", APITruncated: true}
	t.Run("identical records collapse", func(t *testing.T) {
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			return variablePage(request, []d.ProcessInstanceVariable{identical, identical}, 2, 2, d.UserTaskReportedTotalKindExact, false), nil
		}}

		got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

		require.NoError(t, err)
		require.Equal(t, []d.ProcessInstanceVariable{identical}, got)
	})

	t.Run("conflicting records fail", func(t *testing.T) {
		conflict := identical
		conflict.ScopeKey = "other-scope"
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			return variablePage(request, []d.ProcessInstanceVariable{identical, conflict}, 2, 2, d.UserTaskReportedTotalKindExact, false), nil
		}}

		got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

		require.ErrorIs(t, err, d.ErrMalformedResponse)
		require.Nil(t, got)
	})
}

// TestSearchUserTaskEffectiveVariablesRejectsContradictoryMetadata verifies malformed page facts cannot become a complete variable collection.
func TestSearchUserTaskEffectiveVariablesRejectsContradictoryMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*d.UserTaskVariablePage)
	}{
		{name: "request mismatch", mutate: func(page *d.UserTaskVariablePage) { page.Request.From = 99 }},
		{name: "negative raw count", mutate: func(page *d.UserTaskVariablePage) { page.RawItemCount = -1 }},
		{name: "items exceed raw count", mutate: func(page *d.UserTaskVariablePage) { page.RawItemCount = 0 }},
		{name: "invalid total", mutate: func(page *d.UserTaskVariablePage) { page.ReportedTotal.Count = -1 }},
		{name: "invalid total kind", mutate: func(page *d.UserTaskVariablePage) { page.ReportedTotal.Kind = "invalid" }},
		{name: "exact below observed", mutate: func(page *d.UserTaskVariablePage) { page.ReportedTotal.Count = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
				page := variablePage(request, []d.ProcessInstanceVariable{{Name: "a"}}, 1, 1, d.UserTaskReportedTotalKindExact, false)
				tt.mutate(&page)
				return page, nil
			}}

			got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

			require.ErrorIs(t, err, d.ErrMalformedResponse)
			require.Nil(t, got)
		})
	}
}

// TestSearchUserTaskEffectiveVariablesRejectsExactTotalBelowRetainedLowerBound verifies exact metadata cannot contradict an earlier capped minimum.
func TestSearchUserTaskEffectiveVariablesRejectsExactTotalBelowRetainedLowerBound(t *testing.T) {
	t.Parallel()

	var calls int
	api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
		calls++
		if calls == 1 {
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "a"}}, 1, 3, d.UserTaskReportedTotalKindLowerBound, false), nil
		}
		return variablePage(request, []d.ProcessInstanceVariable{{Name: "b"}}, 1, 2, d.UserTaskReportedTotalKindExact, false), nil
	}}

	got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Nil(t, got)
	require.Equal(t, 2, calls)
}

// TestSearchUserTaskEffectiveVariablesHandlesEmptyAndOffsetOverflow verifies empty exact results initialize slices and offset arithmetic cannot wrap.
func TestSearchUserTaskEffectiveVariablesHandlesEmptyAndOffsetOverflow(t *testing.T) {
	t.Parallel()

	t.Run("empty exact result", func(t *testing.T) {
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			return variablePage(request, nil, 0, 0, d.UserTaskReportedTotalKindExact, false), nil
		}}

		got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("offset overflow", func(t *testing.T) {
		var calls int
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			calls++
			return variablePage(request, nil, math.MaxInt32, math.MaxInt64, d.UserTaskReportedTotalKindLowerBound, false), nil
		}}

		got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

		require.ErrorIs(t, err, d.ErrMalformedResponse)
		require.Nil(t, got)
		require.Equal(t, 2, calls)
	})
}

// TestSearchUserTaskEffectiveVariablesPropagatesCancellationAndLaterPageFailure verifies no partial collection escapes interrupted traversal.
func TestSearchUserTaskEffectiveVariablesPropagatesCancellationAndLaterPageFailure(t *testing.T) {
	t.Parallel()

	t.Run("pre-canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		api := &variableUserTaskAPI{searchVariables: func(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error) {
			called = true
			return d.UserTaskVariablePage{}, nil
		}}

		got, err := SearchUserTaskEffectiveVariables(ctx, api, "task-a")

		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, got)
		require.False(t, called)
	})

	t.Run("canceled after page", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			cancel()
			return variablePage(request, []d.ProcessInstanceVariable{{Name: "a"}}, 1, 2, d.UserTaskReportedTotalKindExact, false), nil
		}}

		got, err := SearchUserTaskEffectiveVariables(ctx, api, "task-a")

		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, got)
	})

	t.Run("later page failure", func(t *testing.T) {
		wantErr := errors.New("later page failed")
		var calls int
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			calls++
			if calls == 1 {
				return variablePage(request, []d.ProcessInstanceVariable{{Name: "a"}}, 1, 2, d.UserTaskReportedTotalKindExact, false), nil
			}
			return d.UserTaskVariablePage{}, wantErr
		}}

		got, err := SearchUserTaskEffectiveVariables(context.Background(), api, "task-a")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})
}

// TestEnrichUserTasksWithVariablesPreservesTaskAndScopeOrder verifies sequential attachment retains tasks and backend-selected local scopes.
func TestEnrichUserTasksWithVariablesPreservesTaskAndScopeOrder(t *testing.T) {
	t.Parallel()

	tasks := []d.UserTask{
		{Key: "task-b", Name: "second", ProcessInstanceKey: "pi-b", TenantId: "tenant-b"},
		{Key: "task-a", Name: "first", ProcessInstanceKey: "pi-a", TenantId: "tenant-a"},
	}
	requested := make([]string, 0, 2)
	api := &variableUserTaskAPI{searchVariables: func(_ context.Context, key string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
		requested = append(requested, key)
		return variablePage(request, []d.ProcessInstanceVariable{
			{Name: "zeta", ProcessInstanceKey: "pi-" + key[len(key)-1:], ScopeKey: "local-" + key, TenantId: "actual-tenant"},
			{Name: "alpha", ProcessInstanceKey: "pi-" + key[len(key)-1:], ScopeKey: "pi-" + key[len(key)-1:], TenantId: "actual-tenant"},
		}, 2, 2, d.UserTaskReportedTotalKindExact, false), nil
	}}

	got, err := EnrichUserTasksWithVariables(context.Background(), api, tasks)

	require.NoError(t, err)
	require.Equal(t, []string{"task-b", "task-a"}, requested)
	require.EqualValues(t, 2, got.Total)
	require.Equal(t, tasks[0], got.Items[0].Item)
	require.Equal(t, []string{"alpha", "zeta"}, variableNames(got.Items[0].Variables))
	require.Equal(t, "local-task-b", got.Items[0].Variables[1].ScopeKey)
	require.Equal(t, "actual-tenant", got.Items[0].Variables[1].TenantId)
	require.Equal(t, tasks[1], got.Items[1].Item)
	require.Equal(t, []string{"alpha", "zeta"}, variableNames(got.Items[1].Variables))
}

// TestEnrichUserTasksWithVariablesHandlesEmptyAndTaskFailure verifies empty inputs are initialized and later failures return no partial success.
func TestEnrichUserTasksWithVariablesHandlesEmptyAndTaskFailure(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		got, err := EnrichUserTasksWithVariables(context.Background(), &variableUserTaskAPI{}, nil)

		require.NoError(t, err)
		require.Zero(t, got.Total)
		require.NotNil(t, got.Items)
		require.Empty(t, got.Items)
	})

	t.Run("empty variables initialized", func(t *testing.T) {
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, _ string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			return variablePage(request, nil, 0, 0, d.UserTaskReportedTotalKindExact, false), nil
		}}

		got, err := EnrichUserTasksWithVariables(context.Background(), api, []d.UserTask{{Key: "task-a"}})

		require.NoError(t, err)
		require.EqualValues(t, 1, got.Total)
		require.NotNil(t, got.Items[0].Variables)
		require.Empty(t, got.Items[0].Variables)
	})

	t.Run("later task failure", func(t *testing.T) {
		wantErr := errors.New("second task failed")
		requested := make([]string, 0, 2)
		api := &variableUserTaskAPI{searchVariables: func(_ context.Context, key string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
			requested = append(requested, key)
			if key == "task-b" {
				return d.UserTaskVariablePage{}, wantErr
			}
			return variablePage(request, nil, 0, 0, d.UserTaskReportedTotalKindExact, false), nil
		}}

		got, err := EnrichUserTasksWithVariables(context.Background(), api, []d.UserTask{{Key: "task-a"}, {Key: "task-b"}, {Key: "task-c"}})

		require.ErrorIs(t, err, wantErr)
		require.Empty(t, got)
		require.Equal(t, []string{"task-a", "task-b"}, requested)
	})
}

// variablePage builds one adapter-shaped page while keeping raw count independent of selected records.
func variablePage(request d.UserTaskVariablePageRequest, items []d.ProcessInstanceVariable, rawCount int32, total int64, kind d.UserTaskReportedTotalKind, continuation bool) d.UserTaskVariablePage {
	return d.UserTaskVariablePage{
		Items:                   items,
		Request:                 request,
		RawItemCount:            rawCount,
		ReportedTotal:           d.UserTaskReportedTotal{Count: total, Kind: kind},
		HasContinuationEvidence: continuation,
	}
}

// variableNames returns variable names in their observed order for concise assertions.
func variableNames(items []d.ProcessInstanceVariable) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names
}
