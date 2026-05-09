// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBatchOperationClient struct {
	searchBatchOperationsWithResponse                func(ctx context.Context, body camundav89.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchBatchOperationsResponse, error)
	getBatchOperationWithResponse                    func(ctx context.Context, batchOperationKey camundav89.BatchOperationKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetBatchOperationResponse, error)
	cancelProcessInstancesBatchOperationWithResponse func(ctx context.Context, body camundav89.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstancesBatchOperationResponse, error)
}

func (m *mockBatchOperationClient) SearchBatchOperationsWithResponse(ctx context.Context, body camundav89.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchBatchOperationsResponse, error) {
	if m.searchBatchOperationsWithResponse == nil {
		panic("unexpected call")
	}
	return m.searchBatchOperationsWithResponse(ctx, body, reqEditors...)
}

func (m *mockBatchOperationClient) GetBatchOperationWithResponse(ctx context.Context, batchOperationKey camundav89.BatchOperationKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetBatchOperationResponse, error) {
	if m.getBatchOperationWithResponse == nil {
		panic("unexpected call")
	}
	return m.getBatchOperationWithResponse(ctx, batchOperationKey, reqEditors...)
}

func (m *mockBatchOperationClient) CancelProcessInstancesBatchOperationWithResponse(ctx context.Context, body camundav89.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstancesBatchOperationResponse, error) {
	if m.cancelProcessInstancesBatchOperationWithResponse == nil {
		panic("unexpected call")
	}
	return m.cancelProcessInstancesBatchOperationWithResponse(ctx, body, reqEditors...)
}

func TestService_CheckReadAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("UsesNonMutatingSearchProbe", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			searchBatchOperationsWithResponse: func(ctx context.Context, body camundav89.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchBatchOperationsResponse, error) {
				assertProbePage(t, body.Page)
				return &camundav89.SearchBatchOperationsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/batch-operations/search", http.StatusOK, "200 OK"),
					JSON200: &camundav89.BatchOperationSearchQueryResult{
						Items: []camundav89.BatchOperationResponse{},
						Page:  camundav89.SearchQueryPageResponse{},
					},
				}, nil
			},
		})

		err := svc.CheckReadAccess(ctx)

		require.NoError(t, err)
	})

	t.Run("ForbiddenMapsToDomainForbidden", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			searchBatchOperationsWithResponse: func(ctx context.Context, body camundav89.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchBatchOperationsResponse, error) {
				return &camundav89.SearchBatchOperationsResponse{
					Body:         []byte(`{"title":"FORBIDDEN","status":403,"detail":"Unauthorized to perform operation 'READ' on resource 'BATCH'"}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/batch-operations/search", http.StatusForbidden, "403 Forbidden"),
				}, nil
			},
		})

		err := svc.CheckReadAccess(ctx)

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrForbidden)
	})
}

func TestService_CancelProcessInstances(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, &mockBatchOperationClient{
		cancelProcessInstancesBatchOperationWithResponse: func(ctx context.Context, body camundav89.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstancesBatchOperationResponse, error) {
			assertCancelFilter(t, body.Filter)
			return &camundav89.CancelProcessInstancesBatchOperationResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/cancellation", http.StatusOK, "200 OK"),
				JSON200: &camundav89.BatchOperationCreatedResult{
					BatchOperationKey:  "cancel-batch-1",
					BatchOperationType: camundav89.BatchOperationTypeEnumCANCELPROCESSINSTANCE,
				},
			}, nil
		},
	})

	op, err := svc.CancelProcessInstances(ctx, d.ProcessInstanceFilter{
		ProcessDefinitionKey: "2251799813686441",
		State:                d.StateActive,
	})

	require.NoError(t, err)
	assert.Equal(t, "cancel-batch-1", op.Key)
	assert.Equal(t, "CANCEL_PROCESS_INSTANCE", op.Type)
}

func newTestService(t *testing.T, client GenBatchOperationClientCamunda) *Service {
	t.Helper()

	cfg := &config.Config{}
	cfg.App.Tenant = "tenant-a"
	cfg.APIs.Camunda.BaseURL = "https://camunda.local/v2"
	svc, err := New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client))
	require.NoError(t, err)
	return svc
}

func assertCancelFilter(t *testing.T, filter camundav89.ProcessInstanceFilter) {
	t.Helper()
	raw, err := json.Marshal(filter)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "tenant-a", decoded["tenantId"])
	assert.Equal(t, "2251799813686441", decoded["processDefinitionKey"])
	assert.Equal(t, "ACTIVE", decoded["state"])
}

func assertProbePage(t *testing.T, page *camundav89.SearchQueryPageRequest) {
	t.Helper()
	require.NotNil(t, page)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, float64(0), decoded["from"])
	assert.Equal(t, float64(1), decoded["limit"])
}

func newHTTPResponse(method string, rawURL string, statusCode int, status string) *http.Response {
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
