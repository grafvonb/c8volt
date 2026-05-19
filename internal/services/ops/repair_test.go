// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/testx"
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

// TestRepairProcessInstancesDryRunDiscoversExplicitTargets verifies explicit process-instance repair freezes selected PIs and active incidents.
func TestRepairProcessInstancesDryRunDiscoversExplicitTargets(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 17, 16, 15, 0, 0, time.UTC)
	retries := int32(1)
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		getProcessInstances: func(_ context.Context, keys typex.Keys, _ int, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(keys))
			return []d.ProcessInstance{
				{Key: "2251799813685251", State: d.StateActive},
				{Key: "2251799813685253", State: d.StateActive},
			}, nil
		},
	}, repairIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			switch key {
			case "2251799813685251":
				return []d.ProcessInstanceIncidentDetail{
					{IncidentKey: "2251799813685249", ProcessInstanceKey: key, RootProcessInstanceKey: key, JobKey: "2251799813685252", State: "ACTIVE"},
				}, nil
			case "2251799813685253":
				return []d.ProcessInstanceIncidentDetail{
					{IncidentKey: "2251799813685250", ProcessInstanceKey: key, RootProcessInstanceKey: key, State: "ACTIVE"},
				}, nil
			default:
				t.Fatalf("unexpected process instance key %s", key)
				return nil, nil
			}
		},
	}, nil, nil, stubJobAPI{}, "")

	got, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{
		CommandName:              "ops repair process-instance",
		DiscoveryMode:            d.OpsRepairDiscoveryModeKeyed,
		InputKeys:                typex.Keys{"2251799813685251", "2251799813685251", "2251799813685253"},
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
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.FrozenSet.Status)
	require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(got.FrozenSet.InputKeys))
	require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Equal(t, []string{"2251799813685249", "2251799813685250"}, []string(got.FrozenSet.IncidentKeys))
	require.Equal(t, []string{"2251799813685252"}, []string(got.FrozenSet.JobKeys))
	require.Len(t, got.Plan, 2)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusNotApplicable, got.Plan[1].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Remaining.Status)
	require.Equal(t, "ops.repair.v1", got.Report.SchemaVersion)
	require.Equal(t, started, got.Report.StartedAt)
	require.Equal(t, got.FrozenSet, got.Report.FrozenSet)
}

// TestRepairProcessInstancesDryRunReportsExplicitTargetsWithoutIncidents verifies direct PI keys can include non-applicable instances.
func TestRepairProcessInstancesDryRunReportsExplicitTargetsWithoutIncidents(t *testing.T) {
	t.Parallel()

	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		getProcessInstances: func(_ context.Context, keys typex.Keys, _ int, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, []string{"2251799813685251", "2251799813685255"}, []string(keys))
			return []d.ProcessInstance{
				{Key: "2251799813685251", State: d.StateActive, Incident: true},
				{Key: "2251799813685255", State: d.StateActive, Incident: false},
			}, nil
		},
	}, repairIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			switch key {
			case "2251799813685251":
				return []d.ProcessInstanceIncidentDetail{
					{IncidentKey: "2251799813685249", ProcessInstanceKey: key, RootProcessInstanceKey: key, State: "ACTIVE"},
				}, nil
			case "2251799813685255":
				return nil, nil
			default:
				t.Fatalf("unexpected process instance key %s", key)
				return nil, nil
			}
		},
	}, nil, nil, stubJobAPI{}, "")

	got, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{
		CommandName:   "ops repair process-instance",
		DiscoveryMode: d.OpsRepairDiscoveryModeKeyed,
		InputKeys:     typex.Keys{"2251799813685251", "2251799813685255"},
		DryRun:        true,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"2251799813685251"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Equal(t, []string{"2251799813685249"}, []string(got.FrozenSet.IncidentKeys))
	require.Equal(t, []string{"2251799813685255"}, []string(got.FrozenSet.SkippedProcessInstanceKeys))
	require.Equal(t, got.FrozenSet, got.Report.FrozenSet)
}

