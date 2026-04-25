package v89

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
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService wires the v8.9 resource service with separate resource and
// process definition mocks, making the deployment confirmation dependency
// explicit in each test.
func newTestService(t *testing.T, tenantID string, client *mockResourceClient, processClient *mockProcessDefinitionClient) *Service {
	t.Helper()

	svc, err := New(testConfigWithTenant(t, tenantID), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client, processClient))
	require.NoError(t, err)
	return svc
}

type mockResourceClient struct {
	createDeploymentWithBodyWithResponse func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error)
	deleteResourceWithResponse           func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error)
	getResourceWithResponse              func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error)
}

func (m *mockResourceClient) CreateDeploymentWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
	return m.createDeploymentWithBodyWithResponse(ctx, contentType, body, reqEditors...)
}

func (m *mockResourceClient) DeleteResourceOpWithResponse(ctx context.Context, resourceKey camundav89.ResourceKey, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
	return m.deleteResourceWithResponse(ctx, resourceKey, body, reqEditors...)
}

func (m *mockResourceClient) GetResourceWithResponse(ctx context.Context, resourceKey camundav89.ResourceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
	return m.getResourceWithResponse(ctx, resourceKey, reqEditors...)
}

type mockProcessDefinitionClient struct {
	getProcessDefinitionWithResponse func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionWithResponse(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
	return m.getProcessDefinitionWithResponse(ctx, key, reqEditors...)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionXMLWithResponse(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionXMLResponse, error) {
	panic("unexpected GetProcessDefinitionXMLWithResponse call")
}

func (m *mockProcessDefinitionClient) SearchProcessDefinitionsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessDefinitionsResponse, error) {
	panic("unexpected SearchProcessDefinitionsWithBodyWithResponse call")
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionStatisticsWithResponse(ctx context.Context, key string, body camundav89.GetProcessDefinitionStatisticsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionStatisticsResponse, error) {
	panic("unexpected GetProcessDefinitionStatisticsWithResponse call")
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionInstanceVersionStatisticsWithResponse(ctx context.Context, body camundav89.GetProcessDefinitionInstanceVersionStatisticsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionInstanceVersionStatisticsResponse, error) {
	panic("unexpected GetProcessDefinitionInstanceVersionStatisticsWithResponse call")
}

// TestService_Deploy verifies the v8.9 deployment flow.
// v8.9 defaults to confirmation polling, while no-wait is still expected to
// stop before polling when malformed deployment payloads are detected.
func TestService_Deploy(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant"
	resourceName := "demo.bpmn"
	resourceData := []byte("<xml>demo</xml>")

	t.Run("SuccessWithConfirmation", func(t *testing.T) {
		svc := newTestService(t, tenantID, &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
				return &camundav89.CreateDeploymentResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
					JSON200: &camundav89.DeploymentResult{
						DeploymentKey: "deployment-1",
						TenantId:      tenantID,
						Deployments: []camundav89.DeploymentMetadataResult{
							{
								ProcessDefinition: &camundav89.DeploymentProcessResult{
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
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				assert.Equal(t, "proc-1", key)
				return &camundav89.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-1", http.StatusOK, "200 OK"),
					JSON200: &camundav89.ProcessDefinitionResult{
						ProcessDefinitionId:  "demo",
						ProcessDefinitionKey: "proc-1",
					},
				}, nil
			},
		})

		deployment, err := svc.Deploy(ctx, []d.DeploymentUnitData{{Name: resourceName, Data: resourceData}})

		require.NoError(t, err)
		assert.Equal(t, "deployment-1", deployment.Key)
		assert.Equal(t, tenantID, deployment.TenantId)
		require.Len(t, deployment.Units, 1)
		assert.Equal(t, "proc-1", deployment.Units[0].ProcessDefinition.ProcessDefinitionKey)
	})

	t.Run("MalformedSuccessPayload", func(t *testing.T) {
		svc := newTestService(t, tenantID, &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
				return &camundav89.CreateDeploymentResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
				}, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected confirmation poll")
				return nil, nil
			},
		})

		_, err := svc.Deploy(ctx, []d.DeploymentUnitData{{Name: resourceName, Data: resourceData}}, services.WithNoWait())

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})
}

// TestService_Delete documents the allow-inconsistent guard around v8.9
// resource deletion, preserving the same destructive-operation safety contract
// as older supported Camunda versions.
func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("AllowInconsistentCallsDeletionEndpoint", func(t *testing.T) {
		var called bool
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				called = true
				assert.Equal(t, "resource-1", resourceKey)
				return &camundav89.DeleteResourceOpResponse{
					HTTPResponse: newHTTPResponse(http.MethodDelete, "https://camunda.local/v2/resources/resource-1", http.StatusNoContent, "204 No Content"),
				}, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		err := svc.Delete(ctx, "resource-1", services.WithAllowInconsistent())

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("WithoutAllowInconsistentSkipsDeletionEndpoint", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				t.Fatalf("deletion endpoint should not be called")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		err := svc.Delete(ctx, "resource-1")

		require.NoError(t, err)
	})
}

// TestService_Get verifies v8.9 resource mapping and decoded-empty protection.
// A non-nil generated JSON200 value is not sufficient; required identity fields
// must be present before a resource is considered successfully loaded.
func TestService_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				assert.Equal(t, "resource-1", resourceKey)
				return &camundav89.GetResourceResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
					JSON200: &camundav89.ResourceResult{
						ResourceId:   "demo-process",
						ResourceKey:  "resource-1",
						ResourceName: "demo.bpmn",
						TenantId:     "tenant-a",
						Version:      7,
						VersionTag:   new("v1"),
					},
				}, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		resource, err := svc.Get(ctx, "resource-1")

		require.NoError(t, err)
		assert.Equal(t, d.Resource{
			ID:         "demo-process",
			Key:        "resource-1",
			Name:       "demo.bpmn",
			TenantId:   "tenant-a",
			Version:    7,
			VersionTag: "v1",
		}, resource)
	})

	t.Run("DecodedEmptySuccessPayload", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				return &camundav89.GetResourceResponse{
					Body:         []byte(`{"detail":"missing payload"}`),
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
					JSON200:      &camundav89.ResourceResult{},
				}, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				t.Fatalf("unexpected process definition lookup")
				return nil, nil
			},
		})

		_, err := svc.Get(ctx, "resource-1")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})
}

