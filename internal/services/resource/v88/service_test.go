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

type mockResourceClient struct {
	createDeploymentWithBodyWithResponse func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error)
	deleteResourceWithResponse           func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error)
}

func (m *mockResourceClient) CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
	return m.createDeploymentWithBodyWithResponse(ctx, contentType, body, reqEditors...)
}

func (m *mockResourceClient) DeleteResourceWithResponse(ctx context.Context, resourceKey camundav88.ResourceKey, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
	return m.deleteResourceWithResponse(ctx, resourceKey, body, reqEditors...)
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
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
					t.Fatalf("unexpected delete call")
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
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
					t.Fatalf("unexpected delete call")
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
				deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
			},
			processClient: &mockProcessDefinitionClient{
				getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
					assert.Equal(t, "proc-2", key)
					processDefinitionID := camundav88.ProcessDefinitionId("demo")
					processDefinitionKey := camundav88.ProcessDefinitionKey("proc-2")
					respBody := camundav88.ProcessDefinitionResult{
						ProcessDefinitionId:  &processDefinitionID,
						ProcessDefinitionKey: &processDefinitionKey,
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
		svc, err := New(testConfigWithTenant(t, "tenant"), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
				called = true
				assert.Equal(t, "resource-1", resourceKey)
				return &camundav88.DeleteResourceResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/resources/resource-1", http.StatusNoContent, "204 No Content"),
				}, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		}))
		require.NoError(t, err)

		err = svc.Delete(ctx, "resource-1", services.WithAllowInconsistent())

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("WithoutAllowInconsistentSkipsDeletionEndpoint", func(t *testing.T) {
		svc, err := New(testConfigWithTenant(t, "tenant"), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
				t.Fatalf("deletion endpoint should not be called")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		}))
		require.NoError(t, err)

		err = svc.Delete(ctx, "resource-1")

		require.NoError(t, err)
	})
}

func TestService_ProcessDefinitionDeployPoller(t *testing.T) {
	ctx := context.Background()

	t.Run("MissingProcessDefinitionsKeepPolling", func(t *testing.T) {
		svc, err := New(testConfigWithTenant(t, "tenant"), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
				t.Fatalf("unexpected delete call")
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
		svc, err := New(testConfigWithTenant(t, "tenant"), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav88.DeleteResourceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.DeleteResourceResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
				processDefinitionID := camundav88.ProcessDefinitionId("demo")
				processDefinitionKey := camundav88.ProcessDefinitionKey("proc-1")
				respBody := camundav88.ProcessDefinitionResult{
					ProcessDefinitionId:  &processDefinitionID,
					ProcessDefinitionKey: &processDefinitionKey,
				}
				return &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-1", http.StatusOK, "200 OK"),
					JSON200:      &respBody,
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
			Success: true,
			Message: "process definitions visible: [proc-1]",
		}, status)
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
