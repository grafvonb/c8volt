// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v88 "github.com/grafvonb/c8volt/internal/services/tenant/v88"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockTenantClient struct {
	mock.Mock
}

func (m *mockTenantClient) SearchTenantsWithResponse(ctx context.Context, body camundav88.SearchTenantsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchTenantsResponse, error) {
	args := m.Called(ctx, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*camundav88.SearchTenantsResponse), args.Error(1)
}

func (m *mockTenantClient) GetTenantWithResponse(ctx context.Context, tenantId camundav88.TenantId, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetTenantResponse, error) {
	panic("unexpected GetTenantWithResponse call")
}

func TestService_SearchTenants(t *testing.T) {
	ctx := context.Background()
	desc := "primary tenant"
	mockErr := errors.New("search failed")

	tests := []struct {
		name          string
		setupMock     func(*mockTenantClient)
		expectedError error
		assertResult  func(*testing.T, []domain.Tenant)
	}{
		{
			name: "success",
			setupMock: func(m *mockTenantClient) {
				resp := &camundav88.SearchTenantsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/tenants/search", http.StatusOK, "200 OK"),
					JSON200: &camundav88.TenantSearchQueryResult{
						Items: []camundav88.TenantResult{
							{TenantId: "tenant-b", Name: "Beta"},
							{TenantId: "tenant-a", Name: "Alpha", Description: &desc},
						},
					},
				}
				m.On("SearchTenantsWithResponse", mock.Anything, mock.MatchedBy(func(body camundav88.SearchTenantsJSONRequestBody) bool {
					return body.Page != nil && body.Sort != nil && len(*body.Sort) == 2
				})).Return(resp, nil)
			},
			assertResult: func(t *testing.T, tenants []domain.Tenant) {
				require.Len(t, tenants, 2)
				assert.Equal(t, "tenant-b", tenants[0].TenantId)
				assert.Equal(t, "Beta", tenants[0].Name)
				assert.Empty(t, tenants[0].Description)
				assert.Equal(t, "tenant-a", tenants[1].TenantId)
				assert.Equal(t, "primary tenant", tenants[1].Description)
			},
		},
		{
			name: "HTTP error",
			setupMock: func(m *mockTenantClient) {
				resp := &camundav88.SearchTenantsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/tenants/search", http.StatusInternalServerError, "500"),
					Body:         []byte("boom"),
				}
				m.On("SearchTenantsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
			},
			expectedError: domain.ErrInternal,
		},
		{
			name: "nil payload",
			setupMock: func(m *mockTenantClient) {
				resp := &camundav88.SearchTenantsResponse{
					HTTPResponse: newHTTPResponse(http.MethodPost, "https://example.com/tenants/search", http.StatusOK, "200"),
				}
				m.On("SearchTenantsWithResponse", mock.Anything, mock.Anything).Return(resp, nil)
			},
			expectedError: domain.ErrMalformedResponse,
		},
		{
			name: "client error",
			setupMock: func(m *mockTenantClient) {
				m.On("SearchTenantsWithResponse", mock.Anything, mock.Anything).Return((*camundav88.SearchTenantsResponse)(nil), mockErr)
			},
			expectedError: mockErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockTenantClient{}
			tt.setupMock(m)
			svc, err := v88.New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), v88.WithClient(m))
			require.NoError(t, err)

			got, err := svc.SearchTenants(ctx, 100, services.WithVerbose())
			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			tt.assertResult(t, got)
			m.AssertExpectations(t)
		})
	}
}

func newHTTPResponse(method, rawURL string, statusCode int, status string) *http.Response {
	u, _ := url.Parse(rawURL)
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Request: &http.Request{
			Method: method,
			URL:    u,
		},
	}
}
