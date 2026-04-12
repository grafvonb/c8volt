package v87_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafvonb/c8volt/config"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/processdefinition/v87"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProcessDefinitionClient struct {
	mock.Mock
}

func (m *mockProcessDefinitionClient) SearchProcessDefinitionsWithResponse(ctx context.Context, body operatev87.SearchProcessDefinitionsJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessDefinitionsResponse, error) {
	args := m.Called(ctx, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operatev87.SearchProcessDefinitionsResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessDefinitionByKeyResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operatev87.GetProcessDefinitionByKeyResponse), args.Error(1)
}

func (m *mockProcessDefinitionClient) GetProcessDefinitionAsXmlByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessDefinitionAsXmlByKeyResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*operatev87.GetProcessDefinitionAsXmlByKeyResponse), args.Error(1)
}

func TestService_SearchProcessDefinitions(t *testing.T) {
	ctx := context.Background()
	mockErr := errors.New("search failed")

	samplePD := makeProcessDefinition(1, "proc", 2)
	successResp := &operatev87.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/search", http.StatusOK, "200 OK"),
		JSON200: &operatev87.ResultsProcessDefinition{
			Items: &[]operatev87.ProcessDefinition{samplePD},
		},
	}

	tests := []struct {
		name              string
		opts              []services.CallOption
		setupMock         func(*mockProcessDefinitionClient)
		expectedError     error
		expectedErrSubstr string
		assertResult      func(*testing.T, []domain.ProcessDefinition)
	}{
		{
			name: "Success",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
					Return(successResp, nil)
			},
			assertResult: func(t *testing.T, defs []domain.ProcessDefinition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "proc", defs[0].BpmnProcessId)
			},
		},
		{
			name:              "StatsNotSupported",
			opts:              []services.CallOption{services.WithStat()},
			expectedErrSubstr: "not supported",
		},
		{
			name: "ClientError",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
					Return((*operatev87.SearchProcessDefinitionsResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTPError",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/search", http.StatusInternalServerError, "500"),
					Body:         []byte("boom"),
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
					Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "NilPayload",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.SearchProcessDefinitionsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/search", http.StatusOK, "200"),
				}
				m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).
					Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProcessDefinitionClient{}
			if tt.setupMock != nil {
				tt.setupMock(m)
			}

			svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v87.WithClientOperate(m))
			require.NoError(t, err)

			defs, err := svc.SearchProcessDefinitions(ctx, domain.ProcessDefinitionFilter{BpmnProcessId: "proc"}, 10, tt.opts...)

			if tt.expectedErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrSubstr)
				m.AssertExpectations(t)
				return
			}

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				tt.assertResult(t, defs)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestService_SearchProcessDefinitionsLatest(t *testing.T) {
	ctx := context.Background()
	m := &mockProcessDefinitionClient{}

	resp := &operatev87.SearchProcessDefinitionsResponse{
		HTTPResponse: newHTTPResponse(http.MethodPost, "https://operate.local/search", http.StatusOK, "200"),
		JSON200: &operatev87.ResultsProcessDefinition{
			Items: &[]operatev87.ProcessDefinition{
				makeProcessDefinition(1, "alpha", 1),
				makeProcessDefinition(2, "alpha", 3),
				makeProcessDefinition(3, "beta", 2),
			},
		},
	}

	m.On("SearchProcessDefinitionsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)

	svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v87.WithClientOperate(m))
	require.NoError(t, err)

	defs, err := svc.SearchProcessDefinitionsLatest(ctx, domain.ProcessDefinitionFilter{})
	require.NoError(t, err)
	require.Len(t, defs, 2)
	assert.Equal(t, "alpha", defs[0].BpmnProcessId)
	assert.Equal(t, int32(3), defs[0].ProcessVersion)
	assert.Equal(t, "beta", defs[1].BpmnProcessId)
	m.AssertExpectations(t)
}

