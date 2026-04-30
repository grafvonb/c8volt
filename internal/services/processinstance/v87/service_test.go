// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		opts          []services.CallOption
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
							ProcessDefinitionId:      new("demo"),
							ProcessDefinitionVersion: new(int32(7)),
							TenantId:                 new("tenant-a"),
							Variables:                &map[string]interface{}{"orderId": "42"},
						},
					}, nil
				},
				postProcessInstancesProcessInstanceKeyCancellationWithResponse: func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
					t.Fatalf("unexpected cancellation call")
					return nil, nil
				},
			},
			opts: []services.CallOption{services.WithNoWait()},
			assertResult: func(t *testing.T, creation d.ProcessInstanceCreation) {
				assert.Equal(t, "demo", creation.BpmnProcessId)
				assert.Equal(t, int32(7), creation.ProcessDefinitionVersion)
				assert.Equal(t, "tenant-a", creation.TenantId)
				assert.Equal(t, "42", creation.Variables["orderId"])
				assert.NotEmpty(t, creation.StartDate)
			},
		},
		{
			name: "SuccessSkipsWaitWhenKeyUnknown",
			camunda: &mockCamundaClient{
				postProcessInstancesWithResponse: func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
					return &camundav87.PostProcessInstancesResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
						JSON200: &camundav87.CreateProcessInstanceResult{
							ProcessDefinitionId:      new("demo"),
							ProcessDefinitionVersion: new(int32(7)),
							TenantId:                 new("tenant-a"),
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
				assert.Equal(t, "<unknown in v87>", creation.Key)
				assert.Equal(t, "demo", creation.BpmnProcessId)
				assert.Equal(t, int32(7), creation.ProcessDefinitionVersion)
				assert.Equal(t, "tenant-a", creation.TenantId)
				assert.Equal(t, "42", creation.Variables["orderId"])
				assert.NotEmpty(t, creation.StartDate)
				assert.Empty(t, creation.StartConfirmedAt)
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
			opts:          []services.CallOption{services.WithNoWait()},
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
			}, tt.opts...)

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

func TestService_CreateProcessInstance_DefaultsEmptyTenantToDefaultTenant(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig()
	cfg.App.Tenant = ""
	svc := newTestService(t, cfg, &mockCamundaClient{
		postProcessInstancesWithResponse: func(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error) {
			require.NotNil(t, body.TenantId)
			assert.Equal(t, config.DefaultTenant, *body.TenantId)
			return &camundav87.PostProcessInstancesResponse{
				HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/process-instances", http.StatusOK, "200 OK"),
				JSON200: &camundav87.CreateProcessInstanceResult{
					ProcessDefinitionId:      new("demo"),
					ProcessDefinitionVersion: new(int32(7)),
					TenantId:                 new(config.DefaultTenant),
					Variables:                &map[string]interface{}{},
				},
			}, nil
		},
		postProcessInstancesProcessInstanceKeyCancellationWithResponse: func(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error) {
			t.Fatalf("unexpected cancellation call")
			return nil, nil
		},
	}, newStrictOperateClient(t))

	creation, err := svc.CreateProcessInstance(ctx, d.ProcessInstanceData{BpmnProcessId: "demo"}, services.WithNoWait())

	require.NoError(t, err)
	assert.Equal(t, config.DefaultTenant, creation.TenantId)
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
			name:          "TenantSafeLookupUnsupported",
			key:           "123",
			operate:       newStrictOperateClient(t),
			expectedError: d.ErrUnsupported,
		},
		{
			name:              "KeyConversionError",
			key:               "not-a-number",
			operate:           newStrictOperateClient(t),
			expectedErrSubstr: "converting process instance key",
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
				if errors.Is(tt.expectedError, d.ErrUnsupported) {
					assert.Contains(t, err.Error(), "process-instance direct lookup by key is not tenant-safe in Camunda 8.7")
				}
				assert.NotContains(t, err.Error(), "parent process instances were not found")
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

	t.Run("OmitsTenantFilterWhenConfigTenantIsEmpty", func(t *testing.T) {
		cfg := testConfig()
		cfg.App.Tenant = ""
		svc := newTestService(t, cfg, newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				assert.Nil(t, body.Filter.TenantId)
				items := []operatev87.ProcessInstance{*makeProcessInstanceResponse(123, "ACTIVE", "")}
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

		items, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
	})

	t.Run("ParentKeyConversionError", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), newStrictOperateClient(t))

		_, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{ParentKey: "abc"}, 25)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing parent key")
	})

	t.Run("OmitsUnsupportedPresencePushdownFields", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				assert.Nil(t, body.Filter.ParentKey)

				payload, err := json.Marshal(body)
				require.NoError(t, err)
				assert.NotContains(t, string(payload), "hasIncident")
				assert.NotContains(t, string(payload), "parentProcessInstanceKey")

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
			HasParent:   new(true),
			HasIncident: new(true),
		}, 25)

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "123", items[0].Key)
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

	t.Run("RejectsDateFiltersAsUnsupported", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
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
		})

		_, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			StartDateAfter: "2026-01-01",
		}, 25)

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrUnsupported)
		assert.Contains(t, err.Error(), "date filters require Camunda 8.8")
	})

	t.Run("RejectsAnyDateBoundAsUnsupported", func(t *testing.T) {
		cases := []d.ProcessInstanceFilter{
			{StartDateAfter: "2026-01-01"},
			{StartDateBefore: "2026-01-31"},
			{EndDateAfter: "2026-01-01"},
			{EndDateBefore: "2026-01-31"},
		}

		for _, filter := range cases {
			t.Run(fmt.Sprintf("%+v", filter), func(t *testing.T) {
				svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
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
				})

				_, err := svc.SearchForProcessInstances(ctx, filter, 25)

				require.Error(t, err)
				assert.ErrorIs(t, err, d.ErrUnsupported)
				assert.Contains(t, err.Error(), "date filters require Camunda 8.8")
			})
		}
	})

	t.Run("RejectsDerivedAbsoluteDateRangesAsUnsupported", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
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
		})

		_, err := svc.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{
			StartDateAfter: "2026-03-11",
			EndDateBefore:  "2026-04-03",
		}, 25)

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrUnsupported)
		assert.Contains(t, err.Error(), "date filters require Camunda 8.8")
	})
}

