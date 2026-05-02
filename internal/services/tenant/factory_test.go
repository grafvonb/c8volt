// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/tenant"
	v87 "github.com/grafvonb/c8volt/internal/services/tenant/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/tenant/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/tenant/v89"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// testConfig returns the minimal service configuration needed by tenant factory tests.
func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://localhost:8080/v2",
			},
		},
	}
}

// Verifies the tenant factory selects the implementation matching each supported Camunda version.
func TestFactory_SupportedVersions(t *testing.T) {
	tests := []struct {
		name    string
		version toolx.CamundaVersion
		assert  func(*testing.T, tenant.API)
	}{
		{
			name:    "v87",
			version: toolx.V87,
			assert: func(t *testing.T, svc tenant.API) {
				require.IsType(t, &v87.Service{}, svc)
			},
		},
		{
			name:    "v88",
			version: toolx.V88,
			assert: func(t *testing.T, svc tenant.API) {
				require.IsType(t, &v88.Service{}, svc)
			},
		},
		{
			name:    "v89",
			version: toolx.V89,
			assert: func(t *testing.T, svc tenant.API) {
				require.IsType(t, &v89.Service{}, svc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.App.CamundaVersion = tt.version

			svc, err := tenant.New(cfg, &http.Client{}, slog.Default())

			require.NoError(t, err)
			require.NotNil(t, svc)
			tt.assert(t, svc)
		})
	}
}

// Verifies unknown Camunda versions fail with the shared unknown-version error.
func TestFactory_UnknownVersion(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = "v0"

	svc, err := tenant.New(cfg, &http.Client{}, slog.Default())

	require.Error(t, err)
	require.Nil(t, svc)
	require.ErrorIs(t, err, services.ErrUnknownAPIVersion)
	require.Contains(t, err.Error(), "\"unknown\"")
	require.Contains(t, err.Error(), toolx.ImplementedCamundaVersionsString())
}

// Verifies the current default Camunda version continues to resolve to the v8.8 tenant service.
func TestFactory_CurrentDefaultVersionStillUsesV88(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = toolx.CurrentCamundaVersion

	svc, err := tenant.New(cfg, &http.Client{}, slog.Default())

	require.NoError(t, err)
	require.NotNil(t, svc)
	require.IsType(t, &v88.Service{}, svc)
}
