// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestPurgeAllProcessDefinitionsRecordsControls verifies the foundational request and report model before discovery is implemented.
func TestPurgeAllProcessDefinitionsRecordsControls(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 16, 17, 30, 0, 0, time.UTC)
	request := d.AllProcessDefinitionsPurgeRequest{
		CommandName:  "ops purge all-process-definitions",
		DryRun:       true,
		AutoConfirm:  true,
		Automation:   true,
		OutputMode:   "json",
		Selection:    d.ProcessDefinitionFilter{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "stable", IsLatestVersion: true},
		Workers:      2,
		ReportFile:   "all-pds.md",
		ReportFormat: "markdown",
		DiscoveredCandidateProcessDefinitionKeys: typex.Keys{
			"pd-a",
			"pd-a",
			"pd-b",
		},
		StartedAt: started,
	}

	got, err := NewWithProcessDefinitionPurge(
		stubProcessInstanceAPI{},
		nil,
		stubProcessDefinitionAPI{},
		stubResourceAPI{},
	).PurgeAllProcessDefinitions(
		context.Background(),
		request,
		services.WithNoWait(),
		services.WithForce(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.AllProcessDefinitionsPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, started, got.Request.StartedAt)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.Force)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Equal(t, request.Selection, got.Discovery.Filters)
	require.Equal(t, []string{"pd-a", "pd-b"}, []string(got.Discovery.CandidateProcessDefinitionKeys))
	require.Equal(t, 2, got.Discovery.CandidateProcessDefinitionCount)
	require.True(t, got.Discovery.LatestOnly)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, d.AllProcessDefinitionsPurgeOutcomePlanned, got.Report.Outcome)
	require.True(t, got.Report.NoWait)
	require.True(t, got.Report.Force)
	require.True(t, got.Report.FailFast)
	require.True(t, got.Report.NoWorkerLimit)
	require.Equal(t, request.Selection, got.Report.SelectionFilters)
	require.Equal(t, got.Discovery, got.Report.Discovery)
	require.Empty(t, got.Errors)
}

// TestPurgeAllProcessDefinitionsValidatesServiceDependencies keeps the all-process-definitions service seam explicit.
func TestPurgeAllProcessDefinitionsValidatesServiceDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		api  API
		want string
	}{
		{
			name: "missing process-instance service",
			api:  NewWithProcessDefinitionPurge(nil, nil, stubProcessDefinitionAPI{}, stubResourceAPI{}),
			want: "process-instance service",
		},
		{
			name: "missing process-definition service",
			api:  NewWithProcessDefinitionPurge(stubProcessInstanceAPI{}, nil, nil, stubResourceAPI{}),
			want: "process-definition service",
		},
		{
			name: "missing resource service",
			api:  NewWithProcessDefinitionPurge(stubProcessInstanceAPI{}, nil, stubProcessDefinitionAPI{}, nil),
			want: "resource service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.api.PurgeAllProcessDefinitions(context.Background(), d.AllProcessDefinitionsPurgeRequest{})

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.AllProcessDefinitionsPurgeOutcomeFailed, got.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Discovery.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Discovery.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
		})
	}
}

type stubProcessDefinitionAPI struct {
	pdsvc.API
}

type stubResourceAPI struct {
	rsvc.API
}
