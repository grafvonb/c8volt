package v88_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	operatev88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCamundaClient struct {
	createProcessInstanceWithResponse func(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error)
	getProcessInstanceWithResponse    func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error)
	searchProcessInstancesWithResp    func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error)
	cancelProcessInstanceWithResponse func(ctx context.Context, key string, body camundav88.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstanceResponse, error)
}

func (m *mockCamundaClient) CreateProcessInstanceWithResponse(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error) {
	return m.createProcessInstanceWithResponse(ctx, body, reqEditors...)
}

func (m *mockCamundaClient) GetProcessInstanceWithResponse(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
	return m.getProcessInstanceWithResponse(ctx, key, reqEditors...)
}

func (m *mockCamundaClient) SearchProcessInstancesWithResponse(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
	return m.searchProcessInstancesWithResp(ctx, body, reqEditors...)
}

func (m *mockCamundaClient) CancelProcessInstanceWithResponse(ctx context.Context, key string, body camundav88.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstanceResponse, error) {
	return m.cancelProcessInstanceWithResponse(ctx, key, body, reqEditors...)
}

type mockOperateClient struct {
	deleteProcessInstanceAndAllDependantDataByKeyWithResp func(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error)
}

func (m *mockOperateClient) DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
	return m.deleteProcessInstanceAndAllDependantDataByKeyWithResp(ctx, key, reqEditors...)
}

