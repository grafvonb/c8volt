// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v89 "github.com/grafvonb/c8volt/internal/services/incident/v89"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

type mockIncidentClient struct {
	getIncident                    func(context.Context, camundav89.IncidentKey, ...camundav89.RequestEditorFn) (*camundav89.GetIncidentResponse, error)
	resolveIncident                func(context.Context, camundav89.IncidentKey, camundav89.ResolveIncidentJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.ResolveIncidentResponse, error)
	searchIncidents                func(context.Context, camundav89.SearchIncidentsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchIncidentsResponse, error)
	searchProcessInstanceIncidents func(context.Context, camundav89.ProcessInstanceKey, camundav89.SearchProcessInstanceIncidentsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error)
}

func (m mockIncidentClient) GetIncidentWithResponse(ctx context.Context, key camundav89.IncidentKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetIncidentResponse, error) {
	return m.getIncident(ctx, key, reqEditors...)
}

func (m mockIncidentClient) ResolveIncidentWithResponse(ctx context.Context, key camundav89.IncidentKey, body camundav89.ResolveIncidentJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.ResolveIncidentResponse, error) {
	return m.resolveIncident(ctx, key, body, reqEditors...)
}

func (m mockIncidentClient) SearchIncidentsWithResponse(ctx context.Context, body camundav89.SearchIncidentsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchIncidentsResponse, error) {
	return m.searchIncidents(ctx, body, reqEditors...)
}

func (m mockIncidentClient) SearchProcessInstanceIncidentsWithResponse(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.SearchProcessInstanceIncidentsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error) {
	return m.searchProcessInstanceIncidents(ctx, key, body, reqEditors...)
}

func TestResolveIncidentMapsAcceptedResponse(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, mockIncidentClient{
		resolveIncident: func(_ context.Context, key camundav89.IncidentKey, body camundav89.ResolveIncidentJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.ResolveIncidentResponse, error) {
			require.Equal(t, "2251799813685249", key)
			require.Nil(t, body.OperationReference)
			return &camundav89.ResolveIncidentResponse{HTTPResponse: testHTTPResponse(http.StatusNoContent), Body: nil}, nil
		},
	})

	got, err := svc.ResolveIncident(context.Background(), "2251799813685249")

	require.NoError(t, err)
	require.Equal(t, d.IncidentResolutionResponse{
		Key:        "2251799813685249",
		Ok:         true,
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
	}, got)
}

func TestGetIncidentMapsDetail(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, mockIncidentClient{
		getIncident: func(_ context.Context, key camundav89.IncidentKey, _ ...camundav89.RequestEditorFn) (*camundav89.GetIncidentResponse, error) {
			require.Equal(t, "2251799813685249", key)
			jobKey := "2251799813685251"
			rootKey := "2251799813685250"
			return &camundav89.GetIncidentResponse{
				HTTPResponse: testHTTPResponse(http.StatusOK),
				JSON200: &camundav89.IncidentResult{
					IncidentKey:            "2251799813685249",
					ProcessInstanceKey:     "2251799813685250",
					TenantId:               "tenant-a",
					State:                  camundav89.IncidentStateEnumACTIVE,
					ErrorType:              camundav89.IncidentErrorTypeEnumJOBNORETRIES,
					ErrorMessage:           "no retries left",
					ElementId:              "task-a",
					ElementInstanceKey:     "2251799813685252",
					JobKey:                 &jobKey,
					RootProcessInstanceKey: &rootKey,
					ProcessDefinitionKey:   "2251799813685253",
					ProcessDefinitionId:    "order-process",
				},
			}, nil
		},
	})

	got, err := svc.GetIncident(context.Background(), "2251799813685249")

	require.NoError(t, err)
	require.Equal(t, d.ProcessInstanceIncidentDetail{
		IncidentKey:            "2251799813685249",
		ProcessInstanceKey:     "2251799813685250",
		TenantId:               "tenant-a",
		State:                  "ACTIVE",
		ErrorType:              "JOB_NO_RETRIES",
		ErrorMessage:           "no retries left",
		FlowNodeId:             "task-a",
		FlowNodeInstanceKey:    "2251799813685252",
		JobKey:                 "2251799813685251",
		RootProcessInstanceKey: "2251799813685250",
		ProcessDefinitionKey:   "2251799813685253",
		ProcessDefinitionId:    "order-process",
	}, got)
}

