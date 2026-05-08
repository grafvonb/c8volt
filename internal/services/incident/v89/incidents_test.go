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
	v89 "github.com/grafvonb/c8volt/internal/services/incident/v89"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

type mockIncidentClient struct {
	getIncident                     func(context.Context, camundav89.IncidentKey, ...camundav89.RequestEditorFn) (*camundav89.GetIncidentResponse, error)
	resolveIncident                 func(context.Context, camundav89.IncidentKey, camundav89.ResolveIncidentJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.ResolveIncidentResponse, error)
	resolveProcessInstanceIncidents func(context.Context, camundav89.ProcessInstanceKey, ...camundav89.RequestEditorFn) (*camundav89.ResolveProcessInstanceIncidentsResponse, error)
	searchProcessInstanceIncidents  func(context.Context, camundav89.ProcessInstanceKey, camundav89.SearchProcessInstanceIncidentsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error)
}

func (m mockIncidentClient) GetIncidentWithResponse(ctx context.Context, key camundav89.IncidentKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetIncidentResponse, error) {
	return m.getIncident(ctx, key, reqEditors...)
}

func (m mockIncidentClient) ResolveIncidentWithResponse(ctx context.Context, key camundav89.IncidentKey, body camundav89.ResolveIncidentJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.ResolveIncidentResponse, error) {
	return m.resolveIncident(ctx, key, body, reqEditors...)
}

func (m mockIncidentClient) ResolveProcessInstanceIncidentsWithResponse(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.ResolveProcessInstanceIncidentsResponse, error) {
	return m.resolveProcessInstanceIncidents(ctx, key, reqEditors...)
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

func TestResolveProcessInstanceIncidentsMapsAcceptedResponse(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, mockIncidentClient{
		resolveProcessInstanceIncidents: func(_ context.Context, key camundav89.ProcessInstanceKey, _ ...camundav89.RequestEditorFn) (*camundav89.ResolveProcessInstanceIncidentsResponse, error) {
			require.Equal(t, "2251799813685250", key)
			return &camundav89.ResolveProcessInstanceIncidentsResponse{HTTPResponse: testHTTPResponse(http.StatusOK), Body: nil}, nil
		},
	})

	got, err := svc.ResolveProcessInstanceIncidents(context.Background(), "2251799813685250")

	require.NoError(t, err)
	require.Equal(t, d.IncidentResolutionResponse{
		Key:        "2251799813685250",
		Ok:         true,
		StatusCode: http.StatusOK,
		Status:     "200 OK",
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
