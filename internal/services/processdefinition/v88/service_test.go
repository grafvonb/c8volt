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

func (m *mockProcessDefinitionClient) SearchProcessInstancesWithResponse(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error) {
	args := m.Called(ctx, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav88.SearchProcessInstancesResponse), args.Error(1)
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
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
				mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnumACTIVE, 11)
				mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnumCOMPLETED, 22)
				mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnum(domain.StateTerminated), 33)
				mockProcessInstanceIncidentCount(m, t, "123", "", 2)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.NotNil(t, defs[0].Statistics)
				assert.Equal(t, int64(11), defs[0].Statistics.Active)
				assert.Equal(t, int64(22), defs[0].Statistics.Completed)
				assert.Equal(t, int64(33), defs[0].Statistics.Canceled)
				assert.Equal(t, int64(2), defs[0].Statistics.Incidents)
				assert.True(t, defs[0].Statistics.IncidentCountSupported)
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
				m.On("SearchProcessInstancesWithResponse", mock.Anything, mock.Anything).Return((*camundav88.SearchProcessInstancesResponse)(nil), mockErr)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: mockErr,
		},
		{
			name: "Stats missing payload fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
						Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
					},
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
				m.On("SearchProcessInstancesWithResponse", mock.Anything, mock.Anything).Return(&camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-instances/search", http.StatusOK, "200"),
				}, nil)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: domain.ErrMalformedResponse,
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

	m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			body := args.Get(1).(camundav88.SearchProcessDefinitionsJSONRequestBody)
			require.NotNil(t, body.Filter.IsLatestVersion)
			assert.True(t, *body.Filter.IsLatestVersion)
			assert.Nil(t, body.Filter.TenantId)
			page, err := body.Page.AsCursorForwardPagination()
			require.NoError(t, err)
			assert.Equal(t, camundav88.EndCursor(""), page.After)
			assert.Equal(t, int32(1000), *page.Limit)
			require.NotNil(t, body.Sort)
			require.Len(t, *body.Sort, 2)
			assert.Equal(t, camundav88.ProcessDefinitionSearchQuerySortRequestFieldProcessDefinitionId, (*body.Sort)[0].Field)
			assert.Equal(t, camundav88.ProcessDefinitionSearchQuerySortRequestFieldTenantId, (*body.Sort)[1].Field)
		}).
		Return(resp, nil)
	mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnumACTIVE, 7)
	mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnumCOMPLETED, 8)
	mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnum(domain.StateTerminated), 9)
	mockProcessInstanceIncidentCount(m, t, "123", "", 0)

	svc, err := v88.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
	require.NoError(t, err)
	defs, err := svc.SearchProcessDefinitionsLatest(ctx, domain.ProcessDefinitionFilter{}, services.WithStat())

	require.NoError(t, err)
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].Statistics)
	assert.Equal(t, int64(7), defs[0].Statistics.Active)
	assert.Equal(t, int64(8), defs[0].Statistics.Completed)
	assert.Equal(t, int64(9), defs[0].Statistics.Canceled)
	assert.Zero(t, defs[0].Statistics.Incidents)
	assert.True(t, defs[0].Statistics.IncidentCountSupported)
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

func TestService_SearchProcessDefinitions_IncidentSearchIncludesTenantFilterWhenConfigured(t *testing.T) {
	ctx := context.Background()
	m := &mockProcessDefinitionClient{}

	resp := &camundav88.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200 OK"),
		JSON200: &camundav88.ProcessDefinitionSearchQueryResult{
			Items: []camundav88.ProcessDefinitionResult{makeProcessDefinitionResult("proc", "123", 2)},
		},
	}
	m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
	mockProcessInstanceStateCount(m, t, "123", "tenant-a", camundav88.ProcessInstanceStateEnumACTIVE, 4)
	mockProcessInstanceStateCount(m, t, "123", "tenant-a", camundav88.ProcessInstanceStateEnumCOMPLETED, 5)
	mockProcessInstanceStateCount(m, t, "123", "tenant-a", camundav88.ProcessInstanceStateEnum(domain.StateTerminated), 6)
	mockProcessInstanceIncidentCount(m, t, "123", "tenant-a", 1)

	cfg := testConfig()
	cfg.App.Tenant = "tenant-a"
	svc, err := v88.New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClientCamunda(m))
	require.NoError(t, err)

	defs, err := svc.SearchProcessDefinitions(ctx, domain.ProcessDefinitionFilter{}, 25, services.WithStat())
	require.NoError(t, err)
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].Statistics)
	assert.Equal(t, int64(1), defs[0].Statistics.Incidents)
	assert.True(t, defs[0].Statistics.IncidentCountSupported)
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
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnumACTIVE, 12)
				mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnumCOMPLETED, 13)
				mockProcessInstanceStateCount(m, t, "123", "", camundav88.ProcessInstanceStateEnum(domain.StateTerminated), 14)
				mockProcessInstanceIncidentCount(m, t, "123", "", 9)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, pd domain.ProcessDefinition) {
				require.NotNil(t, pd.Statistics)
				assert.Equal(t, int64(12), pd.Statistics.Active)
				assert.Equal(t, int64(13), pd.Statistics.Completed)
				assert.Equal(t, int64(14), pd.Statistics.Canceled)
				assert.Equal(t, int64(9), pd.Statistics.Incidents)
				assert.True(t, pd.Statistics.IncidentCountSupported)
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
				m.On("SearchProcessInstancesWithResponse", mock.Anything, mock.Anything).Return((*camundav88.SearchProcessInstancesResponse)(nil), mockErr)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: mockErr,
		},
		{
			name: "Stats missing payload fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav88.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				m.On("SearchProcessInstancesWithResponse", mock.Anything, mock.Anything).Return(&camundav88.SearchProcessInstancesResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-instances/search", http.StatusOK, "200"),
				}, nil)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: domain.ErrMalformedResponse,
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

