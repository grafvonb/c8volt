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
	"github.com/stretchr/testify/require"
)

func TestExecuteRetentionPolicyValidatesRequestShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request d.RetentionPolicyRequest
		want    string
	}{
		{
			name:    "negative retention days",
			request: d.RetentionPolicyRequest{RetentionDays: -1},
			want:    "retention-days must be a non-negative integer",
		},
		{
			name: "explicit key selection",
			request: d.RetentionPolicyRequest{
				RetentionDays: 90,
				Selection: d.ProcessInstanceFilter{
					Key: "2251799813685249",
				},
			},
			want: "does not accept explicit process-instance keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(stubProcessInstanceAPI{}).ExecuteRetentionPolicy(context.Background(), tt.request)

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.RetentionPolicyOutcomeFailed, got.Outcome)
			require.Equal(t, d.RetentionPolicyOutcomeFailed, got.Report.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Discovery.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Discovery.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
			require.True(t, strings.Contains(got.Report.Errors[0], tt.want))
		})
	}
}

func TestExecuteRetentionPolicyAcceptsZeroRetentionDaysAndRecordsControls(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			require.Equal(t, d.ProcessInstanceFilter{
				BpmnProcessId: "invoice",
				State:         d.StateCompleted,
				EndDateBefore: "2026-05-14T00:00:00Z",
			}, filter)
			require.EqualValues(t, 100, page.Size)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813685249", EndDate: "2026-05-13T09:00:00Z"},
				},
			}, nil
		},
	}
	request := d.RetentionPolicyRequest{
		CommandName:            "ops execute retention-policy",
		RetentionDays:          0,
		DerivedEndDateBoundary: "2026-05-14T00:00:00Z",
		DryRun:                 true,
		AutoConfirm:            true,
		Automation:             true,
		OutputMode:             "json",
		Selection: d.ProcessInstanceFilter{
			BpmnProcessId: "invoice",
			State:         d.StateCompleted,
		},
		BatchSize:    100,
		Limit:        10,
		Workers:      2,
		ReportFile:   "retention-report.md",
		ReportFormat: "markdown",
		StartedAt:    started,
	}

	got, err := New(piAPI).ExecuteRetentionPolicy(
		context.Background(),
		request,
		services.WithNoWait(),
		services.WithNoStateCheck(),
		services.WithForce(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.RetentionPolicyOutcomePlanned, got.Outcome)
	require.Equal(t, d.RetentionPolicyOutcomePlanned, got.Report.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, []string{"2251799813685249"}, []string(got.Discovery.SeedKeys))
	require.Equal(t, 1, got.Discovery.Count)
	require.Equal(t, started, got.Request.StartedAt)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.NoStateCheck)
	require.True(t, got.Request.Force)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.True(t, got.Report.NoWait)
	require.True(t, got.Report.NoStateCheck)
	require.True(t, got.Report.Force)
	require.True(t, got.Report.FailFast)
	require.True(t, got.Report.NoWorkerLimit)
	require.Equal(t, 0, got.Report.RetentionDays)
	require.Equal(t, "2026-05-14T00:00:00Z", got.Report.DerivedEndDateBoundary)
	require.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "invoice", State: d.StateCompleted}, got.Report.SelectionFilters)
}

func TestExecuteRetentionPolicyDryRunDiscoversFrozenSeedSetAndSkipsDeleteWork(t *testing.T) {
	t.Parallel()

	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			require.Equal(t, "2026-02-13", filter.EndDateBefore)
			require.EqualValues(t, 1000, page.Size)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "seed-1", EndDate: "2026-02-12"},
					{Key: "seed-2", EndDate: "2026-02-11"},
				},
			}, nil
		},
	}
	request := d.RetentionPolicyRequest{
		CommandName:            "ops execute retention-policy",
		RetentionDays:          90,
		DerivedEndDateBoundary: "2026-02-13",
		DryRun:                 true,
		StartedAt:              time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
	}

	got, err := New(piAPI).ExecuteRetentionPolicy(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, d.RetentionPolicyOutcomePlanned, got.Outcome)
	require.Equal(t, d.RetentionDiscoveryResult{
		Status:                 d.OpsWorkflowStepStatusPlanned,
		RetentionDays:          90,
		DerivedEndDateBoundary: "2026-02-13",
		Filters: d.ProcessInstanceFilter{
			EndDateBefore: "2026-02-13",
		},
		SeedKeys: []string{"seed-1", "seed-2"},
		Count:    2,
	}, got.Discovery)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, got.Discovery, got.Report.Discovery)
	require.Empty(t, got.Errors)
}