func TestWaitForProcessInstanceIncidentsResolvedPollsInitialSetOnly(t *testing.T) {
	t.Parallel()

	attempts := 0
	svc := newTestService(t, mockIncidentClient{
		searchProcessInstanceIncidents: func(_ context.Context, key camundav89.ProcessInstanceKey, body camundav89.SearchProcessInstanceIncidentsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error) {
			require.Equal(t, "2251799813685250", key)
			require.NotNil(t, body.Filter)
			require.NotNil(t, body.Filter.State)
			state, err := body.Filter.State.AsIncidentStateFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.IncidentStateEnumACTIVE, state)
			attempts++
			items := []camundav89.IncidentResult{
				{IncidentKey: "other", ProcessInstanceKey: key, State: camundav89.IncidentStateEnumACTIVE},
			}
			if attempts == 1 {
				items = append(items, camundav89.IncidentResult{IncidentKey: "2251799813685249", ProcessInstanceKey: key, State: camundav89.IncidentStateEnumACTIVE})
			}
			return &camundav89.SearchProcessInstanceIncidentsResponse{
				HTTPResponse: testHTTPResponse(http.StatusOK),
				JSON200:      &camundav89.IncidentSearchQueryResult{Items: items},
			}, nil
		},
	})

	got, err := svc.WaitForProcessInstanceIncidentsResolved(context.Background(), "2251799813685250", []string{"2251799813685249"})

	require.NoError(t, err)
	require.True(t, got.Ok)
	require.Equal(t, "2251799813685250", got.Key)
	require.Equal(t, 2, attempts)
}

func TestSearchProcessInstanceIncidentsUsesRequestedStateScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		option    services.CallOption
		wantState *camundav89.IncidentStateEnum
	}{
		{name: "default active", wantState: ptrIncidentState89(camundav89.IncidentStateEnumACTIVE)},
		{name: "pending", option: services.WithIncidentState("pending"), wantState: ptrIncidentState89(camundav89.IncidentStateEnumPENDING)},
		{name: "resolved", option: services.WithIncidentState("resolved"), wantState: ptrIncidentState89(camundav89.IncidentStateEnumRESOLVED)},
		{name: "migrated", option: services.WithIncidentState("migrated"), wantState: ptrIncidentState89(camundav89.IncidentStateEnumMIGRATED)},
		{name: "unknown", option: services.WithIncidentState("unknown"), wantState: ptrIncidentState89(camundav89.IncidentStateEnumUNKNOWN)},
		{name: "all", option: services.WithIncidentState("all"), wantState: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestService(t, mockIncidentClient{
				searchProcessInstanceIncidents: func(_ context.Context, key camundav89.ProcessInstanceKey, body camundav89.SearchProcessInstanceIncidentsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error) {
					require.Equal(t, camundav89.ProcessInstanceKey("2251799813685250"), key)
					require.NotNil(t, body.Filter)
					if tt.wantState == nil {
						require.Nil(t, body.Filter.State)
					} else {
						require.NotNil(t, body.Filter.State)
						state, err := body.Filter.State.AsIncidentStateFilterProperty0()
						require.NoError(t, err)
						require.Equal(t, *tt.wantState, state)
					}
					return &camundav89.SearchProcessInstanceIncidentsResponse{
						HTTPResponse: testHTTPResponse(http.StatusOK),
						JSON200:      &camundav89.IncidentSearchQueryResult{Items: nil},
					}, nil
				},
			})

			opts := []services.CallOption{}
			if tt.option != nil {
				opts = append(opts, tt.option)
			}
			got, err := svc.SearchProcessInstanceIncidents(context.Background(), "2251799813685250", opts...)

			require.NoError(t, err)
			require.Empty(t, got)
		})
	}
}

