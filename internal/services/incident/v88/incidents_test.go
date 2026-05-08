// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

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
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	v88 "github.com/grafvonb/c8volt/internal/services/incident/v88"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

type mockIncidentClient struct {
	getIncident                     func(context.Context, camundav88.IncidentKey, ...camundav88.RequestEditorFn) (*camundav88.GetIncidentResponse, error)
	resolveIncident                 func(context.Context, camundav88.IncidentKey, camundav88.ResolveIncidentJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.ResolveIncidentResponse, error)
	resolveProcessInstanceIncidents func(context.Context, camundav88.ProcessInstanceKey, ...camundav88.RequestEditorFn) (*camundav88.ResolveProcessInstanceIncidentsResponse, error)
	searchProcessInstanceIncidents  func(context.Context, string, camundav88.SearchProcessInstanceIncidentsJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstanceIncidentsResponse, error)
}

func (m mockIncidentClient) GetIncidentWithResponse(ctx context.Context, key camundav88.IncidentKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetIncidentResponse, error) {
	return m.getIncident(ctx, key, reqEditors...)
}

func (m mockIncidentClient) ResolveIncidentWithResponse(ctx context.Context, key camundav88.IncidentKey, body camundav88.ResolveIncidentJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.ResolveIncidentResponse, error) {
	return m.resolveIncident(ctx, key, body, reqEditors...)
}

func (m mockIncidentClient) ResolveProcessInstanceIncidentsWithResponse(ctx context.Context, key camundav88.ProcessInstanceKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.ResolveProcessInstanceIncidentsResponse, error) {
	return m.resolveProcessInstanceIncidents(ctx, key, reqEditors...)
}

func (m mockIncidentClient) SearchProcessInstanceIncidentsWithResponse(ctx context.Context, key string, body camundav88.SearchProcessInstanceIncidentsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstanceIncidentsResponse, error) {
	return m.searchProcessInstanceIncidents(ctx, key, body, reqEditors...)
}

