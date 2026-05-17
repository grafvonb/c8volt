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
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
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

	incAPI := stubIncidentAPI{
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.Equal(t, request.Selection, filter)
			require.EqualValues(t, 5, size)
			return nil, nil
		},
	}

	got, err := New(stubProcessInstanceAPI{}, incAPI).PurgeProcessInstancesWithIncidents(
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

// TestPurgeProcessInstancesWithIncidentsDryRunDiscoversFrozenCandidates verifies incident discovery, dedupe, skips, and limit capping before delete planning exists.
func TestPurgeProcessInstancesWithIncidentsDryRunDiscoversFrozenCandidates(t *testing.T) {
	t.Parallel()

	incAPI := stubIncidentAPI{
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.Equal(t, d.IncidentFilter{State: "ACTIVE", ErrorType: "JOB_NO_RETRIES", Keys: []string{"9001"}}, filter)
			require.EqualValues(t, 3, size)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "9001", ProcessInstanceKey: "1001", State: "ACTIVE"},
				{IncidentKey: "9002", ProcessInstanceKey: "1001", State: "ACTIVE"},
				{IncidentKey: "9003", State: "ACTIVE"},
				{IncidentKey: "9004", ProcessInstanceKey: "1002", State: "ACTIVE"},
			}, nil
		},
	}
	request := d.IncidentPurgeRequest{
		CommandName: "ops purge process-instances-with-incidents",
		DryRun:      true,
		Selection: d.IncidentFilter{
			Keys:      []string{"9001"},
			State:     "ACTIVE",
			ErrorType: "JOB_NO_RETRIES",
		},
		BatchSize: 100,
		Limit:     3,
		StartedAt: time.Date(2026, 5, 16, 11, 0, 0, 0, time.UTC),
	}

	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "1001", startKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: startKey,
				RootKey:  startKey,
				Keys:     []string{startKey},
				Chain: map[string]d.ProcessInstance{
					startKey: {Key: startKey, State: d.StateCompleted},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "1001", rootKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeDescendants,
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{rootKey},
				Chain: map[string]d.ProcessInstance{
					rootKey: {Key: rootKey, State: d.StateCompleted},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := New(piAPI, incAPI).PurgeProcessInstancesWithIncidents(context.Background(), request, services.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, d.IncidentPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Equal(t, []string{"9001", "9002", "9003"}, []string(got.Discovery.IncidentKeys))
	require.Len(t, got.Discovery.CandidateIncidents, 3)
	require.Equal(t, []string{"1001"}, []string(got.Discovery.CandidateProcessInstanceKeys))
	require.Equal(t, []string{"1001"}, []string(got.Discovery.DuplicateCandidateProcessInstanceKeys))
	require.Len(t, got.Discovery.SkippedIncidents, 1)
	require.Equal(t, "9003", got.Discovery.SkippedIncidents[0].Incident.IncidentKey)
	require.Equal(t, "missing process-instance key", got.Discovery.SkippedIncidents[0].Reason)
	require.Equal(t, 3, got.Discovery.IncidentCount)
	require.Equal(t, 1, got.Discovery.CandidateProcessInstanceCount)
	require.Equal(t, []string{"duplicate_candidate_process_instances", "skipped_incidents"}, []string{got.Discovery.Notices[0].Code, got.Discovery.Notices[1].Code})
	require.Len(t, got.Notices, 2)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.Equal(t, []string{"1001"}, []string(got.DeletePlan.CandidateProcessInstanceKeys))
	require.Equal(t, []string{"1001"}, []string(got.DeletePlan.ResolvedRootKeys))
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, got.Discovery, got.Report.Discovery)
	require.Empty(t, got.Errors)
}

// TestPurgeProcessInstancesWithIncidentsDryRunNoTargetsSkipsPlanning records the no-target discovery result without mutation.
func TestPurgeProcessInstancesWithIncidentsDryRunNoTargetsSkipsPlanning(t *testing.T) {
	t.Parallel()

	incAPI := stubIncidentAPI{
		searchIncidents: func(_ context.Context, filter d.IncidentFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.Equal(t, d.IncidentFilter{State: "ACTIVE"}, filter)
			require.EqualValues(t, 1000, size)
			return nil, nil
		},
	}

	got, err := New(stubProcessInstanceAPI{}, incAPI).PurgeProcessInstancesWithIncidents(context.Background(), d.IncidentPurgeRequest{
		DryRun:    true,
		Selection: d.IncidentFilter{State: "ACTIVE"},
	})

	require.NoError(t, err)
	require.Equal(t, d.IncidentPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Zero(t, got.Discovery.IncidentCount)
	require.Zero(t, got.Discovery.CandidateProcessInstanceCount)
	require.Empty(t, got.Discovery.CandidateProcessInstanceKeys)
	require.Equal(t, "no_candidate_incidents", got.Discovery.Notices[0].Code)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
}

// TestPurgeProcessInstancesWithIncidentsDryRunBuildsDeletePlan verifies frozen incident candidates are expanded by the existing delete-plan path.
func TestPurgeProcessInstancesWithIncidentsDryRunBuildsDeletePlan(t *testing.T) {
	t.Parallel()

	incAPI := stubIncidentAPI{
		searchIncidents: func(_ context.Context, _ d.IncidentFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "inc-1", ProcessInstanceKey: "child-1", State: "ACTIVE"},
				{IncidentKey: "inc-2", ProcessInstanceKey: "child-2", State: "ACTIVE"},
			}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			result := pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: startKey,
				RootKey:  "root-1",
				Keys:     []string{startKey, "root-1"},
				Chain: map[string]d.ProcessInstance{
					startKey: {Key: startKey, State: d.StateCompleted},
					"root-1": {Key: "root-1", State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}
			if startKey == "child-2" {
				result.MissingAncestors = []pitraversal.MissingAncestor{{Key: "missing-parent", StartKey: startKey}}
				result.Warning = "one or more parent process instances were not found"
				result.Outcome = pitraversal.OutcomePartial
			}
			return result, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "root-1", rootKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeDescendants,
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{"root-1", "child-1", "child-2"},
				Chain: map[string]d.ProcessInstance{
					"root-1":  {Key: "root-1", State: d.StateTerminated},
					"child-1": {Key: "child-1", State: d.StateCompleted},
					"child-2": {Key: "child-2", State: d.StateCompleted},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := New(piAPI, incAPI).PurgeProcessInstancesWithIncidents(context.Background(), d.IncidentPurgeRequest{
		DryRun:  true,
		Workers: 2,
	})

	require.NoError(t, err)
	require.Equal(t, d.IncidentPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.Equal(t, []string{"child-1", "child-2"}, []string(got.DeletePlan.CandidateProcessInstanceKeys))
	require.Equal(t, []string{"root-1"}, []string(got.DeletePlan.ResolvedRootKeys))
	require.Equal(t, []string{"root-1", "child-1", "child-2"}, []string(got.DeletePlan.AffectedKeys))
	require.Equal(t, []string{"root-1"}, []string(got.DeletePlan.DuplicateResolvedRootKeys))
	require.Len(t, got.DeletePlan.FinalStateItems, 2)
	require.Empty(t, got.DeletePlan.NonFinalAffectedItems)
	require.Equal(t, []d.MissingAncestor{{Key: "missing-parent", StartKey: "child-2"}}, got.DeletePlan.MissingAncestors)
	require.Equal(t, []string{"one or more parent process instances were not found"}, got.DeletePlan.TraversalWarnings)
	require.False(t, got.DeletePlan.RequiresConfirmation)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, got.DeletePlan, got.Report.DeletePlan)
}

// TestPurgeProcessInstancesWithIncidentsBlocksNonFinalDestructivePlan verifies planning stops destructive runs before mutation when --force is absent.
func TestPurgeProcessInstancesWithIncidentsBlocksNonFinalDestructivePlan(t *testing.T) {
	t.Parallel()

	incAPI := stubIncidentAPI{
		searchIncidents: func(_ context.Context, _ d.IncidentFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			return []d.ProcessInstanceIncidentDetail{{IncidentKey: "inc-1", ProcessInstanceKey: "child-1", State: "ACTIVE"}}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: startKey,
				RootKey:  "root-1",
				Keys:     []string{startKey, "root-1"},
				Chain: map[string]d.ProcessInstance{
					startKey: {Key: startKey, State: d.StateActive},
					"root-1": {Key: "root-1", State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				Mode:     pitraversal.ModeDescendants,
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{"root-1", "child-1"},
				Chain: map[string]d.ProcessInstance{
					"root-1":  {Key: "root-1", State: d.StateTerminated},
					"child-1": {Key: "child-1", State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := New(piAPI, incAPI).PurgeProcessInstancesWithIncidents(context.Background(), d.IncidentPurgeRequest{})

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrPrecondition)
	require.Contains(t, err.Error(), "no delete request was submitted")
	require.Equal(t, d.IncidentPurgeOutcomeFailed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.True(t, got.DeletePlan.RequiresConfirmation)
	require.Len(t, got.DeletePlan.NonFinalAffectedItems, 1)
	require.Equal(t, "child-1", got.DeletePlan.NonFinalAffectedItems[0].Key)
	require.Equal(t, d.OpsWorkflowStepStatusBlocked, got.Deletion.Status)
	require.Len(t, got.Deletion.Errors, 1)
	require.Len(t, got.Errors, 1)
}

// TestPurgeProcessInstancesWithIncidentsExecutesFrozenPlanRoots verifies destructive execution submits only resolved plan roots.
func TestPurgeProcessInstancesWithIncidentsExecutesFrozenPlanRoots(t *testing.T) {
	t.Parallel()

	incAPI := stubIncidentAPI{
		searchIncidents: func(_ context.Context, _ d.IncidentFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			return []d.ProcessInstanceIncidentDetail{{IncidentKey: "inc-1", ProcessInstanceKey: "child-1", State: "ACTIVE"}}, nil
		},
	}
	var deleted []string
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "child-1", startKey)
			require.True(t, services.ApplyCallOptions(opts).NoWait)
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: startKey,
				RootKey:  "root-1",
				Keys:     []string{startKey, "root-1"},
				Chain: map[string]d.ProcessInstance{
					startKey: {Key: startKey, State: d.StateCompleted},
					"root-1": {Key: "root-1", State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "root-1", rootKey)
			require.True(t, services.ApplyCallOptions(opts).NoWait)
			return pitraversal.Result{
				Mode:     pitraversal.ModeDescendants,
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{"root-1", "child-1"},
				Chain: map[string]d.ProcessInstance{
					"root-1":  {Key: "root-1", State: d.StateTerminated},
					"child-1": {Key: "child-1", State: d.StateCompleted},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.NoWait)
			require.True(t, cfg.Force)
			require.True(t, cfg.FailFast)
			require.True(t, cfg.NoWorkerLimit)
			deleted = append(deleted, key)
			return d.DeleteResponse{Ok: true, StatusCode: http.StatusAccepted, Status: "accepted"}, nil
		},
	}

	got, err := New(piAPI, incAPI).PurgeProcessInstancesWithIncidents(
		context.Background(),
		d.IncidentPurgeRequest{Workers: 2},
		services.WithNoWait(),
		services.WithForce(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.IncidentPurgeOutcomeDeleted, got.Outcome)
	require.Equal(t, []string{"root-1"}, deleted)
	require.Equal(t, d.OpsWorkflowStepStatusSubmitted, got.Deletion.Status)
	require.Equal(t, []string{"root-1"}, []string(got.Deletion.SubmittedRootKeys))
	require.True(t, got.Deletion.Submitted)
	require.False(t, got.Deletion.Confirmed)
	require.True(t, got.Deletion.NoWait)
	require.Equal(t, got.Deletion, got.Report.Deletion)
}

// TestPurgeProcessInstancesWithIncidentsUsesFrozenCandidatesWithoutRediscovery protects confirmed command execution from scope drift.
func TestPurgeProcessInstancesWithIncidentsUsesFrozenCandidatesWithoutRediscovery(t *testing.T) {
	t.Parallel()

	incAPI := stubIncidentAPI{
		searchIncidents: func(context.Context, d.IncidentFilter, int32, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			t.Fatal("incident discovery should not run when frozen candidate keys are supplied")
			return nil, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "child-1", startKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: startKey,
				RootKey:  "root-1",
				Keys:     []string{startKey, "root-1"},
				Chain: map[string]d.ProcessInstance{
					startKey: {Key: startKey, State: d.StateCompleted},
					"root-1": {Key: "root-1", State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "root-1", rootKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeDescendants,
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{"root-1"},
				Chain: map[string]d.ProcessInstance{
					"root-1": {Key: "root-1", State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "root-1", key)
			return d.DeleteResponse{Ok: true, StatusCode: http.StatusAccepted, Status: "accepted"}, nil
		},
	}

	got, err := New(piAPI, incAPI).PurgeProcessInstancesWithIncidents(context.Background(), d.IncidentPurgeRequest{
		DiscoveredCandidateProcessInstanceKeys: typexKeys("child-1"),
	}, services.WithNoWait())

	require.NoError(t, err)
	require.Equal(t, d.IncidentPurgeOutcomeDeleted, got.Outcome)
	require.Equal(t, []string{"child-1"}, []string(got.Discovery.CandidateProcessInstanceKeys))
	require.Equal(t, 1, got.Discovery.CandidateProcessInstanceCount)
	require.Zero(t, got.Discovery.IncidentCount)
	require.Equal(t, []string{"root-1"}, []string(got.Deletion.SubmittedRootKeys))
}

type stubIncidentAPI struct {
	incsvc.API
	searchIncidents func(context.Context, d.IncidentFilter, int32, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
}

func (s stubIncidentAPI) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchIncidents == nil {
		panic("unexpected incident search")
	}
	return s.searchIncidents(ctx, filter, size, opts...)
}
