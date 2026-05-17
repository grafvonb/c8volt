// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestExecuteSmokeTestValidatesRequestShape protects local validation before any workflow step can mutate state.
func TestExecuteSmokeTestValidatesRequestShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request d.SmokeTestRequest
		want    string
	}{
		{
			name:    "zero count",
			request: d.SmokeTestRequest{Count: 0},
			want:    "count must be a positive integer",
		},
		{
			name:    "negative count",
			request: d.SmokeTestRequest{Count: -3},
			want:    "count must be a positive integer",
		},
		{
			name: "unsupported report format",
			request: d.SmokeTestRequest{
				Count:        1,
				ReportFormat: "xml",
				ReportFile:   "smoke-test.xml",
			},
			want: "report-format must be markdown or json",
		},
		{
			name: "report format without file",
			request: d.SmokeTestRequest{
				Count:        1,
				ReportFormat: "json",
			},
			want: "report-format requires report-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(nil, nil).ExecuteSmokeTest(context.Background(), tt.request)

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.SmokeTestOutcomeFailed, got.Outcome)
			require.Equal(t, d.SmokeTestOutcomeFailed, got.Report.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deployment.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Run.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Walk.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessInstanceCleanup.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionEligibility.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Plan.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
			require.True(t, strings.Contains(got.Report.Errors[0], tt.want))
		})
	}
}

// TestExecuteSmokeTestRecordsFoundationalControls verifies the boundary records reusable call controls in result metadata.
func TestExecuteSmokeTestRecordsFoundationalControls(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 17, 10, 30, 0, 0, time.UTC)
	request := d.SmokeTestRequest{
		CommandName:  "ops execute smoke-test",
		Count:        1,
		DryRun:       true,
		NoCleanup:    true,
		AutoConfirm:  true,
		Automation:   true,
		OutputMode:   "json",
		ReportFile:   "smoke-test.md",
		ReportFormat: "markdown",
		StartedAt:    started,
	}

	got, err := New(nil, nil).ExecuteSmokeTest(
		context.Background(),
		request,
		services.WithNoWait(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Deployment.Status)
	require.Equal(t, started, got.Request.StartedAt)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.True(t, got.Report.NoWait)
	require.True(t, got.Report.NoCleanup)
	require.False(t, got.Report.CleanupRequested)
	require.Equal(t, got.Plan, got.Report.Plan)
}

func TestExecuteSmokeTestDryRunPlansReadOnlyWorkflow(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{
		topology: d.Topology{GatewayVersion: "8.8.2"},
	}
	request := d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       3,
		DryRun:      true,
		NoCleanup:   true,
		ReportFile:  "smoke-test.md",
	}

	got, err := NewWithWorkflowDependencies(cluster, nil, nil, nil, nil, toolx.V88).ExecuteSmokeTest(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, 1, cluster.topologyCalls)
	require.Equal(t, d.SmokeTestOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan.Status)
	require.Equal(t, "8.8", got.Plan.CamundaVersion)
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn", got.Fixture.File)
	require.Equal(t, "C88_MultipleSubProcessesParentProcess", got.Fixture.BpmnProcessID)
	require.True(t, got.Fixture.Available)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Deployment.Status)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Run.Status)
	require.Equal(t, 3, got.Run.RequestedCount)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Walk.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessInstanceCleanup.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionEligibility.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
	require.False(t, got.Plan.CleanupRequested)
	require.Len(t, got.Plan.PlannedSteps, 7)
	require.Equal(t, "connectivity", got.Plan.PlannedSteps[0].Name)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Plan.PlannedSteps[0].Status)
	require.Equal(t, "cleanup", got.Plan.PlannedSteps[5].Name)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Plan.PlannedSteps[5].Status)
	require.Equal(t, "report", got.Plan.PlannedSteps[6].Name)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan.PlannedSteps[6].Status)
	require.Equal(t, got.Plan, got.Report.Plan)
	require.Equal(t, got.Fixture, got.Report.Fixture)
	require.Equal(t, got.Deployment, got.Report.Deployment)
}

func TestExecuteSmokeTestDryRunConnectivityFailureDoesNotPlanMutation(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{err: errors.New("topology unavailable")}

	got, err := NewWithWorkflowDependencies(cluster, nil, nil, nil, nil, toolx.V89).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		DryRun:      true,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "smoke-test connectivity validation")
	require.Equal(t, 1, cluster.topologyCalls)
	require.Equal(t, d.SmokeTestOutcomeFailed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deployment.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Run.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Walk.Status)
	require.Len(t, got.Plan.PlannedSteps, 7)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.PlannedSteps[0].Status)
	require.NotEmpty(t, got.Errors)
}

type stubSmokeTestClusterAPI struct {
	topology      d.Topology
	err           error
	topologyCalls int
}

func (s *stubSmokeTestClusterAPI) GetClusterTopology(context.Context, ...services.CallOption) (d.Topology, error) {
	s.topologyCalls++
	if s.err != nil {
		return d.Topology{}, s.err
	}
	return s.topology, nil
}

func (*stubSmokeTestClusterAPI) GetClusterLicense(context.Context, ...services.CallOption) (d.License, error) {
	return d.License{}, nil
}
