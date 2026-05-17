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
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestRepairWorkflowsValidateFoundationalDependencies verifies the new repair boundary refuses incomplete service wiring.
func TestRepairWorkflowsValidateFoundationalDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		api  API
		want string
	}{
		{
			name: "missing process-instance service",
			api:  NewWithRepairDependencies(nil, nil, stubIncidentAPI{}, nil, nil, stubJobAPI{}, ""),
			want: "process-instance service",
		},
		{
			name: "missing incident service",
			api:  NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, nil, nil, nil, stubJobAPI{}, ""),
			want: "incident service",
		},
		{
			name: "missing job service",
			api:  NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, stubIncidentAPI{}, nil, nil, nil, ""),
			want: "job service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.api.RepairIncidents(context.Background(), d.OpsRepairRequest{CommandName: "ops repair incident"})

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.OpsRepairOutcomeFailed, got.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.FrozenSet.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
		})
	}
}

// TestRepairWorkflowsRecordFoundationalPlan verifies repair constructors inject job support without running concrete target behavior.
func TestRepairWorkflowsRecordFoundationalPlan(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 17, 16, 15, 0, 0, time.UTC)
	retries := int32(1)
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, stubIncidentAPI{}, nil, nil, stubJobAPI{}, "")

	got, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{
		CommandName:              "ops repair process-instance",
		DiscoveryMode:            d.OpsRepairDiscoveryModeKeyed,
		InputKeys:                typex.Keys{"pi-2", "pi-2", "pi-1"},
		ProcessInstanceSelection: d.ProcessInstanceFilter{State: d.StateActive},
		Workers:                  3,
		RequestedRetries:         &retries,
		StartedAt:                started,
	}, services.WithNoWait(), services.WithFailFast(), services.WithNoWorkerLimit(), services.WithDryRun())

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsRepairTargetProcessInstance, got.Request.Target)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.True(t, got.Request.DryRun)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.FrozenSet.Status)
	require.Equal(t, []string{"pi-2", "pi-1"}, []string(got.FrozenSet.InputKeys))
	require.Equal(t, []string{"pi-2", "pi-1"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Empty(t, got.FrozenSet.IncidentKeys)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Remaining.Status)
	require.Equal(t, "ops.repair.v1", got.Report.SchemaVersion)
	require.Equal(t, started, got.Report.StartedAt)
	require.Equal(t, got.FrozenSet, got.Report.FrozenSet)
}

type stubJobAPI struct {
	jsvc.API
}
