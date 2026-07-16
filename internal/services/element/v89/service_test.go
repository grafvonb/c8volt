// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

type mockElementClient struct {
	getElementInstanceWithResponse     func(context.Context, camundav89.ElementInstanceKey, ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error)
	searchElementInstancesWithResponse func(context.Context, camundav89.SearchElementInstancesJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error)
}

func (m *mockElementClient) GetElementInstanceWithResponse(ctx context.Context, key camundav89.ElementInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
	if m.getElementInstanceWithResponse == nil {
		panic("unexpected GetElementInstanceWithResponse call")
	}
	return m.getElementInstanceWithResponse(ctx, key, reqEditors...)
}

func (m *mockElementClient) SearchElementInstancesWithResponse(ctx context.Context, body camundav89.SearchElementInstancesJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
	if m.searchElementInstancesWithResponse == nil {
		panic("unexpected SearchElementInstancesWithResponse call")
	}
	return m.searchElementInstancesWithResponse(ctx, body, reqEditors...)
}

func TestService_GetElement_MapsPayload(t *testing.T) {
	start := time.Date(2026, 7, 15, 10, 12, 1, 123, time.UTC)
	end := time.Date(2026, 7, 15, 10, 13, 2, 456, time.UTC)
	rootKey := camundav89.ProcessInstanceKey("2251799813688001")
	incidentKey := camundav89.IncidentKey("2251799813687777")
	svc := newElementServiceTest(t, &mockElementClient{
		getElementInstanceWithResponse: func(_ context.Context, key camundav89.ElementInstanceKey, _ ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
			require.Equal(t, camundav89.ElementInstanceKey("2251799813689002"), key)
			return &camundav89.GetElementInstanceResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.ElementInstanceResult{
					ElementInstanceKey:     key,
					ElementId:              "ship-order",
					ElementName:            "Ship order",
					Type:                   camundav89.ElementInstanceResultType("SERVICE_TASK"),
					State:                  camundav89.ElementInstanceStateEnum("ACTIVE"),
					StartDate:              start,
					EndDate:                &end,
					ProcessInstanceKey:     "2251799813688001",
					RootProcessInstanceKey: &rootKey,
					ProcessDefinitionId:    "order-process",
					ProcessDefinitionKey:   "2251799813687001",
					TenantId:               "tenant-a",
					HasIncident:            true,
					IncidentKey:            &incidentKey,
				},
			}, nil
		},
	})

	element, err := svc.GetElement(context.Background(), "2251799813689002")

	require.NoError(t, err)
	require.Equal(t, d.Element{
		ElementInstanceKey:     "2251799813689002",
		ElementId:              "ship-order",
		ElementName:            "Ship order",
		Type:                   "SERVICE_TASK",
		State:                  "ACTIVE",
		StartDate:              "2026-07-15T10:12:01.000000123Z",
		EndDate:                "2026-07-15T10:13:02.000000456Z",
		ProcessInstanceKey:     "2251799813688001",
		RootProcessInstanceKey: "2251799813688001",
		ProcessDefinitionId:    "order-process",
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
		HasIncident:            true,
		IncidentKey:            "2251799813687777",
	}, element)
}

func TestService_GetElement_NotFound(t *testing.T) {
	svc := newElementServiceTest(t, &mockElementClient{
		getElementInstanceWithResponse: func(_ context.Context, key camundav89.ElementInstanceKey, _ ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
			require.Equal(t, camundav89.ElementInstanceKey("2251799813689999"), key)
			return &camundav89.GetElementInstanceResponse{
				HTTPResponse: responseWithStatus(http.StatusNotFound),
				Body:         []byte(`{"title":"not found"}`),
			}, nil
		},
	})

	element, err := svc.GetElement(context.Background(), "2251799813689999")

	require.ErrorIs(t, err, d.ErrNotFound)
	require.Empty(t, element)
}

func TestService_SearchElements_ConstructsFiltersAndConvertsRows(t *testing.T) {
	start := time.Date(2026, 7, 15, 10, 12, 1, 0, time.UTC)
	end := time.Date(2026, 7, 15, 10, 13, 2, 0, time.UTC)
	rootKey := camundav89.ProcessInstanceKey("2251799813688001")
	incidentKey := camundav89.IncidentKey("2251799813687777")
	svc := newElementServiceTest(t, &mockElementClient{
		searchElementInstancesWithResponse: func(_ context.Context, body camundav89.SearchElementInstancesJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
			requireElementSearchFilterJSON(t, body, map[string]any{
				"processInstanceKey":   "2251799813688001",
				"elementId":            "ship-order",
				"state":                "ACTIVE",
				"type":                 "SERVICE_TASK",
				"processDefinitionKey": "2251799813687001",
				"processDefinitionId":  "order-process",
			}, 25)
			return &camundav89.SearchElementInstancesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.ElementInstanceSearchQueryResult{
					Items: []camundav89.ElementInstanceResult{{
						ElementInstanceKey:     "2251799813689002",
						ElementId:              "ship-order",
						ElementName:            "Ship order",
						Type:                   camundav89.ElementInstanceResultType("SERVICE_TASK"),
						State:                  camundav89.ElementInstanceStateEnumACTIVE,
						StartDate:              start,
						EndDate:                &end,
						ProcessInstanceKey:     "2251799813688001",
						RootProcessInstanceKey: &rootKey,
						ProcessDefinitionId:    "order-process",
						ProcessDefinitionKey:   "2251799813687001",
						TenantId:               "tenant-a",
						HasIncident:            true,
						IncidentKey:            &incidentKey,
					}},
					Page: camundav89.SearchQueryPageResponse{TotalItems: 1},
				},
			}, nil
		},
	})

	result, err := svc.SearchElements(context.Background(), d.ElementSearchQuery{
		ProcessInstanceKey:   "2251799813688001",
		ElementId:            "ship-order",
		State:                "ACTIVE",
		Type:                 "SERVICE_TASK",
		ProcessDefinitionKey: "2251799813687001",
		BpmnProcessId:        "order-process",
		Limit:                25,
	})

	require.NoError(t, err)
	require.Equal(t, int32(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, d.Element{
		ElementInstanceKey:     "2251799813689002",
		ElementId:              "ship-order",
		ElementName:            "Ship order",
		Type:                   "SERVICE_TASK",
		State:                  "ACTIVE",
		StartDate:              "2026-07-15T10:12:01Z",
		EndDate:                "2026-07-15T10:13:02Z",
		ProcessInstanceKey:     "2251799813688001",
		RootProcessInstanceKey: "2251799813688001",
		ProcessDefinitionId:    "order-process",
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
		HasIncident:            true,
		IncidentKey:            "2251799813687777",
	}, result.Items[0])
}

