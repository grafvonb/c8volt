// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v810 "github.com/grafvonb/c8volt/internal/services/processinstance/v810"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCamundaClient struct {
	createProcessInstanceWithResponse func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error)
	searchProcessInstancesWithResp    func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error)
	searchVariablesWithResponse       func(ctx context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error)
	createElementInstanceVariables    func(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error)
	cancelProcessInstanceWithResponse func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error)
	deleteProcessInstanceWithResponse func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error)
	getProcessInstanceWithResponse    func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error)
}

var _ v810.GenProcessInstanceClientCamunda = (*mockCamundaClient)(nil)

func (m *mockCamundaClient) CreateProcessInstanceWithResponse(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
	return m.createProcessInstanceWithResponse(ctx, body, reqEditors...)
}

func (m *mockCamundaClient) SearchProcessInstancesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
	return m.searchProcessInstancesWithResp(ctx, contentType, body, reqEditors...)
}

func (m *mockCamundaClient) SearchVariablesWithResponse(ctx context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
	return m.searchVariablesWithResponse(ctx, params, body, reqEditors...)
}

func (m *mockCamundaClient) CreateElementInstanceVariablesWithResponse(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
	return m.createElementInstanceVariables(ctx, elementInstanceKey, body, reqEditors...)
}

func (m *mockCamundaClient) CancelProcessInstanceWithResponse(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
	return m.cancelProcessInstanceWithResponse(ctx, key, body, reqEditors...)
}

func (m *mockCamundaClient) DeleteProcessInstanceWithResponse(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
	return m.deleteProcessInstanceWithResponse(ctx, key, body, reqEditors...)
}

func (m *mockCamundaClient) GetProcessInstanceWithResponse(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
	return m.getProcessInstanceWithResponse(ctx, key, reqEditors...)
}

