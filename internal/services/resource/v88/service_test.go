package v88

import (
	"context"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T, tenantID string, client *mockResourceClient, processClient *mockProcessDefinitionClient) *Service {
	t.Helper()

	svc, err := New(testConfigWithTenant(t, tenantID), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client, processClient))
	require.NoError(t, err)
	return svc
}

type mockResourceClient struct {
	createDeploymentWithBodyWithResponse func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error)
	deleteResourceWithResponse           func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error)
	getResourceWithResponse              func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error)
}

func (m *mockResourceClient) CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
	return m.createDeploymentWithBodyWithResponse(ctx, contentType, body, reqEditors...)
}

func (m *mockResourceClient) DeleteResourceOpWithResponse(ctx context.Context, resourceKey camundav88.ResourceKey, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
	return m.deleteResourceWithResponse(ctx, resourceKey, body, reqEditors...)
}

func (m *mockResourceClient) GetResourceWithResponse(ctx context.Context, resourceKey camundav88.ResourceKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
	return m.getResourceWithResponse(ctx, resourceKey, reqEditors...)
}

type mockProcessDefinitionClient struct {
	getProcessDefinitionWithResponse func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionWithResponse(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
	return m.getProcessDefinitionWithResponse(ctx, key, reqEditors...)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionXMLWithResponse(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionXMLResponse, error) {
	panic("unexpected GetProcessDefinitionXMLWithResponse call")
}

func (m *mockProcessDefinitionClient) SearchProcessDefinitionsWithResponse(ctx context.Context, body camundav88.SearchProcessDefinitionsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessDefinitionsResponse, error) {
	panic("unexpected SearchProcessDefinitionsWithResponse call")
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionStatisticsWithResponse(ctx context.Context, key string, body camundav88.GetProcessDefinitionStatisticsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionStatisticsResponse, error) {
	panic("unexpected GetProcessDefinitionStatisticsWithResponse call")
}

func TestService_Deploy(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant"
	resourceName := "demo.bpmn"
	resourceData := []byte("<xml>demo</xml>")

	tests := []struct {
		name          string
		opts          []services.CallOption
		client        *mockResourceClient
		processClient *mockProcessDefinitionClient
		expectedError error
		assertResult  func(*testing.T, d.Deployment)
	}{
		{
			name: "SuccessNoWait",
			opts: []services.CallOption{services.WithNoWait()},
			client: &mockResourceClient{
				createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
					assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
					return &camundav88.CreateDeploymentResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
						JSON200: &camundav88.DeploymentResult{
							DeploymentKey: "deployment-1",
							TenantId:      tenantID,
							Deployments: []camundav88.DeploymentMetadataResult{
								{
									ProcessDefinition: &camundav88.DeploymentProcessResult{
										ProcessDefinitionId:      "demo",
										ProcessDefinitionKey:     "proc-1",
										ProcessDefinitionVersion: 3,
										ResourceName:             resourceName,
										TenantId:                 tenantID,
									},
								},
							},
						},
					}, nil
				},
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
					t.Fatalf("unexpected get call")
					return nil, nil
				},
			},
			processClient: &mockProcessDefinitionClient{
				getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
					t.Fatalf("unexpected confirmation poll")
					return nil, nil
				},
			},
			assertResult: func(t *testing.T, deployment d.Deployment) {
				assert.Equal(t, "deployment-1", deployment.Key)
				assert.Equal(t, tenantID, deployment.TenantId)
				require.Len(t, deployment.Units, 1)
				assert.Equal(t, "proc-1", deployment.Units[0].ProcessDefinition.ProcessDefinitionKey)
			},
		},
		{
			name: "MalformedSuccessPayload",
			client: &mockResourceClient{
				createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
					assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
					return &camundav88.CreateDeploymentResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
					}, nil
				},
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
					t.Fatalf("unexpected get call")
					return nil, nil
				},
			},
			processClient: &mockProcessDefinitionClient{
				getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
					t.Fatalf("unexpected confirmation poll")
					return nil, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
		{
			name: "SuccessWithConfirmation",
			client: &mockResourceClient{
				createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
					assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
					return &camundav88.CreateDeploymentResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
						JSON200: &camundav88.DeploymentResult{
							DeploymentKey: "deployment-2",
							TenantId:      tenantID,
							Deployments: []camundav88.DeploymentMetadataResult{
								{
									ProcessDefinition: &camundav88.DeploymentProcessResult{
										ProcessDefinitionId:      "demo",
										ProcessDefinitionKey:     "proc-2",
										ProcessDefinitionVersion: 4,
										ResourceName:             resourceName,
										TenantId:                 tenantID,
									},
								},
							},
						},
					}, nil
				},
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
					t.Fatalf("unexpected get call")
					return nil, nil
				},
			},
			processClient: &mockProcessDefinitionClient{
				getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
					assert.Equal(t, "proc-2", key)
					respBody := camundav88.ProcessDefinitionResult{
						ProcessDefinitionId:  "demo",
						ProcessDefinitionKey: "proc-2",
					}
					return &camundav88.GetProcessDefinitionResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-2", http.StatusOK, "200 OK"),
						JSON200:      &respBody,
					}, nil
				},
			},
			assertResult: func(t *testing.T, deployment d.Deployment) {
				assert.Equal(t, "deployment-2", deployment.Key)
				assert.Equal(t, tenantID, deployment.TenantId)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(testConfigWithTenant(t, tenantID), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(tt.client, tt.processClient))
			require.NoError(t, err)

			deployment, err := svc.Deploy(ctx, []d.DeploymentUnitData{{Name: resourceName, Data: resourceData}}, tt.opts...)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			if tt.assertResult != nil {
				tt.assertResult(t, deployment)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("AllowInconsistentCallsDeletionEndpoint", func(t *testing.T) {
		var called bool
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
				called = true
				assert.Equal(t, "resource-1", resourceKey)
				return &camundav88.DeleteResourceOpResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/resources/resource-1", http.StatusNoContent, "204 No Content"),
				}, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		err := svc.Delete(ctx, "resource-1", services.WithAllowInconsistent())

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("AllowInconsistentReturnsDeletionStatusErrors", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
				return &camundav88.DeleteResourceOpResponse{
					Body:         []byte(`{"detail":"resource not found"}`),
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/resources/resource-1", http.StatusNotFound, "404 Not Found"),
				}, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		err := svc.Delete(ctx, "resource-1", services.WithAllowInconsistent())

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrNotFound)
	})

	t.Run("WithoutAllowInconsistentSkipsDeletionEndpoint", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
				t.Fatalf("deletion endpoint should not be called")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		err := svc.Delete(ctx, "resource-1")

		require.NoError(t, err)
	})
}