func TestService_CreateProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessNoWait", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error) {
				payload := marshalJSON(t, body)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"processDefinitionVersion":7`)
				assert.Contains(t, payload, `"tenantId":"tenant-a"`)
				assert.Contains(t, payload, `"orderId":"42"`)
				return &camundav88.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav88.CreateProcessInstanceResult{
						ProcessDefinitionId:      "demo",
						ProcessDefinitionKey:     "proc-key",
						ProcessDefinitionVersion: 7,
						ProcessInstanceKey:       "123",
						TenantId:                 "tenant-a",
						Variables:                map[string]any{"orderId": "42"},
					},
				}, nil
			},
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

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

	t.Run("MalformedSuccessPayload", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error) {
				return &camundav88.CreateProcessInstanceResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
				}, nil
			},
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		_, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo"}, services.WithNoWait())

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})

	t.Run("SuccessWaitsForActiveState", func(t *testing.T) {
		getCalls := 0
		svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: func(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error) {
				return &camundav88.CreateProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					JSON200: &camundav88.CreateProcessInstanceResult{
						ProcessDefinitionId:      "demo",
						ProcessDefinitionKey:     "proc-key",
						ProcessDefinitionVersion: 7,
						ProcessInstanceKey:       "123",
						TenantId:                 "tenant-a",
					},
				}, nil
			},
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				getCalls++
				assert.Equal(t, "123", key)
				return &camundav88.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResult("123", "ACTIVE", ""),
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo", TenantId: "tenant-a"})

		require.NoError(t, err)
		assert.Equal(t, "123", creation.Key)
		assert.Equal(t, "2026-03-23T18:00:00Z", creation.StartDate)
		assert.NotEmpty(t, creation.StartConfirmedAt)
		assert.Equal(t, 1, getCalls)
	})
}

func TestService_GetProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				assert.Equal(t, "123", key)
				return &camundav88.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResult("123", "ACTIVE", ""),
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		pi, err := svc.GetProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.Equal(t, "123", pi.Key)
		assert.Equal(t, d.StateActive, pi.State)
		assert.Equal(t, "tenant", pi.TenantId)
	})

	t.Run("MalformedSuccessPayload", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				return &camundav88.GetProcessInstanceResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		_, err := svc.GetProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})
}

func TestService_SearchForProcessInstances(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				payload := marshalJSON(t, body)
				assert.Contains(t, payload, `"tenantId":"tenant"`)
				assert.Contains(t, payload, `"processDefinitionId":"demo"`)
				assert.Contains(t, payload, `"processDefinitionVersion":3`)
				assert.Contains(t, payload, `"processDefinitionVersionTag":"stable"`)
				assert.Contains(t, payload, `"state":"ACTIVE"`)
				assert.Contains(t, payload, `"parentProcessInstanceKey":"456"`)
				assert.Contains(t, payload, `"limit":25`)
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{*makeProcessInstanceResult("123", "ACTIVE", "456")},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			BpmnProcessId:     "demo",
			ProcessVersion:    3,
			ProcessVersionTag: "stable",
			State:             d.StateActive,
			ParentKey:         "456",
		}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
		assert.Equal(t, "456", items[0].ParentKey)
	})

	t.Run("MapsInclusiveStartDateBounds", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				require.NotNil(t, body.Filter.StartDate)

				startDate, err := body.Filter.StartDate.AsAdvancedDateTimeFilter()
				require.NoError(t, err)
				require.NotNil(t, startDate.Gte)
				require.NotNil(t, startDate.Lte)
				assert.Equal(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), *startDate.Gte)
				assert.Equal(t, time.Date(2026, time.January, 31, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC), *startDate.Lte)

				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{*makeProcessInstanceResult("123", "ACTIVE", "")},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			StartDateAfter:  "2026-01-01",
			StartDateBefore: "2026-01-31",
		}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
	})

	t.Run("MapsInclusiveEndDateBoundsAndRequiresExistingEndDate", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				require.NotNil(t, body.Filter.EndDate)

				endDate, err := body.Filter.EndDate.AsAdvancedDateTimeFilter()
				require.NoError(t, err)
				require.NotNil(t, endDate.Gte)
				require.NotNil(t, endDate.Lte)
				require.NotNil(t, endDate.Exists)
				assert.Equal(t, true, *endDate.Exists)
				assert.Equal(t, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), *endDate.Gte)
				assert.Equal(t, time.Date(2026, time.March, 31, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC), *endDate.Lte)
				assert.Nil(t, body.Filter.StartDate)

				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{*makeProcessInstanceResult("123", "COMPLETED", "")},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			EndDateAfter:  "2026-02-01",
			EndDateBefore: "2026-03-31",
		}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
	})

	t.Run("RequiresExistingEndDateForSingleDerivedUpperBound", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				require.NotNil(t, body.Filter.EndDate)

				endDate, err := body.Filter.EndDate.AsAdvancedDateTimeFilter()
				require.NoError(t, err)
				require.Nil(t, endDate.Gte)
				require.NotNil(t, endDate.Lte)
				require.NotNil(t, endDate.Exists)
				assert.Equal(t, true, *endDate.Exists)
				assert.Equal(t, time.Date(2026, time.April, 3, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC), *endDate.Lte)

				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{*makeProcessInstanceResult("123", "COMPLETED", "")},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			EndDateBefore: "2026-04-03",
		}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
	})

	t.Run("OmitsTenantFilterWhenConfigTenantIsEmpty", func(t *testing.T) {
		cfg := testConfig()
		cfg.App.Tenant = ""
		svc := newTestService(t, cfg, &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				payload := marshalJSON(t, body)
				assert.NotContains(t, payload, `"tenantId"`)
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{*makeProcessInstanceResult("123", "ACTIVE", "")},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
	})

	t.Run("MalformedSuccessPayload", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		_, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{}, 25)

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})
}

func TestService_SearchForProcessInstancesPage_UsesNativePageMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("reports has-more when total exceeds the visible window", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{
							*makeProcessInstanceResult("123", "ACTIVE", ""),
							*makeProcessInstanceResult("124", "ACTIVE", ""),
						},
						Page: camundav88.SearchQueryPageResponse{
							TotalItems: 3,
						},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 0, Size: 2})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateHasMore, page.OverflowState)
		require.Len(t, page.Items, 2)
	})

	t.Run("treats an exact boundary final page as no-more", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{
							*makeProcessInstanceResult("123", "ACTIVE", ""),
							*makeProcessInstanceResult("124", "ACTIVE", ""),
						},
						Page: camundav88.SearchQueryPageResponse{
							TotalItems: 2,
						},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 0, Size: 2})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateNoMore, page.OverflowState)
		require.Len(t, page.Items, 2)
	})

	t.Run("respects hasMoreTotalItems when totals are capped", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessInstanceSearchQueryResult{
						Items: []camundav88.ProcessInstanceResult{
							*makeProcessInstanceResult("123", "ACTIVE", ""),
							*makeProcessInstanceResult("124", "ACTIVE", ""),
						},
						Page: camundav88.SearchQueryPageResponse{
							TotalItems:        2,
							HasMoreTotalItems: true,
						},
					},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 0, Size: 2})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateHasMore, page.OverflowState)
		require.Len(t, page.Items, 2)
	})
}

func TestService_GetProcessInstanceStateByKey(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				return &camundav88.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResult("123", "COMPLETED", ""),
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		state, pi, err := svc.GetProcessInstanceStateByKey(ctx, "123")

		require.NoError(t, err)
		assert.Equal(t, d.StateCompleted, state)
		assert.Equal(t, "123", pi.Key)
	})

	t.Run("MalformedSuccessPayload", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				return &camundav88.GetProcessInstanceResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
				}, nil
			},
			searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, newStrictOperateClient(t))

		_, _, err := svc.GetProcessInstanceStateByKey(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
		assert.Contains(t, err.Error(), "fetching process instance with key 123")
	})
}

func TestService_CancelProcessInstance(t *testing.T) {
	ctx := context.Background()
	var cancelled string

	svc := newTestService(t, testConfig(), &mockCamundaClient{
		createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
		getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
		cancelProcessInstanceWithResponse: func(ctx context.Context, key string, body camundav88.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstanceResponse, error) {
			cancelled = key
			return &camundav88.CancelProcessInstanceResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/123/cancellation", http.StatusAccepted, "202 Accepted"),
			}, nil
		},
	}, newStrictOperateClient(t))

	resp, instances, err := svc.CancelProcessInstance(ctx, "123", services.WithNoStateCheck(), services.WithNoWait())

	require.NoError(t, err)
	assert.Equal(t, "123", cancelled)
	assert.True(t, resp.Ok)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Empty(t, instances)
}

func TestService_DeleteProcessInstance(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessNoWait", func(t *testing.T) {
		var deletedKeys []int64
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				return &camundav88.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResult("123", "COMPLETED", ""),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200:      &camundav88.ProcessInstanceSearchQueryResult{},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, &mockOperateClient{
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				deletedKeys = append(deletedKeys, key)
				return &operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      &operatev88.ChangeStatus{},
				}, nil
			},
		})

		resp, err := svc.DeleteProcessInstance(ctx, "123", services.WithNoWait())

		require.NoError(t, err)
		assert.Equal(t, []int64{123}, deletedKeys)
		assert.True(t, resp.Ok)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("WrongStateWithoutForceReturnsConflict", func(t *testing.T) {
		svc := newTestService(t, testConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				return &camundav88.GetProcessInstanceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResult("123", "ACTIVE", ""),
				}, nil
			},
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200:      &camundav88.ProcessInstanceSearchQueryResult{},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, &mockOperateClient{
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				return &operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://operate.local/process-instances/123", http.StatusBadRequest, "400 Bad Request"),
					ApplicationproblemJSON400: &operatev88.Error{
						Message: new(wrongStateMessage()),
					},
				}, nil
			},
		})

		resp, err := svc.DeleteProcessInstance(ctx, "123", services.WithNoWait())

		require.NoError(t, err)
		assert.False(t, resp.Ok)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("SuccessWaitsForAbsentState", func(t *testing.T) {
		getCalls := 0
		svc := newTestService(t, waitTestConfig(), &mockCamundaClient{
			createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
			getProcessInstanceWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
				getCalls++
				switch getCalls {
				case 1:
					return &camundav88.GetProcessInstanceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      makeProcessInstanceResult("123", "COMPLETED", ""),
					}, nil
				case 2:
					return nil, d.ErrNotFound
				default:
					t.Fatalf("unexpected get call #%d", getCalls)
					return nil, nil
				}
			},
			searchProcessInstancesWithResp: func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
				return &camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances/search", http.StatusOK, "200 OK"),
					JSON200:      &camundav88.ProcessInstanceSearchQueryResult{},
				}, nil
			},
			cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
		}, &mockOperateClient{
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				return &operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      &operatev88.ChangeStatus{},
				}, nil
			},
		})

		resp, err := svc.DeleteProcessInstance(ctx, "123")

		require.NoError(t, err)
		assert.True(t, resp.Ok)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, getCalls)
	})
}

func TestService_WithClientAndLoggerOptions(t *testing.T) {
	camundaClient := newStrictCamundaClient(t)
	operateClient := newStrictOperateClient(t)
	svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		v88.WithClientCamunda(camundaClient),
		v88.WithClientOperate(operateClient),
	)
	require.NoError(t, err)
	require.Equal(t, camundaClient, svc.ClientCamunda())
	require.Equal(t, operateClient, svc.ClientOperate())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v88.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v88.WithClientCamunda(nil)(svc)
	v88.WithClientOperate(nil)(svc)
	v88.WithLogger(nil)(svc)
	require.Equal(t, camundaClient, svc.ClientCamunda())
	require.Equal(t, operateClient, svc.ClientOperate())
	require.Equal(t, logger, svc.Logger())
}

func newTestService(t *testing.T, cfg *config.Config, camundaClient *mockCamundaClient, operateClient *mockOperateClient) *v88.Service {
	t.Helper()

	svc, err := v88.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v88.WithClientCamunda(camundaClient),
		v88.WithClientOperate(operateClient),
	)
	require.NoError(t, err)
	return svc
}

func newStrictCamundaClient(t *testing.T) *mockCamundaClient {
	t.Helper()
	return &mockCamundaClient{
		createProcessInstanceWithResponse: unexpectedCreateProcessInstance(t),
		getProcessInstanceWithResponse:    unexpectedGetProcessInstance(t),
		searchProcessInstancesWithResp:    unexpectedSearchProcessInstances(t),
		cancelProcessInstanceWithResponse: unexpectedCancelProcessInstance(t),
	}
}

func newStrictOperateClient(t *testing.T) *mockOperateClient {
	t.Helper()
	return &mockOperateClient{
		deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
			t.Fatalf("unexpected delete call")
			return nil, nil
		},
	}
}

func unexpectedCreateProcessInstance(t *testing.T) func(context.Context, camundav88.CreateProcessInstanceJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error) {
		t.Fatalf("unexpected create call")
		return nil, nil
	}
}

func unexpectedGetProcessInstance(t *testing.T) func(context.Context, string, ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error) {
		t.Fatalf("unexpected get call")
		return nil, nil
	}
}

func unexpectedSearchProcessInstances(t *testing.T) func(context.Context, camundav88.SearchProcessInstancesJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
	t.Helper()
	return func(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
		t.Fatalf("unexpected search call")
		return nil, nil
	}
}

func unexpectedCancelProcessInstance(t *testing.T) func(context.Context, string, camundav88.CancelProcessInstanceJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstanceResponse, error) {
	t.Helper()
	return func(ctx context.Context, key string, body camundav88.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstanceResponse, error) {
		t.Fatalf("unexpected cancellation call")
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
			Operate: config.API{
				BaseURL: "https://operate.local",
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

func makeProcessInstanceResult(key string, state string, parentKey string) *camundav88.ProcessInstanceResult {
	startDate := time.Date(2026, time.March, 23, 18, 0, 0, 0, time.UTC)
	item := &camundav88.ProcessInstanceResult{
		HasIncident:              false,
		ProcessDefinitionId:      "demo",
		ProcessDefinitionKey:     "9001",
		ProcessDefinitionName:    ptr("demo"),
		ProcessDefinitionVersion: 3,
		ProcessDefinitionVersionTag: ptr("stable"),
		ProcessInstanceKey: key,
		StartDate:          startDate,
		State:              camundav88.ProcessInstanceStateEnum(state),
		TenantId:           "tenant",
	}
	if parentKey != "" {
		item.ParentProcessInstanceKey = &parentKey
	}
	return item
}

func wrongStateMessage() string {
	return "Process instances needs to be in one of the states [COMPLETED, CANCELED]"
}

func ptr[T any](v T) *T {
	return &v
}

func marshalJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func newHTTPResponse(method, rawURL string, statusCode int, status string) *http.Response {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Request: &http.Request{
			Method: method,
			URL:    u,
		},
	}
}
