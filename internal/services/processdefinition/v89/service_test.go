// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v89 "github.com/grafvonb/c8volt/internal/services/processdefinition/v89"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProcessDefinitionClient struct {
	mock.Mock
}

func (m *mockProcessDefinitionClient) SearchProcessDefinitionsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessDefinitionsResponse, error) {
	rawBody, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	args := m.Called(ctx, contentType, string(rawBody))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav89.SearchProcessDefinitionsResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionWithResponse(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav89.GetProcessDefinitionResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionXMLWithResponse(ctx context.Context, key string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionXMLResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav89.GetProcessDefinitionXMLResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) SearchProcessInstancesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error) {
	rawBody, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	args := m.Called(ctx, contentType, string(rawBody))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav89.SearchProcessInstancesResponse), args.Error(1)
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
				resp := &camundav89.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200 OK"),
					Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant","version":2,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
					JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
				}
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
			},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "proc", defs[0].BpmnProcessId)
			},
		},
		{
			name: "HTTP error",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusInternalServerError, "500"),
					Body:         []byte("boom"),
				}
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "Nil payload",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
				}
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "Client error",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return((*camundav89.SearchProcessDefinitionsResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "Stats requested",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant","version":2,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
					JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
				}
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
				mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnumACTIVE, 10)
				mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnumCOMPLETED, 20)
				mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnum(domain.StateTerminated), 30)
				mockProcessInstanceIncidentCount(m, t, "123", "", 4)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.NotNil(t, defs[0].Statistics)
				assert.Equal(t, int64(10), defs[0].Statistics.Active)
				assert.Equal(t, int64(20), defs[0].Statistics.Completed)
				assert.Equal(t, int64(30), defs[0].Statistics.Canceled)
				assert.Equal(t, int64(4), defs[0].Statistics.Incidents)
				assert.True(t, defs[0].Statistics.IncidentCountSupported)
			},
		},
		{
			name: "Stats retrieval fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant","version":2,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
					JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
				}
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
				m.On("SearchProcessInstancesWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return((*camundav89.SearchProcessInstancesResponse)(nil), mockErr)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: mockErr,
		},
		{
			name: "Stats missing payload fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
					Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant","version":2,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
					JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
				}
				m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
				m.On("SearchProcessInstancesWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(&camundav89.SearchProcessInstancesResponse{
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

			svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(m))
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

	resp := &camundav89.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200"),
		Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant","version":1,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
		JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
	}

	m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).
		Run(func(args mock.Arguments) {
			body := decodeProcessDefinitionSearchRequest(t, args.String(2))
			require.NotNil(t, body.Filter.IsLatestVersion)
			assert.True(t, *body.Filter.IsLatestVersion)
			assert.Nil(t, body.Filter.TenantID)
			assert.NotNil(t, body.Page.After)
			assert.Equal(t, "", *body.Page.After)
			require.NotNil(t, body.Page.Limit)
			assert.Equal(t, int32(1000), *body.Page.Limit)
			assert.Nil(t, body.Page.From)
			require.Len(t, body.Sort, 2)
			assert.Equal(t, "processDefinitionId", body.Sort[0].Field)
			assert.Equal(t, "tenantId", body.Sort[1].Field)
		}).
		Return(resp, nil)
	mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnumACTIVE, 3)
	mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnumCOMPLETED, 4)
	mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnum(domain.StateTerminated), 5)
	mockProcessInstanceIncidentCount(m, t, "123", "", 0)

	svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(m))
	require.NoError(t, err)
	defs, err := svc.SearchProcessDefinitionsLatest(ctx, domain.ProcessDefinitionFilter{}, services.WithStat())

	require.NoError(t, err)
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].Statistics)
	assert.Equal(t, int64(3), defs[0].Statistics.Active)
	assert.Equal(t, int64(4), defs[0].Statistics.Completed)
	assert.Equal(t, int64(5), defs[0].Statistics.Canceled)
	assert.Zero(t, defs[0].Statistics.Incidents)
	assert.True(t, defs[0].Statistics.IncidentCountSupported)
	m.AssertExpectations(t)
}

