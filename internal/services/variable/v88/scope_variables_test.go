// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grafvonb/c8volt/internal/services"
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
	svc, err := New(&config.Config{App: config.App{Tenant: "tenant-a"}, APIs: config.APIs{Camunda: config.API{BaseURL: "https://camunda.local/v2"}}}, &http.Client{}, slog.Default(), WithClientCamunda(client))
	require.NoError(t, err)
	return svc
}

// scopeVariableHTTPResponse supplies the response metadata required by HTTP normalization.
func scopeVariableHTTPResponse(statusCode int, status string) *http.Response {
	u, _ := url.Parse("https://camunda.local/v2/element-instances/456/variables")
	return &http.Response{StatusCode: statusCode, Status: status, Request: &http.Request{Method: http.MethodPut, URL: u}}
}

// TestScopeVariablesReadsCompleteLocalCollection proves independent identities,
// sparse cursor continuation, raw counts, optional evidence, and stable ordering.
func TestScopeVariablesReadsCompleteLocalCollection(t *testing.T) {
	t.Parallel()
	for _, ignore := range []bool{false, true} {
		t.Run(fmt.Sprint("ignore tenant=", ignore), func(t *testing.T) {
			calls := 0
			tenant := "tenant-b"
			query := d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456", ExpectedTenantId: &tenant}
			svc := scopeReadService(t, func(_ context.Context, params *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
				calls++
				require.LessOrEqual(t, calls, 4)
				require.NotNil(t, params.TruncateValues)
				require.False(t, *params.TruncateValues)
				raw, err := json.Marshal(body)
				require.NoError(t, err)
				require.Contains(t, string(raw), `"processInstanceKey":"123"`)
				require.Contains(t, string(raw), `"scopeKey":"456"`)
				require.Contains(t, string(raw), `"field":"name"`)
				if ignore {
					require.Nil(t, body.Filter.TenantId)
				} else {
					require.Equal(t, camundav88.TenantId("tenant-a"), *body.Filter.TenantId)
				}
				if calls > 1 {
					require.Contains(t, string(raw), fmt.Sprintf(`"after":"c%d"`, calls-1))
				}
				bodies := []string{
					`{"items":[{"name":"same","value":"2","variableKey":"b","processInstanceKey":"123","scopeKey":"456","tenantId":"tenant-b","truncated":true},{"name":"parent","value":"0","processInstanceKey":"123","scopeKey":"123"}],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"c1"}}`,
					`{"items":[],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"c2"}}`,
					`{"items":[{"name":"sibling","value":"0","processInstanceKey":"123","scopeKey":"789"},{"name":"called","value":"0","processInstanceKey":"999","scopeKey":"999"}],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"c3"}}`,
					`{"items":[{"name":"same","value":"1","variableKey":"a","processInstanceKey":"123","scopeKey":"456","isTruncated":true}],"page":{"totalItems":5,"hasMoreTotalItems":false}}`,
				}
				return scopeReadResponse(bodies[calls-1]), nil
			})
			var opts []services.CallOption
			if ignore {
				opts = append(opts, services.WithIgnoreTenant())
			}
			got, err := svc.SearchScopeVariables(context.Background(), query, opts...)
			require.NoError(t, err)
			require.Equal(t, 4, calls)
			require.Equal(t, []d.ProcessInstanceVariable{
				{Name: "same", Value: "1", VariableKey: "a", ProcessInstanceKey: "123", ScopeKey: "456", APITruncated: true},
				{Name: "same", Value: "2", VariableKey: "b", ProcessInstanceKey: "123", ScopeKey: "456", TenantId: "tenant-b", APITruncated: true},
			}, got)
		})
	}
}

