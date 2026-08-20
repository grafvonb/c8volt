// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/element"
	v810 "github.com/grafvonb/c8volt/internal/services/element/v810"
	v87 "github.com/grafvonb/c8volt/internal/services/element/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/element/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/element/v89"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// testConfig returns the minimal config needed to construct versioned element services.
func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{BaseURL: "http://localhost:8080/v2"},
		},
	}
}

// TestFactory_SupportedVersions verifies every implemented version has an explicit factory branch.
func TestFactory_SupportedVersions(t *testing.T) {
	tests := []struct {
		name    string
		version toolx.CamundaVersion
		assert  func(*testing.T, element.API)
	}{
		{name: "v87", version: toolx.V87, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v87.Service{}, svc) }},
		{name: "v88", version: toolx.V88, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v88.Service{}, svc) }},
		{name: "v89", version: toolx.V89, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v89.Service{}, svc) }},
		{name: "v810", version: toolx.V810, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v810.Service{}, svc) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.App.CamundaVersion = tt.version

			svc, err := element.New(cfg, &http.Client{}, slog.Default())

			require.NoError(t, err)
			require.NotNil(t, svc)
			tt.assert(t, svc)
		})
	}
}

// TestFactory_StableVersionSelectionUnchanged proves V810 support does not reroute stable service selection or the current default.
func TestFactory_StableVersionSelectionUnchanged(t *testing.T) {
	tests := []struct {
		name    string
		version toolx.CamundaVersion
		assert  func(*testing.T, element.API)
	}{
		{name: "v87", version: toolx.V87, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v87.Service{}, svc) }},
		{name: "v88", version: toolx.V88, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v88.Service{}, svc) }},
		{name: "v89", version: toolx.V89, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v89.Service{}, svc) }},
		{name: "current-default", version: toolx.CurrentCamundaVersion, assert: func(t *testing.T, svc element.API) { require.IsType(t, &v89.Service{}, svc) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.App.CamundaVersion = tt.version

			svc, err := element.New(cfg, &http.Client{}, slog.Default())

			require.NoError(t, err)
			require.NotNil(t, svc)
			tt.assert(t, svc)
		})
	}
}

// TestFactory_UnknownVersion preserves the shared unknown-version error contract.
func TestFactory_UnknownVersion(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = "v0"

	svc, err := element.New(cfg, &http.Client{}, slog.Default())

	require.Error(t, err)
	require.Nil(t, svc)
	require.ErrorIs(t, err, services.ErrUnknownAPIVersion)
	require.Contains(t, err.Error(), toolx.ImplementedCamundaVersionsString())
}