// TestService_CreateProcessInstance verifies creation request mapping and wait-based confirmation semantics.
func TestService_CreateProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessNoWait", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
				payload := marshalJSON(t, body)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"processDefinitionVersion":7`)
				assert.Contains(t, payload, `"tenantId":"tenant-a"`)
				assert.Contains(t, payload, `"orderId":"42"`)
				return &camundav810.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav810.CreateProcessInstanceResult{
						ProcessDefinitionId:      "demo",
						ProcessDefinitionKey:     "proc-key",
						ProcessDefinitionVersion: 7,
						ProcessInstanceKey:       "123",
						TenantId:                 "tenant-a",
						Variables:                map[string]any{"orderId": "42"},
					},
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{
			BpmnProcessId:            "demo",
			ProcessDefinitionVersion: 7,
			TenantId:                 "tenant-a",
			Variables:                map[string]any{"orderId": "42"},
		}, services.WithNoWait())

		require.NoError(t, err)
		assert.Equal(t, "123", creation.Key)
		assert.Equal(t, "demo", creation.BpmnProcessId)
		assert.Equal(t, int32(7), creation.ProcessDefinitionVersion)
		assert.Equal(t, "tenant-a", creation.TenantId)
		assert.Equal(t, "42", creation.Variables["orderId"])
		assert.NotEmpty(t, creation.StartDate)
	})

	t.Run("DefaultsEmptyTenantToDefaultTenant", func(t *testing.T) {
		cfg := testConfig()
		cfg.App.Tenant = ""
		svc := newTestService(t, cfg, &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
				payload := marshalJSON(t, body)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"tenantId":"\u003cdefault\u003e"`)
				return &camundav810.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav810.CreateProcessInstanceResult{
						ProcessDefinitionId:      "demo",
						ProcessDefinitionKey:     "proc-key",
						ProcessDefinitionVersion: 7,
						ProcessInstanceKey:       "123",
						TenantId:                 config.DefaultTenant,
					},
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo"}, services.WithNoWait())

		require.NoError(t, err)
		assert.Equal(t, config.DefaultTenant, creation.TenantId)
	})

	t.Run("SuccessWaitsForObservableCreationStates", func(t *testing.T) {
		tests := []struct {
			name  string
			state string
		}{
			{name: "Active", state: "ACTIVE"},
			{name: "Completed", state: "COMPLETED"},
			{name: "Terminal", state: "TERMINATED"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				getCalls := 0
				svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
					createProcessInstanceWithResponse: func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
						return &camundav810.CreateProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
							JSON200: &camundav810.CreateProcessInstanceResult{
								ProcessDefinitionId:      "demo",
								ProcessDefinitionKey:     "proc-key",
								ProcessDefinitionVersion: 7,
								ProcessInstanceKey:       "123",
								TenantId:                 "tenant-a",
							},
						}, nil
					},
					searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
					cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
					deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
					getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
						getCalls++
						assert.Equal(t, camundav810.ProcessInstanceKey("123"), key)
						return &camundav810.GetProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
							JSON200:      new(makeProcessInstanceResult("123", tt.state, "")),
						}, nil
					},
				})

				creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo", TenantId: "tenant-a"})

				require.NoError(t, err)
				assert.Equal(t, "123", creation.Key)
				assert.Equal(t, "2026-03-23T18:00:00Z", creation.StartDate)
				assert.Equal(t, d.State(tt.state), creation.State)
				assert.NotEmpty(t, creation.StartConfirmedAt)
				assert.Equal(t, 1, getCalls)
			})
		}
	})

	t.Run("RejectsNotFoundDuringConfirmation", func(t *testing.T) {
		getCalls := 0
		svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
				return &camundav810.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav810.CreateProcessInstanceResult{
						ProcessDefinitionId:      "demo",
						ProcessDefinitionKey:     "proc-key",
						ProcessDefinitionVersion: 7,
						ProcessInstanceKey:       "123",
						TenantId:                 "tenant-a",
					},
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				getCalls++
				return &camundav810.GetProcessInstanceResponse{
					Body:         []byte(`{"message":"not found"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusNotFound, "404 Not Found"),
				}, nil
			},
		})

		_, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo", TenantId: "tenant-a"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "wait for observable state")
		assert.Contains(t, err.Error(), "exceeded max_retries")
		assert.Equal(t, 2, getCalls)
	})

	t.Run("CreateLogOmitsStartAndConfirmedTimestamps", func(t *testing.T) {
		var logBuf bytes.Buffer
		svc, err := v810.New(
			waitTestConfig(),
			&http.Client{},
			slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo)),
			v810.WithClientCamunda(&mockCamundaClient{
				createProcessInstanceWithResponse: func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
					return &camundav810.CreateProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
						JSON200: &camundav810.CreateProcessInstanceResult{
							ProcessDefinitionId:      "demo",
							ProcessDefinitionKey:     "proc-key",
							ProcessDefinitionVersion: 7,
							ProcessInstanceKey:       "123",
							TenantId:                 "tenant-a",
						},
					}, nil
				},
				searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
				cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
				deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
				getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
					return &camundav810.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
					}, nil
				},
			}),
		)
		require.NoError(t, err)

		_, err = svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo", TenantId: "tenant-a"})

		require.NoError(t, err)
		output := logBuf.String()
		assert.Contains(t, output, "INFO pi 123 created; pd proc-key demo v7 tenant-a; state ACTIVE")
		assert.NotContains(t, output, "start ")
		assert.NotContains(t, output, "confirmed")
	})
}

// TestService_SearchAndLookup verifies v8.10 search request mapping, paging,
// lookup behavior, and native filter serialization.
func TestService_SearchAndLookup(t *testing.T) {
	ctx := context.Background()

	t.Run("SearchUsesTenantSafeBodyAndPageMetadata", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"tenantId":"tenant"`)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"processDefinitionKey":"9001"`)
				assert.Contains(t, payload, `"processDefinitionVersion":3`)
				assert.Contains(t, payload, `"processDefinitionVersionTag":"stable"`)
				assert.Contains(t, payload, `"state":"ACTIVE"`)
				assert.Contains(t, payload, `"startDate":{"$gte":"2026-07-18T10:00:00.123Z","$lte":"2026-07-19T23:59:59.999999999Z"}`)
				assert.Contains(t, payload, `"parentProcessInstanceKey":"456"`)
				assert.Contains(t, payload, `"endDate":{"$exists":true,"$lte":"2026-04-03T23:59:59.999999999Z"}`)
				assert.Contains(t, payload, `"limit":25`)
				assert.Contains(t, payload, `"$exists":true`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "456")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 2, HasMoreTotalItems: true},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			BpmnProcessId:        "demo",
			ProcessDefinitionKey: "9001",
			ProcessVersion:       3,
			ProcessVersionTag:    "stable",
			State:                d.StateActive,
			ParentKey:            "456",
			StartDateAfter:       "2026-07-18T10:00:00.123Z",
			StartDateBefore:      "2026-07-19",
			EndDateBefore:        "2026-04-03",
		}, d.ProcessInstancePageRequest{From: 0, Size: 25})

		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "123", page.Items[0].Key)
		assert.Equal(t, "456", page.Items[0].ParentKey)
		assert.Equal(t, d.ProcessInstanceOverflowStateHasMore, page.OverflowState)
		require.NotNil(t, page.ReportedTotal)
		assert.EqualValues(t, 2, page.ReportedTotal.Count)
		assert.Equal(t, d.ProcessInstanceReportedTotalKindLowerBound, page.ReportedTotal.Kind)
	})

	t.Run("DoesNotTreatLowerBoundTotalMetadataAsAnotherPageWhenNoItemsAreReturned", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 10000, HasMoreTotalItems: true},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 10000, Size: 500})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateNoMore, page.OverflowState)
		require.NotNil(t, page.ReportedTotal)
		assert.EqualValues(t, 10000, page.ReportedTotal.Count)
		assert.Equal(t, d.ProcessInstanceReportedTotalKindLowerBound, page.ReportedTotal.Kind)
		require.Empty(t, page.Items)
	})

	t.Run("TreatsExactCursorFinalPageAsNoMoreEvenWhenEndCursorIsPresent", func(t *testing.T) {
		endCursor := camundav810.EndCursor("cursor-final")
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"after":"cursor-prev"`)
				assert.Contains(t, payload, `"limit":2`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("125", "ACTIVE", "")},
					Page: camundav810.SearchQueryPageResponse{
						TotalItems: 3,
						EndCursor:  &endCursor,
					},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 2, Size: 2, After: "cursor-prev"})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateNoMore, page.OverflowState)
		require.NotNil(t, page.ReportedTotal)
		assert.EqualValues(t, 3, page.ReportedTotal.Count)
		assert.Equal(t, d.ProcessInstanceReportedTotalKindExact, page.ReportedTotal.Kind)
		require.Len(t, page.Items, 1)
		require.Equal(t, "cursor-final", page.EndCursor)
	})

	t.Run("MapsVariableExistenceFilters", func(t *testing.T) {
		exists := true
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"variables":[{"name":"customerId","value":{"$exists":true}},{"name":"payload","value":{"$exists":true}}]`)
				assert.Contains(t, payload, `"tenantId":"tenant"`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			VariableFilters: d.ProcessInstanceVariableFilterSet{
				Clauses: []d.ProcessInstanceVariableFilterClause{
					{Name: "customerId", Operator: d.ProcessInstanceVariableFilterOperatorExists, Exists: &exists},
					{Name: "payload", Operator: d.ProcessInstanceVariableFilterOperatorExists, Exists: &exists},
				},
			},
		}, d.ProcessInstancePageRequest{From: 0, Size: 25})

		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "123", page.Items[0].Key)
	})

	t.Run("MapsVariableEqualityFilters", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"variables":[{"name":"status","value":{"$eq":"\"approved\""}},{"name":"payload","value":{"$eq":"\"payload,with,comma\""}}]`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			VariableFilters: d.ProcessInstanceVariableFilterSet{
				Clauses: []d.ProcessInstanceVariableFilterClause{
					{Name: "status", Operator: d.ProcessInstanceVariableFilterOperatorEq, Value: `"approved"`},
					{Name: "payload", Operator: d.ProcessInstanceVariableFilterOperatorEq, Value: `"payload,with,comma"`},
				},
			},
		}, d.ProcessInstancePageRequest{From: 0, Size: 25})

		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "123", page.Items[0].Key)
	})

	t.Run("MapsVariableLikeFilters", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"variables":[{"name":"email","value":{"$like":"*@example.com"}},{"name":"customerId","value":{"$like":"CUST-????"}},{"name":"literal","value":{"$like":"invoice-\\*"}}]`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			VariableFilters: d.ProcessInstanceVariableFilterSet{
				Clauses: []d.ProcessInstanceVariableFilterClause{
					{Name: "email", Operator: d.ProcessInstanceVariableFilterOperatorLike, Value: `*@example.com`},
					{Name: "customerId", Operator: d.ProcessInstanceVariableFilterOperatorLike, Value: `CUST-????`},
					{Name: "literal", Operator: d.ProcessInstanceVariableFilterOperatorLike, Value: `invoice-\*`},
				},
			},
		}, d.ProcessInstancePageRequest{From: 0, Size: 25})

		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "123", page.Items[0].Key)
	})

	t.Run("MapsVariableAdvancedFilters", func(t *testing.T) {
		exists := false
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"variables":[{"name":"status","value":{"$neq":"\"failed\""}},{"name":"active","value":{"$exists":false}},{"name":"kind","value":{"$in":["approved","pending"]}},{"name":"segment","value":{"$notIn":["legacy","test"]}}]`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			VariableFilters: d.ProcessInstanceVariableFilterSet{
				Clauses: []d.ProcessInstanceVariableFilterClause{
					{Name: "status", Operator: d.ProcessInstanceVariableFilterOperatorNeq, Value: `"failed"`},
					{Name: "active", Operator: d.ProcessInstanceVariableFilterOperatorExists, Exists: &exists},
					{Name: "kind", Operator: d.ProcessInstanceVariableFilterOperatorIn, Value: `["approved","pending"]`},
					{Name: "segment", Operator: d.ProcessInstanceVariableFilterOperatorNotIn, Value: `["legacy","test"]`},
				},
			},
		}, d.ProcessInstancePageRequest{From: 0, Size: 25})

		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "123", page.Items[0].Key)
	})

	t.Run("MapsCanceledSearchStateToTerminated", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"state":"TERMINATED"`)
				assert.NotContains(t, payload, `"state":"CANCELED"`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "CANCELED", "")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			State: d.StateCanceled,
		}, d.ProcessInstancePageRequest{From: 0, Size: 25})

		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, d.StateCanceled, page.Items[0].State)
	})

	t.Run("GetProcessInstanceUsesGeneratedClient", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				assert.Equal(t, camundav810.ProcessInstanceKey("123"), key)
				return &camundav810.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
				}, nil
			},
		})

		pi, err := svc.GetProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.Equal(t, "123", pi.Key)
		assert.Equal(t, d.StateActive, pi.State)
		assert.Equal(t, "tenant", pi.TenantId)
	})

	t.Run("GetProcessInstanceTreatsNotFoundAsNotFound", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				return &camundav810.GetProcessInstanceResponse{
					Body:         []byte(`{"message":"not found"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusNotFound, "404 Not Found"),
				}, nil
			},
		})

		_, err := svc.GetProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrNotFound)
		assert.NotContains(t, err.Error(), "parent process instances were not found")
	})

	t.Run("SearchPushesDownParentPresenceAndIncidentPresence", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"hasIncident":true`)
				assert.Contains(t, payload, `"parentProcessInstanceKey":{"$exists":true}`)
				assert.NotContains(t, payload, `"parentProcessInstanceKey":"`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "456")},
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			HasParent:   new(true),
			HasIncident: new(true),
		}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
		assert.Equal(t, "456", items[0].ParentKey)
	})
}