func TestService_SearchProcessDefinitions_IncidentSearchIncludesTenantFilterWhenConfigured(t *testing.T) {
	ctx := context.Background()
	m := &mockProcessDefinitionClient{}

	resp := &camundav89.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200 OK"),
		Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant-a","version":2,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
		JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
	}
	m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(resp, nil)
	mockProcessInstanceStateCount(m, t, "123", "tenant-a", camundav89.ProcessInstanceStateEnumACTIVE, 5)
	mockProcessInstanceStateCount(m, t, "123", "tenant-a", camundav89.ProcessInstanceStateEnumCOMPLETED, 6)
	mockProcessInstanceStateCount(m, t, "123", "tenant-a", camundav89.ProcessInstanceStateEnum(domain.StateTerminated), 7)
	mockProcessInstanceIncidentCount(m, t, "123", "tenant-a", 3)

	cfg := testConfig()
	cfg.App.Tenant = "tenant-a"
	svc, err := v89.New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(m))
	require.NoError(t, err)

	defs, err := svc.SearchProcessDefinitions(ctx, domain.ProcessDefinitionFilter{}, 25, services.WithStat())
	require.NoError(t, err)
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].Statistics)
	assert.Equal(t, int64(3), defs[0].Statistics.Incidents)
	assert.True(t, defs[0].Statistics.IncidentCountSupported)
	m.AssertExpectations(t)
}

func TestService_SearchProcessDefinitions_IncludesTenantFilterWhenConfigured(t *testing.T) {
	ctx := context.Background()
	m := &mockProcessDefinitionClient{}

	resp := &camundav89.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-definitions", http.StatusOK, "200 OK"),
		Body:         []byte(`{"items":[{"hasStartForm":false,"name":"name-proc","processDefinitionId":"proc","processDefinitionKey":"123","resourceName":"proc.bpmn","tenantId":"tenant-a","version":2,"versionTag":"tag"}],"page":{"hasMoreTotalItems":false,"totalItems":1}}`),
		JSON200:      &camundav89.ProcessDefinitionSearchQueryResult{},
	}

	m.On("SearchProcessDefinitionsWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).
		Run(func(args mock.Arguments) {
			body := decodeProcessDefinitionSearchRequest(t, args.String(2))
			require.NotNil(t, body.Filter.TenantID)
			assert.Equal(t, "tenant-a", *body.Filter.TenantID)
		}).
		Return(resp, nil)

	cfg := testConfig()
	cfg.App.Tenant = "tenant-a"
	svc, err := v89.New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(m))
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
				resp := &camundav89.GetProcessDefinitionResponse{
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
				resp := &camundav89.GetProcessDefinitionResponse{
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
				resp := &camundav89.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "Stats requested",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnumACTIVE, 5)
				mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnumCOMPLETED, 15)
				mockProcessInstanceStateCount(m, t, "123", "", camundav89.ProcessInstanceStateEnum(domain.StateTerminated), 25)
				mockProcessInstanceIncidentCount(m, t, "123", "", 9)
			},
			opts: []services.CallOption{services.WithStat()},
			assertResult: func(t *testing.T, pd domain.ProcessDefinition) {
				require.NotNil(t, pd.Statistics)
				assert.Equal(t, int64(5), pd.Statistics.Active)
				assert.Equal(t, int64(15), pd.Statistics.Completed)
				assert.Equal(t, int64(25), pd.Statistics.Canceled)
				assert.Equal(t, int64(9), pd.Statistics.Incidents)
				assert.True(t, pd.Statistics.IncidentCountSupported)
			},
		},
		{
			name: "Stats retrieval fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				m.On("SearchProcessInstancesWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return((*camundav89.SearchProcessInstancesResponse)(nil), mockErr)
			},
			opts:          []services.CallOption{services.WithStat()},
			expectedError: mockErr,
		},
		{
			name: "Stats missing payload fails",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.GetProcessDefinitionResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinitionResult("proc", "123", 2)),
				}
				m.On("GetProcessDefinitionWithResponse", mock.Anything, "123").Return(resp, nil)
				m.On("SearchProcessInstancesWithBodyWithResponse", mock.Anything, "application/json", mock.Anything).Return(&camundav89.SearchProcessInstancesResponse{
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

			svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(m))
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
				resp := &camundav89.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
					XML200:       new(string),
				}
				*resp.XML200 = "<definitions id=\"proc\"/>"
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\"/>",
		},
		{
			name: "Success falls back to raw body",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
					Body:         []byte("<definitions id=\"proc\"/>"),
					XML200:       new(string),
				}
				*resp.XML200 = "\n  \n  \n"
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\"/>",
		},
		{
			name: "Success prefers formatted raw body",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
					Body:         []byte("<definitions id=\"proc\">\n  <process id=\"order\" />\n</definitions>\n"),
					XML200:       new(string),
				}
				*resp.XML200 = "\n  \n    \n"
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\">\n  <process id=\"order\" />\n</definitions>\n",
		},
		{
			name: "HTTP error",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &camundav89.GetProcessDefinitionXMLResponse{
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
				resp := &camundav89.GetProcessDefinitionXMLResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://example.com/v2/process-definitions/123/xml", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "Client error",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("GetProcessDefinitionXMLWithResponse", mock.Anything, "123").Return((*camundav89.GetProcessDefinitionXMLResponse)(nil), mockErr)
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

			svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(m))
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
	svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v89.WithClientCamunda(mc))
	require.NoError(t, err)
	require.Equal(t, mc, svc.ClientCamunda())

	v89.WithClientCamunda(nil)(svc)
	require.Equal(t, mc, svc.ClientCamunda())
}