func TestService_SearchForProcessInstancesPage_FallbackOverflowDetection(t *testing.T) {
	ctx := context.Background()

	t.Run("uses total to report has-more and trim the requested window", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Size)
				assert.Equal(t, int32(4), *body.Size)
				items := []operatev87.ProcessInstance{
					*makeProcessInstanceResponse(101, "ACTIVE", ""),
					*makeProcessInstanceResponse(102, "ACTIVE", ""),
					*makeProcessInstanceResponse(103, "ACTIVE", ""),
					*makeProcessInstanceResponse(104, "ACTIVE", ""),
				}
				return &operatev87.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
					JSON200: &operatev87.ResultsProcessInstance{
						Items: &items,
						Total: new(int64(5)),
					},
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 2, Size: 2})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateHasMore, page.OverflowState)
		require.NotNil(t, page.ReportedTotal)
		assert.EqualValues(t, 5, page.ReportedTotal.Count)
		assert.Equal(t, d.ProcessInstanceReportedTotalKindExact, page.ReportedTotal.Kind)
		require.Len(t, page.Items, 2)
		assert.Equal(t, "103", page.Items[0].Key)
		assert.Equal(t, "104", page.Items[1].Key)
	})

	t.Run("missing total on a full page stays indeterminate", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Size)
				assert.Equal(t, int32(2), *body.Size)
				items := []operatev87.ProcessInstance{
					*makeProcessInstanceResponse(201, "ACTIVE", ""),
					*makeProcessInstanceResponse(202, "ACTIVE", ""),
				}
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

		page, err := svc.SearchForProcessInstancesPage(ctx, d.ProcessInstanceFilter{}, d.ProcessInstancePageRequest{From: 0, Size: 2})

		require.NoError(t, err)
		assert.Equal(t, d.ProcessInstanceOverflowStateIndeterminate, page.OverflowState)
		assert.Nil(t, page.ReportedTotal)
		require.Len(t, page.Items, 2)
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
			name:          "TenantSafeLookupUnsupported",
			key:           "123",
			operate:       newStrictOperateClient(t),
			expectedError: d.ErrUnsupported,
		},
		{
			name:              "KeyConversionError",
			key:               "invalid",
			operate:           newStrictOperateClient(t),
			expectedErrSubstr: "converting process instance key",
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
				assert.Contains(t, err.Error(), "process instance state")
				assert.NotContains(t, err.Error(), "get process instance state")
				if errors.Is(tt.expectedError, d.ErrUnsupported) {
					assert.Contains(t, err.Error(), "process-instance state lookup by key is not tenant-safe in Camunda 8.7")
				}
				return
			}

			require.NoError(t, err)
			tt.assertResult(t, state, pi)
		})
	}
}