func TestService_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		client        *mockResourceClient
		expected      d.Resource
		expectedError error
	}{
		{
			name: "Success",
			client: &mockResourceClient{
				createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
					t.Fatalf("unexpected deploy call")
					return nil, nil
				},
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
					assert.Equal(t, "resource-1", resourceKey)
					return &camundav88.GetResourceResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
						JSON200: &camundav88.ResourceResult{
							ResourceId:   "demo-process",
							ResourceKey:  "resource-1",
							ResourceName: "demo.bpmn",
							TenantId:     "tenant-a",
							Version:      7,
							VersionTag:   new("v1"),
						},
					}, nil
				},
			},
			expected: d.Resource{
				ID:         "demo-process",
				Key:        "resource-1",
				Name:       "demo.bpmn",
				TenantId:   "tenant-a",
				Version:    7,
				VersionTag: "v1",
			},
		},
		{
			name: "MalformedSuccessPayload",
			client: &mockResourceClient{
				createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
					t.Fatalf("unexpected deploy call")
					return nil, nil
				},
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
					return &camundav88.GetResourceResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
					}, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
		{
			name: "DecodedEmptySuccessPayload",
			client: &mockResourceClient{
				createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
					t.Fatalf("unexpected deploy call")
					return nil, nil
				},
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
					return &camundav88.GetResourceResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
						JSON200:      &camundav88.ResourceResult{},
					}, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, "tenant", tt.client, &mockProcessDefinitionClient{
				getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
					t.Fatalf("unexpected process definition lookup")
					return nil, nil
				},
			})

			resource, err := svc.Get(ctx, "resource-1")

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resource)
		})
	}
}

