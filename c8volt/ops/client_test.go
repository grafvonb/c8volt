// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	opsvc "github.com/grafvonb/c8volt/internal/services/ops"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

func TestClientPurgeOrphanProcessInstancesMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 11, 12, 30, 0, 0, time.UTC)
	hasIncident := true
	api := stubOpsService{
		purge: func(_ context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error) {
			require.Equal(t, d.OrphanPurgeRequest{
				CommandName: "ops purge orphan-process-instances",
				DryRun:      true,
				AutoConfirm: true,
				Automation:  true,
				OutputMode:  "json",
				Selection: d.ProcessInstanceFilter{
					BpmnProcessId:     "invoice",
					ProcessVersion:    3,
					ProcessVersionTag: "stable",
					State:             d.StateActive,
					HasIncident:       &hasIncident,
				},
				BatchSize:    250,
				Limit:        10,
				Workers:      4,
				ReportFile:   "report.json",
				ReportFormat: "json",
				DiscoveredKeys: typex.Keys{
					"2251799813685249",
				},
				StartedAt: started,
			}, request)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.OrphanPurgeResult{
				Request: request,
				Discovery: d.OrphanDiscoveryResult{
					Status: d.OpsWorkflowStepStatusPlanned,
					Keys:   []string{"2251799813685249"},
					Count:  1,
				},
				DeletionPlan: d.DeletionPlan{
					Status:               d.OpsWorkflowStepStatusPlanned,
					RequestedKeys:        []string{"2251799813685249"},
					AffectedKeys:         []string{"2251799813685249", "2251799813685250"},
					RootKeys:             []string{"2251799813685248"},
					RequiresConfirmation: true,
					DryRunPreview: d.DryRunPIKeyExpansion{
						Roots:     []string{"2251799813685248"},
						Collected: []string{"2251799813685249", "2251799813685250"},
						Outcome:   d.TraversalOutcomeComplete,
					},
				},
				Deletion: d.DeletionResult{
					Status:    d.OpsWorkflowStepStatusSubmitted,
					Submitted: true,
					Items: []d.Reporter{
						{Key: "2251799813685248", Ok: true, StatusCode: 202, Status: "accepted"},
					},
				},
				DeleteRequested: true,
				Outcome:         d.OrphanPurgeOutcomePlanned,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).PurgeOrphanProcessInstances(context.Background(), OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		DryRun:      true,
		AutoConfirm: true,
		Automation:  true,
		OutputMode:  "json",
		Selection: process.ProcessInstanceFilter{
			BpmnProcessId:     "invoice",
			ProcessVersion:    3,
			ProcessVersionTag: "stable",
			State:             process.StateActive,
			HasIncident:       &hasIncident,
		},
		BatchSize:    250,
		Limit:        10,
		Workers:      4,
		ReportFile:   "report.json",
		ReportFormat: "json",
		DiscoveredKeys: typex.Keys{
			"2251799813685249",
		},
		StartedAt: started,
	}, foptions.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, OrphanPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"2251799813685249"}, []string(got.Discovery.Keys))
	require.Equal(t, []string{"2251799813685248"}, []string(got.DeletionPlan.RootKeys))
	require.Equal(t, process.TraversalOutcomeComplete, got.DeletionPlan.DryRunPreview.Outcome)
	require.True(t, got.DeleteRequested)
	require.Equal(t, WorkflowStepStatusSubmitted, got.Deletion.Status)
	require.Equal(t, []process.DeleteReport{{Key: "2251799813685248", Ok: true, StatusCode: 202, Status: "accepted"}}, got.Deletion.Items)
}

func TestClientExecuteRetentionPolicyMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 14, 9, 30, 0, 0, time.UTC)
	hasIncident := false
	api := stubOpsService{
		retention: func(_ context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error) {
			require.Equal(t, d.RetentionPolicyRequest{
				CommandName:            "ops execute retention-policy",
				RetentionDays:          90,
				DerivedEndDateBoundary: "2026-02-13T00:00:00Z",
				DryRun:                 true,
				AutoConfirm:            true,
				Automation:             true,
				OutputMode:             "json",
				Selection: d.ProcessInstanceFilter{
					BpmnProcessId:        "invoice",
					ProcessDefinitionKey: "2251799813685250",
					ProcessVersion:       3,
					ProcessVersionTag:    "stable",
					State:                d.StateCompleted,
					HasIncident:          &hasIncident,
				},
				BatchSize:     250,
				Limit:         10,
				Workers:       4,
				NoWait:        true,
				NoStateCheck:  true,
				Force:         true,
				FailFast:      true,
				NoWorkerLimit: true,
				ReportFile:    "retention-report.json",
				ReportFormat:  "json",
				StartedAt:     started,
			}, request)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.Verbose)
			require.True(t, cfg.NoWait)
			require.True(t, cfg.Force)
			require.True(t, cfg.FailFast)
			return d.RetentionPolicyResult{
				Request: request,
				Discovery: d.RetentionDiscoveryResult{
					Status:                 d.OpsWorkflowStepStatusPlanned,
					RetentionDays:          90,
					DerivedEndDateBoundary: "2026-02-13T00:00:00Z",
					Filters: d.ProcessInstanceFilter{
						EndDateBefore: "2026-02-13T00:00:00Z",
					},
					SeedKeys: []string{"2251799813685249"},
					Count:    1,
				},
				DeletePlan: d.RetentionDeletePlan{
					Status:               d.OpsWorkflowStepStatusPlanned,
					SeedKeys:             []string{"2251799813685249"},
					ResolvedRootKeys:     []string{"2251799813685248"},
					AffectedKeys:         []string{"2251799813685248", "2251799813685249"},
					RequiresConfirmation: true,
				},
				Deletion: d.RetentionDeletionResult{
					Status:            d.OpsWorkflowStepStatusSubmitted,
					SubmittedRootKeys: []string{"2251799813685248"},
					Submitted:         true,
					NoWait:            true,
					Items: []d.Reporter{
						{Key: "2251799813685248", Ok: true, StatusCode: 202, Status: "accepted"},
					},
				},
				Outcome: d.RetentionPolicyOutcomePlanned,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).ExecuteRetentionPolicy(context.Background(), RetentionPolicyRequest{
		CommandName:            "ops execute retention-policy",
		RetentionDays:          90,
		DerivedEndDateBoundary: "2026-02-13T00:00:00Z",
		DryRun:                 true,
		AutoConfirm:            true,
		Automation:             true,
		OutputMode:             "json",
		Selection: process.ProcessInstanceFilter{
			BpmnProcessId:        "invoice",
			ProcessDefinitionKey: "2251799813685250",
			ProcessVersion:       3,
			ProcessVersionTag:    "stable",
			State:                process.StateCompleted,
			HasIncident:          &hasIncident,
		},
		BatchSize:     250,
		Limit:         10,
		Workers:       4,
		NoWait:        true,
		NoStateCheck:  true,
		Force:         true,
		FailFast:      true,
		NoWorkerLimit: true,
		ReportFile:    "retention-report.json",
		ReportFormat:  "json",
		StartedAt:     started,
	}, foptions.WithVerbose(), foptions.WithNoWait(), foptions.WithForce(), foptions.WithFailFast())

	require.NoError(t, err)
	require.Equal(t, RetentionPolicyOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"2251799813685249"}, []string(got.Discovery.SeedKeys))
	require.Equal(t, "2026-02-13T00:00:00Z", got.Discovery.DerivedEndDateBoundary)
	require.Equal(t, process.ProcessInstanceFilter{EndDateBefore: "2026-02-13T00:00:00Z"}, got.Discovery.Filters)
	require.Equal(t, []string{"2251799813685248"}, []string(got.DeletePlan.ResolvedRootKeys))
	require.Equal(t, WorkflowStepStatusSubmitted, got.Deletion.Status)
	require.True(t, got.Deletion.NoWait)
	require.Equal(t, []process.DeleteReport{{Key: "2251799813685248", Ok: true, StatusCode: 202, Status: "accepted"}}, got.Deletion.Items)
}

func TestClientExecuteRetentionPolicyNormalizesValidationErrors(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		retention: func(_ context.Context, request d.RetentionPolicyRequest, _ ...services.CallOption) (d.RetentionPolicyResult, error) {
			err := errors.New("unexpected")
			err = errors.Join(d.ErrValidation, err)
			return d.RetentionPolicyResult{Request: request, Outcome: d.RetentionPolicyOutcomeFailed}, err
		},
	}

	_, err := New(api, slog.Default()).ExecuteRetentionPolicy(context.Background(), RetentionPolicyRequest{RetentionDays: -1})

	require.ErrorIs(t, err, ferr.ErrInvalidInput)
}

type stubOpsService struct {
	purge     func(context.Context, d.OrphanPurgeRequest, ...services.CallOption) (d.OrphanPurgeResult, error)
	retention func(context.Context, d.RetentionPolicyRequest, ...services.CallOption) (d.RetentionPolicyResult, error)
}

func (s stubOpsService) PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error) {
	if s.purge == nil {
		panic("unexpected call")
	}
	return s.purge(ctx, request, opts...)
}

func (s stubOpsService) ExecuteRetentionPolicy(ctx context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error) {
	if s.retention == nil {
		panic("unexpected call")
	}
	return s.retention(ctx, request, opts...)
}

var _ opsvc.API = stubOpsService{}
