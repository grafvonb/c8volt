// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	tasklistv88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestService_GetNativeUserTask_UsesDirectGetAndPreservesForeignTenant verifies V88 native reads never search or fall back.
func TestService_GetNativeUserTask_UsesDirectGetAndPreservesForeignTenant(t *testing.T) {
	name := "Approve invoice"
	assignee := "alice"
	payload := nativeUserTaskResult(&name, &assignee)
	svc := newTestServiceWithTasklist(t, &mockUserTaskCamundaClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav88.UserTaskKey, _ ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error) {
			require.Equal(t, camundav88.UserTaskKey(payload.UserTaskKey), key)
			return nativeUserTaskHTTPResponse(http.StatusOK, &payload), nil
		},
		searchUserTasksWithResponse: func(context.Context, camundav88.SearchUserTasksJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error) {
			t.Fatal("native lookup must not search")
			return nil, nil
		},
	}, &mockUserTaskTasklistClient{
		getTaskByIdWithResponse: func(context.Context, string, ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error) {
			t.Fatal("native lookup must not use Tasklist fallback")
			return nil, nil
		},
	}, "configured-tenant")

	task, err := svc.GetNativeUserTask(context.Background(), payload.UserTaskKey)

	require.NoError(t, err)
	require.Equal(t, "foreign-tenant", task.TenantId)
	require.Equal(t, "Approve invoice", task.Name)
	require.Equal(t, "alice", task.Assignee)
	require.Equal(t, []string{"bob"}, task.CandidateUsers)
	require.Equal(t, []string{"accounting"}, task.CandidateGroups)
	require.Equal(t, "approve_invoice", task.ElementId)
	require.Equal(t, "2251799815391222", task.ElementInstanceKey)
	require.Equal(t, "2251799813689000", task.ProcessDefinitionKey)
	require.Equal(t, "invoice", task.ProcessDefinitionId)
	require.Equal(t, int32(7), task.ProcessDefinitionVersion)
	payload.CandidateUsers[0] = "changed"
	require.Equal(t, []string{"bob"}, task.CandidateUsers)

	payload = nativeUserTaskResult(nil, nil)
	task, err = svc.GetNativeUserTask(context.Background(), payload.UserTaskKey)
	require.NoError(t, err)
	require.Empty(t, task.Name)
	require.Empty(t, task.Assignee)
}

// TestService_GetNativeUserTask_RejectsBackendAndMalformedResponses covers V88 native failures without a successful partial task.
func TestService_GetNativeUserTask_RejectsBackendAndMalformedResponses(t *testing.T) {
	tests := []struct {
		name      string
		response  *camundav88.GetUserTaskResponse
		transport error
		wantError error
	}{
		{name: "not found", response: nativeUserTaskHTTPResponse(http.StatusNotFound, nil), wantError: d.ErrNotFound},
		{name: "denied", response: nativeUserTaskHTTPResponse(http.StatusForbidden, nil), wantError: d.ErrForbidden},
		{name: "empty payload", response: nativeUserTaskHTTPResponse(http.StatusOK, nil), wantError: d.ErrMalformedResponse},
		{name: "mismatched identity", response: nativeUserTaskHTTPResponse(http.StatusOK, nativeUserTaskResultPtr("wrong", "CREATED", "2251799813711967")), wantError: d.ErrMalformedResponse},
		{name: "missing state", response: nativeUserTaskHTTPResponse(http.StatusOK, nativeUserTaskResultPtr("2251799815391233", "", "2251799813711967")), wantError: d.ErrMalformedResponse},
		{name: "missing process instance", response: nativeUserTaskHTTPResponse(http.StatusOK, nativeUserTaskResultPtr("2251799815391233", "CREATED", "")), wantError: d.ErrMalformedResponse},
		{name: "transport", transport: errors.New("connection reset"), wantError: errors.New("connection reset")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, &mockUserTaskCamundaClient{
				getUserTaskWithResponse: func(context.Context, camundav88.UserTaskKey, ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error) {
					return tt.response, tt.transport
				},
			})

			_, err := svc.GetNativeUserTask(context.Background(), "2251799815391233")

			require.Error(t, err)
			if tt.transport != nil {
				require.ErrorIs(t, err, tt.transport)
				return
			}
			require.ErrorIs(t, err, tt.wantError)
		})
	}
}

// nativeUserTaskResult builds a complete V88 payload while allowing nullable native strings to vary.
func nativeUserTaskResult(name, assignee *string) camundav88.UserTaskResult {
	return camundav88.UserTaskResult{UserTaskKey: "2251799815391233", State: "CREATED", Name: name, ElementId: "approve_invoice", ElementInstanceKey: "2251799815391222", Assignee: assignee, CandidateUsers: []string{"bob"}, CandidateGroups: []string{"accounting"}, ProcessInstanceKey: "2251799813711967", ProcessDefinitionKey: "2251799813689000", ProcessDefinitionId: "invoice", ProcessDefinitionVersion: 7, TenantId: "foreign-tenant"}
}

// nativeUserTaskResultPtr creates the minimum payload needed to exercise V88 identity validation.
func nativeUserTaskResultPtr(key string, state camundav88.UserTaskStateEnum, processInstanceKey string) *camundav88.UserTaskResult {
	return &camundav88.UserTaskResult{UserTaskKey: key, State: state, ProcessInstanceKey: processInstanceKey}
}

// nativeUserTaskHTTPResponse creates a generated V88 response with the requested status and payload.
func nativeUserTaskHTTPResponse(status int, payload *camundav88.UserTaskResult) *camundav88.GetUserTaskResponse {
	return &camundav88.GetUserTaskResponse{Body: []byte(`{"message":"backend response"}`), HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", status, http.StatusText(status)), JSON200: payload}
}
