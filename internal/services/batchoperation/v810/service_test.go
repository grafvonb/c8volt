// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBatchOperationClient struct {
	searchBatchOperationsWithResponse                func(ctx context.Context, body camundav810.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchBatchOperationsResponse, error)
	getBatchOperationWithResponse                    func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error)
	cancelProcessInstancesBatchOperationWithResponse func(ctx context.Context, body camundav810.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstancesBatchOperationResponse, error)
}

// SearchBatchOperationsWithResponse delegates to the configured mock callback and fails on unexpected calls.
func (m *mockBatchOperationClient) SearchBatchOperationsWithResponse(ctx context.Context, body camundav810.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchBatchOperationsResponse, error) {
	if m.searchBatchOperationsWithResponse == nil {
		panic("unexpected call")
	}
	return m.searchBatchOperationsWithResponse(ctx, body, reqEditors...)
}

// GetBatchOperationWithResponse delegates to the configured mock callback and fails on unexpected calls.
func (m *mockBatchOperationClient) GetBatchOperationWithResponse(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
	if m.getBatchOperationWithResponse == nil {
		panic("unexpected call")
	}
	return m.getBatchOperationWithResponse(ctx, batchOperationKey, reqEditors...)
}

// CancelProcessInstancesBatchOperationWithResponse delegates to the configured mock callback and fails on unexpected calls.
func (m *mockBatchOperationClient) CancelProcessInstancesBatchOperationWithResponse(ctx context.Context, body camundav810.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstancesBatchOperationResponse, error) {
	if m.cancelProcessInstancesBatchOperationWithResponse == nil {
		panic("unexpected call")
	}
	return m.cancelProcessInstancesBatchOperationWithResponse(ctx, body, reqEditors...)
}

