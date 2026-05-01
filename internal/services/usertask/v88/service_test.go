// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

import (
	"context"
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
	getUserTaskWithResponse func(context.Context, camundav88.UserTaskKey, ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error)
}

func (m *mockUserTaskCamundaClient) GetUserTaskWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error) {
	return m.getUserTaskWithResponse(ctx, userTaskKey, reqEditors...)
}

var _ v88.GenUserTaskClientCamunda = (*mockUserTaskCamundaClient)(nil)

func TestService_GetUserTask_ResolvesProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		getUserTaskWithResponse: func(_ context.Context, userTaskKey camundav88.UserTaskKey, _ ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error) {
			require.Equal(t, camundav88.UserTaskKey("2251799815391233"), userTaskKey)
			return &camundav88.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &camundav88.UserTaskResult{
					UserTaskKey:        "2251799815391233",
					ProcessInstanceKey: "2251799813711967",
				},
			}, nil
		},
	})

	task, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.NoError(t, err)
	require.Equal(t, "2251799815391233", task.Key)
	require.Equal(t, "2251799813711967", task.ProcessInstanceKey)
}

func TestService_GetUserTask_ReturnsNotFoundForMissingTask(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		getUserTaskWithResponse: func(_ context.Context, userTaskKey camundav88.UserTaskKey, _ ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error) {
			require.Equal(t, camundav88.UserTaskKey("2251799815391233"), userTaskKey)
			return &camundav88.GetUserTaskResponse{
				Body:         []byte(`{"message":"not found"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusNotFound, "404 Not Found"),
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Contains(t, err.Error(), "get user task")
}

func TestService_GetUserTask_RejectsMissingProcessInstanceKey(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		getUserTaskWithResponse: func(_ context.Context, userTaskKey camundav88.UserTaskKey, _ ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error) {
			require.Equal(t, camundav88.UserTaskKey("2251799815391233"), userTaskKey)
			return &camundav88.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", http.StatusOK, "200 OK"),
				JSON200: &camundav88.UserTaskResult{
					UserTaskKey:        "2251799815391233",
					ProcessInstanceKey: "",
				},
			}, nil
		},
	})

	_, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Contains(t, err.Error(), "user task 2251799815391233 has no process instance key")
}

func newTestService(t *testing.T, camundaClient *mockUserTaskCamundaClient) *v88.Service {
	t.Helper()

	svc, err := v88.New(
		testConfig(),
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v88.WithClientCamunda(camundaClient),
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
