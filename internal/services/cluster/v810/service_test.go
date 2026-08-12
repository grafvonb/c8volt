// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	v810 "github.com/grafvonb/c8volt/internal/services/cluster/v810"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockClusterClient struct {
	mock.Mock
}

// GetTopologyWithResponse delegates to testify mock expectations for topology requests.
func (m *mockClusterClient) GetTopologyWithResponse(ctx context.Context, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetTopologyResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav810.GetTopologyResponse), args.Error(1)
}

// GetLicenseWithResponse delegates to testify mock expectations for license requests.
func (m *mockClusterClient) GetLicenseWithResponse(ctx context.Context, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetLicenseResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav810.GetLicenseResponse), args.Error(1)
}

var testTopologyURL = &url.URL{Scheme: "https", Host: "camunda.local", Path: "/cluster/topology"}
var testLicenseURL = &url.URL{Scheme: "https", Host: "camunda.local", Path: "/cluster/license"}

// newHTTPResponse builds a minimal V810 cluster HTTP response for error normalization tests.
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

// TestService_GetClusterTopology verifies V810 topology conversion, transport errors, and malformed payloads.
func TestService_GetClusterTopology(t *testing.T) {
	mockErr := errors.New("network error")
	ctx := context.Background()

	validTopology := camundav810.TopologyResponse{
		ClusterSize:           3,
		GatewayVersion:        "8.10.0-alpha4",
		LastCompletedChangeId: "change-42",
		PartitionsCount:       3,
		ReplicationFactor:     3,
		Brokers: []camundav810.BrokerInfo{
			{
				Host:    "broker-0",
				NodeId:  0,
				Port:    26501,
				Version: "8.10.0-alpha4",
				Partitions: []camundav810.Partition{
					{PartitionId: 1, Role: camundav810.Leader, Health: camundav810.Healthy},
				},
			},
		},
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
				m.On("GetTopologyWithResponse", mock.Anything).Return(&camundav810.GetTopologyResponse{
					JSON200:      &validTopology,
					HTTPResponse: newHTTPResponse(testTopologyURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			validateResp: func(t *testing.T, top d.Topology) {
				assert.Equal(t, "8.10.0-alpha4", top.GatewayVersion)
				assert.Equal(t, "change-42", top.LastCompletedChangeId)
				assert.Equal(t, int32(3), top.ClusterSize)
				assert.Len(t, top.Brokers, 1)
				assert.Equal(t, int32(0), top.Brokers[0].NodeId)
				assert.Equal(t, "8.10.0-alpha4", top.Brokers[0].Version)
			},
		},
		{
			name: "Client Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return((*camundav810.GetTopologyResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTP Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return(&camundav810.GetTopologyResponse{
					HTTPResponse: newHTTPResponse(testTopologyURL, http.StatusInternalServerError, "500 Internal Server Error"),
					Body:         []byte("error"),
				}, nil)
			},
			expectedError: d.ErrInternal,
		},
		{
			name: "Empty Payload",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return(&camundav810.GetTopologyResponse{
					HTTPResponse: newHTTPResponse(testTopologyURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			expectedError: d.ErrMalformedResponse,
		},
		{
			name: "Nil Response",
			setupMock: func(m *mockClusterClient) {
				m.On("GetTopologyWithResponse", mock.Anything).Return((*camundav810.GetTopologyResponse)(nil), nil)
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

			svc, err := v810.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v810.WithClient(m))
			require.NoError(t, err)

			resp, err := svc.GetClusterTopology(ctx)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				tt.validateResp(t, resp)
			}
			m.AssertExpectations(t)
		})
	}
}

// TestService_GetClusterLicense verifies V810 license conversion, transport errors, and malformed payloads.
func TestService_GetClusterLicense(t *testing.T) {
	mockErr := errors.New("network error")
	ctx := context.Background()
	expiresAt := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)

	validLicense := camundav810.LicenseResponse{
		ExpiresAt:    &expiresAt,
		IsCommercial: true,
		LicenseType:  "Self-Managed Enterprise",
		ValidLicense: true,
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
				m.On("GetLicenseWithResponse", mock.Anything).Return(&camundav810.GetLicenseResponse{
					JSON200:      &validLicense,
					HTTPResponse: newHTTPResponse(testLicenseURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			validateResp: func(t *testing.T, lic d.License) {
				require.NotNil(t, lic.ExpiresAt)
				require.NotNil(t, lic.IsCommercial)
				assert.Equal(t, expiresAt, *lic.ExpiresAt)
				assert.True(t, *lic.IsCommercial)
				assert.Equal(t, "Self-Managed Enterprise", lic.LicenseType)
				assert.True(t, lic.ValidLicense)
			},
		},
		{
			name: "Client Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return((*camundav810.GetLicenseResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
		{
			name: "HTTP Error",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return(&camundav810.GetLicenseResponse{
					HTTPResponse: newHTTPResponse(testLicenseURL, http.StatusInternalServerError, "500 Internal Server Error"),
					Body:         []byte("error"),
				}, nil)
			},
			expectedError: d.ErrInternal,
		},
		{
			name: "Empty Payload",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return(&camundav810.GetLicenseResponse{
					HTTPResponse: newHTTPResponse(testLicenseURL, http.StatusOK, "200 OK"),
				}, nil)
			},
			expectedError: d.ErrMalformedResponse,
		},
		{
			name: "Nil Response",
			setupMock: func(m *mockClusterClient) {
				m.On("GetLicenseWithResponse", mock.Anything).Return((*camundav810.GetLicenseResponse)(nil), nil)
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

			svc, err := v810.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v810.WithClient(m))
			require.NoError(t, err)

			resp, err := svc.GetClusterLicense(ctx)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				tt.validateResp(t, resp)
			}
			m.AssertExpectations(t)
		})
	}
}

// TestService_WithClient verifies nil client options cannot accidentally erase a configured client.
func TestService_WithClient(t *testing.T) {
	mc := &mockClusterClient{}
	svc, err := v810.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v810.WithClient(mc))
	require.NoError(t, err)
	require.Equal(t, mc, svc.Client())

	v810.WithClient(nil)(svc)
	require.Equal(t, mc, svc.Client())
}

// TestService_WithLogger verifies nil logger options cannot accidentally erase a configured logger.
func TestService_WithLogger(t *testing.T) {
	svc, err := v810.New(testConfig(), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v810.WithLogger(logger)(svc)
	require.Equal(t, logger, svc.Logger())

	v810.WithLogger(nil)(svc)
	require.Equal(t, logger, svc.Logger())
}

// testConfig returns the minimal config needed to construct a V810 cluster service in tests.
func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "https://camunda.local/v2",
			},
		},
	}
}