func TestService_SearchElementsPagesByBatchSizeUntilLimit(t *testing.T) {
	var requests []camundav89.SearchElementInstancesJSONRequestBody
	svc := newElementServiceTest(t, &mockElementClient{
		searchElementInstancesWithResponse: func(_ context.Context, body camundav89.SearchElementInstancesJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
			requests = append(requests, body)
			key := camundav89.ElementInstanceKey("2251799813689002")
			if len(requests) == 2 {
				key = "2251799813689003"
			}
			return &camundav89.SearchElementInstancesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.ElementInstanceSearchQueryResult{
					Items: []camundav89.ElementInstanceResult{{
						ElementInstanceKey: key,
						ElementId:          "ship-order",
						Type:               camundav89.ElementInstanceResultType("SERVICE_TASK"),
						State:              camundav89.ElementInstanceStateEnumACTIVE,
					}},
					Page: camundav89.SearchQueryPageResponse{TotalItems: 3},
				},
			}, nil
		},
	})

	result, err := svc.SearchElements(context.Background(), d.ElementSearchQuery{BatchSize: 1, Limit: 2})

	require.NoError(t, err)
	require.Equal(t, int32(2), result.Total)
	require.Len(t, result.Items, 2)
	requireElementSearchPageJSON(t, requests[0], 0, 1)
	requireElementSearchPageJSON(t, requests[1], 1, 1)
	require.Equal(t, "2251799813689002", result.Items[0].ElementInstanceKey)
	require.Equal(t, "2251799813689003", result.Items[1].ElementInstanceKey)
}

func TestService_SearchElementsPageReportsLowerBoundTotal(t *testing.T) {
	svc := newElementServiceTest(t, &mockElementClient{
		searchElementInstancesWithResponse: func(_ context.Context, body camundav89.SearchElementInstancesJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
			requireElementSearchPageJSON(t, body, 100, 25)
			return &camundav89.SearchElementInstancesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.ElementInstanceSearchQueryResult{
					Items: []camundav89.ElementInstanceResult{{
						ElementInstanceKey: "2251799813689002",
						ElementId:          "ship-order",
						Type:               camundav89.ElementInstanceResultType("SERVICE_TASK"),
						State:              camundav89.ElementInstanceStateEnumACTIVE,
					}},
					Page: camundav89.SearchQueryPageResponse{TotalItems: 10000, HasMoreTotalItems: true},
				},
			}, nil
		},
	})

	page, err := svc.SearchElementsPage(context.Background(), d.ElementSearchQuery{State: "ACTIVE"}, d.ElementPageRequest{From: 100, Size: 25})

	require.NoError(t, err)
	require.Equal(t, d.ElementPageRequest{From: 100, Size: 25}, page.Request)
	require.Equal(t, d.ProcessInstanceOverflowStateHasMore, page.OverflowState)
	require.NotNil(t, page.ReportedTotal)
	require.Equal(t, int64(10000), page.ReportedTotal.Count)
	require.Equal(t, d.ElementReportedTotalKindLowerBound, page.ReportedTotal.Kind)
}

func newElementServiceTest(t *testing.T, client GenElementClient) *Service {
	t.Helper()

	svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client))
	require.NoError(t, err)
	return svc
}

func okHTTPResponse() *http.Response {
	return responseWithStatus(http.StatusOK)
}

func responseWithStatus(status int) *http.Response {
	u, _ := url.Parse("http://camunda.example.test/v2/element-instances/2251799813689002")
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Request:    &http.Request{Method: http.MethodGet, URL: u},
	}
}

func requireElementSearchFilterJSON(t *testing.T, body camundav89.SearchElementInstancesJSONRequestBody, want map[string]any, wantLimit int32) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	filter := got["filter"].(map[string]any)
	for name, value := range want {
		require.Equal(t, value, filter[name], "filter %s", name)
	}
	page := got["page"].(map[string]any)
	require.Equal(t, float64(wantLimit), page["limit"])
}

func requireElementSearchPageJSON(t *testing.T, body camundav89.SearchElementInstancesJSONRequestBody, wantFrom int32, wantLimit int32) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	page := got["page"].(map[string]any)
	require.Equal(t, float64(wantFrom), page["from"])
	require.Equal(t, float64(wantLimit), page["limit"])
}