func TestService_GetProcessDefinition(t *testing.T) {
	ctx := context.Background()
	mockErr := errors.New("get failed")

	tests := []struct {
		name              string
		key               string
		opts              []services.CallOption
		setupMock         func(*mockProcessDefinitionClient)
		expectedError     error
		expectedErrSubstr string
		assertResult      func(*testing.T, domain.ProcessDefinition)
	}{
		{
			name: "Success",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123", http.StatusOK, "200"),
					JSON200:      new(makeProcessDefinition(1, "proc", 2)),
				}
				m.On("GetProcessDefinitionByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			assertResult: func(t *testing.T, pd domain.ProcessDefinition) {
				assert.Equal(t, "proc", pd.BpmnProcessId)
			},
		},
		{
			name:              "StatsNotSupported",
			key:               "123",
			opts:              []services.CallOption{services.WithStat()},
			expectedErrSubstr: "not supported",
		},
		{
			name:              "KeyConversionError",
			key:               "abc",
			expectedErrSubstr: "converting process definition key",
		},
		{
			name: "ClientError",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("GetProcessDefinitionByKeyWithResponse", mock.Anything, int64(123)).
					Return((*operatev87.GetProcessDefinitionByKeyResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTPError",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123", http.StatusInternalServerError, "500"),
					Body:         []byte("fail"),
				}
				m.On("GetProcessDefinitionByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "NilPayload",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProcessDefinitionClient{}
			if tt.setupMock != nil {
				tt.setupMock(m)
			}

			svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v87.WithClientOperate(m))
			require.NoError(t, err)

			pd, err := svc.GetProcessDefinition(ctx, tt.key, tt.opts...)

			if tt.expectedErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrSubstr)
			} else if tt.expectedError != nil {
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
		name              string
		key               string
		opts              []services.CallOption
		setupMock         func(*mockProcessDefinitionClient)
		expectedError     error
		expectedErrSubstr string
		expectedXML       string
	}{
		{
			name: "Success",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionAsXmlByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123/xml", http.StatusOK, "200"),
					XML200:       new("<definitions id=\"proc\"/>"),
				}
				m.On("GetProcessDefinitionAsXmlByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\"/>",
		},
		{
			name: "SuccessFallsBackToRawBody",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionAsXmlByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123/xml", http.StatusOK, "200"),
					Body:         []byte("<definitions id=\"proc\"/>"),
					XML200:       new("\n  \n  \n"),
				}
				m.On("GetProcessDefinitionAsXmlByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\"/>",
		},
		{
			name: "SuccessPrefersFormattedRawBody",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionAsXmlByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123/xml", http.StatusOK, "200"),
					Body:         []byte("<definitions id=\"proc\">\n  <process id=\"order\" />\n</definitions>\n"),
					XML200:       new("\n  \n    \n"),
				}
				m.On("GetProcessDefinitionAsXmlByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedXML: "<definitions id=\"proc\">\n  <process id=\"order\" />\n</definitions>\n",
		},
		{
			name:              "StatsNotSupported",
			key:               "123",
			opts:              []services.CallOption{services.WithStat()},
			expectedErrSubstr: "not supported",
		},
		{
			name:              "KeyConversionError",
			key:               "abc",
			expectedErrSubstr: "converting process definition key",
		},
		{
			name: "ClientError",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				m.On("GetProcessDefinitionAsXmlByKeyWithResponse", mock.Anything, int64(123)).
					Return((*operatev87.GetProcessDefinitionAsXmlByKeyResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTPError",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionAsXmlByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123/xml", http.StatusInternalServerError, "500"),
					Body:         []byte("fail"),
				}
				m.On("GetProcessDefinitionAsXmlByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "NilPayload",
			key:  "123",
			setupMock: func(m *mockProcessDefinitionClient) {
				resp := &operatev87.GetProcessDefinitionAsXmlByKeyResponse{
					HTTPResponse: newHTTPResponse(http.MethodGet, "https://operate.local/process/123/xml", http.StatusOK, "200"),
				}
				m.On("GetProcessDefinitionAsXmlByKeyWithResponse", mock.Anything, int64(123)).Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProcessDefinitionClient{}
			if tt.setupMock != nil {
				tt.setupMock(m)
			}

			svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v87.WithClientOperate(m))
			require.NoError(t, err)

			xml, err := svc.GetProcessDefinitionXML(ctx, tt.key, tt.opts...)

			if tt.expectedErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrSubstr)
			} else if tt.expectedError != nil {
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

func TestService_WithClientOperate(t *testing.T) {
	mc := &mockProcessDefinitionClient{}
	svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v87.WithClientOperate(mc))
	require.NoError(t, err)
	require.Equal(t, mc, svc.ClientOperate())

	v87.WithClientOperate(nil)(svc)
	require.Equal(t, mc, svc.ClientOperate())
}

func TestService_WithLogger(t *testing.T) {
	svc, err := v87.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v87.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v87.WithLogger(nil)(svc)
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

func makeProcessDefinition(key int64, id string, version int32) operatev87.ProcessDefinition {
	return operatev87.ProcessDefinition{
		Key:           new(key),
		BpmnProcessId: new(id),
		Version:       new(version),
		Name:          new("name-" + id),
	}
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
