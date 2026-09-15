// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type searchUserTaskAPI struct {
	searchPage func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error)
}

// GetUserTask rejects accidental use of the legacy resolver during discovery.
func (a *searchUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("user-task search must not call the legacy resolver")
}

// GetNativeUserTask rejects accidental direct reads during discovery.
func (a *searchUserTaskAPI) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("user-task search must not call the native getter")
}

// SearchUserTasksPage delegates to the page behavior configured by each traversal case.
func (a *searchUserTaskAPI) SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error) {
	return a.searchPage(ctx, query, page, opts...)
}

// SearchUserTaskEffectiveVariablesPage rejects accidental enrichment during task discovery.
func (a *searchUserTaskAPI) SearchUserTaskEffectiveVariablesPage(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error) {
	panic("user-task search must not call effective-variable search")
}

// TestSearchUserTasksPagesPrefersAdvancingCursors verifies cursor metadata takes precedence over offset fallback and a final nonempty cursor receives a terminal probe.
func TestSearchUserTasksPagesPrefersAdvancingCursors(t *testing.T) {
	t.Parallel()

	requests := make([]d.UserTaskPageRequest, 0, 3)
	api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return searchPage(request, []string{"task-a"}, 1, "cursor-a", 2, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		case 2:
			return searchPage(request, []string{"task-b"}, 1, "cursor-b", 2, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		default:
			return searchPage(request, nil, 0, "", 2, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateIndeterminate), nil
		}
	}}

	cumulative := make([]int64, 0, 3)
	result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 1}, func(step d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		cumulative = append(cumulative, step.CumulativeCount)
		return d.UserTaskSearchPageActionContinue, nil
	})

	require.NoError(t, err)
	require.Equal(t, []d.UserTaskPageRequest{{Size: 1}, {Size: 1, After: "cursor-a"}, {Size: 1, After: "cursor-b"}}, requests)
	require.Equal(t, []string{"task-a", "task-b"}, userTaskKeys(result.Items))
	require.EqualValues(t, 2, result.Total)
	require.EqualValues(t, 3, result.Pages)
	require.Equal(t, d.UserTaskSearchCompletionExhausted, result.Completion)
	require.Equal(t, []int64{1, 2, 2}, cumulative)
}

// TestSearchUserTasksPagesUsesRawOffsetProgress verifies short and empty continuing pages advance by raw count or requested size rather than selected count.
func TestSearchUserTasksPagesUsesRawOffsetProgress(t *testing.T) {
	t.Parallel()

	requests := make([]d.UserTaskPageRequest, 0, 4)
	api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return searchPage(request, []string{"task-a"}, 2, "", 3, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		case 2:
			return searchPage(request, nil, 0, "", 3, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		case 3:
			return searchPage(request, []string{"task-b"}, 1, "", 3, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateIndeterminate), nil
		default:
			return searchPage(request, nil, 0, "", 3, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateIndeterminate), nil
		}
	}}

	result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 3}, nil)

	require.NoError(t, err)
	require.Equal(t, []d.UserTaskPageRequest{{Size: 3}, {From: 2, Size: 3}, {From: 5, Size: 3}, {From: 6, Size: 3}}, requests)
	require.Equal(t, []string{"task-a", "task-b"}, userTaskKeys(result.Items))
	require.Equal(t, d.UserTaskSearchCompletionExhausted, result.Completion)
}

// TestSearchUserTasksPagesTrimsMidPageLimitWithoutChangingRawFacts verifies visitor metadata keeps backend progress while only selected rows enter the result.
func TestSearchUserTasksPagesTrimsMidPageLimitWithoutChangingRawFacts(t *testing.T) {
	t.Parallel()

	var calls int
	api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		calls++
		return searchPage(request, []string{"task-a", "task-b", "task-c"}, 3, "cursor-a", 10, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
	}}
	var observed d.UserTaskSearchPageStep
	result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 3, Limit: 2}, func(step d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		observed = step
		return d.UserTaskSearchPageActionContinue, nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"task-a", "task-b"}, userTaskKeys(result.Items))
	require.EqualValues(t, 3, observed.Page.RawItemCount)
	require.Equal(t, []string{"task-a", "task-b"}, userTaskKeys(observed.Page.Items))
	require.EqualValues(t, 2, observed.CumulativeCount)
	require.True(t, observed.LimitReached)
	require.Equal(t, d.UserTaskSearchCompletionLimitReached, result.Completion)
	require.Equal(t, 1, calls)
}

