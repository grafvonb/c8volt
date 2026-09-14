// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"errors"
	"fmt"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// facadeSearchUserTaskAPI supplies one-page fixtures while recording facade-to-service requests.
type facadeSearchUserTaskAPI struct {
	searchPage func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error)
}

// GetUserTask rejects legacy resolver access from public search methods.
func (a facadeSearchUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{}, errors.New("unexpected legacy user-task read")
}

// GetNativeUserTask rejects direct native reads from public search methods.
func (a facadeSearchUserTaskAPI) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{}, errors.New("unexpected native user-task read")
}

// SearchUserTasksPage delegates to the fixture callback for service-owned traversal.
func (a facadeSearchUserTaskAPI) SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error) {
	return a.searchPage(ctx, query, page, opts...)
}

// TestClientSearchUserTasksMapsRequestOptionsAndReturnedCount verifies the
// facade forwards every selector while exposing the limited returned count.
func TestClientSearchUserTasksMapsRequestOptionsAndReturnedCount(t *testing.T) {
	t.Parallel()

	source := []d.UserTask{
		{Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a", CandidateUsers: []string{"alice"}},
		{Key: "task-b", State: "COMPLETED", ProcessInstanceKey: "process-b", CandidateGroups: []string{"accounting"}},
	}
	api := facadeSearchUserTaskAPI{searchPage: func(_ context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error) {
		require.Equal(t, d.UserTaskSearchQuery{
			ProcessInstanceKey: "process-a", ProcessDefinitionKey: "definition-a", BpmnProcessId: "invoice",
			ElementId: "approve_invoice", State: "CREATED", Assignee: "alice", CandidateUser: "bob",
			CandidateGroup: "accounting", BatchSize: 25, Limit: 1,
		}, query)
		require.Equal(t, d.UserTaskPageRequest{Size: 25}, page)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.IgnoreTenant)
		require.True(t, cfg.Verbose)
		return d.UserTaskSearchPage{
			Items: source, Request: page, RawItemCount: 2,
			ReportedTotal:     &d.UserTaskReportedTotal{Count: 2, Kind: d.UserTaskReportedTotalKindExact},
			ContinuationState: d.UserTaskContinuationStateNoMore,
		}, nil
	}}
	client := New(nil, nil, api, nil)

	got, err := client.SearchUserTasks(context.Background(), SearchRequest{
		ProcessInstanceKey: "process-a", ProcessDefinitionKey: "definition-a", BpmnProcessId: "invoice",
		ElementId: "approve_invoice", State: "CREATED", Assignee: "alice", CandidateUser: "bob",
		CandidateGroup: "accounting", BatchSize: 25, Limit: 1,
	}, options.WithIgnoreTenant(), options.WithVerbose())
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Total)
	require.Len(t, got.Items, 1)
	require.Equal(t, []string{"alice"}, got.Items[0].CandidateUsers)

	source[0].CandidateUsers[0] = "changed"
	require.Equal(t, []string{"alice"}, got.Items[0].CandidateUsers)
}