// TestRepairProcessInstancesNoActiveIncidentsNoOpsWithoutError verifies direct PI keys with no incidents are a clean no-op.
func TestRepairProcessInstancesNoActiveIncidentsNoOpsWithoutError(t *testing.T) {
	t.Parallel()

	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		getProcessInstances: func(_ context.Context, keys typex.Keys, _ int, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, []string{"2251799813685255"}, []string(keys))
			return []d.ProcessInstance{{Key: "2251799813685255", State: d.StateActive, Incident: false}}, nil
		},
	}, repairIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.Equal(t, "2251799813685255", key)
			return nil, nil
		},
	}, nil, nil, stubJobAPI{}, "")

	got, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{
		CommandName:   "ops repair process-instance",
		DiscoveryMode: d.OpsRepairDiscoveryModeKeyed,
		InputKeys:     typex.Keys{"2251799813685255"},
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Empty(t, got.FrozenSet.ProcessInstanceKeys)
	require.Empty(t, got.FrozenSet.IncidentKeys)
	require.Equal(t, []string{"2251799813685255"}, []string(got.FrozenSet.SkippedProcessInstanceKeys))
	require.Empty(t, got.Plan)
	require.Len(t, got.Notices, 1)
	require.Equal(t, "no_active_incidents", got.Notices[0].Code)
}

// TestRepairProcessInstancesSearchDedupeAndRoutesThroughIncidentRepair verifies search-selected PIs repair each duplicate incident once.
func TestRepairProcessInstancesSearchDedupeAndRoutesThroughIncidentRepair(t *testing.T) {
	t.Parallel()

	var gotFilter d.ProcessInstanceFilter
	var gotSize int32
	var resolved []string
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			gotFilter = filter
			gotSize = size
			return []d.ProcessInstance{
				{Key: "2251799813685251", State: d.StateActive},
				{Key: "2251799813685253", State: d.StateActive},
			}, nil
		},
	}, repairIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			if key == "2251799813685251" {
				return []d.ProcessInstanceIncidentDetail{
					{IncidentKey: "2251799813685249", ProcessInstanceKey: key, State: "ACTIVE"},
					{IncidentKey: "2251799813685250", ProcessInstanceKey: key, State: "RESOLVED"},
				}, nil
			}
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "2251799813685249", ProcessInstanceKey: "2251799813685251", State: "ACTIVE"},
				{IncidentKey: "2251799813685254", ProcessInstanceKey: key, State: "ACTIVE"},
			}, nil
		},
		resolveIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			resolved = append(resolved, key)
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusNoContent, Status: "accepted"}, nil
		},
		waitForIncidentResolved: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "resolved"}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	hasIncident := true
	got, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{
		CommandName:              "ops repair process-instance",
		DiscoveryMode:            d.OpsRepairDiscoveryModeSearch,
		ProcessInstanceSelection: d.ProcessInstanceFilter{State: d.StateActive, HasIncident: &hasIncident},
		BatchSize:                25,
		Limit:                    2,
		Workers:                  1,
		NoWait:                   true,
	})

	require.NoError(t, err)
	require.Equal(t, d.ProcessInstanceFilter{State: d.StateActive, HasIncident: &hasIncident}, gotFilter)
	require.EqualValues(t, 2, gotSize)
	require.Equal(t, d.OpsRepairOutcomeRepaired, got.Outcome)
	require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Equal(t, []string{"2251799813685249", "2251799813685254"}, []string(got.FrozenSet.IncidentKeys))
	require.Equal(t, []string{"2251799813685249", "2251799813685254"}, resolved)
	require.Len(t, got.Plan, 2)
}