// TestSearchUserTasksPagesVisitorStopAndError verifies visitor decisions are validated and errors never become successful partial results.
func TestSearchUserTasksPagesVisitorStopAndError(t *testing.T) {
	t.Parallel()

	newAPI := func() *searchUserTaskAPI {
		return &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
			return searchPage(request, []string{"task-a"}, 1, "cursor-a", 2, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		}}
	}

	stopped, err := SearchUserTasksPages(context.Background(), newAPI(), d.UserTaskSearchQuery{BatchSize: 1}, func(d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		return d.UserTaskSearchPageActionStop, nil
	})
	require.NoError(t, err)
	require.Equal(t, d.UserTaskSearchCompletionVisitorStopped, stopped.Completion)
	require.Equal(t, []string{"task-a"}, userTaskKeys(stopped.Items))

	errVisitor := errors.New("visitor failed")
	failed, err := SearchUserTasksPages(context.Background(), newAPI(), d.UserTaskSearchQuery{BatchSize: 1}, func(d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		return d.UserTaskSearchPageActionContinue, errVisitor
	})
	require.ErrorIs(t, err, errVisitor)
	require.Empty(t, failed)

	failed, err = SearchUserTasksPages(context.Background(), newAPI(), d.UserTaskSearchQuery{BatchSize: 1}, func(d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		return d.UserTaskSearchPageAction("invalid"), nil
	})
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Empty(t, failed)
}

// TestSearchUserTasksCollectsInitializedResults verifies the collection wrapper preserves an empty non-nil slice and applies the default batch size.
func TestSearchUserTasksCollectsInitializedResults(t *testing.T) {
	t.Parallel()

	api := &searchUserTaskAPI{searchPage: func(_ context.Context, query d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		require.EqualValues(t, consts.MaxPISearchSize, query.BatchSize)
		require.EqualValues(t, consts.MaxPISearchSize, request.Size)
		return searchPage(request, nil, 0, "", 0, d.UserTaskReportedTotalKindExact, d.UserTaskContinuationStateNoMore), nil
	}}

	items, err := SearchUserTasks(context.Background(), api, d.UserTaskSearchQuery{})

	require.NoError(t, err)
	require.NotNil(t, items)
	require.Empty(t, items)
}

// TestSearchUserTasksPagesRejectsCursorCycles verifies both repeated and previously seen cursors fail instead of repeating requests.
func TestSearchUserTasksPagesRejectsCursorCycles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cursors []string
	}{
		{name: "repeated current cursor", cursors: []string{"cursor-a", "cursor-a"}},
		{name: "cycle to earlier cursor", cursors: []string{"cursor-a", "cursor-b", "cursor-a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
				cursor := tt.cursors[calls]
				calls++
				return searchPage(request, []string{"task"}, 1, cursor, 100, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
			}}

			result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 1}, nil)

			require.ErrorIs(t, err, d.ErrMalformedResponse)
			require.Empty(t, result)
			require.Equal(t, len(tt.cursors), calls)
		})
	}
}

