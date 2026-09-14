// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestService_SearchUserTasksPage_MapsAllFiltersAndPaginationUnions verifies v8.8 scalar selectors, equality properties, and page variants.
func TestService_SearchUserTasksPage_MapsAllFiltersAndPaginationUnions(t *testing.T) {
	query := d.UserTaskSearchQuery{
		ProcessInstanceKey: "2251799813711967", ProcessDefinitionKey: "2251799813689000",
		BpmnProcessId: "invoice", ElementId: "approve_invoice", State: "CREATED",
		Assignee: "alice", CandidateUser: "bob", CandidateGroup: "accounting",
	}
	pageRequests := []struct {
		name  string
		page  d.UserTaskPageRequest
		check func(*testing.T, camundav88.SearchQueryPageRequest)
	}{
		{name: "limit", page: d.UserTaskPageRequest{Size: 100}, check: func(t *testing.T, page camundav88.SearchQueryPageRequest) {
			got, err := page.AsLimitPagination()
			require.NoError(t, err)
			require.Equal(t, int32(100), *got.Limit)
		}},
		{name: "offset", page: d.UserTaskPageRequest{From: 200, Size: 50}, check: func(t *testing.T, page camundav88.SearchQueryPageRequest) {
			got, err := page.AsOffsetPagination()
			require.NoError(t, err)
			require.Equal(t, int32(200), *got.From)
			require.Equal(t, int32(50), *got.Limit)
		}},
		{name: "cursor", page: d.UserTaskPageRequest{After: "cursor-a", Size: 25}, check: func(t *testing.T, page camundav88.SearchQueryPageRequest) {
			got, err := page.AsCursorForwardPagination()
			require.NoError(t, err)
			require.Equal(t, "cursor-a", string(got.After))
			require.Equal(t, int32(25), *got.Limit)
		}},
	}
	for _, tt := range pageRequests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, &mockUserTaskCamundaClient{searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
				require.NotNil(t, body.Filter)
				requireV88UserTaskFilter(t, *body.Filter)
				require.NotNil(t, body.Page)
				tt.check(t, *body.Page)
				return v88SearchResponse(http.StatusOK, &camundav88.UserTaskSearchQueryResult{}), nil
			}}, "tenant-a")

			_, err := svc.SearchUserTasksPage(context.Background(), query, tt.page)
			require.NoError(t, err)
		})
	}
}

// TestService_SearchUserTasksPage_MapsItemsAndPageFacts verifies v8.8 response mapping and capped-total normalization.
func TestService_SearchUserTasksPage_MapsItemsAndPageFacts(t *testing.T) {
	endCursor := camundav88.EndCursor("cursor-b")
	payload := &camundav88.UserTaskSearchQueryResult{
		Items: []camundav88.UserTaskResult{{
			UserTaskKey: "task-a", State: camundav88.UserTaskStateEnumCREATED,
			ElementId: "approve_invoice", ProcessInstanceKey: "process-a", TenantId: "tenant-a",
		}},
		Page: camundav88.SearchQueryPageResponse{EndCursor: &endCursor, TotalItems: 500, HasMoreTotalItems: true},
	}
	svc := newTestService(t, &mockUserTaskCamundaClient{searchUserTasksWithResponse: func(context.Context, camundav88.SearchUserTasksJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
		return v88SearchResponse(http.StatusOK, payload), nil
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
	payload.Page = camundav88.SearchQueryPageResponse{TotalItems: 0}
	page, err = svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, request)
	require.NoError(t, err)
	require.NotNil(t, page.Items)
	require.Equal(t, &d.UserTaskReportedTotal{Count: 0, Kind: d.UserTaskReportedTotalKindExact}, page.ReportedTotal)
	require.Equal(t, d.UserTaskContinuationStateNoMore, page.ContinuationState)
}

// TestService_SearchUserTasksPage_RejectsBackendAndMalformedResponses pins v8.8 transport, HTTP, payload, and identity failures.
func TestService_SearchUserTasksPage_RejectsBackendAndMalformedResponses(t *testing.T) {
	transportErr := errors.New("connection reset")
	tests := []struct {
		name      string
		response  *camundav88.SearchUserTasksResponse
		transport error
		want      error
	}{
		{name: "transport", transport: transportErr, want: transportErr},
		{name: "backend", response: v88SearchResponse(http.StatusServiceUnavailable, nil), want: d.ErrUnavailable},
		{name: "empty payload", response: v88SearchResponse(http.StatusOK, nil), want: d.ErrMalformedResponse},
		{name: "malformed item", response: v88SearchResponse(http.StatusOK, &camundav88.UserTaskSearchQueryResult{Items: []camundav88.UserTaskResult{{UserTaskKey: "task-a"}}}), want: d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, &mockUserTaskCamundaClient{searchUserTasksWithResponse: func(context.Context, camundav88.SearchUserTasksJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
				return tt.response, tt.transport
			}})
			_, err := svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, d.UserTaskPageRequest{Size: 10})
			require.ErrorIs(t, err, tt.want)
		})
	}
}

