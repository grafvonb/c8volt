package v89_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v89 "github.com/grafvonb/c8volt/internal/services/processinstance/v89"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCamundaClient struct {
	createProcessInstanceWithResponse func(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error)
	searchProcessInstancesWithResp    func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error)
	cancelProcessInstanceWithResponse func(ctx context.Context, key string, body camundav89.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error)
	deleteProcessInstanceWithResponse func(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error)
	getProcessInstanceWithResponse    func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error)
}

var _ v89.GenProcessInstanceClientCamunda = (*mockCamundaClient)(nil)

func (m *mockCamundaClient) CreateProcessInstanceWithResponse(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error) {
	return m.createProcessInstanceWithResponse(ctx, body, reqEditors...)
}

func (m *mockCamundaClient) SearchProcessInstancesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
	return m.searchProcessInstancesWithResp(ctx, contentType, body, reqEditors...)
}

func (m *mockCamundaClient) CancelProcessInstanceWithResponse(ctx context.Context, key string, body camundav89.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error) {
	return m.cancelProcessInstanceWithResponse(ctx, key, body, reqEditors...)
}

func (m *mockCamundaClient) DeleteProcessInstanceWithResponse(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
	return m.deleteProcessInstanceWithResponse(ctx, key, body, reqEditors...)
}

func (m *mockCamundaClient) GetProcessInstanceWithResponse(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
	return m.getProcessInstanceWithResponse(ctx, key, reqEditors...)
}

func TestService_CreateProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessNoWait", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error) {
				payload := marshalJSON(t, body)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"processDefinitionVersion":7`)
				assert.Contains(t, payload, `"tenantId":"tenant-a"`)
				assert.Contains(t, payload, `"orderId":"42"`)
				return &camundav89.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav89.CreateProcessInstanceResult{
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
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error) {
				payload := marshalJSON(t, body)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"tenantId":"\u003cdefault\u003e"`)
				return &camundav89.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav89.CreateProcessInstanceResult{
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

	t.Run("SuccessWaitsForActiveState", func(t *testing.T) {
		getCalls := 0
		svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error) {
				return &camundav89.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav89.CreateProcessInstanceResult{
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
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				getCalls++
				assert.Equal(t, camundav89.ProcessInstanceKey("123"), key)
				return &camundav89.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
				}, nil
			},
		})

		creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo", TenantId: "tenant-a"})

		require.NoError(t, err)
		assert.Equal(t, "123", creation.Key)
		assert.Equal(t, "2026-03-23T18:00:00Z", creation.StartDate)
		assert.NotEmpty(t, creation.StartConfirmedAt)
		assert.Equal(t, 1, getCalls)
	})
}