func TestSearchProcessInstanceIncidentsUsesServerSideErrorTypeAndPagesMessageFilter(t *testing.T) {
	t.Parallel()

	var fromValues []int32
	svc := newTestService(t, mockIncidentClient{
		searchProcessInstanceIncidents: func(_ context.Context, key camundav89.ProcessInstanceKey, body camundav89.SearchProcessInstanceIncidentsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error) {
			require.Equal(t, camundav89.ProcessInstanceKey("2251799813685250"), key)
			require.NotNil(t, body.Filter)
			require.NotNil(t, body.Filter.ErrorType)
			errorType, err := body.Filter.ErrorType.AsIncidentErrorTypeFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, errorType)
			require.Nil(t, body.Filter.ErrorMessage)

			require.NotNil(t, body.Page)
			page, err := body.Page.AsOffsetPagination()
			require.NoError(t, err)
			require.NotNil(t, page.From)
			fromValues = append(fromValues, *page.From)

			items := []camundav89.IncidentResult{
				{IncidentKey: "first", ProcessInstanceKey: key, State: camundav89.IncidentStateEnumACTIVE, ErrorType: camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, ErrorMessage: "intentional first"},
				{IncidentKey: "skip-message", ProcessInstanceKey: key, State: camundav89.IncidentStateEnumACTIVE, ErrorType: camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, ErrorMessage: "other failure"},
			}
			pageResp := camundav89.SearchQueryPageResponse{TotalItems: 3}
			if *page.From > 0 {
				items = []camundav89.IncidentResult{{IncidentKey: "second", ProcessInstanceKey: key, State: camundav89.IncidentStateEnumACTIVE, ErrorType: camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, ErrorMessage: "INTENTIONAL second"}}
			}
			return &camundav89.SearchProcessInstanceIncidentsResponse{
				HTTPResponse: testHTTPResponse(http.StatusOK),
				JSON200:      &camundav89.IncidentSearchQueryResult{Items: items, Page: pageResp},
			}, nil
		},
	})

	got, err := svc.SearchProcessInstanceIncidents(context.Background(), "2251799813685250",
		services.WithIncidentErrorType("io_mapping_error"),
		services.WithIncidentErrorMessage("intentional"),
	)

	require.NoError(t, err)
	require.Equal(t, []int32{0, 2}, fromValues)
	require.Equal(t, []string{"first", "second"}, incidentDetailKeys(got))
}

