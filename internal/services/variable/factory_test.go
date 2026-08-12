// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package variable_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/variable"
	v810 "github.com/grafvonb/c8volt/internal/services/variable/v810"
	v87 "github.com/grafvonb/c8volt/internal/services/variable/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/variable/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/variable/v89"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestFactory_SupportedVersions verifies every implemented Camunda version selects its native variable service.
func TestFactory_SupportedVersions(t *testing.T) {
	tests := []struct {
		name    string
		version toolx.CamundaVersion
		assert  func(*testing.T, variable.API)
	}{
		{
			name:    "v87",
			version: toolx.V87,
			assert: func(t *testing.T, svc variable.API) {
				require.IsType(t, &v87.Service{}, svc)
			},
		},
		{
			name:    "v88",
			version: toolx.V88,
			assert: func(t *testing.T, svc variable.API) {
				require.IsType(t, &v88.Service{}, svc)
			},
		},
		{
			name:    "v89",
			version: toolx.V89,
			assert: func(t *testing.T, svc variable.API) {
				require.IsType(t, &v89.Service{}, svc)
			},
		},
		{
			name:    "v810",
			version: toolx.V810,
			assert: func(t *testing.T, svc variable.API) {
				require.IsType(t, &v810.Service{}, svc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := variableTestConfig()
			cfg.App.CamundaVersion = tt.version

			svc, err := variable.New(cfg, &http.Client{}, slog.Default())

			require.NoError(t, err)
			require.NotNil(t, svc)
			tt.assert(t, svc)
		})
	}
}

// TestFactory_UnknownVersion verifies unsupported versions use the shared factory error.
func TestFactory_UnknownVersion(t *testing.T) {
	cfg := variableTestConfig()
	cfg.App.CamundaVersion = "v0"

	svc, err := variable.New(cfg, &http.Client{}, slog.Default())

	require.Error(t, err)
	require.Nil(t, svc)
	require.ErrorIs(t, err, services.ErrUnknownAPIVersion)
	require.Contains(t, err.Error(), "\"unknown\"")
	require.Contains(t, err.Error(), toolx.ImplementedCamundaVersionsString())
}

// TestFactory_CurrentDefaultVersionStillUsesV88 protects the unchanged default runtime selection.
func TestFactory_CurrentDefaultVersionStillUsesV88(t *testing.T) {
	cfg := variableTestConfig()
	cfg.App.CamundaVersion = toolx.CurrentCamundaVersion

	svc, err := variable.New(cfg, &http.Client{}, slog.Default())

	require.NoError(t, err)
	require.NotNil(t, svc)
	require.IsType(t, &v88.Service{}, svc)
}

// variableTestConfig returns the minimal Camunda v2 base URL required by service constructors.
func variableTestConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://localhost:8080/v2",
			},
		},
	}
}
