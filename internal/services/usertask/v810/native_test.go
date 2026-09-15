// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestService_GetNativeUserTask_MapsCompleteAndNullablePayloads verifies direct reads preserve native fields and normalize absent optional strings.
func TestService_GetNativeUserTask_MapsCompleteAndNullablePayloads(t *testing.T) {
	name := "Approve invoice"
	assignee := "alice"
	payload := nativeUserTaskResult(&name, &assignee)
	svc := newTestService(t, &mockUserTaskClient{
		getUserTaskWithResponse: func(_ context.Context, key camundav810.UserTaskKey, _ ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
			require.Equal(t, camundav810.UserTaskKey(payload.UserTaskKey), key)
			return &camundav810.GetUserTaskResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/"+string(key), http.StatusOK, "200 OK"),
				JSON200:      &payload,
			}, nil
		},
	}, "configured-tenant")

	task, err := svc.GetNativeUserTask(context.Background(), payload.UserTaskKey)

	require.NoError(t, err)
	require.Equal(t, d.UserTask{
		Key:                      "2251799815391233",
		State:                    "CREATED",
		Name:                     "Approve invoice",
		ElementId:                "approve_invoice",
		ElementInstanceKey:       "2251799815391222",
		Assignee:                 "alice",
		CandidateUsers:           []string{"bob"},
		CandidateGroups:          []string{"accounting"},
		ProcessInstanceKey:       "2251799813711967",
		ProcessDefinitionKey:     "2251799813689000",
		ProcessDefinitionId:      "invoice",
		ProcessDefinitionVersion: 7,
		TenantId:                 "foreign-tenant",
	}, task)
	payload.CandidateUsers[0] = "changed"
	require.Equal(t, []string{"bob"}, task.CandidateUsers)

	payload = nativeUserTaskResult(nil, nil)
	task, err = svc.GetNativeUserTask(context.Background(), payload.UserTaskKey)
	require.NoError(t, err)
	require.Empty(t, task.Name)
	require.Empty(t, task.Assignee)
}

// TestService_GetNativeUserTask_RejectsBackendAndMalformedResponses pins direct-read error classification without tenant post-filtering.
func TestService_GetNativeUserTask_RejectsBackendAndMalformedResponses(t *testing.T) {
	tests := []struct {
		name      string
		response  *camundav810.GetUserTaskResponse
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
			svc := newTestService(t, &mockUserTaskClient{
				getUserTaskWithResponse: func(context.Context, camundav810.UserTaskKey, ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error) {
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

// nativeUserTaskResult builds a complete V810 payload while allowing nullable native strings to vary.
func nativeUserTaskResult(name, assignee *string) camundav810.UserTaskResult {
	return camundav810.UserTaskResult{
		UserTaskKey:              "2251799815391233",
		State:                    "CREATED",
		Name:                     name,
		ElementId:                "approve_invoice",
		ElementInstanceKey:       "2251799815391222",
		Assignee:                 assignee,
		CandidateUsers:           []string{"bob"},
		CandidateGroups:          []string{"accounting"},
		ProcessInstanceKey:       "2251799813711967",
		ProcessDefinitionKey:     "2251799813689000",
		ProcessDefinitionId:      "invoice",
		ProcessDefinitionVersion: 7,
		TenantId:                 "foreign-tenant",
	}
}

// nativeUserTaskResultPtr creates the minimum payload needed to exercise native identity validation.
func nativeUserTaskResultPtr(key string, state camundav810.UserTaskStateEnum, processInstanceKey string) *camundav810.UserTaskResult {
	return &camundav810.UserTaskResult{UserTaskKey: key, State: state, ProcessInstanceKey: processInstanceKey}
}

// nativeUserTaskHTTPResponse creates a generated response with the requested status and payload.
func nativeUserTaskHTTPResponse(status int, payload *camundav810.UserTaskResult) *camundav810.GetUserTaskResponse {
	return &camundav810.GetUserTaskResponse{
		Body:         []byte(`{"message":"backend response"}`),
		HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/user-tasks/2251799815391233", status, http.StatusText(status)),
		JSON200:      payload,
	}
}
