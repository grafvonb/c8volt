// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/job"
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

	gotLatency, err := cli.AnalyseAPILatency(context.Background(), ops.APILatencyRequest{
		CommandName: "ops analyse api-latency",
		Count:       7,
		Workers:     4,
	})
	require.NoError(t, err)
	require.Equal(t, ops.APILatencyOutcomeCompleted, gotLatency.Outcome)
	require.Equal(t, []ops.APILatencyStagePlan{
		{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 2},
		{Index: 2, WorkerCount: 2, PrimarySamples: 2, DerivedRequestLimit: 4},
		{Index: 3, WorkerCount: 4, PrimarySamples: 4, DerivedRequestLimit: 8},
	}, gotLatency.Plan.Stages)

	gotActiveLatency, err := cli.ExecuteAPILatencyTest(context.Background(), ops.APILatencyRequest{
		CommandName: "ops execute api-latency-test",
		Count:       7,
		Workers:     4,
		DryRun:      true,
	})
	require.NoError(t, err)
	require.Equal(t, ops.APILatencyOutcomePlanned, gotActiveLatency.Outcome)
	require.NotNil(t, gotActiveLatency.Plan.Cleanup)
}

// TestNew_V810WiresCompleteNativeRuntime proves the top-level facade can
// construct every V810-aware service family without falling back to unsupported
// bootstrap behavior.
func TestNew_V810WiresCompleteNativeRuntime(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.App.CamundaVersion = toolx.V810
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

	caps, err := cli.Capabilities(context.Background())
	require.NoError(t, err)
	require.Equal(t, "8.10", caps.CamundaVersion)

	_, err = cli.GetClusterTopology(context.Background())
	require.Error(t, err)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotDefinitions, err := cli.SearchProcessDefinitions(context.Background(), process.ProcessDefinitionFilter{})
	require.Error(t, err)
	require.Empty(t, gotDefinitions.Items)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotInstances, err := cli.SearchProcessInstances(context.Background(), process.ProcessInstanceFilter{}, 1)
	require.Error(t, err)
	require.Empty(t, gotInstances.Items)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotVariables, err := cli.SearchProcessInstanceVariables(context.Background(), "2251799813685249")
	require.Error(t, err)
	require.Empty(t, gotVariables)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotIncidents, err := cli.SearchIncidents(context.Background(), incident.Filter{}, 1)
	require.Error(t, err)
	require.Empty(t, gotIncidents.Items)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	_, err = cli.GetResource(context.Background(), "resource-id-123")
	require.Error(t, err)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	err = cli.CheckBatchOperationReadAccess(context.Background())
	require.Error(t, err)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotTenants, err := cli.SearchTenants(context.Background(), tenant.TenantFilter{})
	require.Error(t, err)
	require.Empty(t, gotTenants.Items)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotJobs, err := cli.SearchJobs(context.Background(), job.SearchRequest{Limit: 1})
	require.Error(t, err)
	require.Empty(t, gotJobs.Items)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotElements, err := cli.SearchElements(context.Background(), element.SearchRequest{Limit: 1})
	require.Error(t, err)
	require.Empty(t, gotElements.Items)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	_, err = cli.ResolveProcessInstanceKeyFromUserTask(context.Background(), "2251799813685250")
	require.Error(t, err)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)

	gotPurge, err := cli.PurgeOrphanProcessInstances(context.Background(), ops.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		DryRun:      true,
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, ferrors.ErrUnsupported)
	require.Equal(t, "ops purge orphan-process-instances", gotPurge.Request.CommandName)
}
