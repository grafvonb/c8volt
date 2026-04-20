package v88_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v88 "github.com/grafvonb/c8volt/internal/services/processdefinition/v88"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProcessDefinitionClient struct {
	mock.Mock
}

func (m *mockProcessDefinitionClient) SearchProcessDefinitionsWithResponse(ctx context.Context, body camundav88.SearchProcessDefinitionsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessDefinitionsResponse, error) {
	args := m.Called(ctx, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav88.SearchProcessDefinitionsResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionWithResponse(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav88.GetProcessDefinitionResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionXMLWithResponse(ctx context.Context, key string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionXMLResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav88.GetProcessDefinitionXMLResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionStatisticsWithResponse(ctx context.Context, key string, body camundav88.GetProcessDefinitionStatisticsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionStatisticsResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav88.GetProcessDefinitionStatisticsResponse), args.Error(1)
}

func TestService_SearchProcessDefinitions(t *testing.T) {
	ctx := context.Background()
	mockErr := errors.New("search failed")

	tests := []struct {
		name          string
		setupMock     func(*mockProcessDefinitionClient)
		opts          []services.CallOption
		expectedError error
		assertResult  func(*testing.T, []domain.ProcessDefinition)
	}{
		{
			name: "Success without stats",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200 OK"),
					JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
						Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
					},
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
			},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "proc", defs[0].BpmnProcessId)
			},
		},
		{
			name: "HTTP error",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusInternalServerError, "500"),
					Body:         []byte("boom"),
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "Nil payload",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "Client error",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return((*camundav88.SearchProcessDefinitionsResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "Stats requested",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
						Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
					},
				}
				statsResp := &camundav88.GetProcessDefinitionStatisticsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions/123/statistics", http.StatusOK, "200"),
					JSON200: &camundav88.ProcessDefinitionElementStatisticsQueryResult{
						Items: []camundav88.ProcessElementStatisticsResult{{Active: 1}},
					},
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
				m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).Return(statsResp, nil)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.NotNil(t, defs[0].Statistics)
				assert.Equal(t, int64(1), defs[0].Statistics.Active)
			},
		},
		{
			name: "Stats retrieval fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
						Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
					},
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
				m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).Return((*camundav88.GetProcessDefinitionStatisticsResponse)(nil), mockErr)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: mockErr,
		},
		{
			name: "Stats missing payload is tolerated",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
						Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
					},
				}
				statsResp := &camundav88.GetProcessDefinitionStatisticsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions/123/statistics", http.StatusOK, "200"),
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
				m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).Return(statsResp, nil)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.Len(t, defs, 1)
				assert.Nil(t, defs[0].Statistics)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProcessDefinitionClient{}
			if tt.setupMock != nil {
				tt := tt // capture
				tt.setupMock(m)
			}

			svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
			require.NoError(t, err)
			defs, err := svc.SearchProcessDefinitions(ctx, domain.ProcessDefinitionFilter{BpmnProcessId: "proc"}, 25, tt.opts...)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.assertResult != nil {
					tt.assertResult(t, defs)
				}
			}
			m.AssertExpectations(t)
		})
	}
}

func TestService_SearchProcessDefinitionsLatestForcesLatest(t *testing.T) {
	ctx := context.Background()
	m := &mockProcessDefinitionClient{}

	resp := &camundav88.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
		JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
			Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 1)},
		},
	}

	statsResp := &camundav88.GetProcessDefinitionStatisticsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions/123/statistics", http.StatusOK, "200"),
		JSON200: &camundav88.ProcessDefinitionElementStatisticsQueryResult{
			Items: []camundav88.ProcessElementStatisticsResult{{Active: 2}},
		},
	}

	m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			body := args.Get(1).(camundav88.SearchProcessDefinitionsJSONRequestBody)
			require.NotNil(t, body.Filter.IsLatestVersion)
			assert.True(t, *body.Filter.IsLatestVersion)
			assert.Nil(t, body.Filter.TenantId)
		}).
		Return(resp, nil)

	m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).
		Return(statsResp, nil)

	svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
	require.NoError(t, err)
	defs, err := svc.SearchProcessDefinitionsLatest(ctx, domain.ProcessDefinitionFilter{}, services.WithStat())

	require.NoError(t, err)
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].Statistics)
	assert.Equal(t, int64(2), defs[0].Statistics.Active)
	m.AssertExpectations(t)
}

func TestService_SearchProcessDefinitions_IncludesTenantFilterWhenConfigured(t *testing.T) {
	ctx := context.Background()
	m := &mockProcessDefinitionClient{}

	resp := &camundav88.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200 OK"),
		JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
			Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
		},
	}

	m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			body := args.Get(1).(camundav88.SearchProcessDefinitionsJSONRequestBody)
			require.NotNil(t, body.Filter)
			require.NotNil(t, body.Filter.TenantId)
			assert.Equal(t, "tenant-a", *body.Filter.TenantId)
		}).
		Return(resp, nil)

	cfg := testConfig()
	cfg.App.Tenant = "tenant-a"
	svc, err := v88.New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
	require.NoError(t, err)

	_, err = svc.SearchProcessDefinitions(ctx, domain.ProcessDefinitionFilter{}, 25)
	require.NoError(t, err)
	m.AssertExpectations(t)
}