func TestService_ProcessDefinitionDeployPoller(t *testing.T) {
	ctx := context.Background()

	t.Run("MissingProcessDefinitionsKeepPolling", func(t *testing.T) {
		svc, err := New(testConfigWithTenant(t, "tenant"), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				return &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-1", http.StatusNotFound, "404 Not Found"),
				}, nil
			},
		}))
		require.NoError(t, err)

		poll := svc.processDefinitionDeployPoller(camundav88.DeploymentResult{
			Deployments: []camundav88.DeploymentMetadataResult{
				{
					ProcessDefinition: &camundav88.DeploymentProcessResult{ProcessDefinitionKey: "proc-1"},
				},
			},
		})

		status, err := poll(ctx)

		require.NoError(t, err)
		assert.Equal(t, poller.JobPollStatus{
			Success: false,
			Message: "process definitions not visible yet, waiting: [proc-1]",
		}, status)
	})

	t.Run("VisibleProcessDefinitionsCompletePolling", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				respBody := camundav88.ProcessDefinitionResult{
					ProcessDefinitionId:  "demo",
					ProcessDefinitionKey: "proc-1",
				}
				return &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-1", http.StatusOK, "200 OK"),
					JSON200:      &respBody,
				}, nil
			},
		})

		poll := svc.processDefinitionDeployPoller(camundav88.DeploymentResult{
			Deployments: []camundav88.DeploymentMetadataResult{
				{
					ProcessDefinition: &camundav88.DeploymentProcessResult{ProcessDefinitionKey: "proc-1"},
				},
			},
		})

		status, err := poll(ctx)

		require.NoError(t, err)
		assert.Equal(t, poller.JobPollStatus{
			Success: true,
			Message: "process definitions visible: [proc-1]",
		}, status)
	})

	t.Run("UnexpectedStatusFailsPolling", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceOpJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				return &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-1", http.StatusInternalServerError, "500 Internal Server Error"),
				}, nil
			},
		})

		poll := svc.processDefinitionDeployPoller(camundav88.DeploymentResult{
			Deployments: []camundav88.DeploymentMetadataResult{
				{
					ProcessDefinition: &camundav88.DeploymentProcessResult{ProcessDefinitionKey: "proc-1"},
				},
			},
		})

		status, err := poll(ctx)

		require.Error(t, err)
		assert.Equal(t, poller.JobPollStatus{}, status)
		assert.ErrorContains(t, err, `get process definition "proc-1": unexpected status 500`)
	})
}

func assertMultipartDeploymentRequest(t *testing.T, contentType string, body io.Reader, tenantID string, resourceName string, resourceData []byte) {
	t.Helper()

	mediaType, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)

	reader := multipart.NewReader(body, params["boundary"])
	parts := map[string]string{}
	filenames := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		data, err := io.ReadAll(part)
		require.NoError(t, err)
		parts[part.FormName()] = string(data)
		filenames[part.FormName()] = part.FileName()
	}

	assert.Equal(t, tenantID, parts["tenantId"])
	assert.Equal(t, string(resourceData), parts["resources"])
	assert.Equal(t, resourceName, filenames["resources"])
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

func testConfigWithTenant(t *testing.T, tenantID string) *config.Config {
	t.Helper()

	cfg := testx.TestConfig(t)
	cfg.App.Tenant = tenantID
	return cfg
}

func testStringPtr(v string) *string { return &v }

func testInt32Ptr(v int32) *int32 { return &v }