func TestService_V87SearchBackedChildrenRemainSupported(t *testing.T) {
	ctx := context.Background()

	svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
		getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
			t.Fatalf("unexpected get call")
			return nil, nil
		},
		searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
			require.NotNil(t, body.Filter)
			require.Equal(t, int64(123), *body.Filter.ParentKey)
			require.Equal(t, "tenant", *body.Filter.TenantId)
			items := []operatev87.ProcessInstance{*makeProcessInstanceResponse(456, "ACTIVE", "123")}
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

	children, err := svc.GetDirectChildrenOfProcessInstance(ctx, "123")

	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, "456", children[0].Key)
	assert.Equal(t, "123", children[0].ParentKey)
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
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
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
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

		_, err := svc.DeleteProcessInstance(ctx, "123", services.WithNoWait())

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrUnsupported)
	})

	t.Run("WrongStateWithoutForceReturnsConflict", func(t *testing.T) {
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
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
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

		_, err := svc.DeleteProcessInstance(ctx, "123", services.WithNoWait())

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrUnsupported)
	})

	t.Run("SuccessWaitsForAbsentState", func(t *testing.T) {
		svc := newTestService(t, waitTestConfig(), newStrictCamundaClient(t), &mockOperateClient{
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
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

		_, err := svc.DeleteProcessInstance(ctx, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrUnsupported)
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

func TestService_TraversalResults(t *testing.T) {
	ctx := context.Background()

	t.Run("AncestryResultUsesTenantSafeLookupForPartialParentChains", func(t *testing.T) {
		searchCalls := 0
		svc := newTestService(t, testConfig(), newStrictCamundaClient(t), &mockOperateClient{
			getProcessInstanceByKeyWithResponse: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error) {
				t.Fatalf("unexpected direct get call")
				return nil, nil
			},
			searchProcessInstancesWithResponse: func(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error) {
				require.NotNil(t, body.Filter)
				require.NotNil(t, body.Size)
				assert.Equal(t, int32(2), *body.Size)
				searchCalls++
				if searchCalls == 1 {
					items := []operatev87.ProcessInstance{*makeProcessInstanceResponse(123, "ACTIVE", "999")}
					return &operatev87.SearchProcessInstancesResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
						JSON200:      &operatev87.ResultsProcessInstance{Items: &items},
					}, nil
				}
				return &operatev87.SearchProcessInstancesResponse{
					Body:         []byte(`{"items":[]}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/process-instances/search", http.StatusOK, "200 OK"),
					JSON200:      &operatev87.ResultsProcessInstance{},
				}, nil
			},
			deleteProcessInstanceAndAllDependantDataByKeyWithResp: func(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		})

		result, err := svc.AncestryResult(ctx, "123")

		require.NoError(t, err)
		assert.Equal(t, "123", result.RootKey)
		assert.Equal(t, []string{"123"}, result.Keys)
		assert.Equal(t, "one or more parent process instances were not found", result.Warning)
		assert.Equal(t, "partial", string(result.Outcome))
		require.Len(t, result.MissingAncestors, 1)
		assert.Equal(t, "999", result.MissingAncestors[0].Key)
		assert.Equal(t, 2, searchCalls)
	})
}

// newTestService creates a v8.7 process-instance service with strict injected clients.
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

// newStrictCamundaClient returns a v8.7 Camunda client mock that fails on unexpected calls.
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

// newStrictOperateClient returns a v8.7 Operate client mock that fails on unexpected calls.
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
	item := &operatev87.ProcessInstance{
		Key:                  new(key),
		BpmnProcessId:        new("demo"),
		ProcessDefinitionKey: new(int64(9001)),
		ProcessVersion:       new(int32(3)),
		ProcessVersionTag:    new("stable"),
		StartDate:            new("2026-03-23T18:00:00Z"),
		State:                new(operatev87.ProcessInstanceState(state)),
		TenantId:             new("tenant"),
		Incident:             new(false),
	}
	if parentKey != "" {
		parsedParentKey, err := toolx.StringToInt64(parentKey)
		if err != nil {
			panic(err)
		}
		item.ParentKey = new(parsedParentKey)
	}
	return item
}

func wrongStateMessage() string {
	return "Process instances needs to be in one of the states [COMPLETED, CANCELED]"
}

// newHTTPResponse builds a minimal HTTP response for v8.7 process-instance error handling tests.
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