// Variable lookup must request process-scope values and decode fields missing from the generated model.
func TestService_SearchProcessInstanceVariables_UsesProcessInstanceAndScopeFilters(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testConfig(), &mockCamundaClient{
		createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
		searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
		searchVariablesWithResponse: func(ctx context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
			require.NotNil(t, params)
			require.NotNil(t, params.TruncateValues)
			assert.False(t, *params.TruncateValues)
			payload := marshalJSON(t, body)
			assert.Contains(t, payload, `"processInstanceKey":"123"`)
			assert.Contains(t, payload, `"scopeKey":"123"`)
			assert.Contains(t, payload, `"tenantId":"tenant"`)
			assert.Contains(t, payload, `"field":"name"`)
			assert.Contains(t, payload, `"order":"ASC"`)
			rawBody := []byte(`{"items":[{"name":"zeta","value":"2","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"},{"name":"alpha","value":"1","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant","isTruncated":true}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`)
			return &camundav810.SearchVariablesResponse{
				Body:         rawBody,
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/variables/search", http.StatusOK, "200 OK"),
				JSON200:      &camundav810.VariableSearchQueryResult{},
			}, nil
		},
		cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
		getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
	})

	got, err := svc.SearchProcessInstanceVariables(ctx, "123")

	require.NoError(t, err)
	require.Equal(t, []d.ProcessInstanceVariable{
		{Name: "alpha", Value: "1", VariableKey: "901", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant", APITruncated: true},
		{Name: "zeta", Value: "2", VariableKey: "902", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant"},
	}, got)
}

func TestElementInstanceVariablesUpdate_UsesProcessInstanceKeyAsElementInstanceKey(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, testConfig(), &mockCamundaClient{
		createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
		searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
		searchVariablesWithResponse:       unexpectedSearchVariables(t),
		createElementInstanceVariables: func(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			assert.Equal(t, camundav810.ElementInstanceKey("123"), elementInstanceKey)
			assert.Equal(t, map[string]any{"foo": "bar"}, body.Variables)
			return &camundav810.CreateElementInstanceVariablesResponse{
				Body:         []byte{},
				HTTPResponse: newHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/123/variables", http.StatusNoContent, "204 No Content"),
			}, nil
		},
		cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
		getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
	})

	got, err := svc.UpdateProcessInstanceVariables(ctx, "123", map[string]any{"foo": "bar"}, services.WithNoWait())

	require.NoError(t, err)
	assert.Equal(t, d.ProcessInstanceVariableUpdateResponse{
		Key:        "123",
		Ok:         true,
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
	}, got)
}