// TestRepairIncidentsFreezesExplicitTargetsAndPlansMixedJobs verifies explicit incident repair freezes lookup results before mutation.
func TestRepairIncidentsFreezesExplicitTargetsAndPlansMixedJobs(t *testing.T) {
	t.Parallel()

	retries := int32(1)
	started := time.Date(2026, 5, 17, 18, 0, 0, 0, time.UTC)
	incidents := map[string]d.ProcessInstanceIncidentDetail{
		"2251799813685249": {
			IncidentKey:            "2251799813685249",
			ProcessInstanceKey:     "2251799813685251",
			RootProcessInstanceKey: "2251799813685251",
			JobKey:                 "2251799813685252",
			State:                  "ACTIVE",
			ErrorType:              "JOB_NO_RETRIES",
		},
		"2251799813685250": {
			IncidentKey:            "2251799813685250",
			ProcessInstanceKey:     "2251799813685253",
			RootProcessInstanceKey: "2251799813685253",
			State:                  "ACTIVE",
			ErrorType:              "IO_MAPPING_ERROR",
		},
	}
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{
		getIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			return incidents[key], nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName:      "ops repair incident",
		DiscoveryMode:    d.OpsRepairDiscoveryModeKeyed,
		InputKeys:        typex.Keys{"2251799813685249", "2251799813685250", "2251799813685249"},
		RequestedRetries: &retries,
		DryRun:           true,
		StartedAt:        started,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.FrozenSet.Status)
	require.Equal(t, []string{"2251799813685249", "2251799813685250"}, []string(got.FrozenSet.IncidentKeys))
	require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Equal(t, []string{"2251799813685252"}, []string(got.FrozenSet.JobKeys))
	require.Len(t, got.FrozenSet.OriginalIncidents, 2)
	require.Len(t, got.Plan, 2)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusNotApplicable, got.Plan[1].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusNotApplicable, got.JobApplicability[1].TimeoutStatus)
	require.Equal(t, got.FrozenSet, got.Report.FrozenSet)
}

// TestRepairIncidentsUpdatesJobAndConfirmsResolution verifies the mutation path composes job and incident primitives.
func TestRepairIncidentsUpdatesJobAndConfirmsResolution(t *testing.T) {
	t.Parallel()

	retries := int32(1)
	var jobRequests []d.JobUpdateRequest
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{
		getIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			return d.ProcessInstanceIncidentDetail{
				IncidentKey:        key,
				ProcessInstanceKey: "2251799813685251",
				JobKey:             "2251799813685252",
				State:              "ACTIVE",
			}, nil
		},
		resolveIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusNoContent, Status: "accepted"}, nil
		},
		waitForIncidentResolved: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "resolved"}, nil
		},
	}, nil, nil, repairJobAPI{
		updateJob: func(_ context.Context, request d.JobUpdateRequest, _ ...services.CallOption) (d.JobUpdateResult, error) {
			jobRequests = append(jobRequests, request)
			return d.JobUpdateResult{
				Key:                request.Key,
				MutationAccepted:   true,
				SubmittedRetries:   request.Retries,
				ConfirmedRetries:   request.Retries,
				ConfirmationStatus: "confirmed",
			}, nil
		},
	}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName:      "ops repair incident",
		InputKeys:        typex.Keys{"2251799813685249"},
		RequestedRetries: &retries,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomeRepaired, got.Outcome)
	require.Len(t, jobRequests, 1)
	require.Equal(t, "2251799813685252", jobRequests[0].Key)
	require.Equal(t, &retries, jobRequests[0].Retries)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Plan[0].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusSubmitted, got.Plan[0].ResolutionStatus)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Plan[0].ConfirmationStatus)
}

// TestRepairIncidentsSearchModeFreezesFilteredTargets verifies filtered discovery is bounded and converted into a dry-run plan.
func TestRepairIncidentsSearchModeFreezesFilteredTargets(t *testing.T) {
	t.Parallel()

	retries := int32(1)
	var gotFilter d.IncidentFilter
	var gotSize int32
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			gotFilter = filter
			gotSize = size
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "2251799813685249", ProcessInstanceKey: "2251799813685251", JobKey: "2251799813685252", State: "ACTIVE", ErrorType: "IO_MAPPING_ERROR"},
				{IncidentKey: "2251799813685250", ProcessInstanceKey: "2251799813685253", State: "ACTIVE", ErrorType: "IO_MAPPING_ERROR"},
				{IncidentKey: "2251799813685254", ProcessInstanceKey: "2251799813685255", State: "ACTIVE", ErrorType: "IO_MAPPING_ERROR"},
			}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName:       "ops repair incident",
		DiscoveryMode:     d.OpsRepairDiscoveryModeSearch,
		IncidentSelection: d.IncidentFilter{State: "active", ErrorType: "io_mapping_error"},
		BatchSize:         25,
		Limit:             2,
		RequestedRetries:  &retries,
		DryRun:            true,
	})

	require.NoError(t, err)
	require.Equal(t, d.IncidentFilter{State: "active", ErrorType: "io_mapping_error"}, gotFilter)
	require.EqualValues(t, 2, gotSize)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsRepairDiscoveryModeSearch, got.FrozenSet.DiscoveryMode)
	require.Equal(t, []string{"2251799813685249", "2251799813685250"}, []string(got.FrozenSet.IncidentKeys))
	require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Equal(t, []string{"2251799813685252"}, []string(got.FrozenSet.JobKeys))
	require.Len(t, got.FrozenSet.OriginalIncidents, 2)
	require.Len(t, got.Plan, 2)
	require.Equal(t, d.OpsWorkflowStepStatusNotApplicable, got.Plan[1].RetryUpdateStatus)
	require.Equal(t, got.FrozenSet, got.Report.FrozenSet)
}

