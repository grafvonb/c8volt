// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
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

// TestElementSearchTenant applies configured tenant selection at the server for pages, collections and both total paths.
func TestElementSearchTenant(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, tenant string
		ignore       bool
		filtered     bool
	}{
		{name: "configured", tenant: "tenant-a", filtered: true},
		{name: "ignored", tenant: "tenant-a", ignore: true},
		{name: "absent"},
		{name: "blank", tenant: "   "},
	} {
		for _, mode := range []string{"page", "collected", "pages", "total", "fallback total"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				requests := 0
				svc := newElementServiceTest(t, &mockElementClient{searchElementInstancesWithResponse: func(_ context.Context, body camundav89.SearchElementInstancesJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
					requests++
					require.NotNil(t, body.Filter)
					if tc.filtered {
						require.NotNil(t, body.Filter.TenantId)
						require.Equal(t, "tenant-a", string(*body.Filter.TenantId))
					} else {
						require.Nil(t, body.Filter.TenantId)
					}
					// This fake acts like the server: filter before paging and reporting totals.
					all := []camundav89.ElementInstanceResult{{ElementInstanceKey: "E1", TenantId: "tenant-a"}, {ElementInstanceKey: "E2", TenantId: "tenant-b"}, {ElementInstanceKey: "E3", TenantId: "tenant-a"}}
					selected := []camundav89.ElementInstanceResult{}
					for _, item := range all {
						if body.Filter.TenantId == nil || item.TenantId == *body.Filter.TenantId {
							selected = append(selected, item)
						}
					}
					raw, err := json.Marshal(body.Page)
					require.NoError(t, err)
					var page struct {
						From  int `json:"from"`
						Limit int `json:"limit"`
					}
					require.NoError(t, json.Unmarshal(raw, &page))
					require.Equal(t, 1, page.Limit)
					items := []camundav89.ElementInstanceResult{}
					if page.From < len(selected) {
						items = append(items, selected[page.From])
					}
					return &camundav89.SearchElementInstancesResponse{HTTPResponse: okHTTPResponse(), JSON200: &camundav89.ElementInstanceSearchQueryResult{Items: items, Page: camundav89.SearchQueryPageResponse{TotalItems: int64(len(selected)), HasMoreTotalItems: mode == "fallback total" && len(items) > 0}}}, nil
				}})
				svc.cfg.App.Tenant = tc.tenant
				opts := []services.CallOption{}
				if tc.ignore {
					opts = append(opts, services.WithIgnoreTenant())
				}
				query := d.ElementSearchQuery{BatchSize: 1}
				want := 3
				if tc.filtered {
					want = 2
				}
				switch mode {
				case "page":
					got, err := svc.SearchElementsPage(context.Background(), query, d.ElementPageRequest{Size: 1}, opts...)
					require.NoError(t, err)
					require.Len(t, got.Items, 1)
					require.Equal(t, int64(want), got.ReportedTotal.Count)
					require.Equal(t, 1, requests)
				case "collected":
					got, err := svc.SearchElements(context.Background(), query, opts...)
					require.NoError(t, err)
					require.Len(t, got.Items, want)
					require.Equal(t, int32(want), got.Total)
					require.Equal(t, want, requests)
					keys := []string{}
					for _, item := range got.Items {
						keys = append(keys, item.ElementInstanceKey)
					}
					if tc.filtered {
						require.Equal(t, []string{"E1", "E3"}, keys)
					} else {
						require.Equal(t, []string{"E1", "E2", "E3"}, keys)
					}
				case "pages":
					visits := 0
					got, err := svc.SearchElementsPages(context.Background(), query, func(step d.ElementSearchPageStep) (d.ElementSearchPageAction, error) {
						visits++
						require.Equal(t, int32(visits), step.CumulativeCount)
						return d.ElementSearchPageActionContinue, nil
					}, opts...)
					require.NoError(t, err)
					require.Len(t, got.Items, want)
					require.Equal(t, want, visits)
					require.Equal(t, want, requests)
				default:
					got, err := svc.SearchElementsTotal(context.Background(), query, opts...)
					require.NoError(t, err)
					require.Equal(t, int64(want), got)
					if mode == "total" {
						require.Equal(t, 1, requests)
					} else {
						require.Equal(t, want+1, requests)
					}
				}
			})
		}
	}
}