// TestService_CancelAndDeleteProcessInstance verifies cancellation and deletion lifecycle behavior for v8.10.
func TestService_CancelAndDeleteProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("CancelPreservesOptOutReadAndSubmissionBoundaries", func(t *testing.T) {
		tests := []struct {
			name          string
			opts          []services.CallOption
			states        []string
			wantReads     int
			wantDiscovery int
		}{
			{name: "NoWait", opts: []services.CallOption{services.WithNoWait()}, states: []string{"ACTIVE"}, wantReads: 1},
			{name: "NoStateCheck", opts: []services.CallOption{services.WithNoStateCheck()}, states: []string{"ACTIVE", "ACTIVE", "CANCELED"}, wantReads: 3, wantDiscovery: 1},
			{name: "NoWaitAndNoStateCheck", opts: []services.CallOption{services.WithNoWait(), services.WithNoStateCheck()}, states: nil},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fixture := newCancellationFixture(t, map[string][]string{"123": tt.states}, nil)

				resp, instances, err := fixture.service(t).CancelProcessInstance(ctx, "123", tt.opts...)

				require.NoError(t, err)
				assert.True(t, resp.Ok)
				assert.Equal(t, 1, fixture.cancellationCount())
				assert.Equal(t, tt.wantReads, fixture.readCount("123"))
				assert.Equal(t, tt.wantDiscovery, fixture.discoveryCount())
				if len(tt.opts) == 2 {
					assert.Empty(t, instances)
				}
			})
		}
	})

	t.Run("CancelNoWait", func(t *testing.T) {
		var cancelled string
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
				cancelled = key
				return &camundav810.CancelProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusAccepted, "202 Accepted"),
				}, nil
			},
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		resp, items, err := svc.CancelProcessInstance(ctx, "123", services.WithNoStateCheck(), services.WithNoWait())

		require.NoError(t, err)
		assert.Equal(t, "123", cancelled)
		assert.True(t, resp.Ok)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		assert.Empty(t, items)
	})

	t.Run("CancelPropagatesUnrelatedPrecheckReadError", func(t *testing.T) {
		cancellations := 0
		camunda := newStrictCamundaClient(t)
		camunda.getProcessInstanceWithResponse = func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
			return &camundav810.GetProcessInstanceResponse{
				Body:         []byte(`{"message":"backend unavailable"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusInternalServerError, "500 Internal Server Error"),
			}, nil
		}
		camunda.cancelProcessInstanceWithResponse = func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
			cancellations++
			return nil, errors.New("unexpected cancellation submission")
		}

		resp, instances, err := newTestService(t, waitTestConfig(), camunda).CancelProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrInternal)
		assert.False(t, resp.Ok)
		assert.Empty(t, instances)
		assert.Zero(t, cancellations)
	})

	t.Run("CancelRetriesTransientSubmissionAndPreservesPermanentError", func(t *testing.T) {
		t.Run("RetriesRateLimit", func(t *testing.T) {
			calls := 0
			camunda := newStrictCamundaClient(t)
			camunda.cancelProcessInstanceWithResponse = func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
				calls++
				if calls == 1 {
					httpResp := newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusTooManyRequests, "429 Too Many Requests")
					httpResp.Header = http.Header{"Retry-After": []string{"0"}}
					return &camundav810.CancelProcessInstanceResponse{
						Body:         []byte(`{"message":"rate limited"}`),
						HTTPResponse: httpResp,
					}, nil
				}
				return &camundav810.CancelProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusAccepted, "202 Accepted"),
				}, nil
			}

			resp, _, err := newTestService(t, waitTestConfig(), camunda).CancelProcessInstance(ctx, "123", services.WithNoStateCheck(), services.WithNoWait())

			require.NoError(t, err)
			assert.True(t, resp.Ok)
			assert.Equal(t, 2, calls)
		})

		t.Run("ReturnsPermanentSubmissionError", func(t *testing.T) {
			calls := 0
			camunda := newStrictCamundaClient(t)
			camunda.cancelProcessInstanceWithResponse = func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
				calls++
				return &camundav810.CancelProcessInstanceResponse{
					Body:         []byte(`{"message":"forbidden"}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusForbidden, "403 Forbidden"),
				}, nil
			}

			resp, instances, err := newTestService(t, waitTestConfig(), camunda).CancelProcessInstance(ctx, "123", services.WithNoStateCheck(), services.WithNoWait())

			require.Error(t, err)
			assert.ErrorIs(t, err, d.ErrForbidden)
			assert.False(t, resp.Ok)
			assert.Empty(t, instances)
			assert.Equal(t, 1, calls)
		})
	})

	t.Run("CancelFamilyDiscoveryErrorStopsBeforePolling", func(t *testing.T) {
		cancellations := 0
		keyReads := 0
		discoveryReads := 0
		camunda := newStrictCamundaClient(t)
		camunda.getProcessInstanceWithResponse = func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
			keyReads++
			return &camundav810.GetProcessInstanceResponse{
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
				JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
			}, nil
		}
		camunda.searchProcessInstancesWithResp = func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
			discoveryReads++
			return &camundav810.SearchProcessInstancesResponse{
				Body:         []byte(`{"message":"discovery failed"}`),
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusInternalServerError, "500 Internal Server Error"),
			}, nil
		}
		camunda.cancelProcessInstanceWithResponse = func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
			cancellations++
			return &camundav810.CancelProcessInstanceResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusAccepted, "202 Accepted"),
			}, nil
		}

		resp, _, err := newTestService(t, waitTestConfig(), camunda).CancelProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrInternal)
		assert.Contains(t, err.Error(), "cancel family")
		assert.False(t, resp.Ok)
		assert.Equal(t, 1, cancellations)
		assert.Equal(t, 3, keyReads, "precheck and family traversal reads must finish before polling")
		assert.Equal(t, 1, discoveryReads)
	})

	t.Run("CancelAcceptsCompletedDescendant", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "ACTIVE", "ACTIVE", "CANCELED"},
			"456": {"COMPLETED"},
		}, map[string][]cancellationChild{
			"123": {{key: "456", state: "COMPLETED"}},
		})

		resp, _, err := fixture.service(t).CancelProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, 1, fixture.cancellationCount())
	})

	t.Run("CancelAcceptsNaturalCompletionDuringPolling", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "ACTIVE", "ACTIVE", "CANCELED"},
			"456": {"ACTIVE", "COMPLETED"},
		}, map[string][]cancellationChild{
			"123": {{key: "456", state: "ACTIVE"}},
		})

		resp, _, err := fixture.service(t).CancelProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, 2, fixture.readCount("456"))
	})

	t.Run("CancelAcceptsDisappearanceAfterDiscovery", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "ACTIVE", "ACTIVE", "TERMINATED"},
			"456": {""},
		}, map[string][]cancellationChild{
			"123": {{key: "456", state: "ACTIVE"}},
		})

		resp, _, err := fixture.service(t).CancelProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, 1, fixture.readCount("456"))
	})

	t.Run("CancelAcceptsMixedTerminalFamily", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "ACTIVE", "ACTIVE", "CANCELED"},
			"201": {"COMPLETED"},
			"202": {"CANCELED"},
			"203": {"TERMINATED"},
			"204": {""},
		}, map[string][]cancellationChild{
			"123": {
				{key: "201", state: "COMPLETED"},
				{key: "202", state: "CANCELED"},
				{key: "203", state: "TERMINATED"},
				{key: "204", state: "ACTIVE"},
			},
		})

		resp, _, err := fixture.service(t).CancelProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, 1, fixture.cancellationCount())
	})

	t.Run("CancelRootCompletionDoesNotHideActiveDescendant", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "ACTIVE", "ACTIVE", "COMPLETED"},
			"456": {"ACTIVE"},
		}, map[string][]cancellationChild{
			"123": {{key: "456", state: "ACTIVE"}},
		})

		resp, _, err := fixture.service(t).CancelProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeded max_retries")
		assert.False(t, resp.Ok)
		assert.Equal(t, 2, fixture.readCount("456"))
	})

	for _, state := range []string{"ACTIVE", "UNKNOWN"} {
		t.Run("Cancel"+state+"RemainsUnsatisfied", func(t *testing.T) {
			fixture := newCancellationFixture(t, map[string][]string{
				"123": {"ACTIVE", "ACTIVE", "ACTIVE", state},
			}, nil)

			resp, _, err := fixture.service(t).CancelProcessInstance(ctx, "123")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "exceeded max_retries")
			assert.False(t, resp.Ok)
			assert.Equal(t, 5, fixture.readCount("123"))
			assert.Equal(t, 1, fixture.cancellationCount())
		})
	}

	t.Run("CancelContextInterruptionRemainsAnError", func(t *testing.T) {
		waitCtx, cancel := context.WithCancel(ctx)
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE"},
		}, nil)
		fixture.cancelOnRead("123", 4, cancel)

		resp, _, err := fixture.service(t).CancelProcessInstance(waitCtx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.False(t, resp.Ok)
	})

	for _, state := range []string{"COMPLETED", "CANCELED", "TERMINATED"} {
		t.Run("CancelTerminalRootNoOp"+state, func(t *testing.T) {
			fixture := newCancellationFixture(t, map[string][]string{"123": {state}}, nil)

			resp, instances, err := fixture.service(t).CancelProcessInstance(ctx, "123")

			require.NoError(t, err)
			assert.Equal(t, d.CancelResponse{
				Ok:         true,
				StatusCode: http.StatusOK,
				Status:     fmt.Sprintf("process instance with key 123 is already in state %s, no need to cancel", state),
			}, resp)
			assert.Empty(t, instances)
			assert.Zero(t, fixture.cancellationCount())
		})
	}

	t.Run("CancelAbsentRootNoOp", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{"123": {""}}, nil)

		resp, instances, err := fixture.service(t).CancelProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.Equal(t, d.CancelResponse{
			Ok:         true,
			StatusCode: http.StatusOK,
			Status:     "process instance with key 123 is already in state ABSENT, no need to cancel",
		}, resp)
		assert.Empty(t, instances)
		assert.Zero(t, fixture.cancellationCount())
	})

	t.Run("CancelNoWaitSuppressesProcessInstanceDetailLogs", func(t *testing.T) {
		runCancelNoWaitLogTest := func(t *testing.T, suppress bool) string {
			t.Helper()
			var logBuf bytes.Buffer
			svc, err := v810.New(
				testConfig(),
				&http.Client{},
				slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo)),
				v810.WithClientCamunda(&mockCamundaClient{
					createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
					searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
					cancelProcessInstanceWithResponse: func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
						assert.Equal(t, "123", key)
						return &camundav810.CancelProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusAccepted, "202 Accepted"),
						}, nil
					},
					deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
					getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
				}),
			)
			require.NoError(t, err)

			opts := []services.CallOption{services.WithNoStateCheck(), services.WithNoWait()}
			if suppress {
				opts = append(opts, services.WithSuppressProcessInstanceDetailLogs())
			}
			_, _, err = svc.CancelProcessInstance(context.Background(), "123", opts...)

			require.NoError(t, err)
			return logBuf.String()
		}

		normalLog := runCancelNoWaitLogTest(t, false)
		suppressedLog := runCancelNoWaitLogTest(t, true)

		assert.Contains(t, normalLog, "INFO pi 123 cancel requested; no-wait")
		assert.NotContains(t, suppressedLog, "pi 123 cancel requested; no-wait")
	})

	t.Run("ForceCancelLogsKeyListOnlyWhenVerbose", func(t *testing.T) {
		runForceCancelLogTest := func(t *testing.T, verbose bool, suppress bool) string {
			t.Helper()
			var logBuf bytes.Buffer
			svc, err := v810.New(
				testConfig(),
				&http.Client{},
				slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)),
				v810.WithClientCamunda(&mockCamundaClient{
					createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
					getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
						parentKey := ""
						if key == camundav810.ProcessInstanceKey("124") {
							parentKey = "123"
						}
						return &camundav810.GetProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/"+string(key), http.StatusOK, "200 OK"),
							JSON200:      new(makeProcessInstanceResult(string(key), "ACTIVE", parentKey)),
						}, nil
					},
					searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
						payload := readBody(t, body)
						switch {
						case strings.Contains(payload, `"parentProcessInstanceKey":"123"`):
							return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
								Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("124", "ACTIVE", "123")},
								Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
							}), nil
						case strings.Contains(payload, `"parentProcessInstanceKey":"124"`):
							return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
								Items: []camundav810.ProcessInstanceResult{},
								Page:  camundav810.SearchQueryPageResponse{TotalItems: 0},
							}), nil
						default:
							t.Fatalf("unexpected search payload: %s", payload)
							return nil, nil
						}
					},
					cancelProcessInstanceWithResponse: func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
						assert.Equal(t, "123", key)
						return &camundav810.CancelProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusAccepted, "202 Accepted"),
						}, nil
					},
					deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
				}),
			)
			require.NoError(t, err)

			opts := []services.CallOption{services.WithForce(), services.WithNoWait()}
			if verbose {
				opts = append(opts, services.WithVerbose())
			}
			if suppress {
				opts = append(opts, services.WithSuppressProcessInstanceDetailLogs())
			}
			_, _, err = svc.CancelProcessInstance(context.Background(), "124", opts...)

			require.NoError(t, err)
			return logBuf.String()
		}

		quietLog := runForceCancelLogTest(t, false, false)
		verboseLog := runForceCancelLogTest(t, true, false)
		suppressedLog := runForceCancelLogTest(t, false, true)

		assert.Contains(t, quietLog, "force: cancelling 2 pi")
		assert.NotContains(t, quietLog, "with keys [123 124]")
		assert.Contains(t, verboseLog, "force: cancelling 2 pi; keys [123 124]")
		assert.NotContains(t, suppressedLog, "force: cancelling 2 pi")
	})

	t.Run("DeleteNoWait", func(t *testing.T) {
		var deleted []string
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				return &camundav810.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      new(makeProcessInstanceResult("123", "COMPLETED", "")),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: nil,
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 0},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
				deleted = append(deleted, key)
				return &camundav810.DeleteProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
				}, nil
			},
		})

		resp, err := svc.DeleteProcessInstance(ctx, "123", services.WithNoWait())

		require.NoError(t, err)
		assert.Equal(t, []string{"123"}, deleted)
		assert.True(t, resp.Ok)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DeleteWaitsForAbsentState", func(t *testing.T) {
		getCalls := 0
		svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				getCalls++
				if getCalls == 1 {
					return &camundav810.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("123", "COMPLETED", "")),
					}, nil
				}
				return &camundav810.GetProcessInstanceResponse{
					Body:         []byte(`{"message":"not found"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusNotFound, "404 Not Found"),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: nil,
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 0},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
				return &camundav810.DeleteProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
				}, nil
			},
		})

		resp, err := svc.DeleteProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, 2, getCalls)
	})

	t.Run("DeleteRejectedByApiReturnsErrorBeforeWait", func(t *testing.T) {
		getCalls := 0
		svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				getCalls++
				return &camundav810.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: nil,
					Page:  camundav810.SearchQueryPageResponse{TotalItems: 0},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
				return &camundav810.DeleteProcessInstanceResponse{
					Body:         []byte(`{"message":"cannot delete active instance"}`),
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", http.StatusBadRequest, "400 Bad Request"),
				}, nil
			},
		})

		_, err := svc.DeleteProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrBadRequest)
		assert.Equal(t, 1, getCalls)
	})

	for _, tc := range []struct {
		name          string
		recoveryState string
	}{
		{name: "ForceRecoveryAcceptsCompletedWithNoWait", recoveryState: "COMPLETED"},
		{name: "ForceRecoveryAcceptsAbsentWithNoWait", recoveryState: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newCancellationFixture(t, map[string][]string{
				"123": {"ACTIVE", tc.recoveryState},
			}, nil)
			fixture.deleteResponse = func(call int) *camundav810.DeleteProcessInstanceResponse {
				status := http.StatusOK
				if call == 1 {
					status = http.StatusConflict
				}
				return &camundav810.DeleteProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", status, http.StatusText(status)),
				}
			}

			resp, err := fixture.service(t).DeleteProcessInstance(ctx, "123", services.WithForce(), services.WithNoStateCheck(), services.WithNoWait())

			require.NoError(t, err)
			assert.True(t, resp.Ok)
			assert.Equal(t, 1, fixture.cancellationCount())
			assert.Equal(t, 2, fixture.deletionCount())
			assert.Equal(t, 2, fixture.readCount("123"))
		})
	}

	t.Run("ForceRecoveryRetainsFinalAbsenceVerification", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "ACTIVE", "ACTIVE", "CANCELED", "COMPLETED", ""},
		}, nil)
		fixture.deleteResponse = func(call int) *camundav810.DeleteProcessInstanceResponse {
			status := http.StatusOK
			if call == 1 {
				status = http.StatusConflict
			}
			return &camundav810.DeleteProcessInstanceResponse{
				HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", status, http.StatusText(status)),
			}
		}

		resp, err := fixture.service(t).DeleteProcessInstance(ctx, "123", services.WithForce(), services.WithNoStateCheck())

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, 2, fixture.deletionCount())
		assert.Equal(t, 6, fixture.readCount("123"))
	})

	t.Run("ForceRecoveryPreservesRetryDeleteFailure", func(t *testing.T) {
		fixture := newCancellationFixture(t, map[string][]string{
			"123": {"ACTIVE", "COMPLETED"},
		}, nil)
		fixture.deleteResponse = func(call int) *camundav810.DeleteProcessInstanceResponse {
			if call == 1 {
				return &camundav810.DeleteProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", http.StatusConflict, "409 Conflict"),
				}
			}
			return &camundav810.DeleteProcessInstanceResponse{
				Body:         []byte(`{"message":"forbidden"}`),
				HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", http.StatusForbidden, "403 Forbidden"),
			}
		}

		resp, err := fixture.service(t).DeleteProcessInstance(ctx, "123", services.WithForce(), services.WithNoStateCheck(), services.WithNoWait())

		require.Error(t, err)
		assert.False(t, resp.Ok)
		assert.Equal(t, 2, fixture.deletionCount())
		assert.Equal(t, 2, fixture.readCount("123"))
	})

	t.Run("DeleteWrongStateLogsOnlyWhenVerbose", func(t *testing.T) {
		runDeleteWrongStateLogTest := func(t *testing.T, verbose bool) string {
			t.Helper()
			var logBuf bytes.Buffer
			svc, err := v810.New(
				testConfig(),
				&http.Client{},
				slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)),
				v810.WithClientCamunda(&mockCamundaClient{
					createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
					getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
						return &camundav810.GetProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
							JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
						}, nil
					},
					searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
						return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
							Items: nil,
							Page:  camundav810.SearchQueryPageResponse{TotalItems: 0},
						}), nil
					},
					cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
					deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
						return &camundav810.DeleteProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/process-instances/123", http.StatusConflict, "409 Conflict"),
						}, nil
					},
				}),
			)
			require.NoError(t, err)

			var opts []services.CallOption
			if verbose {
				opts = append(opts, services.WithVerbose())
			}
			resp, err := svc.DeleteProcessInstance(context.Background(), "123", opts...)

			require.NoError(t, err)
			assert.Equal(t, http.StatusConflict, resp.StatusCode)
			return logBuf.String()
		}

		quietLog := runDeleteWrongStateLogTest(t, false)
		verboseLog := runDeleteWrongStateLogTest(t, true)

		assert.NotContains(t, quietLog, "pi 123 delete blocked; state not terminal")
		assert.Contains(t, verboseLog, "pi 123 delete blocked; state not terminal")
	})
}

// TestService_WaitForProcessInstanceExpectation verifies the real v8.10 service keeps explicit canceled matching strict.
func TestService_WaitForProcessInstanceExpectation(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		state   string
		wantOK  bool
		wantErr bool
		reads   int
	}{
		{state: "CANCELED", wantOK: true, reads: 1},
		{state: "TERMINATED", wantOK: true, reads: 1},
		{state: "COMPLETED", wantErr: true, reads: 2},
		{state: "", wantErr: true, reads: 2},
	} {
		name := tc.state
		if name == "" {
			name = "ABSENT"
		}
		t.Run(name, func(t *testing.T) {
			fixture := newCancellationFixture(t, map[string][]string{"123": {tc.state}}, nil)

			resp, pi, err := fixture.service(t).WaitForProcessInstanceExpectation(ctx, "123", d.ProcessInstanceExpectationRequest{States: d.States{d.StateCanceled}})

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "exceeded max_retries")
				assert.False(t, resp.Ok)
				assert.Empty(t, pi)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantOK, resp.Ok)
				assert.Equal(t, d.State(tc.state), resp.State)
				assert.Equal(t, d.State(tc.state), pi.State)
			}
			assert.Equal(t, tc.reads, fixture.readCount("123"))
		})
	}
}

func TestService_WithClientAndLoggerOptions(t *testing.T) {
	camundaClient := newStrictCamundaClient(t)
	svc, err := v810.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		v810.WithClientCamunda(camundaClient),
	)
	require.NoError(t, err)
	require.Equal(t, camundaClient, svc.ClientCamunda())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v810.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v810.WithClientCamunda(nil)(svc)
	v810.WithLogger(nil)(svc)
	require.Equal(t, camundaClient, svc.ClientCamunda())
	require.Equal(t, logger, svc.Logger())
}

func TestService_FinalV810BoundaryUsesVersionLocalCamundaContract(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, testConfig(), newStrictCamundaClient(t))

	require.Implements(t, (*v810.GenProcessInstanceClientCamunda)(nil), svc.ClientCamunda())
}

func TestService_TraversalResults(t *testing.T) {
	ctx := context.Background()

	t.Run("FamilyResultReturnsPartialTreeWhenParentIsMissing", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				switch {
				case strings.Contains(payload, `"parentProcessInstanceKey":"123"`):
					return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
						Items: []camundav810.ProcessInstanceResult{makeProcessInstanceResult("124", "ACTIVE", "123")},
						Page:  camundav810.SearchQueryPageResponse{TotalItems: 1},
					}), nil
				case strings.Contains(payload, `"parentProcessInstanceKey":"124"`):
					return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
						Items: []camundav810.ProcessInstanceResult{},
						Page:  camundav810.SearchQueryPageResponse{TotalItems: 0},
					}), nil
				default:
					t.Fatalf("unexpected search payload: %s", payload)
					return nil, nil
				}
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
				switch key {
				case camundav810.ProcessInstanceKey("123"):
					return &camundav810.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "999")),
					}, nil
				case camundav810.ProcessInstanceKey("124"):
					return &camundav810.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/124", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("124", "ACTIVE", "123")),
					}, nil
				case camundav810.ProcessInstanceKey("999"):
					return &camundav810.GetProcessInstanceResponse{
						Body:         []byte(`{"message":"not found"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/999", http.StatusNotFound, "404 Not Found"),
					}, nil
				default:
					t.Fatalf("unexpected key: %s", key)
					return nil, nil
				}
			},
		})

		result, err := svc.FamilyResult(ctx, "123")

		require.NoError(t, err)
		assert.Equal(t, []string{"123", "124"}, result.Keys)
		assert.Equal(t, "123", result.RootKey)
		assert.Equal(t, "partial", string(result.Outcome))
		assert.Equal(t, "one or more parent process instances were not found", result.Warning)
		require.Len(t, result.MissingAncestors, 1)
		assert.Equal(t, "999", result.MissingAncestors[0].Key)
		assert.Equal(t, []string{"124"}, result.Edges["123"])
	})
}

