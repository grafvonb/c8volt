// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89_test

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
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	tasklistv89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	v89 "github.com/grafvonb/c8volt/internal/services/usertask/v89"
	"github.com/stretchr/testify/require"
)

type mockUserTaskCamundaClient struct {
	searchUserTasksWithResponse func(context.Context, camundav89.SearchUserTasksJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error)
}

func (m *mockUserTaskCamundaClient) SearchUserTasksWithResponse(ctx context.Context, body camundav89.SearchUserTasksJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
	return m.searchUserTasksWithResponse(ctx, body, reqEditors...)
}

var _ v89.GenUserTaskClientCamunda = (*mockUserTaskCamundaClient)(nil)

type mockUserTaskTasklistClient struct {
	searchTasksWithResponse func(context.Context, tasklistv89.SearchTasksJSONRequestBody, ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error)
}

func (m *mockUserTaskTasklistClient) SearchTasksWithResponse(ctx context.Context, body tasklistv89.SearchTasksJSONRequestBody, reqEditors ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
	return m.searchTasksWithResponse(ctx, body, reqEditors...)
}

var _ v89.GenUserTaskClientTasklist = (*mockUserTaskTasklistClient)(nil)

// TestService_GetUserTask_ResolvesProcessInstanceKey verifies task lookup returns the owning process-instance key from the native search result.
func TestService_GetUserTask_ResolvesProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav89.UserTaskSearchQueryResult{
					Items: []camundav89.UserTaskResult{{
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

// TestService_GetUserTask_FallsBackToTasklistAfterPrimaryMiss verifies a native lookup miss can resolve through Tasklist V1.
func TestService_GetUserTask_FallsBackToTasklistAfterPrimaryMiss(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "tenant-a")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav89.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		searchTasksWithResponse: func(_ context.Context, body tasklistv89.SearchTasksJSONRequestBody, _ ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
			requireFallbackTaskSearchBody(t, body, "2251799815391233", "tenant-a")
			return &tasklistv89.SearchTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://tasklist.local/v1/tasks/search", http.StatusOK, "200 OK"),
				JSON200: &[]tasklistv89.TaskSearchResponse{{
					Id:                 ptr("2251799815391233"),
					ProcessInstanceKey: ptr("2251799813711967"),
					TenantId:           ptr("tenant-a"),
				}},
			}, nil
		},
	}, "tenant-a")

	task, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.NoError(t, err)
	require.Equal(t, "2251799815391233", task.Key)
	require.Equal(t, "2251799813711967", task.ProcessInstanceKey)
	require.Equal(t, "tenant-a", task.TenantId)
}

