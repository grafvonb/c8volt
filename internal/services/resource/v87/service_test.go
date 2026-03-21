package v87

import (
	"context"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResourceClient struct {
	postDeploymentsWithBodyWithResponse              func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error)
	postResourcesResourceKeyDeletionWithResponseFunc func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error)
	getResourcesResourceKeyWithResponseFunc          func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error)
}

func (m *mockResourceClient) PostDeploymentsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
	return m.postDeploymentsWithBodyWithResponse(ctx, contentType, body, reqEditors...)
}

func (m *mockResourceClient) PostResourcesResourceKeyDeletionWithResponse(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
	return m.postResourcesResourceKeyDeletionWithResponseFunc(ctx, resourceKey, body, reqEditors...)
}

func (m *mockResourceClient) GetResourcesResourceKeyWithResponse(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
	return m.getResourcesResourceKeyWithResponseFunc(ctx, resourceKey, reqEditors...)
}

func TestService_Deploy(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant"
	resourceName := "demo.bpmn"
	resourceData := []byte("<xml>demo</xml>")

	tests := []struct {
		name          string
		client        *mockResourceClient
		expectedError error
		assertResult  func(*testing.T, d.Deployment)
	}{
		{
			name: "Success",
			client: &mockResourceClient{
				postDeploymentsWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
					assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
					return &camundav87.PostDeploymentsResponse{
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
						JSON200: &camundav87.DeploymentResult{
							TenantId: &tenantID,
						},
					}, nil
				},
				postResourcesResourceKeyDeletionWithResponseFunc: func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourcesResourceKeyWithResponseFunc: func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
					t.Fatalf("unexpected get call")
					return nil, nil
				},
			},
			assertResult: func(t *testing.T, deployment d.Deployment) {
				assert.Equal(t, tenantID, deployment.TenantId)
				assert.Equal(t, "<unknown>", deployment.Key)
			},
		},
		{
			name: "MalformedSuccessPayload",
			client: &mockResourceClient{
				postDeploymentsWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
					assertMultipartDeploymentRequest(t, contentType, body, tenantID, resourceName, resourceData)
					return &camundav87.PostDeploymentsResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/deployments", http.StatusOK, "200 OK"),
					}, nil
				},
				postResourcesResourceKeyDeletionWithResponseFunc: func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourcesResourceKeyWithResponseFunc: func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
					t.Fatalf("unexpected get call")
					return nil, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(tt.client))
			require.NoError(t, err)

			deployment, err := svc.Deploy(ctx, []d.DeploymentUnitData{{Name: resourceName, Data: resourceData}})

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			tt.assertResult(t, deployment)
		})
	}
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("AllowInconsistentCallsDeletionEndpoint", func(t *testing.T) {
		var called bool
		svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			postDeploymentsWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			postResourcesResourceKeyDeletionWithResponseFunc: func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
				called = true
				assert.Equal(t, "resource-1", resourceKey)
				return &camundav87.PostResourcesResourceKeyDeletionResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://camunda.local/v2/resources/resource-1/deletion", http.StatusNoContent, "204 No Content"),
				}, nil
			},
			getResourcesResourceKeyWithResponseFunc: func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}))
		require.NoError(t, err)

		err = svc.Delete(ctx, "resource-1", services.WithAllowInconsistent())

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("WithoutAllowInconsistentSkipsDeletionEndpoint", func(t *testing.T) {
		svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(&mockResourceClient{
			postDeploymentsWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
				t.Fatalf("unexpected deploy call")
				return nil, nil
			},
			postResourcesResourceKeyDeletionWithResponseFunc: func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
				t.Fatalf("deletion endpoint should not be called")
				return nil, nil
			},
			getResourcesResourceKeyWithResponseFunc: func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
				t.Fatalf("unexpected get call")
				return nil, nil
			},
		}))
		require.NoError(t, err)

		err = svc.Delete(ctx, "resource-1")

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
				postDeploymentsWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
					t.Fatalf("unexpected deploy call")
					return nil, nil
				},
				postResourcesResourceKeyDeletionWithResponseFunc: func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourcesResourceKeyWithResponseFunc: func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
					assert.Equal(t, "resource-1", resourceKey)
					return &camundav87.GetResourcesResourceKeyResponse{
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
						JSON200: &camundav87.ResourceResult{
							ResourceId:   testStringPtr("demo-process"),
							ResourceKey:  testStringPtr("resource-1"),
							ResourceName: testStringPtr("demo.bpmn"),
							TenantId:     testStringPtr("tenant-a"),
							Version:      testInt32Ptr(7),
							VersionTag:   testStringPtr("v1"),
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
				postDeploymentsWithBodyWithResponse: func(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostDeploymentsResponse, error) {
					t.Fatalf("unexpected deploy call")
					return nil, nil
				},
				postResourcesResourceKeyDeletionWithResponseFunc: func(ctx context.Context, resourceKey string, body camundav87.PostResourcesResourceKeyDeletionJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostResourcesResourceKeyDeletionResponse, error) {
					t.Fatalf("unexpected delete call")
					return nil, nil
				},
				getResourcesResourceKeyWithResponseFunc: func(ctx context.Context, resourceKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetResourcesResourceKeyResponse, error) {
					return &camundav87.GetResourcesResourceKeyResponse{
						Body:         []byte(`{"detail":"missing payload"}`),
						HTTPResponse: newHTTPResponse(http.MethodGet, "https://camunda.local/v2/resources/resource-1", http.StatusOK, "200 OK"),
					}, nil
				},
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(tt.client))
			require.NoError(t, err)

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

func testStringPtr(v string) *string { return &v }

func testInt32Ptr(v int32) *int32 { return &v }