// cancellationChild describes one discovered family edge and its discovery-time state.
type cancellationChild struct {
	key   string
	state string
}

// cancellationFixture models family discovery and state observations independently for cancellation regressions.
type cancellationFixture struct {
	mu             sync.Mutex
	states         map[string][]string
	children       map[string][]cancellationChild
	reads          map[string]int
	discoveries    int
	cancellations  int
	deleteResponse func(call int) *camundav810.DeleteProcessInstanceResponse
	deletions      int
	cancelReadKey  string
	cancelReadAt   int
	cancelReadFunc context.CancelFunc
}

// newCancellationFixture creates a strict v8.10 backend fixture with deterministic per-key observations.
func newCancellationFixture(t *testing.T, states map[string][]string, children map[string][]cancellationChild) *cancellationFixture {
	t.Helper()
	return &cancellationFixture{
		states:   states,
		children: children,
		reads:    make(map[string]int),
	}
}

// service creates the real v8.10 service wired to a controlled Camunda client.
func (f *cancellationFixture) service(t *testing.T) *v810.Service {
	t.Helper()
	camunda := newStrictCamundaClient(t)
	camunda.cancelProcessInstanceWithResponse = func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
		f.mu.Lock()
		f.cancellations++
		f.mu.Unlock()
		return &camundav810.CancelProcessInstanceResponse{
			HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/"+key+"/cancellation", http.StatusAccepted, "202 Accepted"),
		}, nil
	}
	camunda.getProcessInstanceWithResponse = func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		keyString := string(key)
		states, ok := f.states[keyString]
		require.True(t, ok, "unexpected state lookup for key %s", key)
		read := f.reads[keyString]
		f.reads[keyString]++
		if keyString == f.cancelReadKey && f.reads[keyString] == f.cancelReadAt && f.cancelReadFunc != nil {
			f.cancelReadFunc()
		}
		state := states[min(read, len(states)-1)]
		if state == "" {
			return &camundav810.GetProcessInstanceResponse{
				Body:         []byte(`{"message":"not found"}`),
				HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/"+keyString, http.StatusNotFound, "404 Not Found"),
			}, nil
		}
		return &camundav810.GetProcessInstanceResponse{
			HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/"+keyString, http.StatusOK, "200 OK"),
			JSON200:      new(makeProcessInstanceResult(keyString, state, "")),
		}, nil
	}
	camunda.searchProcessInstancesWithResp = func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.discoveries++
		payload := readBody(t, body)
		var items []camundav810.ProcessInstanceResult
		for parentKey, children := range f.children {
			if !strings.Contains(payload, `"parentProcessInstanceKey":"`+parentKey+`"`) {
				continue
			}
			for _, child := range children {
				items = append(items, makeProcessInstanceResult(child.key, child.state, parentKey))
			}
			break
		}
		return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
			Items: items,
			Page:  camundav810.SearchQueryPageResponse{TotalItems: int64(len(items))},
		}), nil
	}
	if f.deleteResponse != nil {
		camunda.deleteProcessInstanceWithResponse = func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.deletions++
			return f.deleteResponse(f.deletions), nil
		}
	}
	return newTestService(t, waitTestConfig(), camunda)
}