// TestSearchUserTasksPagesRejectsInconsistentMetadata verifies malformed page facts cannot establish partial success.
func TestSearchUserTasksPagesRejectsInconsistentMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*d.UserTaskSearchPage)
	}{
		{name: "request mismatch", mutate: func(page *d.UserTaskSearchPage) { page.Request.From = 9 }},
		{name: "negative raw count", mutate: func(page *d.UserTaskSearchPage) { page.RawItemCount = -1 }},
		{name: "selected exceeds raw", mutate: func(page *d.UserTaskSearchPage) { page.RawItemCount = 0 }},
		{name: "invalid continuation", mutate: func(page *d.UserTaskSearchPage) { page.ContinuationState = "invalid" }},
		{name: "invalid total", mutate: func(page *d.UserTaskSearchPage) { page.ReportedTotal.Count = -1 }},
		{name: "exact total below observed", mutate: func(page *d.UserTaskSearchPage) { page.ReportedTotal.Count = 0 }},
		{name: "no-more before exact total", mutate: func(page *d.UserTaskSearchPage) { page.ReportedTotal.Count = 2 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
				page := searchPage(request, []string{"task-a"}, 1, "", 1, d.UserTaskReportedTotalKindExact, d.UserTaskContinuationStateNoMore)
				tt.mutate(&page)
				return page, nil
			}}

			result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 1}, nil)

			require.ErrorIs(t, err, d.ErrMalformedResponse)
			require.Empty(t, result)
		})
	}
}

// TestSearchUserTasksPagesRejectsOffsetOverflow verifies sparse offset fallback cannot wrap an int32 page position.
func TestSearchUserTasksPagesRejectsOffsetOverflow(t *testing.T) {
	t.Parallel()

	api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		return searchPage(request, nil, 0, "", math.MaxInt64, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
	}}

	result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: math.MaxInt32, Limit: 1}, nil)

	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Empty(t, result)
}

// TestSearchUserTasksPagesPropagatesCancellation verifies cancellation before and during traversal returns no successful result.
func TestSearchUserTasksPagesPropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	api := &searchUserTaskAPI{searchPage: func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
		called = true
		return d.UserTaskSearchPage{}, nil
	}}
	result, err := SearchUserTasksPages(ctx, api, d.UserTaskSearchQuery{BatchSize: 1}, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, called)
	require.Empty(t, result)

	ctx, cancel = context.WithCancel(context.Background())
	api = &searchUserTaskAPI{searchPage: func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
		cancel()
		return d.UserTaskSearchPage{}, context.Canceled
	}}
	result, err = SearchUserTasksPages(ctx, api, d.UserTaskSearchQuery{BatchSize: 1}, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, result)
}

// TestSearchUserTasksTotalUsesExactMetadataImmediately verifies count mode trusts a valid exact total without applying a user limit or visitor path.
func TestSearchUserTasksTotalUsesExactMetadataImmediately(t *testing.T) {
	t.Parallel()

	calls := 0
	api := &searchUserTaskAPI{searchPage: func(_ context.Context, query d.UserTaskSearchQuery, request d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error) {
		calls++
		require.Zero(t, query.Limit)
		require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
		return searchPage(request, []string{"task-a"}, 1, "cursor-a", math.MaxInt64, d.UserTaskReportedTotalKindExact, d.UserTaskContinuationStateHasMore), nil
	}}

	total, err := SearchUserTasksTotal(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 10, Limit: 1}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.EqualValues(t, math.MaxInt64, total)
	require.Equal(t, 1, calls)
}

// TestSearchUserTasksTotalCountsCappedPagesWithoutSelectedObjects verifies fallback uses raw int64 progress and stops on an empty terminal probe after the lower bound.
func TestSearchUserTasksTotalCountsCappedPagesWithoutSelectedObjects(t *testing.T) {
	t.Parallel()

	requests := make([]d.UserTaskPageRequest, 0, 3)
	api := &searchUserTaskAPI{searchPage: func(_ context.Context, query d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		require.Zero(t, query.Limit)
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return searchPage(request, nil, math.MaxInt32, "cursor-a", math.MaxInt32, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		case 2:
			return searchPage(request, nil, 10, "cursor-b", math.MaxInt32, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateHasMore), nil
		default:
			return searchPage(request, nil, 0, "", math.MaxInt32, d.UserTaskReportedTotalKindLowerBound, d.UserTaskContinuationStateIndeterminate), nil
		}
	}}

	total, err := SearchUserTasksTotal(context.Background(), api, d.UserTaskSearchQuery{BatchSize: math.MaxInt32, Limit: 1})

	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt32)+10, total)
	require.Equal(t, []d.UserTaskPageRequest{{Size: math.MaxInt32}, {Size: math.MaxInt32, After: "cursor-a"}, {Size: math.MaxInt32, After: "cursor-b"}}, requests)
}

