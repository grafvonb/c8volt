// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"
	"testing"
	"time"

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

type stubOpsService struct {
	purge func(context.Context, d.OrphanPurgeRequest, ...services.CallOption) (d.OrphanPurgeResult, error)
}

func (s stubOpsService) PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error) {
	if s.purge == nil {
		panic("unexpected call")
	}
	return s.purge(ctx, request, opts...)
}

var _ opsvc.API = stubOpsService{}
