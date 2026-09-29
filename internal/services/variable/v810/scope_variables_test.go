// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"errors"
	"net/http"
	"testing"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestUpdateScopeVariables verifies v8.10 sends one local-only scope write and reports its acceptance.
func TestUpdateScopeVariables(t *testing.T) {
	svc := newVariableTestService(t, &mockVariableClient{
		createElementInstanceVariablesResponse: func(_ context.Context, key camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			require.Equal(t, camundav810.ElementInstanceKey("456"), key)
			require.Equal(t, map[string]any{"foo": "bar"}, body.Variables)
			require.NotNil(t, body.Local)
			require.True(t, *body.Local)
			return &camundav810.CreateElementInstanceVariablesResponse{HTTPResponse: variableHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/456/variables", http.StatusNoContent, "204 No Content")}, nil
		},
	})

	got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})

	require.NoError(t, err)
	require.Equal(t, d.ScopeVariableUpdateResponse{ScopeKey: "456", Accepted: true, StatusCode: http.StatusNoContent, Status: "204 No Content"}, got)
}

// TestUpdateScopeVariablesRejectsInvalidInput verifies invalid scope writes fail before transport access.
func TestUpdateScopeVariablesRejectsInvalidInput(t *testing.T) {
	calls := 0
	svc := newVariableTestService(t, &mockVariableClient{
		createElementInstanceVariablesResponse: func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			calls++
			return nil, nil
		},
	})

	_, err := svc.UpdateScopeVariables(context.Background(), " ", map[string]any{"foo": "bar"})
	require.ErrorIs(t, err, d.ErrValidation)
	_, err = svc.UpdateScopeVariables(context.Background(), "456", nil)
	require.ErrorIs(t, err, d.ErrValidation)
	require.Zero(t, calls)
}

// TestUpdateScopeVariablesErrors verifies transport, missing-response, and HTTP failures remain unsuccessful.
func TestUpdateScopeVariablesErrors(t *testing.T) {
	transportErr := errors.New("transport failed")
	tests := []struct {
		name    string
		resp    *camundav810.CreateElementInstanceVariablesResponse
		err     error
		wantErr error
	}{
		{name: "transport", err: transportErr, wantErr: transportErr},
		{name: "nil response", wantErr: d.ErrMalformedResponse},
		{name: "http", resp: &camundav810.CreateElementInstanceVariablesResponse{HTTPResponse: variableHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/456/variables", http.StatusBadRequest, "400 Bad Request"), Body: []byte(`{"message":"bad variable"}`)}, wantErr: d.ErrBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newVariableTestService(t, &mockVariableClient{
				createElementInstanceVariablesResponse: func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
					return tt.resp, tt.err
				},
			})

			got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})

			require.ErrorIs(t, err, tt.wantErr)
			require.False(t, got.Accepted)
		})
	}
}
