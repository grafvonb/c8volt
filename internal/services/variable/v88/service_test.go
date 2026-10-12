// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

type searchVariableClient struct {
	search func(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error)
}

func (m *searchVariableClient) SearchVariablesWithResponse(ctx context.Context, params *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, opts ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
	return m.search(ctx, params, body, opts...)
}

func (*searchVariableClient) CreateElementInstanceVariablesWithResponse(context.Context, camundav88.ElementInstanceKey, camundav88.CreateElementInstanceVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error) {
	panic("unexpected variable mutation")
}

func TestSearchProcessInstanceVariablesCompletesCursorPages(t *testing.T) {
	calls := 0
	svc := newScopeVariableTestService(t, &searchVariableClient{search: func(_ context.Context, params *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
		calls++
		require.NotNil(t, params.TruncateValues)
		require.False(t, *params.TruncateValues)
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		require.Contains(t, string(raw), `"processInstanceKey":"123"`)
		require.Contains(t, string(raw), `"scopeKey":"123"`)
		require.Contains(t, string(raw), `"tenantId":"tenant-a"`)
		if calls == 1 {
			offset, err := body.Page.AsOffsetPagination()
			require.NoError(t, err)
			require.Zero(t, *offset.From)
			return searchVariablesSuccess([]byte(`{"items":[{"name":"zeta","value":"2","variableKey":"v2","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-a","truncated":true},{"name":"nested","value":"0","variableKey":"vn","processInstanceKey":"123","scopeKey":"child","tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"next"}}`)), nil
		}
		cursor, err := body.Page.AsCursorForwardPagination()
		require.NoError(t, err)
		require.Equal(t, camundav88.EndCursor("next"), cursor.After)
		return searchVariablesSuccess([]byte(`{"items":[{"name":"alpha","value":"1","variableKey":"v1","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-a","isTruncated":false}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`)), nil
	}})

	got, err := svc.SearchProcessInstanceVariables(context.Background(), "123")

	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, []d.ProcessInstanceVariable{
		{Name: "alpha", Value: "1", VariableKey: "v1", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant-a"},
		{Name: "zeta", Value: "2", VariableKey: "v2", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant-a", APITruncated: true},
	}, got)
}

func TestSearchProcessInstanceVariablesRejectsMalformedMetadata(t *testing.T) {
	svc := newScopeVariableTestService(t, &searchVariableClient{search: func(context.Context, *camundav88.SearchVariablesParams, camundav88.SearchVariablesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error) {
		return searchVariablesSuccess([]byte(`{"items":[],"page":{"totalItems":0}}`)), nil
	}})
	_, err := svc.SearchProcessInstanceVariables(context.Background(), "123")
	require.ErrorIs(t, err, d.ErrMalformedResponse)
}

func searchVariablesSuccess(body []byte) *camundav88.SearchVariablesResponse {
	u, _ := url.Parse("https://camunda.local/v2/variables/search")
	return &camundav88.SearchVariablesResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Request: &http.Request{Method: http.MethodPost, URL: u}}, Body: body, JSON200: &camundav88.VariableSearchQueryResult{}}
}