func TestResolveIncidentMapsAcceptedResponse(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, mockIncidentClient{
		resolveIncident: func(_ context.Context, key camundav88.IncidentKey, body camundav88.ResolveIncidentJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.ResolveIncidentResponse, error) {
			require.Equal(t, "2251799813685249", key)
			require.Nil(t, body.OperationReference)
			return &camundav88.ResolveIncidentResponse{HTTPResponse: testHTTPResponse(http.StatusNoContent), Body: nil}, nil
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
		getIncident: func(_ context.Context, key camundav88.IncidentKey, _ ...camundav88.RequestEditorFn) (*camundav88.GetIncidentResponse, error) {
			require.Equal(t, "2251799813685249", key)
			jobKey := "2251799813685251"
			rootKey := "2251799813685250"
			return &camundav88.GetIncidentResponse{
				HTTPResponse: testHTTPResponse(http.StatusOK),
				JSON200: &camundav88.IncidentResult{
					IncidentKey:            "2251799813685249",
					ProcessInstanceKey:     "2251799813685250",
					TenantId:               "tenant-a",
					State:                  camundav88.IncidentStateEnumACTIVE,
					ErrorType:              camundav88.IncidentErrorTypeEnumJOBNORETRIES,
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
		resolveProcessInstanceIncidents: func(_ context.Context, key camundav88.ProcessInstanceKey, _ ...camundav88.RequestEditorFn) (*camundav88.ResolveProcessInstanceIncidentsResponse, error) {
			require.Equal(t, "2251799813685250", key)
			return &camundav88.ResolveProcessInstanceIncidentsResponse{HTTPResponse: testHTTPResponse(http.StatusOK), Body: nil}, nil
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

func TestWaitForIncidentResolvedPollsUntilNotFound(t *testing.T) {
	t.Parallel()

	attempts := 0
	svc := newTestService(t, mockIncidentClient{
		getIncident: func(_ context.Context, key camundav88.IncidentKey, _ ...camundav88.RequestEditorFn) (*camundav88.GetIncidentResponse, error) {
			attempts++
			if attempts == 1 {
				return &camundav88.GetIncidentResponse{
					HTTPResponse: testHTTPResponse(http.StatusOK),
					JSON200:      &camundav88.IncidentResult{IncidentKey: key, State: camundav88.IncidentStateEnumACTIVE},
				}, nil
			}
			return &camundav88.GetIncidentResponse{HTTPResponse: testHTTPResponse(http.StatusNotFound), Body: []byte(`{"message":"not found"}`)}, nil
		},
	})

	got, err := svc.WaitForIncidentResolved(context.Background(), "2251799813685249")

	require.NoError(t, err)
	require.True(t, got.Ok)
	require.Equal(t, "2251799813685249", got.Key)
	require.Equal(t, 2, attempts)
}

func TestWaitForProcessInstanceIncidentsResolvedPollsInitialSetOnly(t *testing.T) {
	t.Parallel()

	attempts := 0
	svc := newTestService(t, mockIncidentClient{
		searchProcessInstanceIncidents: func(_ context.Context, key string, _ camundav88.SearchProcessInstanceIncidentsJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstanceIncidentsResponse, error) {
			require.Equal(t, "2251799813685250", key)
			attempts++
			items := []camundav88.IncidentResult{
				{IncidentKey: "other", ProcessInstanceKey: key, State: camundav88.IncidentStateEnumACTIVE},
			}
			if attempts == 1 {
				items = append(items, camundav88.IncidentResult{IncidentKey: "2251799813685249", ProcessInstanceKey: key, State: camundav88.IncidentStateEnumACTIVE})
			}
			return &camundav88.SearchProcessInstanceIncidentsResponse{
				HTTPResponse: testHTTPResponse(http.StatusOK),
				JSON200:      &camundav88.IncidentSearchQueryResult{Items: items},
			}, nil
		},
	})

	got, err := svc.WaitForProcessInstanceIncidentsResolved(context.Background(), "2251799813685250", []string{"2251799813685249"})

	require.NoError(t, err)
	require.True(t, got.Ok)
	require.Equal(t, "2251799813685250", got.Key)
	require.Equal(t, 2, attempts)
}

func TestResolveIncidentMapsHTTPError(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, mockIncidentClient{
		resolveIncident: func(context.Context, camundav88.IncidentKey, camundav88.ResolveIncidentJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.ResolveIncidentResponse, error) {
			return &camundav88.ResolveIncidentResponse{HTTPResponse: testHTTPResponse(http.StatusConflict), Body: []byte(`{"message":"already resolving"}`)}, nil
		},
	})

	got, err := svc.ResolveIncident(context.Background(), "2251799813685249")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrConflict)
	require.False(t, got.Ok)
	require.Equal(t, http.StatusConflict, got.StatusCode)
}

func newTestService(t *testing.T, client mockIncidentClient) *v88.Service {
	t.Helper()
	cfg := &config.Config{
		App: config.App{
			CamundaVersion: toolx.V88,
			Backoff: config.BackoffConfig{
				Strategy:     config.BackoffFixed,
				InitialDelay: time.Millisecond,
				MaxRetries:   3,
				Timeout:      50 * time.Millisecond,
			},
		},
		APIs: config.APIs{Camunda: config.API{BaseURL: "http://localhost:8080/v2"}},
	}
	svc, err := v88.New(cfg, &http.Client{}, slog.Default(), v88.WithClientCamunda(client))
	require.NoError(t, err)
	return svc
}

func testHTTPResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(statusCodeOrOK(status))),
		Body:       io.NopCloser(nil),
		Request:    &http.Request{Method: http.MethodPost, URL: &url.URL{Scheme: "http", Host: "localhost", Path: "/v2/incidents/1/resolution"}},
	}
}

func statusCodeOrOK(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

var _ v88.GenIncidentClientCamunda = mockIncidentClient{}