// TestRepairIncidentsSearchModeReportsBoundedSearchScope verifies search-mode previews identify batch-limited candidate scope.
func TestRepairIncidentsSearchModeReportsBoundedSearchScope(t *testing.T) {
	t.Parallel()

	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{
		searchIncidents: func(_ context.Context, _ d.IncidentFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.EqualValues(t, 2, size)
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "2251799813685249", ProcessInstanceKey: "2251799813685251", State: "ACTIVE"},
				{IncidentKey: "2251799813685250", ProcessInstanceKey: "2251799813685253", State: "ACTIVE"},
			}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName:       "ops repair incident",
		DiscoveryMode:     d.OpsRepairDiscoveryModeSearch,
		IncidentSelection: d.IncidentFilter{State: "ACTIVE"},
		BatchSize:         2,
		DryRun:            true,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Len(t, got.Notices, 1)
	require.Equal(t, "bounded_search_scope", got.Notices[0].Code)
	require.Contains(t, got.Notices[0].Message, "candidate incidents reached batch size 2")
	require.Equal(t, got.Notices, got.Report.Notices)
}

// TestRepairIncidentsSearchModeDoesNotExpandAfterFreeze verifies mutation uses only the initially discovered incident set.
func TestRepairIncidentsSearchModeDoesNotExpandAfterFreeze(t *testing.T) {
	t.Parallel()

	var resolved []string
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{
		searchIncidents: func(_ context.Context, _ d.IncidentFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "2251799813685249", ProcessInstanceKey: "2251799813685251", State: "ACTIVE", ErrorType: "IO_MAPPING_ERROR"},
			}, nil
		},
		resolveIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			resolved = append(resolved, key)
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusNoContent, Status: "accepted"}, nil
		},
		waitForIncidentResolved: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "resolved"}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName:       "ops repair incident",
		DiscoveryMode:     d.OpsRepairDiscoveryModeSearch,
		IncidentSelection: d.IncidentFilter{State: "active"},
		BatchSize:         10,
		NoWait:            true,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomeRepaired, got.Outcome)
	require.Equal(t, []string{"2251799813685249"}, resolved)
	require.Equal(t, []string{"2251799813685249"}, []string(got.FrozenSet.IncidentKeys))
	require.Len(t, got.Plan, 1)
}

// TestOpsRepairIncidentsDryRunPlansVariablesJobsAndResolutionWithoutMutation verifies dry-run builds the full plan and skips every mutation primitive.
func TestOpsRepairIncidentsDryRunPlansVariablesJobsAndResolutionWithoutMutation(t *testing.T) {
	t.Parallel()

	retries := int32(3)
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{
		getIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			return d.ProcessInstanceIncidentDetail{
				IncidentKey:        key,
				ProcessInstanceKey: "2251799813685251",
				JobKey:             "2251799813685252",
				State:              "ACTIVE",
			}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName:         "ops repair incident",
		InputKeys:           typex.Keys{"2251799813685249"},
		Variables:           map[string]any{"customerTier": "gold"},
		RequestedRetries:    &retries,
		RequestedJobTimeout: 5 * time.Minute,
		DryRun:              true,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Len(t, got.VariableUpdates, 1)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.VariableUpdates[0].Status)
	require.Len(t, got.Plan, 1)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].VariableUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].TimeoutUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].ResolutionStatus)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Plan[0].ConfirmationStatus)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Remaining.Status)
	require.Equal(t, []string{"2251799813685252"}, []string(got.FrozenSet.JobKeys))
	require.Equal(t, []string{"2251799813685251"}, []string(got.FrozenSet.VariableScopes))
}