// TestSearchUserTasksTotalRejectsCountOverflowAndFailures verifies fallback arithmetic, adapter errors, and cancellation never return a numeric success.
func TestSearchUserTasksTotalRejectsCountOverflowAndFailures(t *testing.T) {
	t.Parallel()

	errRead := errors.New("read failed")
	tests := []struct {
		name string
		api  *searchUserTaskAPI
		ctx  context.Context
		err  error
	}{
		{
			name: "adapter failure",
			ctx:  context.Background(),
			err:  errRead,
			api: &searchUserTaskAPI{searchPage: func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
				return d.UserTaskSearchPage{}, errRead
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			total, err := SearchUserTasksTotal(tt.ctx, tt.api, d.UserTaskSearchQuery{BatchSize: 1})

			require.Zero(t, total)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

// TestSearchUserTasksTotalPropagatesPreCanceledContext verifies count mode does not issue a request or return a numeric success after cancellation.
func TestSearchUserTasksTotalPropagatesPreCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	api := &searchUserTaskAPI{searchPage: func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
		called = true
		return d.UserTaskSearchPage{}, nil
	}}

	total, err := SearchUserTasksTotal(ctx, api, d.UserTaskSearchQuery{BatchSize: 1, Limit: 1})

	require.Zero(t, total)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, called)
}

// searchPage constructs stable page facts while allowing tests to vary backend progress independently from selected objects.
func searchPage(request d.UserTaskPageRequest, keys []string, rawCount int32, endCursor string, total int64, totalKind d.UserTaskReportedTotalKind, continuation d.UserTaskContinuationState) d.UserTaskSearchPage {
	items := make([]d.UserTask, len(keys))
	for i, key := range keys {
		items[i] = d.UserTask{Key: key, State: "CREATED", ProcessInstanceKey: "process-" + key}
	}
	return d.UserTaskSearchPage{
		Items:             items,
		Request:           request,
		RawItemCount:      rawCount,
		EndCursor:         endCursor,
		ReportedTotal:     &d.UserTaskReportedTotal{Count: total, Kind: totalKind},
		ContinuationState: continuation,
	}
}

// userTaskKeys extracts stable task identities for concise traversal assertions.
func userTaskKeys(tasks []d.UserTask) []string {
	keys := make([]string, len(tasks))
	for i, task := range tasks {
		keys[i] = task.Key
	}
	return keys
}

// TestSearchUserTasksPagesResolvesExactContinuationBeforeVisitor verifies that
// cumulative service progress determines whether cursor pages require another page.
func TestSearchUserTasksPagesResolvesExactContinuationBeforeVisitor(t *testing.T) {
	t.Parallel()
	calls := 0
	api := &searchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, request d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		calls++
		require.LessOrEqual(t, calls, 3)
		return searchPage(request, []string{"task"}, 1, fmt.Sprintf("cursor-%d", calls), 3, d.UserTaskReportedTotalKindExact, d.UserTaskContinuationStateIndeterminate), nil
	}}
	var states []d.UserTaskContinuationState
	result, err := SearchUserTasksPages(context.Background(), api, d.UserTaskSearchQuery{BatchSize: 1}, func(step d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		states = append(states, step.Page.ContinuationState)
		return d.UserTaskSearchPageActionContinue, nil
	})
	require.NoError(t, err)
	require.Equal(t, []d.UserTaskContinuationState{d.UserTaskContinuationStateHasMore, d.UserTaskContinuationStateHasMore, d.UserTaskContinuationStateNoMore}, states)
	require.EqualValues(t, 3, result.Total)
	require.Equal(t, d.UserTaskSearchCompletionExhausted, result.Completion)
}
