// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestNew_V89WiresSupportedRuntime ensures the top-level client factory wires
// every facade for the newest supported runtime. The calls intentionally fail
// against localhost, but reaching those methods proves the v8.9 services were
// constructed instead of rejected as unsupported.
func TestNew_V89WiresSupportedRuntime(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.App.CamundaVersion = toolx.V89
	cfg.APIs.Camunda.BaseURL = "http://localhost:8080/v2"

	cli, err := New(
		WithConfig(cfg),
		WithHTTPClient(&http.Client{}),
		WithLogger(slog.Default()),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	got, err := cli.SearchProcessInstances(context.Background(), process.ProcessInstanceFilter{}, 1)
	require.Error(t, err)
	require.Empty(t, got.Items)

	_, err = cli.GetResource(context.Background(), "resource-id-123")
	require.Error(t, err)

	gotTenants, err := cli.SearchTenants(context.Background())
	require.Error(t, err)
	require.Empty(t, gotTenants.Items)
	require.IsType(t, tenant.Tenants{}, gotTenants)
}
