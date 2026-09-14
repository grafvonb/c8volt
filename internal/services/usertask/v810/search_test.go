// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestService_SearchUserTasksPage_MapsAllFiltersAndPaginationUnions verifies v8.10 uses equality unions and one page variant per request.
func TestService_SearchUserTasksPage_MapsAllFiltersAndPaginationUnions(t *testing.T) {
	query := d.UserTaskSearchQuery{
		ProcessInstanceKey: "2251799813711967", ProcessDefinitionKey: "2251799813689000",
		BpmnProcessId: "invoice", ElementId: "approve_invoice", State: "CREATED",
		Assignee: "alice", CandidateUser: "bob", CandidateGroup: "accounting",
	}
	pageRequests := []struct {
		name  string
		page  d.UserTaskPageRequest
		check func(*testing.T, camundav810.SearchQueryPageRequest)
	}{
		{name: "limit", page: d.UserTaskPageRequest{Size: 100}, check: func(t *testing.T, page camundav810.SearchQueryPageRequest) {
			got, err := page.AsLimitPagination()
			require.NoError(t, err)
			require.Equal(t, int32(100), *got.Limit)
		}},
		{name: "offset", page: d.UserTaskPageRequest{From: 200, Size: 50}, check: func(t *testing.T, page camundav810.SearchQueryPageRequest) {
			got, err := page.AsOffsetPagination()
			require.NoError(t, err)
			require.Equal(t, int32(200), *got.From)
			require.Equal(t, int32(50), *got.Limit)
		}},
		{name: "cursor", page: d.UserTaskPageRequest{After: "cursor-a", Size: 25}, check: func(t *testing.T, page camundav810.SearchQueryPageRequest) {
			got, err := page.AsCursorForwardPagination()
			require.NoError(t, err)
			require.Equal(t, "cursor-a", string(*got.After))
			require.Equal(t, int32(25), *got.Limit)
		}},
	}
	for _, tt := range pageRequests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, &mockUserTaskClient{searchUserTasksWithResponse: func(_ context.Context, body camundav810.SearchUserTasksJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.SearchUserTasksResponse, error) {
				require.NotNil(t, body.Filter)
				requireV810UserTaskFilter(t, *body.Filter)
				require.NotNil(t, body.Page)
				tt.check(t, *body.Page)
				return v810SearchResponse(http.StatusOK, &camundav810.UserTaskSearchQueryResult{}), nil
			}}, "tenant-a")

			_, err := svc.SearchUserTasksPage(context.Background(), query, tt.page)
			require.NoError(t, err)
		})
	}
}

// TestService_SearchUserTasksPage_MapsItemsAndPageFacts verifies nullable fields, raw counts, cursors, totals, and continuation normalization.
func TestService_SearchUserTasksPage_MapsItemsAndPageFacts(t *testing.T) {
	endCursor := camundav810.EndCursor("cursor-b")
	payload := &camundav810.UserTaskSearchQueryResult{
		Items: []camundav810.UserTaskResult{{
			UserTaskKey: "task-a", State: camundav810.UserTaskStateEnumCREATED,
			ElementId: "approve_invoice", ProcessInstanceKey: "process-a", TenantId: "tenant-a",
		}},
		Page: camundav810.SearchQueryPageResponse{EndCursor: &endCursor, TotalItems: 500, HasMoreTotalItems: true},
	}
	svc := newTestService(t, &mockUserTaskClient{searchUserTasksWithResponse: func(context.Context, camundav810.SearchUserTasksJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchUserTasksResponse, error) {
		return v810SearchResponse(http.StatusOK, payload), nil
	}})
	request := d.UserTaskPageRequest{Size: 100}

	page, err := svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, request)

	require.NoError(t, err)
	require.Equal(t, request, page.Request)
	require.Equal(t, int32(1), page.RawItemCount)
	require.Equal(t, "cursor-b", page.EndCursor)
	require.Equal(t, &d.UserTaskReportedTotal{Count: 500, Kind: d.UserTaskReportedTotalKindLowerBound}, page.ReportedTotal)
	require.Equal(t, d.UserTaskContinuationStateHasMore, page.ContinuationState)
	require.Equal(t, d.UserTask{Key: "task-a", State: "CREATED", ElementId: "approve_invoice", ProcessInstanceKey: "process-a", TenantId: "tenant-a"}, page.Items[0])
	require.Empty(t, page.Items[0].Name)
	require.Empty(t, page.Items[0].Assignee)

	payload.Items = nil
	payload.Page = camundav810.SearchQueryPageResponse{TotalItems: 0}
	page, err = svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, request)
	require.NoError(t, err)
	require.NotNil(t, page.Items)
	require.Equal(t, &d.UserTaskReportedTotal{Count: 0, Kind: d.UserTaskReportedTotalKindExact}, page.ReportedTotal)
	require.Equal(t, d.UserTaskContinuationStateNoMore, page.ContinuationState)
}