// TestV88UserTaskStatesMatchStableMembership pins the nine lifecycle states exposed by the checked-in contract.
func TestV88UserTaskStatesMatchStableMembership(t *testing.T) {
	states := []camundav88.UserTaskStateEnum{
		camundav88.UserTaskStateEnumASSIGNING, camundav88.UserTaskStateEnumCANCELED,
		camundav88.UserTaskStateEnumCANCELING, camundav88.UserTaskStateEnumCOMPLETED,
		camundav88.UserTaskStateEnumCOMPLETING, camundav88.UserTaskStateEnumCREATED,
		camundav88.UserTaskStateEnumCREATING, camundav88.UserTaskStateEnumFAILED,
		camundav88.UserTaskStateEnumUPDATING,
	}
	require.Equal(t, []camundav88.UserTaskStateEnum{
		"ASSIGNING", "CANCELED", "CANCELING", "COMPLETED", "COMPLETING",
		"CREATED", "CREATING", "FAILED", "UPDATING",
	}, states)
}

// requireV88UserTaskFilter proves all scalar and equality predicates are present in one AND-combined request.
func requireV88UserTaskFilter(t *testing.T, filter camundav88.UserTaskFilter) {
	t.Helper()
	require.Equal(t, "2251799813711967", string(*filter.ProcessInstanceKey))
	require.Equal(t, "2251799813689000", string(*filter.ProcessDefinitionKey))
	require.Equal(t, "invoice", string(*filter.ProcessDefinitionId))
	require.Equal(t, "approve_invoice", string(*filter.ElementId))
	state, err := filter.State.AsUserTaskStateFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, camundav88.UserTaskStateEnumCREATED, state)
	for expected, generated := range map[string]*camundav88.StringFilterProperty{
		"alice": filter.Assignee, "bob": filter.CandidateUser, "accounting": filter.CandidateGroup, "tenant-a": filter.TenantId,
	} {
		actual, err := generated.AsStringFilterProperty0()
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

// v88SearchResponse constructs a generated response with stable HTTP classification data.
func v88SearchResponse(status int, payload *camundav88.UserTaskSearchQueryResult) *camundav88.SearchUserTasksResponse {
	return &camundav88.SearchUserTasksResponse{
		Body:         []byte(`{"message":"backend response"}`),
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", status, http.StatusText(status)),
		JSON200:      payload,
	}
}

// TestService_SearchUserTasksPage_IgnoreTenantOmitsDiscoveryScope verifies the existing all-tenant option removes only the tenant predicate.
func TestService_SearchUserTasksPage_IgnoreTenantOmitsDiscoveryScope(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
		require.NotNil(t, body.Filter)
		require.Nil(t, body.Filter.TenantId)
		return v88SearchResponse(http.StatusOK, &camundav88.UserTaskSearchQueryResult{}), nil
	}}, "tenant-a")

	_, err := svc.SearchUserTasksPage(context.Background(), d.UserTaskSearchQuery{}, d.UserTaskPageRequest{Size: 10}, services.WithIgnoreTenant())
	require.NoError(t, err)
}