func makeSearchProcessInstancesResponse(total int64) *camundav88.SearchProcessInstancesResponse {
	return &camundav88.SearchProcessInstancesResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-instances/search", http.StatusOK, "200"),
		JSON200: &camundav88.ProcessInstanceSearchQueryResult{
			Items: []camundav88.ProcessInstanceResult{},
			Page: camundav88.SearchQueryPageResponse{
				TotalItems: total,
			},
		},
	}
}

func mockProcessInstanceStateCount(m *mockProcessDefinitionClient, t *testing.T, processDefinitionKey, tenantID string, state camundav88.ProcessInstanceStateEnum, total int64) {
	m.On("SearchProcessInstancesWithResponse", mock.Anything, mock.MatchedBy(func(body camundav88.SearchProcessInstancesJSONRequestBody) bool {
		return processInstanceSearchMatches(body, processDefinitionKey, tenantID, state)
	})).Return(makeSearchProcessInstancesResponse(total), nil).Once()
}

func mockProcessInstanceIncidentCount(m *mockProcessDefinitionClient, t *testing.T, processDefinitionKey, tenantID string, total int64) {
	m.On("SearchProcessInstancesWithResponse", mock.Anything, mock.MatchedBy(func(body camundav88.SearchProcessInstancesJSONRequestBody) bool {
		return processInstanceIncidentSearchMatches(body, processDefinitionKey, tenantID)
	})).Return(makeSearchProcessInstancesResponse(total), nil).Once()
}

func processInstanceSearchMatches(body camundav88.SearchProcessInstancesJSONRequestBody, processDefinitionKey, tenantID string, expectedState camundav88.ProcessInstanceStateEnum) bool {
	if body.Filter == nil || body.Filter.ProcessDefinitionKey == nil || body.Filter.State == nil || body.Page == nil {
		return false
	}
	pdKey, err := body.Filter.ProcessDefinitionKey.AsProcessDefinitionKeyFilterProperty0()
	if err != nil || string(pdKey) != processDefinitionKey {
		return false
	}
	state, err := body.Filter.State.AsProcessInstanceStateFilterProperty0()
	if err != nil || state != expectedState {
		return false
	}
	if tenantID == "" {
		if body.Filter.TenantId != nil {
			return false
		}
	} else {
		if body.Filter.TenantId == nil {
			return false
		}
		actualTenant, err := body.Filter.TenantId.AsStringFilterProperty0()
		if err != nil || actualTenant != tenantID {
			return false
		}
	}
	page, err := body.Page.AsOffsetPagination()
	if err != nil || page.From == nil || page.Limit == nil {
		return false
	}
	return *page.From == 0 && *page.Limit == 1
}

func processInstanceIncidentSearchMatches(body camundav88.SearchProcessInstancesJSONRequestBody, processDefinitionKey, tenantID string) bool {
	if body.Filter == nil || body.Filter.ProcessDefinitionKey == nil || body.Page == nil || body.Filter.HasIncident == nil || !*body.Filter.HasIncident {
		return false
	}
	if body.Filter.State != nil {
		return false
	}
	pdKey, err := body.Filter.ProcessDefinitionKey.AsProcessDefinitionKeyFilterProperty0()
	if err != nil || string(pdKey) != processDefinitionKey {
		return false
	}
	if tenantID == "" {
		if body.Filter.TenantId != nil {
			return false
		}
	} else {
		if body.Filter.TenantId == nil {
			return false
		}
		actualTenant, err := body.Filter.TenantId.AsStringFilterProperty0()
		if err != nil || actualTenant != tenantID {
			return false
		}
	}
	page, err := body.Page.AsOffsetPagination()
	if err != nil || page.From == nil || page.Limit == nil {
		return false
	}
	return *page.From == 0 && *page.Limit == 1
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
