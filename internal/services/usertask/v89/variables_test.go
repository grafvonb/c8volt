// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	v89 "github.com/grafvonb/c8volt/internal/services/usertask/v89"
	"github.com/stretchr/testify/require"
)

// TestService_SearchUserTaskEffectiveVariablesPage_RequestsNativeOffsetPage verifies the v8.9 route and generated request contract.
func TestService_SearchUserTaskEffectiveVariablesPage_RequestsNativeOffsetPage(t *testing.T) {
	var requests int
	var method, path, truncateValues string
	var requestBody camundav89.SearchUserTaskEffectiveVariablesJSONRequestBody
	var decodeErr, writeErr error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		method = r.Method
		path = r.URL.Path
		truncateValues = r.URL.Query().Get("truncateValues")
		decodeErr = json.NewDecoder(r.Body).Decode(&requestBody)
		w.Header().Set("Content-Type", "application/json")
		_, writeErr = io.WriteString(w, `{"items":[{"name":"alpha","value":"","variableKey":"0","processInstanceKey":"process-a","scopeKey":"task-scope","tenantId":"tenant-b","isTruncated":false,"truncated":true},{"name":"zeta","value":"{\"ok\":true}","variableKey":"variable-z","processInstanceKey":"process-a","scopeKey":"process-a","tenantId":"","truncated":true}],"page":{"totalItems":2,"hasMoreTotalItems":false,"endCursor":"next-page"}}`)
	}))
	t.Cleanup(server.Close)

	svc, err := v89.New(&config.Config{APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}}}, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	request := d.UserTaskVariablePageRequest{From: 40, Size: 20}

	page, err := svc.SearchUserTaskEffectiveVariablesPage(context.Background(), "task-a", request)

	require.NoError(t, err)
	require.Equal(t, 1, requests)
	require.NoError(t, decodeErr)
	require.NoError(t, writeErr)
	require.Equal(t, http.MethodPost, method)
	require.Equal(t, "/v2/user-tasks/task-a/effective-variables/search", path)
	require.Equal(t, "false", truncateValues)
	require.Nil(t, requestBody.Filter)
	require.NotNil(t, requestBody.Page)
	require.Equal(t, int32(40), *requestBody.Page.From)
	require.Equal(t, int32(20), *requestBody.Page.Limit)
	require.NotNil(t, requestBody.Sort)
	require.Len(t, *requestBody.Sort, 1)
	require.Equal(t, camundav89.UserTaskVariableSearchQuerySortRequestFieldName, (*requestBody.Sort)[0].Field)
	require.NotNil(t, (*requestBody.Sort)[0].Order)
	require.Equal(t, camundav89.ASC, *(*requestBody.Sort)[0].Order)
	require.Equal(t, request, page.Request)
	require.Equal(t, int32(2), page.RawItemCount)
	require.Equal(t, d.UserTaskReportedTotal{Count: 2, Kind: d.UserTaskReportedTotalKindExact}, page.ReportedTotal)
	require.True(t, page.HasContinuationEvidence)
	require.Equal(t, []d.ProcessInstanceVariable{
		{Name: "alpha", Value: "", VariableKey: "0", ProcessInstanceKey: "process-a", ScopeKey: "task-scope", TenantId: "tenant-b"},
		{Name: "zeta", Value: `{"ok":true}`, VariableKey: "variable-z", ProcessInstanceKey: "process-a", ScopeKey: "process-a", TenantId: "", APITruncated: true},
	}, page.Items)
}

// TestService_SearchUserTaskEffectiveVariablesPage_PreservesCappedTotal verifies lower-bound metadata and truncation-field precedence.
func TestService_SearchUserTaskEffectiveVariablesPage_PreservesCappedTotal(t *testing.T) {
	svc := newTestService(t, &mockUserTaskCamundaClient{
		searchUserTaskEffectiveVariablesWithResponse: func(_ context.Context, key camundav89.UserTaskKey, params *camundav89.SearchUserTaskEffectiveVariablesParams, _ camundav89.SearchUserTaskEffectiveVariablesJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchUserTaskEffectiveVariablesResponse, error) {
			require.Equal(t, camundav89.UserTaskKey("task-a"), key)
			require.NotNil(t, params.TruncateValues)
			require.False(t, *params.TruncateValues)
			return effectiveVariablesSuccess([]byte(`{"items":[{"name":"empty","value":"","variableKey":"","processInstanceKey":"","scopeKey":"","tenantId":"","isTruncated":true,"truncated":false}],"page":{"totalItems":0,"hasMoreTotalItems":true}}`)), nil
		},
	})

	page, err := svc.SearchUserTaskEffectiveVariablesPage(context.Background(), "task-a", d.UserTaskVariablePageRequest{Size: 10})

	require.NoError(t, err)
	require.Equal(t, d.UserTaskReportedTotal{Count: 0, Kind: d.UserTaskReportedTotalKindLowerBound}, page.ReportedTotal)
	require.False(t, page.HasContinuationEvidence)
	require.Equal(t, []d.ProcessInstanceVariable{{Name: "empty", APITruncated: true}}, page.Items)
}

