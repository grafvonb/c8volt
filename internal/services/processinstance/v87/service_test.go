package v87_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/processinstance/v87"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCamundaClient struct {
	postProcessInstancesWithResponse                               func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error)
	postProcessInstancesProcessInstanceKeyCancellationWithResponse func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error)
}

func (m *mockCamundaClient) PostProcessInstancesWithResponse(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
	return m.postProcessInstancesWithResponse(ctx, body, reqEditors...)
}

func (m *mockCamundaClient) PostProcessInstancesProcessInstanceKeyCancellationWithResponse(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
	return m.postProcessInstancesProcessInstanceKeyCancellationWithResponse(ctx, processInstanceKey, body, reqEditors...)
}

type mockOperateClient struct {
	getProcessInstanceByKeyWithResponse                   func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error)
	searchProcessInstancesWithResponse                    func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error)
	deleteProcessInstanceAndAllDependantDataByKeyWithResp func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error)
}

func (m *mockOperateClient) GetProcessInstanceByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
	return m.getProcessInstanceByKeyWithResponse(ctx, key, reqEditors...)
}

func (m *mockOperateClient) SearchProcessInstancesWithResponse(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
	return m.searchProcessInstancesWithResponse(ctx, body, reqEditors...)
}

func (m *mockOperateClient) DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
	return m.deleteProcessInstanceAndAllDependantDataByKeyWithResp(ctx, key, reqEditors...)
}

func TestService_CreateProcessInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		camunda       *mockCamundaClient
		expectedError error
		assertResult  func(*testing.T, d.ProcessInstanceCreation)
	}{
		{
			name: "SuccessNoWait",
			camunda: &mockCamundaClient{
				postProcessInstancesWithResponse: func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
					require.NotNil(t, body.ProcessDefinitionId)
					assert.Equal(t, "demo", *body.ProcessDefinitionId)
					require.NotNil(t, body.ProcessDefinitionVersion)
					assert.Equal(t, int32(7), *body.ProcessDefinitionVersion)
					require.NotNil(t, body.TenantId)
					assert.Equal(t, "tenant-a", *body.TenantId)
					require.NotNil(t, body.Variables)
					assert.Equal(t, map[string]any{"orderId": "42"}, *body.Variables)
					return &camundav87.PostProcessInstancesResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
						JSON200: &camundav87.CreateProcessInstanceResult{
							ProcessDefinitionId:      toolx.Ptr("demo"),
							ProcessDefinitionVersion: toolx.Ptr(int32(7)),
							TenantId:                 toolx.Ptr("tenant-a"),
							Variables:                &map[string]interface{}{"orderId": "42"},
						},
					}, nil
				},
				postProcessInstancesProcessInstanceKeyCancellationWithResponse: func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
					t.Fatalf("unexpected cancellation call")
					return nil, nil
				},
			},
			assertResult: func(t *testing.T, creation d.ProcessInstanceCreation) {
				assert.Equal(t, "demo", creation.BpmnProcessId)
				assert.Equal(t, int32(7), creation.ProcessDefinitionVersion)
				assert.Equal(t, "tenant-a", creation.TenantId)
				assert.Equal(t, "42", creation.Variables["orderId"])
				assert.NotEmpty(t, creation.StartDate)
			},
		},
		{
			name: "MalformedSuccessPayload",
			camunda: &mockCamundaClient{
				postProcessInstancesWithResponse: func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
					return &camundav87.PostProcessInstancesResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
					}, nil
				},
				postProcessInstancesProcessInstanceKeyCancellationWithResponse: func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
					t.Fatalf("unexpected cancellation call")
					return nil, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, testConfig(), tt.camunda, newStrictOperateClient(t))

			creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{
				BpmnProcessId:            "demo",
				ProcessDefinitionVersion: 7,
				TenantId:                 "tenant-a",
				Variables:                map[string]any{"orderId": "42"},
			}, services.WithNoWait())

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			tt.assertResult(t, creation)
		})
	}
}