// TestScopeVariablesIdentity rejects incomplete or contradictory scope evidence
// while treating absent optional tenant/key fields as unknown.
func TestScopeVariablesIdentity(t *testing.T) {
	t.Parallel()
	tenant := "tenant-b"
	tests := []struct {
		name, item string
		wantErr    bool
	}{
		{"optional evidence", `{"name":"v","value":"null","processInstanceKey":"123","scopeKey":"456"}`, false},
		{"empty tenant", `{"name":"v","value":"null","processInstanceKey":"123","scopeKey":"456","tenantId":""}`, false},
		{"wrong owner", `{"name":"v","value":"1","processInstanceKey":"999","scopeKey":"456"}`, true},
		{"wrong tenant", `{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"456","tenantId":"tenant-c"}`, true},
		{"missing owner", `{"name":"v","value":"1","scopeKey":"456"}`, true},
		{"blank owner", `{"name":"v","value":"1","processInstanceKey":" ","scopeKey":"456"}`, true},
		{"missing scope", `{"name":"v","value":"1","processInstanceKey":"123"}`, true},
		{"blank scope", `{"name":"v","value":"1","processInstanceKey":"123","scopeKey":" "}`, true},
		{"missing name", `{"value":"1","processInstanceKey":"123","scopeKey":"456"}`, true},
		{"missing value", `{"name":"v","processInstanceKey":"123","scopeKey":"456"}`, true},
		{"wrong identity type", `{"name":"v","value":"1","processInstanceKey":123,"scopeKey":"456"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := scopeReadService(t, func(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
				return scopeReadResponse(`{"items":[` + tt.item + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`), nil
			})
			got, err := svc.SearchScopeVariables(context.Background(), d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456", ExpectedTenantId: &tenant}, services.WithIgnoreTenant())
			if tt.wantErr {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Len(t, got, 1)
			}
		})
	}
	// Without expected evidence, a returned tenant must never be compared with configured discovery tenant.
	svc := scopeReadService(t, func(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
		return scopeReadResponse(`{"items":[{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"456","tenantId":"tenant-c"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`), nil
	})
	got, err := svc.SearchScopeVariables(context.Background(), d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456"}, services.WithIgnoreTenant())
	require.NoError(t, err)
	require.Len(t, got, 1)
}

