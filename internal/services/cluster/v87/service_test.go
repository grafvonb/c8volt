// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	v87 "github.com/grafvonb/c8volt/internal/services/cluster/v87"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/grafvonb/c8volt/config"
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

type mockClusterClient struct {
	mock.Mock
}

func (m *mockClusterClient) GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetTopologyResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav87.GetTopologyResponse), args.Error(1)
}

func (m *mockClusterClient) GetLicenseWithResponse(ctx context.Context, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetLicenseResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav87.GetLicenseResponse), args.Error(1)
}

var testTopologyURL = &url.URL{Scheme: "https", Host: "camunda.local", Path: "/cluster/topology"}
var testLicenseURL = &url.URL{Scheme: "https", Host: "camunda.local", Path: "/cluster/license"}

// newHTTPResponse builds a minimal v8.7 cluster HTTP response for error normalization tests.
func newHTTPResponse(u *url.URL, statusCode int, status string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Request: &http.Request{
			Method: http.MethodGet,
			URL:    u,
		},
	}
}

func TestService_GetClusterTopology(t *testing.T) {
	mockErr := errors.New("network error")
	ctx := context.Background()

	validTopology := camundav87.TopologyResponse{
		ClusterSize:       new(int32(3)),
		GatewayVersion:    new("8.7.0"),
		PartitionsCount:   new(int32(3)),
		ReplicationFactor: new(int32(3)),
		Brokers: new([]camundav87.BrokerInfo{
			{
				Host:   new("broker-0"),
				NodeId: new(int32(0)),
				Port:   new(int32(26501)),
				Partitions: new([]camundav87.Partition{
					{
						PartitionId: new(int32(1)),
						Role:        new(camundav87.Leader),
						Health:      new(camundav87.Healthy),
					},
				}),
			},
		}),
	}

	tests := []struct {
		name          string
		setupMock     func(*mockClusterClient)
		expectedError error
		validateResp  func(*testing.T, d.Topology)
	}{
		{
			name: "Success",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return(&camundav87.GetTopologyResponse{
					JSON200:      &validTopology,
					HTTPResponse: newHTTPResponse(testTopologyURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			validateResp: func(t *testing.T, top d.Topology) {
				assert.Equal(t, "8.7.0", top.GatewayVersion)
				assert.Equal(t, int32(3), top.ClusterSize)
				assert.Len(t, top.Brokers, 1)
				assert.Equal(t, int32(0), top.Brokers[0].NodeId)
			},
		},
		{
			name: "Client Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return((*camundav87.GetTopologyResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTP Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return(&camundav87.GetTopologyResponse{
					HTTPResponse: newHTTPResponse(testTopologyURL, http.StatusInternalServerError, "500 Internal Server Error"),
					Body:         []byte("error"),
				}, nil)
			},
			expectedError: d.ErrInternal,
		},
		{
			name: "Empty Payload",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return(&camundav87.GetTopologyResponse{
					JSON200:      nil,
					HTTPResponse: newHTTPResponse(testTopologyURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			expectedError: d.ErrMalformedResponse,
		},
		{
			name: "Nil Response",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return((*camundav87.GetTopologyResponse)(nil), nil)
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockClusterClient{}
			if tt.setupMock != nil {
				tt.setupMock(m)
			}

			svc, err := v87.New(&config.Config{
				APIs: config.APIs{
					Camunda: config.API{
						BaseURL: "http://localhost:8080/v2",
					},
				},
			}, &http.Client{}, slog.Default(), v87.WithClient(m))
			require.NoError(t, err)

			resp, err := svc.GetClusterTopology(ctx)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.validateResp != nil {
					tt.validateResp(t, resp)
				}
			}
			m.AssertExpectations(t)
		})
	}
}

func TestService_GetClusterLicense(t *testing.T) {
	mockErr := errors.New("network error")
	ctx := context.Background()

	validLicense := camundav87.LicenseResponse{
		LicenseType:  new("SaaS"),
		ValidLicense: new(true),
	}

	tests := []struct {
		name          string
		setupMock     func(*mockClusterClient)
		expectedError error
		validateResp  func(*testing.T, d.License)
	}{
		{
			name: "Success",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return(&camundav87.GetLicenseResponse{
					JSON200:      &validLicense,
					HTTPResponse: newHTTPResponse(testLicenseURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			validateResp: func(t *testing.T, lic d.License) {
				assert.Equal(t, "SaaS", lic.LicenseType)
				assert.True(t, lic.ValidLicense)
				assert.Nil(t, lic.ExpiresAt)
				assert.Nil(t, lic.IsCommercial)
			},
		},
		{
			name: "Client Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return((*camundav87.GetLicenseResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTP Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return(&camundav87.GetLicenseResponse{
					HTTPResponse: newHTTPResponse(testLicenseURL, http.StatusInternalServerError, "500 Internal Server Error"),
					Body:         []byte("error"),
				}, nil)
			},
			expectedError: d.ErrInternal,
		},
		{
			name: "Empty Payload",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return(&camundav87.GetLicenseResponse{
					JSON200:      nil,
					HTTPResponse: newHTTPResponse(testLicenseURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			expectedError: d.ErrMalformedResponse,
		},
		{
			name: "Nil Response",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return((*camundav87.GetLicenseResponse)(nil), nil)
			},
			expectedError: d.ErrMalformedResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockClusterClient{}
			if tt.setupMock != nil {
				tt.setupMock(m)
			}

			svc, err := v87.New(&config.Config{
				APIs: config.APIs{
					Camunda: config.API{
						BaseURL: "http://localhost:8080/v2",
					},
				},
			}, &http.Client{}, slog.Default(), v87.WithClient(m))
			require.NoError(t, err)

			resp, err := svc.GetClusterLicense(ctx)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.validateResp != nil {
					tt.validateResp(t, resp)
				}
			}
			m.AssertExpectations(t)
		})
	}
}

func TestService_GetClusterTopology_FakeServer(t *testing.T) {
	fs := testx.NewFakeServer(t)
	cfg := testx.TestConfig(t)
	cfg.APIs.Camunda.BaseURL = fs.BaseURL
	log := testx.Logger(t)
	ctx := testx.ITCtx(t, time.Second*10)

	svc, err := v87.New(cfg, fs.FS.Client(), log)
	require.NoError(t, err)

	resp, err := svc.GetClusterTopology(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.ClusterSize)
	require.Len(t, resp.Brokers, 1)
	require.Equal(t, int32(0), resp.Brokers[0].NodeId)
	require.Equal(t, "8.7.0", resp.GatewayVersion)
}

func TestService_GetClusterLicense_FakeServer(t *testing.T) {
	fs := testx.NewFakeServer(t)
	cfg := testx.TestConfig(t)
	cfg.APIs.Camunda.BaseURL = fs.BaseURL
	log := testx.Logger(t)
	ctx := testx.ITCtx(t, time.Second*10)

	svc, err := v87.New(cfg, fs.FS.Client(), log)
	require.NoError(t, err)

	resp, err := svc.GetClusterLicense(ctx)
	require.NoError(t, err)
	require.Equal(t, "SaaS", resp.LicenseType)
	require.True(t, resp.ValidLicense)
	require.Nil(t, resp.ExpiresAt)
	require.Nil(t, resp.IsCommercial)
}

func TestService_WithClient(t *testing.T) {
	t.Run("non-nil client", func(t *testing.T) {
		mc := &mockClusterClient{}
		svc, err := v87.New(&config.Config{
			APIs: config.APIs{
				Camunda: config.API{
					BaseURL: "http://localhost:8080/v2",
				},
			},
		}, &http.Client{}, slog.Default())
		require.NoError(t, err)
		v87.WithClient(mc)(svc)
		require.Equal(t, mc, svc.Client())
	})
	t.Run("nil client does not override", func(t *testing.T) {
		originalClient := &mockClusterClient{}
		svc, err := v87.New(&config.Config{
			APIs: config.APIs{
				Camunda: config.API{
					BaseURL: "http://localhost:8080/v2",
				},
			},
		}, &http.Client{}, slog.Default(), v87.WithClient(originalClient))
		require.NoError(t, err)
		v87.WithClient(nil)(svc)
		require.Equal(t, originalClient, svc.Client())
	})
}

func TestService_WithLogger(t *testing.T) {
	t.Run("non-nil logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		svc, err := v87.New(&config.Config{
			APIs: config.APIs{
				Camunda: config.API{
					BaseURL: "http://localhost:8080/v2",
				},
			},
		}, &http.Client{}, slog.Default(), v87.WithLogger(logger))
		require.NoError(t, err)
		require.Equal(t, logger, svc.Logger())
	})
	t.Run("nil logger does not override", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		svc, err := v87.New(&config.Config{
			APIs: config.APIs{
				Camunda: config.API{
					BaseURL: "http://localhost:8080/v2",
				},
			},
		}, &http.Client{}, logger, v87.WithLogger(nil))
		require.NoError(t, err)
		require.Equal(t, logger, svc.Logger())
	})
}

func TestService_New_AppliesOptions(t *testing.T) {
	fs := testx.NewFakeServer(t)
	cfg := testx.TestConfig(t)
	cfg.APIs.Camunda.BaseURL = fs.BaseURL
	log := testx.Logger(t)

	customClient := &mockClusterClient{}
	customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc, err := v87.New(cfg, fs.FS.Client(), log, v87.WithClient(customClient), v87.WithLogger(customLogger))
	require.NoError(t, err)
	require.Equal(t, customClient, svc.Client())
	require.Equal(t, customLogger, svc.Logger())
}