func TestService_GetProcessInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		key               string
		operate           *mockOperateClient
		expectedError     error
		expectedErrSubstr string
		assertResult      func(*testing.T, d.ProcessInstance)
	}{
		{
			name: "Success",
			key:  "123",
			operate: &mockOperateClient{
				getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
					assert.Equal(t, int64(123), key)
					return &operatev87.GetProcessInstanceByKeyResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      makeProcessInstanceResponse(123, "ACTIVE", ""),
					}, nil
				},
				searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
					t.Fatalf("unexpected search call")
					return nil, nil
				},
				deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
			},
			assertResult: func(t *testing.T, pi d.ProcessInstance) {
				assert.Equal(t, "123", pi.Key)
				assert.Equal(t, d.StateActive, pi.State)
				assert.Equal(t, "tenant", pi.TenantId)
			},
		},
		{
			name:              "KeyConversionError",
			key:               "not-a-number",
			operate:           newStrictOperateClient(t),
			expectedErrSubstr: "converting process instance key",
		},
		{
			name: "MalformedSuccessPayload",
			key:  "123",
			operate: &mockOperateClient{
				getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
					return &operatev87.GetProcessInstanceByKeyResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					}, nil
				},
				searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
					t.Fatalf("unexpected search call")
					return nil, nil
				},
				deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, testConfig(), newStrictCamundaClient(t), tt.operate)

			pi, err := svc.GetProcessInstance(ctx, tt.key)

			if tt.expectedErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrSubstr)
				return
			}
			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			tt.assertResult(t, pi)
		})
	}
}

func TestService_SearchForProcessInstances(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				require.NotNil(t, body.Size)
				assert.Equal(t, int32(25), *body.Size)
				assert.Equal(t, "tenant", *body.Filter.TenantId)
				assert.Equal(t, "demo", *body.Filter.BpmnProcessId)
				assert.Equal(t, int32(3), *body.Filter.ProcessVersion)
				assert.Equal(t, "stable", *body.Filter.ProcessVersionTag)
				assert.Equal(t, operatev87.ProcessInstanceState("ACTIVE"), *body.Filter.State)
				assert.Equal(t, int64(456), *body.Filter.ParentKey)
				items := []operatev87.ProcessInstance{*makeProcessInstanceResponse(123, "ACTIVE", "456")}
				return &operatev87.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &operatev87.ResultsProcessInstance{
						Items: &items,
					},
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

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

	t.Run("ParentKeyConversionError", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), newStrictOperateClient(t))

		_, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{ParentKey: "abc"}, 25)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing parent key")
	})

	t.Run("MalformedSuccessPayload", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				return &operatev87.SearchProcessInstancesResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

		_, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{}, 25)

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})
}

func TestService_GetProcessInstanceStateByKey(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		key               string
		operate           *mockOperateClient
		expectedError     error
		expectedErrSubstr string
		assertResult      func(*testing.T, d.State, d.ProcessInstance)
	}{
		{
			name: "Success",
			key:  "123",
			operate: &mockOperateClient{
				getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
					return &operatev87.GetProcessInstanceByKeyResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      makeProcessInstanceResponse(123, "COMPLETED", ""),
					}, nil
				},
				searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
					t.Fatalf("unexpected search call")
					return nil, nil
				},
				deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
			},
			assertResult: func(t *testing.T, state d.State, pi d.ProcessInstance) {
				assert.Equal(t, d.StateCompleted, state)
				assert.Equal(t, "123", pi.Key)
			},
		},
		{
			name:              "KeyConversionError",
			key:               "invalid",
			operate:           newStrictOperateClient(t),
			expectedErrSubstr: "converting process instance key",
		},
		{
			name: "MalformedSuccessPayload",
			key:  "123",
			operate: &mockOperateClient{
				getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
					return &operatev87.GetProcessInstanceByKeyResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					}, nil
				},
				searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
					t.Fatalf("unexpected search call")
					return nil, nil
				},
				deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, testConfig(), newStrictCamundaClient(t), tt.operate)

			state, pi, err := svc.GetProcessInstanceStateByKey(ctx, tt.key)

			if tt.expectedErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrSubstr)
				return
			}
			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Contains(t, err.Error(), "fetching process instance with key")
				return
			}

			require.NoError(t, err)
			tt.assertResult(t, state, pi)
		})
	}
}