func TestService_WithLogger(t *testing.T) {
	svc, err := v89.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v89.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v89.WithLogger(nil)(svc)
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

func makeProcessDefinitionResult(id, key string, version int32) camundav89.ProcessDefinitionResult {
	return camundav89.ProcessDefinitionResult{
		ProcessDefinitionId:  id,
		ProcessDefinitionKey: key,
		Name:                 new("name-" + id),
		Version:              version,
		TenantId:             "tenant",
		VersionTag:           new("tag"),
	}
}

func makeSearchProcessInstancesResponse(total int64) *camundav89.SearchProcessInstancesResponse {
	return &camundav89.SearchProcessInstancesResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/v2/process-instances/search", http.StatusOK, "200"),
		JSON200: &camundav89.ProcessInstanceSearchQueryResult{
			Page: camundav89.SearchQueryPageResponse{
				TotalItems: total,
			},
		},
	}
}

func mockProcessInstanceStateCount(m *mockProcessDefinitionClient, t *testing.T, processDefinitionKey, tenantID string, state camundav89.ProcessInstanceStateEnum, total int64) {
	m.On("SearchProcessInstancesWithBodyWithResponse", mock.Anything, "application/json", mock.MatchedBy(func(raw string) bool {
		return processInstanceSearchMatches(raw, processDefinitionKey, tenantID, state)
	})).Return(makeSearchProcessInstancesResponse(total), nil).Once()
}

func mockProcessInstanceIncidentCount(m *mockProcessDefinitionClient, t *testing.T, processDefinitionKey, tenantID string, total int64) {
	m.On("SearchProcessInstancesWithBodyWithResponse", mock.Anything, "application/json", mock.MatchedBy(func(raw string) bool {
		return processInstanceIncidentSearchMatches(raw, processDefinitionKey, tenantID)
	})).Return(makeSearchProcessInstancesResponse(total), nil).Once()
}

func processInstanceSearchMatches(raw string, processDefinitionKey, tenantID string, expectedState camundav89.ProcessInstanceStateEnum) bool {
	var body camundav89.SearchProcessInstancesJSONRequestBody
	err := json.Unmarshal([]byte(raw), &body)
	if err != nil || body.Filter == nil || body.Filter.ProcessDefinitionKey == nil || body.Filter.State == nil || body.Page == nil {
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

func processInstanceIncidentSearchMatches(raw string, processDefinitionKey, tenantID string) bool {
	var body camundav89.SearchProcessInstancesJSONRequestBody
	err := json.Unmarshal([]byte(raw), &body)
	if err != nil || body.Filter == nil || body.Filter.ProcessDefinitionKey == nil || body.Page == nil || body.Filter.HasIncident == nil || !*body.Filter.HasIncident {
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

type processDefinitionSearchRequest struct {
	Filter struct {
		TenantID        *string `json:"tenantId"`
		IsLatestVersion *bool   `json:"isLatestVersion"`
	} `json:"filter"`
	Page struct {
		After *string `json:"after"`
		From  *int32  `json:"from"`
		Limit *int32  `json:"limit"`
	} `json:"page"`
	Sort []struct {
		Field string `json:"field"`
		Order string `json:"order"`
	} `json:"sort"`
}

func decodeProcessDefinitionSearchRequest(t *testing.T, raw string) processDefinitionSearchRequest {
	t.Helper()

	var body processDefinitionSearchRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &body))
	return body
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
