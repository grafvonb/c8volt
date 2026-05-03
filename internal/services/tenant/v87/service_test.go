// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService builds a v8.7 tenant service with discard logging for unsupported-version tests.
func newTestService(t *testing.T) *Service {
	t.Helper()

	svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	return svc
}

// Verifies v8.7 tenant search fails with an explicit unsupported capability error.
func TestService_SearchTenants_Unsupported(t *testing.T) {
	svc := newTestService(t)

	tenants, err := svc.SearchTenants(context.Background(), domain.TenantFilter{}, 1000)

	require.Error(t, err)
	assert.Nil(t, tenants)
	assert.ErrorIs(t, err, domain.ErrUnsupported)
	assert.Contains(t, err.Error(), "tenant search")
	assert.Contains(t, err.Error(), "Camunda 8.8")
}

// Verifies v8.7 keyed tenant lookup fails with an explicit unsupported capability error.
func TestService_GetTenant_Unsupported(t *testing.T) {
	svc := newTestService(t)

	tenant, err := svc.GetTenant(context.Background(), "tenant-a")

	require.Error(t, err)
	assert.Empty(t, tenant)
	assert.ErrorIs(t, err, domain.ErrUnsupported)
	assert.Contains(t, err.Error(), "tenant lookup")
	assert.Contains(t, err.Error(), "Camunda 8.8")
}