// TestElementSearchTenantExplicitGet preserves native cross-tenant exact-key lookup without a search.
func TestElementSearchTenantExplicitGet(t *testing.T) {
	t.Parallel()
	calls := 0
	svc := newElementServiceTest(t, &mockElementClient{getElementInstanceWithResponse: func(_ context.Context, key camundav89.ElementInstanceKey, _ ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
		calls++
		require.Equal(t, camundav89.ElementInstanceKey("E"), key)
		return &camundav89.GetElementInstanceResponse{HTTPResponse: okHTTPResponse(), JSON200: &camundav89.ElementInstanceResult{ElementInstanceKey: key, ProcessInstanceKey: "P", TenantId: "other"}}, nil
	}})
	svc.cfg.App.Tenant = "configured"
	for _, opts := range [][]services.CallOption{nil, {services.WithIgnoreTenant()}} {
		got, err := svc.GetElement(context.Background(), "E", opts...)
		require.NoError(t, err)
		require.Equal(t, "other", got.TenantId)
	}
	require.Equal(t, 2, calls)
}

// TestElementScopeVariablesSparseSelection continues through empty pages with evidence of more data and preserves bounds and visitor stops.
func TestElementScopeVariablesSparseSelection(t *testing.T) {
	for _, mode := range []string{"pages", "collected", "total", "stop", "limit"} {
		for _, lowerBound := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/lower=%t", mode, lowerBound), func(t *testing.T) {
				calls := 0
				svc := newElementServiceTest(t, &mockElementClient{searchElementInstancesWithResponse: func(_ context.Context, body camundav89.SearchElementInstancesJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
					calls++
					require.LessOrEqual(t, calls, 4)
					requireElementSearchPageJSON(t, body, int32(calls-1), 1)
					var items []camundav89.ElementInstanceResult
					if calls == 2 || calls == 4 {
						items = []camundav89.ElementInstanceResult{{ElementInstanceKey: camundav89.ElementInstanceKey(fmt.Sprintf("%d", calls)), ProcessInstanceKey: "100"}}
					}
					total := int64(4)
					more := lowerBound && calls < 4
					if lowerBound {
						total = 0
						if calls == 4 {
							total = 2
						}
					}
					return &camundav89.SearchElementInstancesResponse{HTTPResponse: okHTTPResponse(), JSON200: &camundav89.ElementInstanceSearchQueryResult{Items: items, Page: camundav89.SearchQueryPageResponse{TotalItems: total, HasMoreTotalItems: more}}}, nil
				}})
				query := d.ElementSearchQuery{BatchSize: 1}
				if mode == "limit" {
					query.Limit = 1
				}
				switch mode {
				case "total":
					total, err := svc.SearchElementsTotal(context.Background(), query)
					require.NoError(t, err)
					if lowerBound {
						require.Equal(t, int64(2), total)
						require.Equal(t, 4, calls)
					} else {
						require.Equal(t, int64(4), total)
						require.Equal(t, 1, calls)
					}
				case "collected":
					result, err := svc.SearchElements(context.Background(), query)
					require.NoError(t, err)
					require.Len(t, result.Items, 2)
					require.Equal(t, 4, calls)
				default:
					var steps []d.ElementSearchPageStep
					result, err := svc.SearchElementsPages(context.Background(), query, func(step d.ElementSearchPageStep) (d.ElementSearchPageAction, error) {
						steps = append(steps, step)
						if mode == "stop" && len(step.Page.Items) > 0 {
							return d.ElementSearchPageActionStop, nil
						}
						return d.ElementSearchPageActionContinue, nil
					})
					require.NoError(t, err)
					require.Empty(t, steps[0].Page.Items)
					require.Equal(t, d.ProcessInstanceOverflowStateHasMore, steps[0].Page.OverflowState)
					if mode == "pages" {
						require.Len(t, result.Items, 2)
						require.Equal(t, 4, calls)
					} else {
						require.Len(t, result.Items, 1)
						require.Equal(t, 2, calls)
					}
				}
			})
		}
	}
}