// TestService_SearchUserTasksPage_RejectsBackendAndMalformedResponses pins transport, HTTP, payload, and identity failures.
func TestService_SearchUserTasksPage_RejectsBackendAndMalformedResponses(t *testing.T) {
	transportErr := errors.New("connection reset")
	tests := []struct {
		name      string
		response  *camundav810.SearchUserTasksResponse
		transport error
		want      error
	}{
		{name: "transport", transport: transportErr, want: transportErr},
		{name: "backend", response: v810SearchResponse(http.StatusServiceUnavailable, nil), want: d.ErrUnavailable},
		{name: "empty payload", response: v810SearchResponse(http.StatusOK, nil), want: d.ErrMalformedResponse},
		{name: "malformed item", response: v810SearchResponse(http.StatusOK, &camundav810.UserTaskSearchQueryResult{Items: []camundav810.UserTaskResult{{UserTaskKey: "task-a"}}}), want: d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, &mockUserTaskClient{searchUserTasksWithResponse: func(context.Context, camundav810.SearchUserTasksJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchUserTasksResponse, error) {
				return tt.response, tt.transport
			}})
			_, err := svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, d.UserTaskPageRequest{Size: 10})
			require.ErrorIs(t, err, tt.want)
		})
	}
}

// TestV810UserTaskStatesMatchStableMembership pins the nine lifecycle states exposed by the checked-in contract.
func TestV810UserTaskStatesMatchStableMembership(t *testing.T) {
	states := []camundav810.UserTaskStateEnum{
		camundav810.UserTaskStateEnumASSIGNING, camundav810.UserTaskStateEnumCANCELED,
		camundav810.UserTaskStateEnumCANCELING, camundav810.UserTaskStateEnumCOMPLETED,
		camundav810.UserTaskStateEnumCOMPLETING, camundav810.UserTaskStateEnumCREATED,
		camundav810.UserTaskStateEnumCREATING, camundav810.UserTaskStateEnumFAILED,
		camundav810.UserTaskStateEnumUPDATING,
	}
	require.Equal(t, []camundav810.UserTaskStateEnum{
		"ASSIGNING", "CANCELED", "CANCELING", "COMPLETED", "COMPLETING",
		"CREATED", "CREATING", "FAILED", "UPDATING",
	}, states)
}

// requireV810UserTaskFilter decodes each union to prove all predicates are combined as exact matches.
func requireV810UserTaskFilter(t *testing.T, filter camundav810.UserTaskFilter) {
	t.Helper()
	pi, err := filter.ProcessInstanceKey.AsProcessInstanceKeyFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, "2251799813711967", string(pi))
	pd, err := filter.ProcessDefinitionKey.AsProcessDefinitionKeyFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, "2251799813689000", string(pd))
	bpmn, err := filter.ProcessDefinitionId.AsProcessDefinitionIdFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, "invoice", string(bpmn))
	require.Equal(t, "approve_invoice", string(*filter.ElementId))
	state, err := filter.State.AsUserTaskStateFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, camundav810.UserTaskStateEnumCREATED, state)
	for expected, generated := range map[string]*camundav810.StringFilterProperty{
		"alice": filter.Assignee, "bob": filter.CandidateUser, "accounting": filter.CandidateGroup, "tenant-a": filter.TenantId,
	} {
		actual, err := generated.AsStringFilterProperty0()
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

// v810SearchResponse constructs a generated response with stable HTTP classification data.
func v810SearchResponse(status int, payload *camundav810.UserTaskSearchQueryResult) *camundav810.SearchUserTasksResponse {
	return &camundav810.SearchUserTasksResponse{
		Body:         []byte(`{"message":"backend response"}`),
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", status, http.StatusText(status)),
		JSON200:      payload,
	}
}

// TestService_SearchUserTasksPage_IgnoreTenantOmitsDiscoveryScope verifies the existing all-tenant option removes only the tenant predicate.
func TestService_SearchUserTasksPage_IgnoreTenantOmitsDiscoveryScope(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{searchUserTasksWithResponse: func(_ context.Context, body camundav810.SearchUserTasksJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.SearchUserTasksResponse, error) {
		require.NotNil(t, body.Filter)
		require.Nil(t, body.Filter.TenantId)
		return v810SearchResponse(http.StatusOK, &camundav810.UserTaskSearchQueryResult{}), nil
	}}, "tenant-a")

	_, err := svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, d.UserTaskPageRequest{Size: 10}, services.WithIgnoreTenant())
	require.NoError(t, err)
}