// TestOpsRepairProcessInstancesSearchDryRunPlansMixedApplicabilityWithoutMutation verifies process-instance search gets the same dry-run plan shape.
func TestOpsRepairProcessInstancesSearchDryRunPlansMixedApplicabilityWithoutMutation(t *testing.T) {
	t.Parallel()

	hasIncident := true
	retries := int32(1)
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, d.ProcessInstanceFilter{State: d.StateActive, HasIncident: &hasIncident}, filter)
			require.EqualValues(t, 2, size)
			return []d.ProcessInstance{
				{Key: "2251799813685251", State: d.StateActive},
				{Key: "2251799813685253", State: d.StateActive},
			}, nil
		},
	}, repairIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			switch key {
			case "2251799813685251":
				return []d.ProcessInstanceIncidentDetail{{IncidentKey: "2251799813685249", ProcessInstanceKey: key, JobKey: "2251799813685252", State: "ACTIVE"}}, nil
			case "2251799813685253":
				return []d.ProcessInstanceIncidentDetail{{IncidentKey: "2251799813685250", ProcessInstanceKey: key, State: "ACTIVE"}}, nil
			default:
				t.Fatalf("unexpected process instance key %s", key)
				return nil, nil
			}
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{
		CommandName:              "ops repair process-instance",
		DiscoveryMode:            d.OpsRepairDiscoveryModeSearch,
		ProcessInstanceSelection: d.ProcessInstanceFilter{State: d.StateActive, HasIncident: &hasIncident},
		BatchSize:                2,
		RequestedRetries:         &retries,
		Variables:                map[string]any{"customerTier": "gold"},
		DryRun:                   true,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"2251799813685251", "2251799813685253"}, []string(got.FrozenSet.ProcessInstanceKeys))
	require.Equal(t, []string{"2251799813685249", "2251799813685250"}, []string(got.FrozenSet.IncidentKeys))
	require.Len(t, got.VariableUpdates, 2)
	require.Len(t, got.Plan, 2)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].VariableUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan[0].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusNotApplicable, got.Plan[1].RetryUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Remaining.Status)
	require.Len(t, got.Notices, 1)
	require.Equal(t, "bounded_search_scope", got.Notices[0].Code)
}

// TestRepairIncidentsDedupesVariableScopesAndConfirmsRequestedNames verifies shared scope updates run once before dependent incidents resolve.
func TestRepairIncidentsDedupesVariableScopesAndConfirmsRequestedNames(t *testing.T) {
	t.Parallel()

	var variableUpdates testx.SafeSlice[string]
	var resolved testx.SafeSlice[string]
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		updateVariables: func(_ context.Context, key string, variables map[string]any, _ ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
			variableUpdates.Append(key)
			require.Equal(t, map[string]any{"customerTier": "gold"}, variables)
			return d.ProcessInstanceVariableUpdateResponse{Key: key, Ok: true, StatusCode: http.StatusNoContent, Status: "204 No Content"}, nil
		},
		searchVariables: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			return []d.ProcessInstanceVariable{{Name: "customerTier", Value: `"gold"`, ProcessInstanceKey: key, ScopeKey: key}}, nil
		},
	}, repairIncidentAPI{
		getIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			return d.ProcessInstanceIncidentDetail{IncidentKey: key, ProcessInstanceKey: "2251799813685251", State: "ACTIVE"}, nil
		},
		resolveIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			resolved.Append(key)
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusNoContent, Status: "accepted"}, nil
		},
		waitForIncidentResolved: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "resolved"}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName: "ops repair incident",
		InputKeys:   typex.Keys{"2251799813685249", "2251799813685250"},
		Variables:   map[string]any{"customerTier": "gold"},
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsRepairOutcomeRepaired, got.Outcome)
	require.Equal(t, []string{"2251799813685251"}, variableUpdates.Snapshot())
	require.ElementsMatch(t, []string{"2251799813685249", "2251799813685250"}, resolved.Snapshot())
	require.Len(t, got.VariableUpdates, 1)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.VariableUpdates[0].Status)
	require.Equal(t, []string{"customerTier"}, got.VariableUpdates[0].VariableNames)
	require.Equal(t, []string{"2251799813685249", "2251799813685250"}, []string(got.VariableUpdates[0].DependentIncidentKeys))
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Plan[0].VariableUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Plan[1].VariableUpdateStatus)
}

