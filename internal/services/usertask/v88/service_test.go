// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	v88 "github.com/grafvonb/c8volt/internal/services/usertask/v88"
	"github.com/stretchr/testify/require"
)

type mockUserTaskCamundaClient struct {
	searchUserTasksWithResponse func(context.Context, camundav88.SearchUserTasksJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error)
}

func (m *mockUserTaskCamundaClient) SearchUserTasksWithResponse(ctx context.Context, body camundav88.SearchUserTasksJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
	return m.searchUserTasksWithResponse(ctx, body, reqEditors...)
}

var _ v88.GenUserTaskClientCamunda = (*mockUserTaskCamundaClient)(nil)

func TestService_GetUserTask_ResolvesProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav88.UserTaskSearchQueryResult{
					Items: []camundav88.UserTaskResult{{
						UserTaskKey:        "2251799815391233",
						ProcessInstanceKey: "2251799813711967",
						TenantId:           "tenant-a",
					}},
				},
			}, nil
		},
	})

	task, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.NoError(t, err)
	require.Equal(t, "2251799815391233", task.Key)
	require.Equal(t, "2251799813711967", task.ProcessInstanceKey)
	require.Equal(t, "tenant-a", task.TenantId)
}

func TestService_GetUserTask_IncludesConfiguredTenantFilter(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "tenant-a")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav88.UserTaskSearchQueryResult{
					Items: []camundav88.UserTaskResult{{
						UserTaskKey:        "2251799815391233",
						ProcessInstanceKey: "2251799813711967",
						TenantId:           "tenant-a",
					}},
				},
			}, nil
		},
	}, "tenant-a")

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.NoError(t, err)
}

func TestService_GetUserTask_ReturnsNotFoundForMissingTask(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav88.UserTaskSearchQueryResult{},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "user task 2251799815391233 was not found or is not visible to the configured tenant")
}

func TestService_GetUserTask_RejectsMissingProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav88.UserTaskSearchQueryResult{
					Items: []camundav88.UserTaskResult{{
						UserTaskKey:        "2251799815391233",
						ProcessInstanceKey: "",
					}},
				},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Contains(t, err.Error(), "user task 2251799815391233 has no process instance key")
}

func newTestService(t *testing.T, camundaClient *mockUserTaskCamundaClient, tenantID ...string) *v88.Service {
	t.Helper()

	cfg := testConfig()
	if len(tenantID) > 0 {
		cfg.App.Tenant = tenantID[0]
	}
	svc, err := v88.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v88.WithClientCamunda(camundaClient),
	)
	require.NoError(t, err)
	return svc
}

func requireUserTaskSearchBody(t *testing.T, body camundav88.SearchUserTasksJSONRequestBody, taskKey, tenantID string) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.Contains(t, string(raw), fmt.Sprintf(`"userTaskKey":"%s"`, taskKey))
	if tenantID == "" {
		require.NotContains(t, string(raw), `"tenantId"`)
		return
	}
	require.Contains(t, string(raw), fmt.Sprintf(`"tenantId":"%s"`, tenantID))
}

func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "https://camunda.local/v2",
			},
		},
	}
}

func newHTTPResponse(method, rawURL string, statusCode int, status string) *http.Response {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Request: &http.Request{
			Method: method,
			URL:    u,
		},
	}
}
