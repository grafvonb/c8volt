// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

type scopeVariableClient struct {
	update func(context.Context, camundav88.ElementInstanceKey, camundav88.CreateElementInstanceVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error)
}

// SearchVariablesWithResponse panics because scope writes must not perform variable reads.
func (m *scopeVariableClient) SearchVariablesWithResponse(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
	panic("unexpected variable search")
}

// CreateElementInstanceVariablesWithResponse delegates the mutation to the test callback.
func (m *scopeVariableClient) CreateElementInstanceVariablesWithResponse(ctx context.Context, key camundav88.ElementInstanceKey, body camundav88.CreateElementInstanceVariablesJSONRequestBody, opts ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error) {
	return m.update(ctx, key, body, opts...)
}

// TestUpdateScopeVariables verifies v8.8 sends one local-only scope write and reports its acceptance.
func TestUpdateScopeVariables(t *testing.T) {
	svc := newScopeVariableTestService(t, &scopeVariableClient{update: func(_ context.Context, key camundav88.ElementInstanceKey, body camundav88.CreateElementInstanceVariablesJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error) {
		require.Equal(t, camundav88.ElementInstanceKey("456"), key)
		require.Equal(t, map[string]any{"foo": "bar"}, body.Variables)
		require.NotNil(t, body.Local)
		require.True(t, *body.Local)
		return &camundav88.CreateElementInstanceVariablesResponse{HTTPResponse: scopeVariableHTTPResponse(http.StatusNoContent, "204 No Content")}, nil
	}})

	got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})

	require.NoError(t, err)
	require.Equal(t, d.ScopeVariableUpdateResponse{ScopeKey: "456", Accepted: true, StatusCode: http.StatusNoContent, Status: "204 No Content"}, got)
}

// TestUpdateScopeVariablesRejectsInvalidInput verifies invalid v8.8 scope writes fail before transport access.
func TestUpdateScopeVariablesRejectsInvalidInput(t *testing.T) {
	calls := 0
	svc := newScopeVariableTestService(t, &scopeVariableClient{update: func(context.Context, camundav88.ElementInstanceKey, camundav88.CreateElementInstanceVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error) {
		calls++
		return nil, nil
	}})

	_, err := svc.UpdateScopeVariables(context.Background(), "", map[string]any{"foo": "bar"})
	require.ErrorIs(t, err, d.ErrValidation)
	_, err = svc.UpdateScopeVariables(context.Background(), "456", map[string]any{})
	require.ErrorIs(t, err, d.ErrValidation)
	require.Zero(t, calls)
}

// TestUpdateScopeVariablesErrors verifies v8.8 transport, missing-response, and HTTP failures.
func TestUpdateScopeVariablesErrors(t *testing.T) {
	transportErr := errors.New("transport failed")
	tests := []struct {
		name    string
		resp    *camundav88.CreateElementInstanceVariablesResponse
		err     error
		wantErr error
	}{
		{name: "transport", err: transportErr, wantErr: transportErr},
		{name: "nil response", wantErr: d.ErrMalformedResponse},
		{name: "http", resp: &camundav88.CreateElementInstanceVariablesResponse{HTTPResponse: scopeVariableHTTPResponse(http.StatusBadRequest, "400 Bad Request"), Body: []byte(`{"message":"bad variable"}`)}, wantErr: d.ErrBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newScopeVariableTestService(t, &scopeVariableClient{update: func(context.Context, camundav88.ElementInstanceKey, camundav88.CreateElementInstanceVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error) {
				return tt.resp, tt.err
			}})
			got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})
			require.ErrorIs(t, err, tt.wantErr)
			require.False(t, got.Accepted)
		})
	}
}

// newScopeVariableTestService constructs the v8.8 adapter with a strict generated-client double.
func newScopeVariableTestService(t *testing.T, client GenVariableClientCamunda) *Service {
	t.Helper()
	svc, err := New(&config.Config{APIs: config.APIs{Camunda: config.API{BaseURL: "https://camunda.local/v2"}}}, &http.Client{}, slog.Default(), WithClientCamunda(client))
	require.NoError(t, err)
	return svc
}

// scopeVariableHTTPResponse supplies the response metadata required by HTTP normalization.
func scopeVariableHTTPResponse(statusCode int, status string) *http.Response {
	u, _ := url.Parse("https://camunda.local/v2/element-instances/456/variables")
	return &http.Response{StatusCode: statusCode, Status: status, Request: &http.Request{Method: http.MethodPut, URL: u}}
}