// TestScopeVariablesValidateBeforeRequests proves neither incomplete identities
// nor invalid page positions can reach the generated client.
func TestScopeVariablesValidateBeforeRequests(t *testing.T) {
	t.Parallel()
	calls := 0
	svc := scopeReadService(t, func(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
		calls++
		return nil, errors.New("unexpected request")
	})
	for _, query := range []d.ScopeVariableQuery{{}, {ProcessInstanceKey: "123"}, {ScopeKey: "456"}, {ProcessInstanceKey: " ", ScopeKey: "456"}, {ProcessInstanceKey: "123", ScopeKey: " "}} {
		_, err := svc.SearchScopeVariables(context.Background(), query)
		require.ErrorIs(t, err, d.ErrValidation)
		_, err = svc.SearchScopeVariablesPage(context.Background(), query, d.ProcessInstanceVariablePageRequest{Size: 10})
		require.ErrorIs(t, err, d.ErrValidation)
	}
	for _, page := range []d.ProcessInstanceVariablePageRequest{{}, {Size: -1}, {Size: 1001}, {Size: 1, From: -1}, {Size: 1, From: 1, After: "c"}} {
		_, err := svc.SearchScopeVariablesPage(context.Background(), d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456"}, page)
		require.Error(t, err)
	}
	require.Zero(t, calls)
}

// TestScopeVariablesCompletionAndFailures ensures incomplete traversal cannot
// masquerade as successful emptiness, including cancellation after a response.
func TestScopeVariablesCompletionAndFailures(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"empty", "offset", "repeated cursor", "changed total", "later error", "canceled before", "canceled after", "malformed metadata", "nil response"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "canceled before" {
				cancel()
			}
			calls := 0
			laterErr := errors.New("later page failed")
			svc := scopeReadService(t, func(_ context.Context, _ *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
				calls++
				require.LessOrEqual(t, calls, 3)
				switch mode {
				case "empty":
					return scopeReadResponse(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`), nil
				case "offset":
					if calls == 1 {
						return scopeReadResponse(`{"items":[{"name":"other","value":"1","processInstanceKey":"123","scopeKey":"789"},{"name":"local","value":"2","processInstanceKey":"123","scopeKey":"456"}],"page":{}}`), nil
					}
					offset, err := body.Page.AsOffsetPagination()
					require.NoError(t, err)
					require.Equal(t, int32(2), *offset.From)
					return scopeReadResponse(`{"items":[],"page":{}}`), nil
				case "repeated cursor":
					return scopeReadResponse(`{"items":[],"page":{"endCursor":"same"}}`), nil
				case "changed total":
					if calls > 1 {
						return scopeReadResponse(`{"items":[],"page":{"totalItems":3,"hasMoreTotalItems":false}}`), nil
					}
				case "later error":
					if calls > 1 {
						return nil, laterErr
					}
				case "canceled after":
					cancel()
				case "malformed metadata":
					return scopeReadResponse(`{"items":[],"page":{"totalItems":0}}`), nil
				case "nil response":
					return nil, nil
				}
				return scopeReadResponse(`{"items":[{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"456"}],"page":{"totalItems":2,"hasMoreTotalItems":false,"endCursor":"next"}}`), nil
			})
			got, err := svc.SearchScopeVariables(ctx, d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456"})
			switch mode {
			case "empty":
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Empty(t, got)
				require.Equal(t, 1, calls)
			case "offset":
				require.NoError(t, err)
				require.Len(t, got, 1)
				require.Equal(t, 2, calls)
			case "canceled before", "canceled after":
				require.ErrorIs(t, err, context.Canceled)
				require.Nil(t, got)
			case "later error":
				require.ErrorIs(t, err, laterErr)
				require.Nil(t, got)
				require.Equal(t, 2, calls)
			default:
				require.ErrorIs(t, err, d.ErrMalformedResponse)
				require.Nil(t, got)
			}
			if mode == "canceled before" {
				require.Zero(t, calls)
			}
		})
	}
}

// scopeReadResponse preserves raw fields absent from generated variable models.
func scopeReadResponse(body string) *camundav88.SearchVariablesResponse {
	return searchVariablesSuccess([]byte(body))
}

// scopeReadService installs only a generated-client fake; no transport is used.
func scopeReadService(t *testing.T, read func(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error)) *Service {
	t.Helper()
	return newScopeVariableTestService(t, &searchVariableClient{search: read})
}

// TestProcessInstanceVariableRootCompatibility keeps legacy filtering, required
// raw fields and page errors even when the same adapter supports strict scopes.
func TestProcessInstanceVariableRootCompatibility(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, items string
		wantErr     bool
	}{
		{"wrong owner filtered", `[{"name":"v","value":"1","variableKey":"a","processInstanceKey":"other","scopeKey":"123","tenantId":"tenant-a"}]`, false},
		{"different tenant retained", `[{"name":"v","value":"1","variableKey":"a","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-b"}]`, false},
		{"missing tenant still malformed", `[{"name":"v","value":"1","variableKey":"a","processInstanceKey":"123","scopeKey":"123"}]`, true},
		{"missing variable key still malformed", `[{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-a"}]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := scopeReadService(t, func(_ context.Context, _ *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
				calls++
				raw, err := json.Marshal(body)
				require.NoError(t, err)
				require.Contains(t, string(raw), `"processInstanceKey":"123"`)
				require.Contains(t, string(raw), `"scopeKey":"123"`)
				return scopeReadResponse(`{"items":` + tc.items + `,"page":{"totalItems":1,"hasMoreTotalItems":false}}`), nil
			})
			got, err := svc.SearchProcessInstanceVariables(context.Background(), "123", services.WithIgnoreTenant())
			if tc.wantErr {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
				require.EqualError(t, err, "decode process-instance variables: malformed response: variable item 0 is missing required fields")
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				if tc.name == "wrong owner filtered" {
					require.Empty(t, got)
				} else {
					require.Len(t, got, 1)
					require.Equal(t, "tenant-b", got[0].TenantId)
				}
			}
			require.Equal(t, 1, calls)
			_, err = svc.SearchProcessInstanceVariablesPage(context.Background(), "123", d.ProcessInstanceVariablePageRequest{})
			require.EqualError(t, err, "validate process-instance variable page: process-instance variable page request size must be positive")
			require.Equal(t, 1, calls)
		})
	}
}