// TestService_GetUserTask_DoesNotCallFallbackAfterPrimarySuccess verifies native lookup remains the first and terminal success path.
func TestService_GetUserTask_DoesNotCallFallbackAfterPrimarySuccess(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav89.UserTaskSearchQueryResult{
					Items: []camundav89.UserTaskResult{{
						UserTaskKey:        "2251799815391233",
						ProcessInstanceKey: "2251799813711967",
						TenantId:           "tenant-a",
					}},
				},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		searchTasksWithResponse: func(context.Context, tasklistv89.SearchTasksJSONRequestBody, ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
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
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "tenant-a")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav89.UserTaskSearchQueryResult{
					Items: []camundav89.UserTaskResult{{
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

// TestService_GetUserTask_ReturnsNotFoundForMissingTask keeps empty tenant-scoped search results mapped to a lookup-style not-found error.
func TestService_GetUserTask_ReturnsNotFoundForMissingTask(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav89.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		searchTasksWithResponse: func(_ context.Context, body tasklistv89.SearchTasksJSONRequestBody, _ ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
			requireFallbackTaskSearchBody(t, body, "2251799815391233", "")
			return &tasklistv89.SearchTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://tasklist.local/v1/tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &[]tasklistv89.TaskSearchResponse{},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "fallback user task 2251799815391233 was not found or is not visible to the configured tenant")
}

// TestService_GetUserTask_RejectsFallbackMissingProcessInstanceKey protects fallback lookup from returning unresolved ownership.
func TestService_GetUserTask_RejectsFallbackMissingProcessInstanceKey(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav89.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		searchTasksWithResponse: func(_ context.Context, body tasklistv89.SearchTasksJSONRequestBody, _ ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
			requireFallbackTaskSearchBody(t, body, "2251799815391233", "")
			return &tasklistv89.SearchTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://tasklist.local/v1/tasks/search", http.StatusOK, "200 OK"),
				JSON200: &[]tasklistv89.TaskSearchResponse{{
					Id: ptr("2251799815391233"),
				}},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Contains(t, err.Error(), "fallback user task 2251799815391233 has no process instance key")
}

// TestService_GetUserTask_SurfacesFallbackServerFailure verifies fallback operational failures are not collapsed into not found.
func TestService_GetUserTask_SurfacesFallbackServerFailure(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav89.UserTaskSearchQueryResult{},
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		searchTasksWithResponse: func(_ context.Context, body tasklistv89.SearchTasksJSONRequestBody, _ ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
			requireFallbackTaskSearchBody(t, body, "2251799815391233", "")
			return &tasklistv89.SearchTasksResponse{
				Body:         []byte(`{"message":"tasklist unavailable"}`),
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://tasklist.local/v1/tasks/search", http.StatusInternalServerError, "500 Internal Server Error"),
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrInternal)
	require.NotErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "search fallback task")
	require.Contains(t, err.Error(), "tasklist unavailable")
}

// TestService_GetUserTask_DoesNotFallbackAfterPrimaryServerFailure keeps non-not-found primary errors terminal.
func TestService_GetUserTask_DoesNotFallbackAfterPrimaryServerFailure(t *testing.T) {
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				Body:         []byte(`{"message":"camunda unavailable"}`),
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusInternalServerError, "500 Internal Server Error"),
			}, nil
		},
	}, &mockUserTaskTasklistClient{
		searchTasksWithResponse: func(context.Context, tasklistv89.SearchTasksJSONRequestBody, ...tasklistv89.RequestEditorFn) (*tasklistv89.SearchTasksResponse, error) {
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
		searchUserTasksWithResponse: func(_ context.Context, body camundav89.SearchUserTasksJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error) {
			requireUserTaskSearchBody(t, body, "2251799815391233", "")
			return &camundav89.SearchUserTasksResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/search", http.StatusOK, "200 OK"),
				JSON200: &camundav89.UserTaskSearchQueryResult{
					Items: []camundav89.UserTaskResult{{
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

func newTestService(t *testing.T, camundaClient *mockUserTaskCamundaClient, tenantID ...string) *v89.Service {
	t.Helper()
	return newTestServiceWithTasklist(t, camundaClient, nil, tenantID...)
}

func newTestServiceWithTasklist(t *testing.T, camundaClient *mockUserTaskCamundaClient, tasklistClient *mockUserTaskTasklistClient, tenantID ...string) *v89.Service {
	t.Helper()

	cfg := testConfig()
	if len(tenantID) > 0 {
		cfg.App.Tenant = tenantID[0]
	}
	opts := []v89.Option{v89.WithClientCamunda(camundaClient)}
	if tasklistClient != nil {
		opts = append(opts, v89.WithClientTasklist(tasklistClient))
	}
	svc, err := v89.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		opts...,
	)
	require.NoError(t, err)
	return svc
}

func requireUserTaskSearchBody(t *testing.T, body camundav89.SearchUserTasksJSONRequestBody, taskKey, tenantID string) {
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

func requireFallbackTaskSearchBody(t *testing.T, body tasklistv89.SearchTasksJSONRequestBody, taskKey, tenantID string) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.Contains(t, string(raw), fmt.Sprintf(`"processInstanceKey":"%s"`, taskKey))
	require.Contains(t, string(raw), `"implementation":"JOB_WORKER"`)
	require.Contains(t, string(raw), `"pageSize":2`)
	if tenantID == "" {
		require.NotContains(t, string(raw), `"tenantIds"`)
		return
	}
	require.Contains(t, string(raw), fmt.Sprintf(`"tenantIds":["%s"]`, tenantID))
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