// TestService_SearchUserTaskEffectiveVariablesPage_RejectsBackendAndMalformedResponses pins errors without fallback or recovery calls.
func TestService_SearchUserTaskEffectiveVariablesPage_RejectsBackendAndMalformedResponses(t *testing.T) {
	transportErr := errors.New("connection reset")
	tests := []struct {
		name      string
		response  *camundav89.SearchUserTaskEffectiveVariablesResponse
		transport error
		want      error
	}{
		{name: "transport", transport: transportErr, want: transportErr},
		{name: "http", response: effectiveVariablesResponse(http.StatusServiceUnavailable, []byte(`{"message":"unavailable"}`)), want: d.ErrUnavailable},
		{name: "missing generated payload", response: effectiveVariablesResponse(http.StatusOK, []byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
		{name: "missing items", response: effectiveVariablesSuccess([]byte(`{"page":{"totalItems":0,"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
		{name: "missing page", response: effectiveVariablesSuccess([]byte(`{"items":[]}`)), want: d.ErrMalformedResponse},
		{name: "missing total", response: effectiveVariablesSuccess([]byte(`{"items":[],"page":{"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
		{name: "missing capped flag", response: effectiveVariablesSuccess([]byte(`{"items":[],"page":{"totalItems":0}}`)), want: d.ErrMalformedResponse},
		{name: "missing value", response: effectiveVariablesSuccess([]byte(`{"items":[{"name":"a","variableKey":"v","processInstanceKey":"p","scopeKey":"s","tenantId":"t"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
		{name: "missing metadata", response: effectiveVariablesSuccess([]byte(`{"items":[{"value":"0"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
		{name: "wrong value type", response: effectiveVariablesSuccess([]byte(`{"items":[{"name":"a","value":0,"variableKey":"v","processInstanceKey":"p","scopeKey":"s","tenantId":"t"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
		{name: "negative total", response: effectiveVariablesSuccess([]byte(`{"items":[],"page":{"totalItems":-1,"hasMoreTotalItems":false}}`)), want: d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			svc := newTestService(t, &mockUserTaskCamundaClient{
				searchUserTaskEffectiveVariablesWithResponse: func(context.Context, camundav89.UserTaskKey, *camundav89.SearchUserTaskEffectiveVariablesParams, camundav89.SearchUserTaskEffectiveVariablesJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchUserTaskEffectiveVariablesResponse, error) {
					calls++
					return tt.response, tt.transport
				},
			})

			_, err := svc.SearchUserTaskEffectiveVariablesPage(context.Background(), "task-a", d.UserTaskVariablePageRequest{Size: 10})

			require.ErrorIs(t, err, tt.want)
			require.Equal(t, 1, calls)
		})
	}
}

// effectiveVariablesSuccess creates an HTTP-success response with a present generated payload and caller-controlled raw body.
func effectiveVariablesSuccess(body []byte) *camundav89.SearchUserTaskEffectiveVariablesResponse {
	response := effectiveVariablesResponse(http.StatusOK, body)
	response.JSON200 = &camundav89.VariableSearchQueryResult{}
	return response
}

// effectiveVariablesResponse creates stable HTTP metadata for effective-variable adapter tests.
func effectiveVariablesResponse(status int, body []byte) *camundav89.SearchUserTaskEffectiveVariablesResponse {
	return &camundav89.SearchUserTaskEffectiveVariablesResponse{
		Body:         body,
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/user-tasks/task-a/effective-variables/search", status, http.StatusText(status)),
	}
}
