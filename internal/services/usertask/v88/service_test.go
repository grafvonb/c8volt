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
	tasklistv88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/tasklist"
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

type mockUserTaskTasklistClient struct {
	getTaskByIdWithResponse func(context.Context, string, ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error)
}

func (m *mockUserTaskTasklistClient) GetTaskByIdWithResponse(ctx context.Context, body string, reqEditors ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
	return m.getTaskByIdWithResponse(ctx, body, reqEditors...)
}

var _ v88.GenUserTaskClientTasklist = (*mockUserTaskTasklistClient)(nil)

// TestService_GetUserTask_ResolvesProcessInstanceKey verifies task lookup returns the owning process-instance key from the native search result.
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

// TestService_GetUserTask_FallsBackToTasklistAfterPrimaryMiss protects legacy URL task ids that only Tasklist V1 can resolve.
func TestService_GetUserTask_FallsBackToTasklistAfterPrimaryMiss(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "tenant-a")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav88.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(_ context.Context, taskID string, _ ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			require.Equal(t, "2251799815391233", taskID)
			return &tasklistv88.GetTaskByIdResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://tasklist.local/v1/tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &tasklistv88.TaskResponse{
					Id:                 ptr("2251799815391233"),
					ProcessInstanceKey: ptr("2251799813711967"),
					TenantId:           ptr("tenant-a"),
				},
			}, nil
		},
	}, "tenant-a")

	task, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.NoError(t, err)
	require.Equal(t, "2251799815391233", task.Key)
	require.Equal(t, "2251799813711967", task.ProcessInstanceKey)
	require.Equal(t, "tenant-a", task.TenantId)
}

// TestService_GetUserTask_DoesNotCallFallbackAfterPrimarySuccess keeps deprecated Tasklist calls out of the modern v2 success path.
func TestService_GetUserTask_DoesNotCallFallbackAfterPrimarySuccess(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
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
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(context.Context, string, ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			t.Fatal("Tasklist fallback should not be called after primary success")
			return nil, nil
		},
	})

	task, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.NoError(t, err)
	require.Equal(t, "2251799815391233", task.Key)
	require.Equal(t, "2251799813711967", task.ProcessInstanceKey)
}

// TestService_GetUserTask_IncludesConfiguredTenantFilter verifies task lookup stays scoped to the configured tenant.
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

// TestService_GetUserTask_ReturnsNotFoundForMissingTask keeps a primary miss plus Tasklist 404 in the existing not-found class.
func TestService_GetUserTask_ReturnsNotFoundForMissingTask(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav88.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(_ context.Context, taskID string, _ ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			require.Equal(t, "2251799815391233", taskID)
			return &tasklistv88.GetTaskByIdResponse{
				Body:         []byte(`{"message":"task not found"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://tasklist.local/v1/tasks/2251799815391233", http.StatusNotFound, "404 Not Found"),
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "fallback user task 2251799815391233 was not found or is not visible to the configured tenant")
}

// TestService_GetUserTask_RejectsFallbackMissingProcessInstanceKey prevents rendering a task whose owning process instance is unknown.
func TestService_GetUserTask_RejectsFallbackMissingProcessInstanceKey(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav88.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(_ context.Context, taskID string, _ ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			require.Equal(t, "2251799815391233", taskID)
			return &tasklistv88.GetTaskByIdResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://tasklist.local/v1/tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &tasklistv88.TaskResponse{
					Id: ptr("2251799815391233"),
				},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Contains(t, err.Error(), "fallback user task 2251799815391233 has no process instance key")
}

// TestService_GetUserTask_SurfacesFallbackServerFailure keeps Tasklist outages distinct from genuinely missing tasks.
func TestService_GetUserTask_SurfacesFallbackServerFailure(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav88.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(_ context.Context, taskID string, _ ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			require.Equal(t, "2251799815391233", taskID)
			return &tasklistv88.GetTaskByIdResponse{
				Body:         []byte(`{"message":"tasklist unavailable"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://tasklist.local/v1/tasks/2251799815391233", http.StatusInternalServerError, "500 Internal Server Error"),
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrInternal)
	require.NotErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "get fallback task")
	require.Contains(t, err.Error(), "tasklist unavailable")
}

// TestService_GetUserTask_DoesNotFallbackAfterPrimaryServerFailure keeps non-not-found primary errors terminal.
func TestService_GetUserTask_DoesNotFallbackAfterPrimaryServerFailure(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav88.SearchUserTasksJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav88.SearchUserTasksResponse{
				Body:         []byte(`{"message":"camunda unavailable"}`),
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusInternalServerError, "500 Internal Server Error"),
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(context.Context, string, ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			t.Fatal("Tasklist fallback should not be called after primary server failure")
			return nil, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrInternal)
	require.NotErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "search user task")
	require.Contains(t, err.Error(), "camunda unavailable")
}

// TestService_GetUserTask_RejectsMissingProcessInstanceKey protects the command path from rendering a task lookup with no owning process instance.
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
	return newTestServiceWithTasklist(t, camundaClient, nil, tenantID...)
}

func newTestServiceWithTasklist(t *testing.T, camundaClient *mockUserTaskCamundaClient, tasklistClient *mockUserTaskTasklistClient, tenantID ...string) *v88.Service {
	t.Helper()

	cfg := testConfig()
	if len(tenantID) > 0 {
		cfg.App.Tenant = tenantID[0]
	}
	opts := []v88.Option{v88.WithClientCamunda(camundaClient)}
	if tasklistClient != nil {
		opts = append(opts, v88.WithClientTasklist(tasklistClient))
	}
	svc, err := v88.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		opts...,
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

func ptr[T any](v T) *T {
	return &v
}