func TestSearchIncidentsPageUsesServerFiltersAndLocalMessageFiltering(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, mockIncidentClient{
		searchIncidents: func(_ context.Context, body camundav89.SearchIncidentsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchIncidentsResponse, error) {
			require.NotNil(t, body.Filter)
			state, err := body.Filter.State.AsIncidentStateFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.IncidentStateEnumRESOLVED, state)
			errorType, err := body.Filter.ErrorType.AsIncidentErrorTypeFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, errorType)
			processInstanceKey, err := body.Filter.ProcessInstanceKey.AsProcessInstanceKeyFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.ProcessInstanceKey("pi-a"), processInstanceKey)
			processDefinitionKey, err := body.Filter.ProcessDefinitionKey.AsProcessDefinitionKeyFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.ProcessDefinitionKey("pd-key"), processDefinitionKey)
			processDefinitionID, err := body.Filter.ProcessDefinitionId.AsStringFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, "pd-id", processDefinitionID)
			elementID, err := body.Filter.ElementId.AsStringFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, "task-a", elementID)
			elementInstanceKey, err := body.Filter.ElementInstanceKey.AsElementInstanceKeyFilterProperty0()
			require.NoError(t, err)
			require.Equal(t, camundav89.ElementInstanceKey("fni-a"), elementInstanceKey)
			require.Nil(t, body.Filter.ErrorMessage)
			require.NotNil(t, body.Filter.CreationTime)
			creationTime, err := body.Filter.CreationTime.AsAdvancedDateTimeFilter()
			require.NoError(t, err)
			require.NotNil(t, creationTime.Gte)
			require.NotNil(t, creationTime.Lte)
			require.Equal(t, time.Date(2026, 5, 9, 9, 0, 0, 0, time.UTC), *creationTime.Gte)
			require.Equal(t, time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC), *creationTime.Lte)

			require.NotNil(t, body.Page)
			page, err := body.Page.AsOffsetPagination()
			require.NoError(t, err)
			require.NotNil(t, page.From)
			require.EqualValues(t, 0, *page.From)
			require.NotNil(t, page.Limit)
			require.EqualValues(t, 2, *page.Limit)
			rootKey := "root-a"
			return &camundav89.SearchIncidentsResponse{
				HTTPResponse: testHTTPResponse(http.StatusOK),
				JSON200: &camundav89.IncidentSearchQueryResult{
					Items: []camundav89.IncidentResult{
						{IncidentKey: "match", ProcessInstanceKey: "pi-a", State: camundav89.IncidentStateEnumRESOLVED, ErrorType: camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, ErrorMessage: "INTENTIONAL failure", RootProcessInstanceKey: &rootKey},
						{IncidentKey: "skip-message", ProcessInstanceKey: "pi-a", State: camundav89.IncidentStateEnumRESOLVED, ErrorType: camundav89.IncidentErrorTypeEnumIOMAPPINGERROR, ErrorMessage: "other failure", RootProcessInstanceKey: &rootKey},
					},
					Page: camundav89.SearchQueryPageResponse{TotalItems: 2},
				},
			}, nil
		},
	})

	got, err := svc.SearchIncidentsPage(context.Background(), d.IncidentFilter{
		State:                  "resolved",
		ErrorType:              "io_mapping_error",
		ErrorMessage:           "intentional",
		ProcessInstanceKey:     "pi-a",
		ProcessDefinitionKey:   "pd-key",
		ProcessDefinitionId:    "pd-id",
		FlowNodeId:             "task-a",
		FlowNodeInstanceKey:    "fni-a",
		CreationTimeAfter:      "2026-05-09T09:00:00Z",
		CreationTimeBefore:     "2026-05-09T11:00:00Z",
		RootProcessInstanceKey: "root-a",
	}, d.IncidentPageRequest{Size: 2})

	require.NoError(t, err)
	require.Nil(t, got.ReportedTotal)
	require.Equal(t, []string{"match"}, incidentDetailKeys(got.Items))
}

func ptrIncidentState89(v camundav89.IncidentStateEnum) *camundav89.IncidentStateEnum {
	return &v
}

func incidentDetailKeys(items []d.ProcessInstanceIncidentDetail) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.IncidentKey)
	}
	return keys
}

func newTestService(t *testing.T, client mockIncidentClient) *v89.Service {
	t.Helper()
	cfg := &config.Config{
		App: config.App{
			CamundaVersion: toolx.V89,
			Backoff: config.BackoffConfig{
				Strategy:     config.BackoffFixed,
				InitialDelay: time.Millisecond,
				MaxRetries:   3,
				Timeout:      50 * time.Millisecond,
			},
		},
		APIs: config.APIs{Camunda: config.API{BaseURL: "http://localhost:8080/v2"}},
	}
	svc, err := v89.New(cfg, &http.Client{}, slog.Default(), v89.WithClientCamunda(client))
	require.NoError(t, err)
	return svc
}

func testHTTPResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(nil),
		Request:    &http.Request{Method: http.MethodPost, URL: &url.URL{Scheme: "http", Host: "localhost", Path: "/v2/incidents/1/resolution"}},
	}
}

var _ v89.GenIncidentClientCamunda = mockIncidentClient{}