func TestService_SearchAndLookup(t *testing.T) {
	ctx := context.Background()

	t.Run("SearchUsesTenantSafeBodyAndPageMetadata", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"tenantId":"tenant"`)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"processDefinitionVersion":3`)
				assert.Contains(t, payload, `"processDefinitionVersionTag":"stable"`)
				assert.Contains(t, payload, `"state":"ACTIVE"`)
				assert.Contains(t, payload, `"parentProcessInstanceKey":"456"`)
				assert.Contains(t, payload, `"limit":25`)
				assert.Contains(t, payload, `"$exists":true`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav89.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "456")},
					Page:  camundav89.SearchQueryPageResponse{TotalItems: 2, HasMoreTotalItems: true},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{
			BpmnProcessId:     "demo",
			ProcessVersion:    3,
			ProcessVersionTag: "stable",
			State:             d.StateActive,
			ParentKey:         "456",
			EndDateBefore:     "2026-04-03",
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

	t.Run("MapsCanceledSearchStateToTerminated", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"state":"TERMINATED"`)
				assert.NotContains(t, payload, `"state":"CANCELED"`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav89.ProcessInstanceResult{makeProcessInstanceResult("123", "CANCELED", "")},
					Page:  camundav89.SearchQueryPageResponse{TotalItems: 1},
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
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				assert.Equal(t, camundav89.ProcessInstanceKey("123"), key)
				return &camundav89.GetProcessInstanceResponse{
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
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				return &camundav89.GetProcessInstanceResponse{
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
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				assert.Contains(t, payload, `"hasIncident":true`)
				assert.Contains(t, payload, `"parentProcessInstanceKey":{"$exists":true}`)
				assert.NotContains(t, payload, `"parentProcessInstanceKey":"`)
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: []camundav89.ProcessInstanceResult{makeProcessInstanceResult("123", "ACTIVE", "456")},
					Page:  camundav89.SearchQueryPageResponse{TotalItems: 1},
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

func TestService_CancelAndDeleteProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("CancelNoWait", func(t *testing.T) {
		var cancelled string
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: func(ctx context.Context, key string, body camundav89.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error) {
				cancelled = key
				return &camundav89.CancelProcessInstanceResponse{
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

	t.Run("ForceCancelLogsKeyListOnlyWhenVerbose", func(t *testing.T) {
		runForceCancelLogTest := func(t *testing.T, verbose bool) string {
			t.Helper()
			var logBuf bytes.Buffer
			svc, err := v89.New(
				testConfig(),
				&http.Client{},
				slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)),
				v89.WithClientCamunda(&mockCamundaClient{
					createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
					getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
						parentKey := ""
						if key == camundav89.ProcessInstanceKey("124") {
							parentKey = "123"
						}
						return &camundav89.GetProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/"+string(key), http.StatusOK, "200 OK"),
							JSON200:      new(makeProcessInstanceResult(string(key), "ACTIVE", parentKey)),
						}, nil
					},
					searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
						payload := readBody(t, body)
						switch {
						case strings.Contains(payload, `"parentProcessInstanceKey":"123"`):
							return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
								Items: []camundav89.ProcessInstanceResult{makeProcessInstanceResult("124", "ACTIVE", "123")},
								Page:  camundav89.SearchQueryPageResponse{TotalItems: 1},
							}), nil
						case strings.Contains(payload, `"parentProcessInstanceKey":"124"`):
							return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
								Items: []camundav89.ProcessInstanceResult{},
								Page:  camundav89.SearchQueryPageResponse{TotalItems: 0},
							}), nil
						default:
							t.Fatalf("unexpected search payload: %s", payload)
							return nil, nil
						}
					},
					cancelProcessInstanceWithResponse: func(ctx context.Context, key string, body camundav89.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error) {
						assert.Equal(t, "123", key)
						return &camundav89.CancelProcessInstanceResponse{
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
			_, _, err = svc.CancelProcessInstance(context.Background(), "124", opts...)

			require.NoError(t, err)
			return logBuf.String()
		}

		quietLog := runForceCancelLogTest(t, false)
		verboseLog := runForceCancelLogTest(t, true)

		assert.Contains(t, quietLog, "force flag is set, cancelling 2 process instances")
		assert.NotContains(t, quietLog, "with keys [123 124]")
		assert.Contains(t, verboseLog, "force flag is set, cancelling 2 process instances with keys [123 124]")
	})

	t.Run("DeleteNoWait", func(t *testing.T) {
		var deleted []string
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				return &camundav89.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      new(makeProcessInstanceResult("123", "COMPLETED", "")),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: nil,
					Page:  camundav89.SearchQueryPageResponse{TotalItems: 0},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
				deleted = append(deleted, key)
				return &camundav89.DeleteProcessInstanceResponse{
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
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				getCalls++
				if getCalls == 1 {
					return &camundav89.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("123", "COMPLETED", "")),
					}, nil
				}
				return &camundav89.GetProcessInstanceResponse{
					Body:         []byte(`{"message":"not found"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusNotFound, "404 Not Found"),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: nil,
					Page:  camundav89.SearchQueryPageResponse{TotalItems: 0},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
				return &camundav89.DeleteProcessInstanceResponse{
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
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				getCalls++
				return &camundav89.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
					Items: nil,
					Page:  camundav89.SearchQueryPageResponse{TotalItems: 0},
				}), nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
				return &camundav89.DeleteProcessInstanceResponse{
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

	t.Run("DeleteWrongStateLogsOnlyWhenVerbose", func(t *testing.T) {
		runDeleteWrongStateLogTest := func(t *testing.T, verbose bool) string {
			t.Helper()
			var logBuf bytes.Buffer
			svc, err := v89.New(
				testConfig(),
				&http.Client{},
				slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)),
				v89.WithClientCamunda(&mockCamundaClient{
					createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
					getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
						return &camundav89.GetProcessInstanceResponse{
							HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
							JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "")),
						}, nil
					},
					searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
						return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
							Items: nil,
							Page:  camundav89.SearchQueryPageResponse{TotalItems: 0},
						}), nil
					},
					cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
					deleteProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
						return &camundav89.DeleteProcessInstanceResponse{
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

		assert.NotContains(t, quietLog, "cannot delete, process instance 123 is not in one of terminated states")
		assert.Contains(t, verboseLog, "cannot delete, process instance 123 is not in one of terminated states")
	})
}

func TestService_WithClientAndLoggerOptions(t *testing.T) {
	camundaClient := newStrictCamundaClient(t)
	svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		v89.WithClientCamunda(camundaClient),
	)
	require.NoError(t, err)
	require.Equal(t, camundaClient, svc.ClientCamunda())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v89.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v89.WithClientCamunda(nil)(svc)
	v89.WithLogger(nil)(svc)
	require.Equal(t, camundaClient, svc.ClientCamunda())
	require.Equal(t, logger, svc.Logger())
}

func TestService_FinalV89BoundaryUsesVersionLocalCamundaContract(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, testConfig(), newStrictCamundaClient(t))

	require.Implements(t, (*v89.GenProcessInstanceClientCamunda)(nil), svc.ClientCamunda())
}

func TestService_TraversalResults(t *testing.T) {
	ctx := context.Background()

	t.Run("FamilyResultReturnsPartialTreeWhenParentIsMissing", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
				payload := readBody(t, body)
				switch {
				case strings.Contains(payload, `"parentProcessInstanceKey":"123"`):
					return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
						Items: []camundav89.ProcessInstanceResult{makeProcessInstanceResult("124", "ACTIVE", "123")},
						Page:  camundav89.SearchQueryPageResponse{TotalItems: 1},
					}), nil
				case strings.Contains(payload, `"parentProcessInstanceKey":"124"`):
					return searchResponse(t, http.StatusOK, searchProcessInstancesResult{
						Items: []camundav89.ProcessInstanceResult{},
						Page:  camundav89.SearchQueryPageResponse{TotalItems: 0},
					}), nil
				default:
					t.Fatalf("unexpected search payload: %s", payload)
					return nil, nil
				}
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
			deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
				switch key {
				case camundav89.ProcessInstanceKey("123"):
					return &camundav89.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("123", "ACTIVE", "999")),
					}, nil
				case camundav89.ProcessInstanceKey("124"):
					return &camundav89.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/124", http.StatusOK, "200 OK"),
						JSON200:      new(makeProcessInstanceResult("124", "ACTIVE", "123")),
					}, nil
				case camundav89.ProcessInstanceKey("999"):
					return &camundav89.GetProcessInstanceResponse{
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

type searchProcessInstancesResult struct {
	Items []camundav89.ProcessInstanceResult `json:"items"`
	Page  camundav89.SearchQueryPageResponse `json:"page"`
}

func searchResponse(t *testing.T, statusCode int, payload searchProcessInstancesResult) *camundav89.SearchProcessInstancesResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return &camundav89.SearchProcessInstancesResponse{
		Body:         body,
		HTTPResponse: newHTTPResponseWithContentType(http.MethodPost, "https://camunda.local/v2/process-instances/search", statusCode, http.StatusText(statusCode), "application/json"),
		JSON200:      &camundav89.ProcessInstanceSearchQueryResult{Page: payload.Page},
	}
}

func newTestService(t *testing.T, cfg *config.Config, camundaClient *mockCamundaClient) *v89.Service {
	t.Helper()

	svc, err := v89.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v89.WithClientCamunda(camundaClient),
	)
	require.NoError(t, err)
	return svc
}

func newStrictCamundaClient(t *testing.T) *mockCamundaClient {
	t.Helper()
	return &mockCamundaClient{
		createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
		searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
		cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		deleteProcessInstanceWithResponse: unexpectedDeleteProcessInstance(t),
		getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
	}
}

func unexpectedCreateProcessInstance(t *testing.T) func(context.Context, camundav89.CreateProcessInstanceJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error) {
		t.Fatalf("unexpected create call")
		return nil, nil
	}
}

func unexpectedSearchProcessInstances(t *testing.T) func(context.Context, string, io.Reader, ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
	t.Helper()
	return func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
		t.Fatalf("unexpected search call")
		return nil, nil
	}
}

func unexpectedCancelProcessInstance(t *testing.T) func(context.Context, string, camundav89.CancelProcessInstanceJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key string, body camundav89.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error) {
		t.Fatalf("unexpected cancellation call")
		return nil, nil
	}
}

func unexpectedDeleteProcessInstance(t *testing.T) func(context.Context, camundav89.ProcessInstanceKey, camundav89.DeleteProcessInstanceJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error) {
		t.Fatalf("unexpected delete call")
		return nil, nil
	}
}

func unexpectedGetProcessInstance(t *testing.T) func(context.Context, camundav89.ProcessInstanceKey, ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error) {
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

func makeProcessInstanceResult(key string, state string, parentKey string) camundav89.ProcessInstanceResult {
	startDate := time.Date(2026, time.March, 23, 18, 0, 0, 0, time.UTC)
	item := camundav89.ProcessInstanceResult{
		HasIncident:                 false,
		ProcessDefinitionId:         "demo",
		ProcessDefinitionKey:        "9001",
		ProcessDefinitionName:       new("demo"),
		ProcessDefinitionVersion:    3,
		ProcessDefinitionVersionTag: new("stable"),
		ProcessInstanceKey:          key,
		StartDate:                   startDate,
		State:                       camundav89.ProcessInstanceStateEnum(state),
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

func newHTTPResponse(method, rawURL string, statusCode int, status string) *http.Response {
	return newHTTPResponseWithContentType(method, rawURL, statusCode, status, "")
}

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