// discoveryCount returns the synchronized child-discovery request count.
func (f *cancellationFixture) discoveryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.discoveries
}

// cancellationCount returns the synchronized cancellation request count.
func (f *cancellationFixture) cancellationCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cancellations
}

// deletionCount returns the synchronized deletion request count.
func (f *cancellationFixture) deletionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deletions
}

// readCount returns the synchronized state-read count for a process-instance key.
func (f *cancellationFixture) readCount(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads[key]
}

// cancelOnRead interrupts the supplied context at the selected state observation.
func (f *cancellationFixture) cancelOnRead(key string, read int, cancel context.CancelFunc) {
	f.cancelReadKey = key
	f.cancelReadAt = read
	f.cancelReadFunc = cancel
}

type searchProcessInstancesResult struct {
	Items []camundav810.ProcessInstanceResult `json:"items"`
	Page  camundav810.SearchQueryPageResponse `json:"page"`
}

func searchResponse(t *testing.T, statusCode int, payload searchProcessInstancesResult) *camundav810.SearchProcessInstancesResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return &camundav810.SearchProcessInstancesResponse{
		Body:         body,
		HTTPResponse: newHTTPResponseWithContentType(http.MethodPost, "https://camunda.local/v2/process-instances/search", statusCode, http.StatusText(statusCode), "application/json"),
		JSON200:      &camundav810.ProcessInstanceSearchQueryResult{Page: payload.Page},
	}
}