// TestService_ProcessDefinitionDeployPoller covers the happy confirmation path
// for v8.9 deployments, where all returned process definition keys are visible.
func TestService_ProcessDefinitionDeployPoller(t *testing.T) {
	ctx := context.Background()

	t.Run("VisibleProcessDefinitionsCompletePolling", func(t *testing.T) {
		svc := newTestService(t, "tenant", &mockResourceClient{
			createDeploymentWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateDeploymentResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			deleteResourceWithResponse: func(ctx context.Context, resourceKey string, body camundav89.DeleteResourceOpJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteResourceOpResponse, error) {
				t.Fatalf("unexpected delete call")
				return nil, nil
			},
			getResourceWithResponse: func(ctx context.Context, resourceKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetResourceResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}, &mockProcessDefinitionClient{
			getProcessDefinitionWithResponse: func(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
				return &camundav89.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/process-definitions/proc-1", http.StatusOK, "200 OK"),
					JSON200: &camundav89.ProcessDefinitionResult{
						ProcessDefinitionId:  "demo",
						ProcessDefinitionKey: "proc-1",
					},
				}, nil
			},
		})

		poll := svc.processDefinitionDeployPoller(camundav89.DeploymentResult{
			Deployments: []camundav89.DeploymentMetadataResult{
				{
					ProcessDefinition: &camundav89.DeploymentProcessResult{ProcessDefinitionKey: "proc-1"},
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

// assertMultipartDeploymentRequest checks the exact multipart contract sent to
// Camunda: tenantId is sent as a field, resources carries the BPMN bytes, and
// the uploaded part keeps the caller's filename.
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

// newHTTPResponse builds the minimum response metadata required by status
// normalization helpers, including method and URL for useful error text.
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

// testConfigWithTenant supplies the effective tenant used by deployment request
// bodies and tenant-aware service behavior.
func testConfigWithTenant(t *testing.T, tenantID string) *config.Config {
	t.Helper()

	cfg := testx.TestConfig(t)
	cfg.App.Tenant = tenantID
	return cfg
}

// testStringPtr mirrors nullable fields in generated v8.9 response types.
func testStringPtr(v string) *string { return &v }
