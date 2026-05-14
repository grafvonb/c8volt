// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

type clientTestRoundTripFunc func(*http.Request) (*http.Response, error)

func (f clientTestRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestNew_V89WiresSupportedRuntime ensures the top-level client factory wires
// every facade for the newest supported runtime. The calls intentionally fail
// through a local transport, but reaching those methods proves the v8.9
// services were constructed instead of rejected as unsupported.
func TestNew_V89WiresSupportedRuntime(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.App.CamundaVersion = toolx.V89
	cfg.APIs.Camunda.BaseURL = "http://localhost:8080/v2"
	transportErr := errors.New("test transport blocked")

	cli, err := New(
		WithConfig(cfg),
		WithHTTPClient(&http.Client{Transport: clientTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, transportErr
		})}),
		WithLogger(slog.Default()),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	got, err := cli.SearchProcessInstances(context.Background(), process.ProcessInstanceFilter{}, 1)
	require.Error(t, err)
	require.Empty(t, got.Items)

	// Command code receives the top-level facade, so process capabilities must
	// survive the c8volt.API embedding boundary.
	gotOrphans, err := cli.DiscoverOrphanProcessInstances(context.Background(), process.OrphanDiscoveryRequest{})
	require.Error(t, err)
	require.Empty(t, gotOrphans.Items)

	_, err = cli.GetResource(context.Background(), "resource-id-123")
	require.Error(t, err)

	gotTenants, err := cli.SearchTenants(context.Background(), tenant.TenantFilter{})
	require.Error(t, err)
	require.Empty(t, gotTenants.Items)
	require.IsType(t, tenant.Tenants{}, gotTenants)

	gotPurge, err := cli.PurgeOrphanProcessInstances(context.Background(), ops.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		DryRun:      true,
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)
	require.Equal(t, "ops purge orphan-process-instances", gotPurge.Request.CommandName)
}