// newTestService creates a v8.10 process-instance service with a strict injected Camunda client.
func newTestService(t *testing.T, cfg *config.Config, camundaClient *mockCamundaClient) *v810.Service {
	t.Helper()

	svc, err := v810.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v810.WithClientCamunda(camundaClient),
	)
	require.NoError(t, err)
	return svc
}

// TestService_GetProcessInstanceStateByKeyLoggingScope keeps direct diagnostics while allowing waiter-owned suppression.
func TestService_GetProcessInstanceStateByKeyLoggingScope(t *testing.T) {
	var logBuf bytes.Buffer
	client := newStrictCamundaClient(t)
	client.getProcessInstanceWithResponse = func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
		return &camundav810.GetProcessInstanceResponse{
			HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
			JSON200:      new(makeProcessInstanceResult("123", "COMPLETED", "")),
		}, nil
	}
	svc, err := v810.New(
		testConfig(), &http.Client{}, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)),
		v810.WithClientCamunda(client),
	)
	require.NoError(t, err)

	_, _, err = svc.GetProcessInstanceStateByKey(context.Background(), "123")
	require.NoError(t, err)
	assert.Contains(t, logBuf.String(), "checking pi 123 state")
	assert.Contains(t, logBuf.String(), "fetching pi 123")
	assert.Contains(t, logBuf.String(), "pi 123 state COMPLETED")

	logBuf.Reset()
	_, _, err = svc.GetProcessInstanceStateByKey(context.Background(), "123", services.WithSuppressNestedProcessInstanceLookupLogs())
	require.NoError(t, err)
	assert.Empty(t, logBuf.String())
}

