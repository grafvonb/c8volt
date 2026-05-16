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
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/stretchr/testify/require"
)

// TestPurgeProcessInstancesWithIncidentsRecordsControls verifies the foundational request and report model before discovery is implemented.
func TestPurgeProcessInstancesWithIncidentsRecordsControls(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	request := d.IncidentPurgeRequest{
		CommandName:  "ops purge process-instances-with-incidents",
		DryRun:       true,
		AutoConfirm:  true,
		Automation:   true,
		OutputMode:   "json",
		Selection:    d.IncidentFilter{State: "ACTIVE", ErrorType: "JOB_NO_RETRIES"},
		BatchSize:    25,
		Limit:        5,
		Workers:      2,
		ReportFile:   "incident-purge.md",
		ReportFormat: "markdown",
		StartedAt:    started,
	}

	got, err := New(stubProcessInstanceAPI{}, stubIncidentAPI{}).PurgeProcessInstancesWithIncidents(
		context.Background(),
		request,
		services.WithNoWait(),
		services.WithForce(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.IncidentPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, started, got.Request.StartedAt)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.Force)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Equal(t, request.Selection, got.Discovery.Filters)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, d.IncidentPurgeOutcomePlanned, got.Report.Outcome)
	require.True(t, got.Report.NoWait)
	require.True(t, got.Report.Force)
	require.True(t, got.Report.FailFast)
	require.True(t, got.Report.NoWorkerLimit)
	require.Equal(t, request.Selection, got.Report.SelectionFilters)
	require.Empty(t, got.Errors)
}

// TestPurgeProcessInstancesWithIncidentsValidatesServiceDependencies keeps the service seam explicit for later discovery work.
func TestPurgeProcessInstancesWithIncidentsValidatesServiceDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		api  API
		want string
	}{
		{
			name: "missing process-instance service",
			api:  New(nil, stubIncidentAPI{}),
			want: "process-instance service",
		},
		{
			name: "missing incident service",
			api:  New(stubProcessInstanceAPI{}, nil),
			want: "incident service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.api.PurgeProcessInstancesWithIncidents(context.Background(), d.IncidentPurgeRequest{})

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.IncidentPurgeOutcomeFailed, got.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Discovery.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Discovery.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
		})
	}
}

type stubIncidentAPI struct {
	incsvc.API
}
