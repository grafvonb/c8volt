// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	v810 "github.com/grafvonb/c8volt/internal/services/usertask/v810"
	"github.com/stretchr/testify/require"
)

type mockUserTaskClient struct {
	getUserTaskWithResponse func(context.Context, camundav810.UserTaskKey, ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error)
}

func (m *mockUserTaskClient) GetUserTaskWithResponse(ctx context.Context, key camundav810.UserTaskKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
	return m.getUserTaskWithResponse(ctx, key, reqEditors...)
}

var _ v810.GenUserTaskClientCamunda = (*mockUserTaskClient)(nil)

// TestService_GetUserTask_ResolvesProcessInstanceKey verifies V810 resolves user tasks through the unified Camunda client only.
func TestService_GetUserTask_ResolvesProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey("2251799815391233"), key)
			return &camundav810.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &camundav810.UserTaskResult{
					UserTaskKey:        "2251799815391233",
					ProcessInstanceKey: "2251799813711967",
					TenantId:           "tenant-a",
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

// TestService_GetUserTask_ReturnsNotFoundForMissingTask keeps V810 misses explicit without a Tasklist fallback.
func TestService_GetUserTask_ReturnsNotFoundForMissingTask(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey("2251799815391233"), key)
			return &camundav810.GetUserTaskResponse{
				Body:         []byte(`{"message":"task not found"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusNotFound, "404 Not Found"),
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "user task 2251799815391233")
	require.NotContains(t, err.Error(), "fallback")
}

// TestService_GetUserTask_ReturnsUnavailableForServerUnavailable preserves the shared unavailable classification.
func TestService_GetUserTask_ReturnsUnavailableForServerUnavailable(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey("2251799815391233"), key)
			return &camundav810.GetUserTaskResponse{
				Body:         []byte(`{"message":"gateway unavailable"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusServiceUnavailable, "503 Service Unavailable"),
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrUnavailable)
	require.Contains(t, err.Error(), "get user task")
	require.Contains(t, err.Error(), "gateway unavailable")
}

// TestService_GetUserTask_ReturnsNotFoundForTenantMismatch preserves tenant visibility without Tasklist fallback.
func TestService_GetUserTask_ReturnsNotFoundForTenantMismatch(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey("2251799815391233"), key)
			return &camundav810.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &camundav810.UserTaskResult{
					UserTaskKey:        "2251799815391233",
					ProcessInstanceKey: "2251799813711967",
					TenantId:           "tenant-b",
				},
			}, nil
		},
	}, "tenant-a")

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "not found or is not visible to the configured tenant")
	require.NotContains(t, err.Error(), "fallback")
}

// TestService_GetUserTask_RejectsIdentityMismatch pins V810's native response-key validation before resolver use.
func TestService_GetUserTask_RejectsIdentityMismatch(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey("2251799815391233"), key)
			return &camundav810.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &camundav810.UserTaskResult{
					UserTaskKey:        "2251799815399999",
					ProcessInstanceKey: "2251799813711967",
					TenantId:           "tenant-a",
				},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Contains(t, err.Error(), "user task 2251799815391233 returned mismatched task 2251799815399999")
}

// TestService_GetUserTask_RejectsMissingProcessInstanceKey protects command callers from rendering incomplete lookup results.
func TestService_GetUserTask_RejectsMissingProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey("2251799815391233"), key)
			return &camundav810.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &camundav810.UserTaskResult{
					UserTaskKey: "2251799815391233",
				},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Contains(t, err.Error(), "user task 2251799815391233 has no process instance key")
}

func newTestService(t *testing.T, client *mockUserTaskClient, tenantID ...string) *v810.Service {
	t.Helper()

	cfg := testConfig()
	if len(tenantID) > 0 {
		cfg.App.Tenant = tenantID[0]
	}
	svc, err := v810.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v810.WithClientCamunda(client),
	)
	require.NoError(t, err)
	return svc
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