// newStrictCamundaClient returns a v8.10 Camunda client mock that fails on unexpected calls.
func newStrictCamundaClient(t *testing.T) *mockCamundaClient {
	t.Helper()
	return &mockCamundaClient{
		createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
		searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
		searchVariablesWithResponse:       unexpectedSearchVariables(t),
		createElementInstanceVariables:    unexpectedCreateElementInstanceVariables(t),
		cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
		getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
	}
}

func unexpectedCreateProcessInstance(t *testing.T) func(context.Context, camundav810.CreateProcessInstanceJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error) {
		t.Fatalf("unexpected create call")
		return nil, nil
	}
}

func unexpectedSearchProcessInstances(t *testing.T) func(context.Context, string, io.Reader, ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
	t.Helper()
	return func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error) {
		t.Fatalf("unexpected search call")
		return nil, nil
	}
}

func unexpectedSearchVariables(t *testing.T) func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
	t.Helper()
	return func(ctx context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
		t.Fatalf("unexpected variable search call")
		return nil, nil
	}
}

func unexpectedCreateElementInstanceVariables(t *testing.T) func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
	t.Helper()
	return func(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
		t.Fatalf("unexpected element-instance variable update call")
		return nil, nil
	}
}

func unexpectedCancelProcessInstance(t *testing.T) func(context.Context, string, camundav810.CancelProcessInstanceJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error) {
		t.Fatalf("unexpected cancellation call")
		return nil, nil
	}
}

func unexpectedDeleteProcessInstance(t *testing.T) func(context.Context, camundav810.ProcessInstanceKey, camundav810.DeleteProcessInstanceJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error) {
		t.Fatalf("unexpected delete call")
		return nil, nil
	}
}

func unexpectedGetProcessInstance(t *testing.T) func(context.Context, camundav810.ProcessInstanceKey, ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
		t.Fatalf("unexpected get call")
		return nil, nil
	}
}

func testConfig() *config.Config {
	return &config.Config{
		App: config.App{
			Tenant: "tenant",
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "https://camunda.local/v2",
			},
		},
	}
}

func waitTestConfig() *config.Config {
	cfg := testConfig()
	cfg.App.Backoff = config.BackoffConfig{
		Strategy:     config.BackoffFixed,
		InitialDelay: time.Millisecond,
		MaxRetries:   2,
		Timeout:      25 * time.Millisecond,
	}
	return cfg
}

func makeProcessInstanceResult(key string, state string, parentKey string) camundav810.ProcessInstanceResult {
	startDate := time.Date(2026, time.March, 23, 18, 0, 0, 0, time.UTC)
	item := camundav810.ProcessInstanceResult{
		HasIncident:                 false,
		ProcessDefinitionId:         "demo",
		ProcessDefinitionKey:        "9001",
		ProcessDefinitionName:       new("demo"),
		ProcessDefinitionVersion:    3,
		ProcessDefinitionVersionTag: new("stable"),
		ProcessInstanceKey:          key,
		StartDate:                   startDate,
		State:                       camundav810.ProcessInstanceStateEnum(state),
		TenantId:                    "tenant",
	}
	if parentKey != "" {
		item.ParentProcessInstanceKey = &parentKey
	}
	return item
}

func marshalJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func readBody(t *testing.T, body io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(b)
}

// newHTTPResponse builds a minimal HTTP response for v8.10 process-instance error handling tests.
func newHTTPResponse(method, rawURL string, statusCode int, status string) *http.Response {
	return newHTTPResponseWithContentType(method, rawURL, statusCode, status, "")
}

// newHTTPResponseWithContentType builds a v8.10 HTTP response with an explicit content type.
func newHTTPResponseWithContentType(method, rawURL string, statusCode int, status string, contentType string) *http.Response {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Header:     header,
		Request: &http.Request{
			Method: method,
			URL:    u,
		},
	}
}