func TestService_CancelProcessInstance(t *testing.T) {
	ctx := context.Background()
	var cancelled string

	svc := newTestService(t, testConfig(), &mockCamundaClient{
		postProcessInstancesWithResponse: func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
			t.Fatalf("unexpected create call")
			return nil, nil
		},
		postProcessInstancesProcessInstanceKeyCancellationWithResponse: func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
			cancelled = processInstanceKey
			return &camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse{
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
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				assert.Equal(t, int64(123), key)
				return &operatev87.GetProcessInstanceByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResponse(123, "COMPLETED", ""),
				}, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				items := []operatev87.ProcessInstance{}
				return &operatev87.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &operatev87.ResultsProcessInstance{
						Items: &items,
					},
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				deletedKeys = append(deletedKeys, key)
				return &operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      &operatev87.ChangeStatus{},
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
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				assert.Equal(t, int64(123), key)
				return &operatev87.GetProcessInstanceByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      makeProcessInstanceResponse(123, "ACTIVE", ""),
				}, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				items := []operatev87.ProcessInstance{}
				return &operatev87.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &operatev87.ResultsProcessInstance{
						Items: &items,
					},
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				msg := wrongStateMessage()
				return &operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://operate.local/process-instances/123", http.StatusBadRequest, "400 Bad Request"),
					ApplicationproblemJSON400: &operatev87.Error{
						Message: &msg,
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
		svc := newTestService(t, waitTestConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				getCalls++
				switch getCalls {
				case 1:
					return &operatev87.GetProcessInstanceByKeyResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
						JSON200:      makeProcessInstanceResponse(123, "COMPLETED", ""),
					}, nil
				case 2:
					return nil, d.ErrNotFound
				default:
					t.Fatalf("unexpected get call #%d", getCalls)
					return nil, nil
				}
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				items := []operatev87.ProcessInstance{}
				return &operatev87.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &operatev87.ResultsProcessInstance{
						Items: &items,
					},
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				return &operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://operate.local/process-instances/123", http.StatusOK, "200 OK"),
					JSON200:      &operatev87.ChangeStatus{},
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
	svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		v87.WithClientCamunda(camundaClient),
		v87.WithClientOperate(operateClient),
	)
	require.NoError(t, err)
	require.Equal(t, camundaClient, svc.ClientCamunda())
	require.Equal(t, operateClient, svc.ClientOperate())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v87.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v87.WithClientCamunda(nil)(svc)
	v87.WithClientOperate(nil)(svc)
	v87.WithLogger(nil)(svc)
	require.Equal(t, camundaClient, svc.ClientCamunda())
	require.Equal(t, operateClient, svc.ClientOperate())
	require.Equal(t, logger, svc.Logger())
}

func newTestService(t *testing.T, cfg *config.Config, camundaClient *mockCamundaClient, operateClient *mockOperateClient) *v87.Service {
	t.Helper()

	svc, err := v87.New(
		cfg,
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		v87.WithClientCamunda(camundaClient),
		v87.WithClientOperate(operateClient),
	)
	require.NoError(t, err)
	return svc
}

func newStrictCamundaClient(t *testing.T) *mockCamundaClient {
	t.Helper()
	return &mockCamundaClient{
		postProcessInstancesWithResponse: func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
			t.Fatalf("unexpected create call")
			return nil, nil
		},
		postProcessInstancesProcessInstanceKeyCancellationWithResponse: func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
			t.Fatalf("unexpected cancellation call")
			return nil, nil
		},
	}
}

func newStrictOperateClient(t *testing.T) *mockOperateClient {
	t.Helper()
	return &mockOperateClient{
		getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
			t.Fatalf("unexpected get call")
			return nil, nil
		},
		searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
			t.Fatalf("unexpected search call")
			return nil, nil
		},
		deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
			t.Fatalf("unexpected delete call")
			return nil, nil
		},
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

func makeProcessInstanceResponse(key int64, state string, parentKey string) *operatev87.ProcessInstance {
	processState := operatev87.ProcessInstanceState(state)
	item := &operatev87.ProcessInstance{
		Key:                  toolx.Ptr(key),
		BpmnProcessId:        toolx.Ptr("demo"),
		ProcessDefinitionKey: toolx.Ptr(int64(9001)),
		ProcessVersion:       toolx.Ptr(int32(3)),
		ProcessVersionTag:    toolx.Ptr("stable"),
		StartDate:            toolx.Ptr("2026-03-23T18:00:00Z"),
		State:                &processState,
		TenantId:             toolx.Ptr("tenant"),
		Incident:             toolx.Ptr(false),
	}
	if parentKey != "" {
		parsedParentKey, err := toolx.StringToInt64(parentKey)
		if err != nil {
			panic(err)
		}
		item.ParentKey = toolx.Ptr(parsedParentKey)
	}
	return item
}

func wrongStateMessage() string {
	return "Process instances needs to be in one of the states [COMPLETED, CANCELED]"
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