// TestRepairIncidentsBlocksResolutionWhenVariableScopeFails verifies failed variable repair gates dependent incident resolution.
func TestRepairIncidentsBlocksResolutionWhenVariableScopeFails(t *testing.T) {
	t.Parallel()

	var resolved []string
	api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{
		updateVariables: func(_ context.Context, key string, variables map[string]any, _ ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
			require.Equal(t, "2251799813685251", key)
			require.Equal(t, map[string]any{"customerTier": "gold"}, variables)
			return d.ProcessInstanceVariableUpdateResponse{Key: key, Ok: false, StatusCode: http.StatusConflict, Status: "409 Conflict"}, errors.New("variable update rejected")
		},
	}, repairIncidentAPI{
		getIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
			return d.ProcessInstanceIncidentDetail{IncidentKey: key, ProcessInstanceKey: "2251799813685251", State: "ACTIVE"}, nil
		},
		resolveIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
			resolved = append(resolved, key)
			return d.IncidentResolutionResponse{Key: key, Ok: true, StatusCode: http.StatusNoContent, Status: "accepted"}, nil
		},
	}, nil, nil, repairJobAPI{}, "")

	got, err := api.RepairIncidents(context.Background(), d.OpsRepairRequest{
		CommandName: "ops repair incident",
		InputKeys:   typex.Keys{"2251799813685249"},
		Variables:   map[string]any{"customerTier": "gold"},
		NoWait:      true,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "variable update rejected")
	require.Equal(t, d.OpsRepairOutcomeFailed, got.Outcome)
	require.Empty(t, resolved)
	require.Len(t, got.VariableUpdates, 1)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.VariableUpdates[0].Status)
	require.Len(t, got.Plan, 1)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan[0].VariableUpdateStatus)
	require.Equal(t, d.OpsWorkflowStepStatusBlocked, got.Plan[0].ResolutionStatus)
}

type stubJobAPI struct {
	jsvc.API
}

type repairIncidentAPI struct {
	incsvc.API
	getIncident                     func(context.Context, string, ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	resolveIncident                 func(context.Context, string, ...services.CallOption) (d.IncidentResolutionResponse, error)
	waitForIncidentResolved         func(context.Context, string, ...services.CallOption) (d.IncidentResolutionResponse, error)
	searchIncidents                 func(context.Context, d.IncidentFilter, int32, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	searchProcessInstanceIncidents  func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	waitForProcessIncidentsResolved func(context.Context, string, []string, ...services.CallOption) (d.IncidentResolutionResponse, error)
}

func (s repairIncidentAPI) GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	if s.getIncident == nil {
		panic("unexpected get incident")
	}
	return s.getIncident(ctx, key, opts...)
}

func (s repairIncidentAPI) ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.resolveIncident == nil {
		panic("unexpected resolve incident")
	}
	return s.resolveIncident(ctx, key, opts...)
}

func (s repairIncidentAPI) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.waitForIncidentResolved == nil {
		panic("unexpected wait for incident resolved")
	}
	return s.waitForIncidentResolved(ctx, key, opts...)
}

func (s repairIncidentAPI) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchIncidents == nil {
		panic("unexpected search incidents")
	}
	return s.searchIncidents(ctx, filter, size, opts...)
}

func (s repairIncidentAPI) SearchIncidentsPage(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error) {
	panic("unexpected search incidents page")
}

func (s repairIncidentAPI) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchProcessInstanceIncidents == nil {
		panic("unexpected search process instance incidents")
	}
	return s.searchProcessInstanceIncidents(ctx, key, opts...)
}

func (s repairIncidentAPI) WaitForProcessInstanceIncidentsResolved(ctx context.Context, key string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.waitForProcessIncidentsResolved == nil {
		panic("unexpected wait for process instance incidents resolved")
	}
	return s.waitForProcessIncidentsResolved(ctx, key, incidentKeys, opts...)
}

type repairJobAPI struct {
	jsvc.API
	updateJob func(context.Context, d.JobUpdateRequest, ...services.CallOption) (d.JobUpdateResult, error)
}

func (s repairJobAPI) GetJob(context.Context, string, ...services.CallOption) (d.Job, error) {
	panic("unexpected get job")
}

func (s repairJobAPI) UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error) {
	if s.updateJob == nil {
		panic("unexpected update job")
	}
	return s.updateJob(ctx, request, opts...)
}