func TestService_GetProcessDefinition(t *testing.T) {
	ctx := context.Background()
	mockErr := errors.New("get failed")

	tests := []struct {
		name          string
		setupMock     func(*mockProcessDefinitionClient)
		opts          []services.CallOption
		expectedError error
		assertResult  func(*testing.T, domain.ProcessDefinition)
	}{
		{
			name: "Success without stats",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			assertResult: func(t *testing.T, pd domain.ProcessDefinition) {
				assert.Equal(t, "proc", pd.BpmnProcessId)
			},
		},
		{
			name: "HTTP error",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusInternalServerError, "500"),
					Body:         []byte("fail"),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "Nil payload",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "Stats requested",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				statsResp := &camundav88.GetProcessDefinitionStatisticsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions/123/statistics", http.StatusOK, "200"),
					JSON200: &camundav88.ProcessDefinitionElementStatisticsQueryResult{
						Items: []camundav88.ProcessElementStatisticsResult{{Completed: 5}},
					},
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).Return(statsResp, nil)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, pd domain.ProcessDefinition) {
				require.NotNil(t, pd.Statistics)
				assert.Equal(t, int64(5), pd.Statistics.Completed)
			},
		},
		{
			name: "Stats retrieval fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).Return((*camundav88.GetProcessDefinitionStatisticsResponse)(nil), mockErr)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: mockErr,
		},
		{
			name: "Stats missing payload is tolerated",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				statsResp := &camundav88.GetProcessDefinitionStatisticsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions/123/statistics", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				m.On("GetProcessDefinitionStatisticsWithResponse", mock.Anything, "123", mock.Anything).Return(statsResp, nil)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, pd domain.ProcessDefinition) {
				assert.Nil(t, pd.Statistics)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProcessDefinitionClient{}
			if tt.setupMock != nil {
				tt := tt
				tt.setupMock(m)
			}

			svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
			require.NoError(t, err)
			pd, err := svc.GetProcessDefinition(ctx, "123", tt.opts...)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.assertResult != nil {
					tt.assertResult(t, pd)
				}
			}
			m.AssertExpectations(t)
		})
	}
}

func TestService_GetProcessDefinitionXML(t *testing.T) {
	ctx := context.Background()
	mockErr := errors.New("xml failed")

	tests := []struct {
		name          string
		setupMock     func(*mockProcessDefinitionClient)
		expectedError error
		expectedXML   string
	}{
		{
			name: "Success",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
					XML200:       new("<definitions id=\"proc\"/>"),
				}
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\"/>",
		},
		{
			name: "Success falls back to raw body",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
					Body:         []byte("<definitions id=\"proc\"/>"),
					XML200:       new("\n  \n  \n"),
				}
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\"/>",
		},
		{
			name: "Success prefers formatted raw body",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
					Body:         []byte("<definitions id=\"proc\">\n  <process id=\"order\" />\n</definitions>\n"),
					XML200:       new("\n  \n    \n"),
				}
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\">\n  <process id=\"order\" />\n</definitions>\n",
		},
		{
			name: "HTTP error",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusInternalServerError, "500"),
					Body:         []byte("fail"),
				}
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "Nil payload",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "Client error",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return((*camundav88.GetProcessDefinitionXMLResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProcessDefinitionClient{}
			if tt.setupMock != nil {
				tt := tt
				tt.setupMock(m)
			}

			svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
			require.NoError(t, err)
			xml, err := svc.GetProcessDefinitionXML(ctx, "123")

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedXML, xml)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestService_WithClientCamunda(t *testing.T) {
	mc := &mockProcessDefinitionClient{}
	svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(mc))
	require.NoError(t, err)
	require.Equal(t, mc, svc.ClientCamunda())

	v88.WithClientCamunda(nil)(svc)
	require.Equal(t, mc, svc.ClientCamunda())
}

func TestService_WithLogger(t *testing.T) {
	svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v88.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v88.WithLogger(nil)(svc)
	require.Equal(t, logger, svc.Logger())
}

func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "https://camunda.local/v2",
			},
		},
	}
}

func makeProcessDefinitionResult(id, key string, version int32) camundav88.ProcessDefinitionResult {
	return camundav88.ProcessDefinitionResult{
		ProcessDefinitionId:  id,
		ProcessDefinitionKey: key,
		Name:                 new("name-" + id),
		Version:              version,
		TenantId:             "tenant",
		VersionTag:           new("tag"),
	}
}

func ptr[T any](v T) *T {
	return &v
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