// TestClientSearchUserTasksPagesMapsVisitorFactsAndAction verifies public page
// observation does not take ownership of traversal or internal task slices.
func TestClientSearchUserTasksPagesMapsVisitorFactsAndAction(t *testing.T) {
	t.Parallel()

	source := []d.UserTask{{Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a", CandidateGroups: []string{"ops"}}}
	api := facadeSearchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, page d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		return d.UserTaskSearchPage{
			Items: source, Request: page, RawItemCount: 1, EndCursor: "cursor-a",
			ReportedTotal:     &d.UserTaskReportedTotal{Count: 10, Kind: d.UserTaskReportedTotalKindLowerBound},
			ContinuationState: d.UserTaskContinuationStateHasMore,
		}, nil
	}}
	client := New(nil, nil, api, nil)

	got, err := client.SearchUserTasksPages(context.Background(), SearchRequest{BatchSize: 10}, func(step SearchPageStep) (SearchPageAction, error) {
		require.Equal(t, int64(1), step.CumulativeCount)
		require.False(t, step.LimitReached)
		require.Equal(t, PageRequest{Size: 10}, step.Page.Request)
		require.Equal(t, int32(1), step.Page.RawItemCount)
		require.Equal(t, "cursor-a", step.Page.EndCursor)
		require.Equal(t, ReportedTotal{Count: 10, Kind: ReportedTotalKindLowerBound}, *step.Page.ReportedTotal)
		require.Equal(t, ContinuationStateHasMore, step.Page.ContinuationState)
		step.Page.Items[0].CandidateGroups[0] = "changed"
		return SearchPageActionStop, nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Total)
	require.Equal(t, int32(1), got.Pages)
	require.Equal(t, SearchCompletionVisitorStopped, got.Completion)
	require.Equal(t, []string{"ops"}, got.Items[0].CandidateGroups)
	require.Equal(t, []string{"ops"}, source[0].CandidateGroups)
}

// TestClientSearchUserTasksPreservesEmptyCollections verifies successful empty
// search and visitor results always expose non-nil public item arrays.
func TestClientSearchUserTasksPreservesEmptyCollections(t *testing.T) {
	t.Parallel()

	api := facadeSearchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, page d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
		return d.UserTaskSearchPage{
			Request: page, ReportedTotal: &d.UserTaskReportedTotal{Kind: d.UserTaskReportedTotalKindExact},
			ContinuationState: d.UserTaskContinuationStateNoMore,
		}, nil
	}}
	client := New(nil, nil, api, nil)

	collected, err := client.SearchUserTasks(context.Background(), SearchRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(0), collected.Total)
	require.NotNil(t, collected.Items)

	pages, err := client.SearchUserTasksPages(context.Background(), SearchRequest{}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), pages.Total)
	require.NotNil(t, pages.Items)
	require.Equal(t, SearchCompletionExhausted, pages.Completion)
}

// TestClientSearchUserTasksMapsVisitorAndServiceErrors verifies visitor and
// backend failures cross the facade through the established error classes.
func TestClientSearchUserTasksMapsVisitorAndServiceErrors(t *testing.T) {
	t.Parallel()

	t.Run("visitor", func(t *testing.T) {
		t.Parallel()
		api := facadeSearchUserTaskAPI{searchPage: func(_ context.Context, _ d.UserTaskSearchQuery, page d.UserTaskPageRequest, _ ...services.CallOption) (d.UserTaskSearchPage, error) {
			return d.UserTaskSearchPage{Items: []d.UserTask{{Key: "task-a"}}, Request: page, RawItemCount: 1, ContinuationState: d.UserTaskContinuationStateHasMore}, nil
		}}
		client := New(nil, nil, api, nil)
		visitorErr := fmt.Errorf("render page: %w", d.ErrForbidden)

		got, err := client.SearchUserTasksPages(context.Background(), SearchRequest{}, func(SearchPageStep) (SearchPageAction, error) {
			return SearchPageActionContinue, visitorErr
		})
		require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
		require.ErrorContains(t, err, visitorErr.Error())
		require.NotNil(t, got.Items)
		require.Empty(t, got.Items)
	})

	t.Run("service", func(t *testing.T) {
		t.Parallel()
		serviceErr := fmt.Errorf("search page: %w", d.ErrUnavailable)
		api := facadeSearchUserTaskAPI{searchPage: func(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
			return d.UserTaskSearchPage{}, serviceErr
		}}
		client := New(nil, nil, api, nil)

		got, err := client.SearchUserTasks(context.Background(), SearchRequest{})
		require.ErrorIs(t, err, ferr.ErrUnavailable)
		require.ErrorContains(t, err, serviceErr.Error())
		require.Equal(t, UserTasks{}, got)
	})
}

// TestClientSearchUserTasksTotalDelegatesExactCount verifies count requests
// preserve facade options and return the matching count rather than a list size.
func TestClientSearchUserTasksTotalDelegatesExactCount(t *testing.T) {
	t.Parallel()

	api := facadeSearchUserTaskAPI{searchPage: func(_ context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error) {
		require.Equal(t, "accounting", query.CandidateGroup)
		require.Equal(t, int32(50), query.BatchSize)
		require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
		return d.UserTaskSearchPage{
			Request: page, ReportedTotal: &d.UserTaskReportedTotal{Count: int64(1) << 40, Kind: d.UserTaskReportedTotalKindExact},
			ContinuationState: d.UserTaskContinuationStateIndeterminate,
		}, nil
	}}
	client := New(nil, nil, api, nil)

	total, err := client.SearchUserTasksTotal(context.Background(), SearchRequest{CandidateGroup: "accounting", BatchSize: 50, Limit: 1}, options.WithIgnoreTenant())
	require.NoError(t, err)
	require.Equal(t, int64(1)<<40, total)
}
