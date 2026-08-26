// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type mockVariableClient struct {
	searchVariablesWithResponse            func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error)
	createElementInstanceVariablesResponse func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error)
}

func (m *mockVariableClient) SearchVariablesWithResponse(ctx context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
	if m.searchVariablesWithResponse == nil {
		panic("unexpected SearchVariablesWithResponse call")
	}
	return m.searchVariablesWithResponse(ctx, params, body, reqEditors...)
}

func (m *mockVariableClient) CreateElementInstanceVariablesWithResponse(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
	if m.createElementInstanceVariablesResponse == nil {
		panic("unexpected CreateElementInstanceVariablesWithResponse call")
	}
	return m.createElementInstanceVariablesResponse(ctx, elementInstanceKey, body, reqEditors...)
}

// TestSearchProcessInstanceVariables verifies v8.10 variable lookup requests process-scope values and decodes raw value fields.
func TestSearchProcessInstanceVariables(t *testing.T) {
	svc := newVariableTestService(t, &mockVariableClient{
		searchVariablesWithResponse: func(_ context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
			require.NotNil(t, params)
			require.NotNil(t, params.TruncateValues)
			require.False(t, *params.TruncateValues)
			rawBody := variableJSON(t, body)
			require.Contains(t, rawBody, `"processInstanceKey":"123"`)
			require.Contains(t, rawBody, `"scopeKey":"123"`)
			require.Contains(t, rawBody, `"tenantId":"tenant-a"`)
			require.Contains(t, rawBody, `"field":"name"`)
			return &camundav810.SearchVariablesResponse{
				HTTPResponse: variableHTTPResponse(http.MethodPost, "https://camunda.local/v2/variables/search", http.StatusOK, "200 OK"),
				Body:         []byte(`{"items":[{"name":"zeta","value":"2","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-a"},{"name":"alpha","value":"1","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-a","isTruncated":true}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`),
				JSON200:      &camundav810.VariableSearchQueryResult{},
			}, nil
		},
	})

	variables, err := svc.SearchProcessInstanceVariables(context.Background(), "123")

	require.NoError(t, err)
	require.Equal(t, []d.ProcessInstanceVariable{
		{Name: "alpha", Value: "1", VariableKey: "901", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant-a", APITruncated: true},
		{Name: "zeta", Value: "2", VariableKey: "902", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant-a"},
	}, variables)
}

// TestSearchProcessInstanceVariables_Errors verifies transport and malformed response handling.
func TestSearchProcessInstanceVariables_Errors(t *testing.T) {
	transportErr := errors.New("transport failed")
	tests := []struct {
		name    string
		resp    *camundav810.SearchVariablesResponse
		err     error
		wantErr error
	}{
		{name: "transport", err: transportErr, wantErr: transportErr},
		{
			name: "http",
			resp: &camundav810.SearchVariablesResponse{
				HTTPResponse: variableHTTPResponse(http.MethodPost, "https://camunda.local/v2/variables/search", http.StatusInternalServerError, "500 Internal Server Error"),
				Body:         []byte("boom"),
			},
			wantErr: d.ErrInternal,
		},
		{
			name: "empty body",
			resp: &camundav810.SearchVariablesResponse{
				HTTPResponse: variableHTTPResponse(http.MethodPost, "https://camunda.local/v2/variables/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav810.VariableSearchQueryResult{},
			},
			wantErr: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newVariableTestService(t, &mockVariableClient{
				searchVariablesWithResponse: func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
					return tt.resp, tt.err
				},
			})

			_, err := svc.SearchProcessInstanceVariables(context.Background(), "123")

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestUpdateProcessInstanceVariables verifies v8.10 variable updates target the process instance element scope.
func TestUpdateProcessInstanceVariables(t *testing.T) {
	svc := newVariableTestService(t, &mockVariableClient{
		createElementInstanceVariablesResponse: func(_ context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			require.Equal(t, camundav810.ElementInstanceKey("123"), elementInstanceKey)
			require.Equal(t, map[string]any{"foo": "bar"}, body.Variables)
			return &camundav810.CreateElementInstanceVariablesResponse{
				HTTPResponse: variableHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/123/variables", http.StatusNoContent, "204 No Content"),
			}, nil
		},
	})

	result, err := svc.UpdateProcessInstanceVariables(context.Background(), "123", map[string]any{"foo": "bar"}, services.WithNoWait())

	require.NoError(t, err)
	require.Equal(t, d.ProcessInstanceVariableUpdateResponse{
		Key:        "123",
		Ok:         true,
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
	}, result)
}

// TestUpdateProcessInstanceVariables_HTTPError verifies mutation failures return the shared HTTP error and unsuccessful result.
func TestUpdateProcessInstanceVariables_HTTPError(t *testing.T) {
	svc := newVariableTestService(t, &mockVariableClient{
		createElementInstanceVariablesResponse: func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			return &camundav810.CreateElementInstanceVariablesResponse{
				HTTPResponse: variableHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/123/variables", http.StatusBadRequest, "400 Bad Request"),
				Body:         []byte(`{"message":"bad variable"}`),
			}, nil
		},
	})

	result, err := svc.UpdateProcessInstanceVariables(context.Background(), "123", map[string]any{"foo": "bar"}, services.WithNoWait())

	require.ErrorIs(t, err, d.ErrBadRequest)
	require.False(t, result.Ok)
	require.Equal(t, http.StatusBadRequest, result.StatusCode)
}

// newVariableTestService builds a v8.10 variable service with a generated-client mock.
func newVariableTestService(t *testing.T, client GenVariableClientCamunda) *Service {
	t.Helper()
	svc, err := New(&config.Config{
		App: config.App{Tenant: "tenant-a"},
		APIs: config.APIs{Camunda: config.API{
			BaseURL: "https://camunda.local/v2",
		}},
	}, &http.Client{}, slog.Default(), WithClientCamunda(client))
	require.NoError(t, err)
	return svc
}

// variableJSON serializes generated request unions for shape assertions.
func variableJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

// variableHTTPResponse builds the minimal response metadata used by common payload helpers.
func variableHTTPResponse(method, rawURL string, statusCode int, status string) *http.Response {
	u, _ := url.Parse(rawURL)
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Request: &http.Request{
			Method: method,
			URL:    u,
		},
	}
}