// TestService_CheckReadAccess proves V810 uses a non-mutating batch-operation search probe and preserves transport errors.
func TestService_CheckReadAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("UsesNonMutatingSearchProbe", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			searchBatchOperationsWithResponse: func(ctx context.Context, body camundav810.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchBatchOperationsResponse, error) {
				assertProbePage(t, body.Page)
				return &camundav810.SearchBatchOperationsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/batch-operations/search", http.StatusOK, "200 OK"),
					JSON200: &camundav810.BatchOperationSearchQueryResult{
						Items: []camundav810.BatchOperationResponse{},
						Page:  camundav810.SearchQueryPageResponse{},
					},
				}, nil
			},
		})

		err := svc.CheckReadAccess(ctx)

		require.NoError(t, err)
	})

	t.Run("ForbiddenMapsToDomainForbidden", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			searchBatchOperationsWithResponse: func(ctx context.Context, body camundav810.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchBatchOperationsResponse, error) {
				return &camundav810.SearchBatchOperationsResponse{
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

// TestService_CancelProcessInstances verifies V810 filter conversion and mutation result conversion.
func TestService_CancelProcessInstances(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, &mockBatchOperationClient{
		cancelProcessInstancesBatchOperationWithResponse: func(ctx context.Context, body camundav810.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstancesBatchOperationResponse, error) {
			assertCancelFilter(t, body.Filter)
			return &camundav810.CancelProcessInstancesBatchOperationResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/cancellation", http.StatusOK, "200 OK"),
				JSON200: &camundav810.BatchOperationCreatedResult{
					BatchOperationKey:  "cancel-batch-1",
					BatchOperationType: camundav810.BatchOperationTypeEnumCANCELPROCESSINSTANCE,
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

// TestService_CancelProcessInstances_MalformedPayload keeps nil-success payloads classified as malformed.
func TestService_CancelProcessInstances_MalformedPayload(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, &mockBatchOperationClient{
		cancelProcessInstancesBatchOperationWithResponse: func(ctx context.Context, body camundav810.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstancesBatchOperationResponse, error) {
			return &camundav810.CancelProcessInstancesBatchOperationResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/cancellation", http.StatusOK, "200 OK"),
			}, nil
		},
	})

	op, err := svc.CancelProcessInstances(ctx, d.ProcessInstanceFilter{})

	require.Error(t, err)
	assert.ErrorIs(t, err, d.ErrMalformedResponse)
	assert.Empty(t, op.Key)
}

// TestService_WaitForCompletion verifies V810 poll conversion, success state, and failed-item errors.
func TestService_WaitForCompletion(t *testing.T) {
	ctx := context.Background()
	operationType := camundav810.BatchOperationTypeEnumCANCELPROCESSINSTANCE

	t.Run("MapsCompletionCounts", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			getBatchOperationWithResponse: func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
				assert.Equal(t, "batch-1", batchOperationKey)
				return &camundav810.GetBatchOperationResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/batch-1", http.StatusOK, "200 OK"),
					JSON200: &camundav810.BatchOperationResponse{
						BatchOperationKey:        "batch-1",
						BatchOperationType:       &operationType,
						State:                    camundav810.BatchOperationStateEnumCOMPLETED,
						OperationsTotalCount:     10,
						OperationsCompletedCount: 10,
						OperationsFailedCount:    0,
					},
				}, nil
			},
		})

		op, err := svc.WaitForCompletion(ctx, "batch-1")

		require.NoError(t, err)
		assert.Equal(t, "batch-1", op.Key)
		assert.Equal(t, "CANCEL_PROCESS_INSTANCE", op.Type)
		assert.Equal(t, int32(10), op.OperationsTotalCount)
		assert.Equal(t, int32(10), op.OperationsCompletedCount)
		assert.Equal(t, int32(0), op.OperationsFailedCount)
	})

	t.Run("CompletesWithoutOptionalOperationType", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			getBatchOperationWithResponse: func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
				var payload camundav810.BatchOperationResponse
				require.NoError(t, json.Unmarshal([]byte(`{"batchOperationKey":"batch-1","state":"COMPLETED","operationsTotalCount":1,"operationsCompletedCount":1,"operationsFailedCount":0}`), &payload))
				return &camundav810.GetBatchOperationResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/batch-1", http.StatusOK, "200 OK"),
					JSON200:      &payload,
				}, nil
			},
		})

		op, err := svc.WaitForCompletion(ctx, "batch-1")

		require.NoError(t, err)
		assert.Equal(t, "batch-1", op.Key)
		assert.Empty(t, op.Type)
		assert.Equal(t, "COMPLETED", op.State)
		assert.Equal(t, int32(1), op.OperationsCompletedCount)
		assert.Equal(t, http.StatusOK, op.StatusCode)
	})

	t.Run("FailsWhenCompletedBatchHasFailedItems", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			getBatchOperationWithResponse: func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
				return &camundav810.GetBatchOperationResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/batch-2", http.StatusOK, "200 OK"),
					JSON200: &camundav810.BatchOperationResponse{
						BatchOperationKey:        "batch-2",
						BatchOperationType:       &operationType,
						State:                    camundav810.BatchOperationStateEnumCOMPLETED,
						OperationsTotalCount:     10,
						OperationsCompletedCount: 9,
						OperationsFailedCount:    1,
						Errors: []camundav810.BatchOperationError{
							{PartitionId: 1, Type: camundav810.QUERYFAILED, Message: "query rejected"},
						},
					},
				}, nil
			},
		})

		op, err := svc.WaitForCompletion(ctx, "batch-2")

		require.Error(t, err)
		assert.Equal(t, int32(1), op.OperationsFailedCount)
		assert.Contains(t, err.Error(), "1/10 failed item")
		assert.Contains(t, err.Error(), "query rejected")
	})

	t.Run("RetriesTemporaryInvisibility", func(t *testing.T) {
		calls := 0
		svc := newTestService(t, &mockBatchOperationClient{
			getBatchOperationWithResponse: func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
				calls++
				if calls == 1 {
					return &camundav810.GetBatchOperationResponse{
						Body:         []byte(`{"title":"NOT_FOUND","status":404,"detail":"batch is not visible yet"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/batch-late", http.StatusNotFound, "404 Not Found"),
					}, nil
				}
				return completedBatchResponse("batch-late", operationType, 2, 2, 0), nil
			},
		})

		op, err := svc.WaitForCompletion(ctx, "batch-late")

		require.NoError(t, err)
		assert.Equal(t, 2, calls)
		assert.Equal(t, "COMPLETED", op.State)
	})

	t.Run("RejectsTerminalFailureWithoutFailedItemCount", func(t *testing.T) {
		svc := newTestService(t, &mockBatchOperationClient{
			getBatchOperationWithResponse: func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
				return &camundav810.GetBatchOperationResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/batch-failed", http.StatusOK, "200 OK"),
					JSON200: &camundav810.BatchOperationResponse{
						BatchOperationKey:        "batch-failed",
						BatchOperationType:       &operationType,
						State:                    camundav810.BatchOperationStateEnumFAILED,
						OperationsTotalCount:     3,
						OperationsCompletedCount: 1,
					},
				}, nil
			},
		})

		op, err := svc.WaitForCompletion(ctx, "batch-failed")

		require.Error(t, err)
		assert.Equal(t, "FAILED", op.State)
		assert.Contains(t, err.Error(), "finished with state FAILED")
	})

	t.Run("HonorsContextDeadlineWhileActive", func(t *testing.T) {
		waitCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()
		svc := newTestService(t, &mockBatchOperationClient{
			getBatchOperationWithResponse: func(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error) {
				return &camundav810.GetBatchOperationResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/batch-active", http.StatusOK, "200 OK"),
					JSON200: &camundav810.BatchOperationResponse{
						BatchOperationKey:  "batch-active",
						BatchOperationType: &operationType,
						State:              camundav810.BatchOperationStateEnumACTIVE,
					},
				}, nil
			},
		})

		op, err := svc.WaitForCompletion(waitCtx, "batch-active")

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
		assert.Equal(t, "ACTIVE", op.State)
	})
}

// TestBatchOperationFromPayloadPreservesDocumentedLifecycle verifies every
// generated 8.10 batch state is decoded literally without inventing DRAINING.
func TestBatchOperationFromPayloadPreservesDocumentedLifecycle(t *testing.T) {
	states := []camundav810.BatchOperationStateEnum{
		camundav810.BatchOperationStateEnumCREATED,
		camundav810.BatchOperationStateEnumACTIVE,
		camundav810.BatchOperationStateEnumSUSPENDED,
		camundav810.BatchOperationStateEnumCOMPLETED,
		camundav810.BatchOperationStateEnumPARTIALLYCOMPLETED,
		camundav810.BatchOperationStateEnumFAILED,
		camundav810.BatchOperationStateEnumCANCELED,
	}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			op := batchOperationFromPayload(&camundav810.BatchOperationResponse{
				BatchOperationKey: "batch-1",
				State:             state,
			}, http.StatusOK, "200 OK")

			assert.Equal(t, string(state), op.State)
			assert.NotEqual(t, "DRAINING", op.State)
		})
	}
}

// completedBatchResponse builds a terminal generated response for polling tests.
func completedBatchResponse(key string, operationType camundav810.BatchOperationTypeEnum, total int32, completed int32, failed int32) *camundav810.GetBatchOperationResponse {
	return &camundav810.GetBatchOperationResponse{
		HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/batch-operations/"+key, http.StatusOK, "200 OK"),
		JSON200: &camundav810.BatchOperationResponse{
			BatchOperationKey:        key,
			BatchOperationType:       &operationType,
			State:                    camundav810.BatchOperationStateEnumCOMPLETED,
			OperationsTotalCount:     total,
			OperationsCompletedCount: completed,
			OperationsFailedCount:    failed,
		},
	}
}

// newTestService creates a V810 service around the supplied mock generated client.
func newTestService(t *testing.T, client GenBatchOperationClientCamunda) *Service {
	t.Helper()

	cfg := &config.Config{}
	cfg.App.Tenant = "tenant-a"
	cfg.APIs.Camunda.BaseURL = "https://camunda.local/v2"
	svc, err := New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client))
	require.NoError(t, err)
	return svc
}

// assertCancelFilter checks the JSON shape emitted to the generated V810 client.
func assertCancelFilter(t *testing.T, filter camundav810.ProcessInstanceFilter) {
	t.Helper()
	raw, err := json.Marshal(filter)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "tenant-a", decoded["tenantId"])
	assert.Equal(t, "2251799813686441", decoded["processDefinitionKey"])
	assert.Equal(t, "ACTIVE", decoded["state"])
}

// assertProbePage checks the V810 read probe stays bounded to one item.
func assertProbePage(t *testing.T, page *camundav810.SearchQueryPageRequest) {
	t.Helper()
	require.NotNil(t, page)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, float64(0), decoded["from"])
	assert.Equal(t, float64(1), decoded["limit"])
}

// newHTTPResponse builds a minimal generated-client HTTP response for error normalization.
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
