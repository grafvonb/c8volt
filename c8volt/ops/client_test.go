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
	"github.com/grafvonb/c8volt/c8volt/incident"
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

func TestClientPurgeProcessInstancesWithIncidentsMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 16, 8, 45, 0, 0, time.UTC)
	api := stubOpsService{
		incidentPurge: func(_ context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error) {
			require.Equal(t, d.IncidentPurgeRequest{
				CommandName:   "ops purge process-instances-with-incidents",
				DryRun:        true,
				AutoConfirm:   true,
				Automation:    true,
				OutputMode:    "json",
				Selection:     d.IncidentFilter{State: "ACTIVE", ErrorType: "JOB_NO_RETRIES", ProcessInstanceKey: "pi-a"},
				BatchSize:     50,
				Limit:         5,
				Workers:       3,
				FailFast:      true,
				NoWorkerLimit: true,
				NoWait:        true,
				Force:         true,
				ReportFile:    "incident-purge.json",
				ReportFormat:  "json",
				DiscoveredCandidateProcessInstanceKeys: typex.Keys{
					"pi-a",
				},
				StartedAt: started,
			}, request)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.Verbose)
			require.True(t, cfg.NoWait)
			require.True(t, cfg.Force)
			require.True(t, cfg.FailFast)
			return d.IncidentPurgeResult{
				Request: request,
				Discovery: d.IncidentDiscoveryResult{
					Status:                                d.OpsWorkflowStepStatusPlanned,
					Filters:                               request.Selection,
					CandidateIncidents:                    []d.ProcessInstanceIncidentDetail{{IncidentKey: "inc-a", ProcessInstanceKey: "pi-a"}, {IncidentKey: "inc-b", ProcessInstanceKey: "pi-a"}, {IncidentKey: "inc-c"}},
					IncidentKeys:                          typex.Keys{"inc-a", "inc-b", "inc-c"},
					CandidateProcessInstanceKeys:          typex.Keys{"pi-a"},
					DuplicateCandidateProcessInstanceKeys: typex.Keys{"pi-a"},
					SkippedIncidents:                      []d.IncidentPurgeSkippedIncident{{Incident: d.ProcessInstanceIncidentDetail{IncidentKey: "inc-c"}, Reason: "missing process-instance key"}},
					IncidentCount:                         3,
					CandidateProcessInstanceCount:         1,
					Notices:                               []d.IncidentPurgeWorkflowNotice{{Code: "candidate_duplicates", Severity: "info", Message: "duplicates found", Details: map[string]string{"processInstanceKey": "pi-a"}}},
				},
				DeletePlan: d.IncidentPurgeDeletePlan{
					Status:                       d.OpsWorkflowStepStatusPlanned,
					CandidateProcessInstanceKeys: typex.Keys{"pi-a"},
					ResolvedRootKeys:             typex.Keys{"root-a"},
					AffectedKeys:                 typex.Keys{"root-a", "pi-a"},
					DuplicateResolvedRootKeys:    typex.Keys{"root-a"},
					RequiresConfirmation:         true,
				},
				Deletion: d.IncidentPurgeDeletionResult{
					Status:            d.OpsWorkflowStepStatusSubmitted,
					SubmittedRootKeys: typex.Keys{"root-a"},
					Submitted:         true,
					NoWait:            true,
					Items: []d.Reporter{
						{Key: "root-a", Ok: true, StatusCode: 202, Status: "accepted"},
					},
				},
				Outcome: d.IncidentPurgeOutcomePlanned,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).PurgeProcessInstancesWithIncidents(context.Background(), IncidentPurgeRequest{
		CommandName:   "ops purge process-instances-with-incidents",
		DryRun:        true,
		AutoConfirm:   true,
		Automation:    true,
		OutputMode:    "json",
		Selection:     incident.Filter{State: "ACTIVE", ErrorType: "JOB_NO_RETRIES", ProcessInstanceKey: "pi-a"},
		BatchSize:     50,
		Limit:         5,
		Workers:       3,
		FailFast:      true,
		NoWorkerLimit: true,
		NoWait:        true,
		Force:         true,
		ReportFile:    "incident-purge.json",
		ReportFormat:  "json",
		DiscoveredCandidateProcessInstanceKeys: typex.Keys{
			"pi-a",
		},
		StartedAt: started,
	}, foptions.WithVerbose(), foptions.WithNoWait(), foptions.WithForce(), foptions.WithFailFast())

	require.NoError(t, err)
	require.Equal(t, IncidentPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"inc-a", "inc-b", "inc-c"}, []string(got.Discovery.IncidentKeys))
	require.Equal(t, []string{"pi-a"}, []string(got.Discovery.CandidateProcessInstanceKeys))
	require.Equal(t, []string{"pi-a"}, []string(got.Discovery.DuplicateCandidateProcessInstanceKeys))
	require.Len(t, got.Discovery.SkippedIncidents, 1)
	require.Equal(t, "inc-c", got.Discovery.SkippedIncidents[0].Incident.IncidentKey)
	require.Equal(t, "missing process-instance key", got.Discovery.SkippedIncidents[0].Reason)
	require.Equal(t, "candidate_duplicates", got.Discovery.Notices[0].Code)
	require.Equal(t, "pi-a", got.Discovery.Notices[0].Details["processInstanceKey"])
	require.Equal(t, []string{"root-a"}, []string(got.DeletePlan.ResolvedRootKeys))
	require.Equal(t, []string{"root-a"}, []string(got.DeletePlan.DuplicateResolvedRootKeys))
	require.Equal(t, WorkflowStepStatusSubmitted, got.Deletion.Status)
	require.True(t, got.Deletion.NoWait)
	require.Equal(t, []process.DeleteReport{{Key: "root-a", Ok: true, StatusCode: 202, Status: "accepted"}}, got.Deletion.Items)
}

func TestClientPurgeAllProcessDefinitionsMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 16, 18, 0, 0, 0, time.UTC)
	api := stubOpsService{
		allProcessDefinitionsPurge: func(_ context.Context, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error) {
			require.Equal(t, d.AllProcessDefinitionsPurgeRequest{
				CommandName:   "ops purge all-process-definitions",
				DryRun:        true,
				AutoConfirm:   true,
				Automation:    true,
				OutputMode:    "json",
				Selection:     d.ProcessDefinitionFilter{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "stable", IsLatestVersion: true},
				Workers:       3,
				FailFast:      true,
				NoWorkerLimit: true,
				NoWait:        true,
				Force:         true,
				ReportFile:    "all-pds.json",
				ReportFormat:  "json",
				DiscoveredCandidateProcessDefinitionKeys: typex.Keys{
					"pd-a",
				},
				StartedAt: started,
			}, request)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.Verbose)
			require.True(t, cfg.NoWait)
			require.True(t, cfg.Force)
			require.True(t, cfg.FailFast)
			return d.AllProcessDefinitionsPurgeResult{
				Request: request,
				Discovery: d.ProcessDefinitionDiscoveryResult{
					Status:                         d.OpsWorkflowStepStatusPlanned,
					Filters:                        request.Selection,
					CandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
					CandidateProcessDefinitions: []d.ProcessDefinition{{
						Key:               "pd-a",
						BpmnProcessId:     "invoice",
						Name:              "Invoice",
						TenantId:          "tenant-a",
						ProcessVersion:    3,
						ProcessVersionTag: "stable",
						Statistics:        &d.ProcessDefinitionStatistics{Active: 2, Completed: 5, IncidentCountSupported: true},
					}},
					DuplicateCandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
					CandidateProcessDefinitionCount:         1,
					LatestOnly:                              true,
					Notices:                                 []d.AllProcessDefinitionsPurgeWorkflowNotice{{Code: "candidate_duplicates", Severity: "info", Message: "duplicates found", Details: map[string]string{"processDefinitionKey": "pd-a"}}},
				},
				DeletePlan: d.AllProcessDefinitionsPurgeDeletePlan{
					Status:                         d.OpsWorkflowStepStatusPlanned,
					CandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
					Items: []d.DeleteProcessDefinitionPlanItem{{
						Key:                        "pd-a",
						ActiveProcessInstanceCount: 2,
						ActiveProcessInstanceKeys:  []string{"pi-a", "pi-b"},
						CancellationPlan: d.DryRunPIKeyExpansion{
							Roots:     typex.Keys{"pi-a"},
							Collected: typex.Keys{"pi-a", "pi-b"},
							Outcome:   d.TraversalOutcomeComplete,
						},
					}},
					DuplicateCandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
					AffectedProcessInstanceCount:            2,
					ActiveProcessInstanceCount:              2,
					RequiresConfirmation:                    true,
					RequiresForce:                           true,
				},
				Deletion: d.AllProcessDefinitionsPurgeDeletionResult{
					Status:                         d.OpsWorkflowStepStatusSubmitted,
					SubmittedProcessDefinitionKeys: typex.Keys{"pd-a"},
					Submitted:                      true,
					NoWait:                         true,
					Items: []d.ResourceDeleteResponse{{
						Ok:                true,
						StatusCode:        202,
						Status:            "accepted",
						BatchOperationKey: "batch-a",
						BatchState:        "ACTIVE",
						DeleteHistory:     true,
					}},
				},
				Outcome: d.AllProcessDefinitionsPurgeOutcomePlanned,
				Notices: []d.AllProcessDefinitionsPurgeWorkflowNotice{{Code: "candidate_duplicates", Severity: "info", Message: "duplicates found", Details: map[string]string{"processDefinitionKey": "pd-a"}}},
			}, nil
		},
	}

	got, err := New(api, slog.Default()).PurgeAllProcessDefinitions(context.Background(), AllProcessDefinitionsPurgeRequest{
		CommandName:   "ops purge all-process-definitions",
		DryRun:        true,
		AutoConfirm:   true,
		Automation:    true,
		OutputMode:    "json",
		Selection:     ProcessDefinitionSelection{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "stable", LatestOnly: true},
		Workers:       3,
		FailFast:      true,
		NoWorkerLimit: true,
		NoWait:        true,
		Force:         true,
		ReportFile:    "all-pds.json",
		ReportFormat:  "json",
		DiscoveredCandidateProcessDefinitionKeys: typex.Keys{
			"pd-a",
		},
		StartedAt: started,
	}, foptions.WithVerbose(), foptions.WithNoWait(), foptions.WithForce(), foptions.WithFailFast())

	require.NoError(t, err)
	require.Equal(t, AllProcessDefinitionsPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"pd-a"}, []string(got.Discovery.CandidateProcessDefinitionKeys))
	require.Equal(t, []string{"pd-a"}, []string(got.Discovery.DuplicateCandidateProcessDefinitionKeys))
	require.True(t, got.Discovery.LatestOnly)
	require.Equal(t, "invoice", got.Discovery.CandidateProcessDefinitions[0].BpmnProcessId)
	require.EqualValues(t, 2, got.Discovery.CandidateProcessDefinitions[0].Statistics.Active)
	require.Equal(t, "candidate_duplicates", got.Discovery.Notices[0].Code)
	require.Equal(t, "pd-a", got.Discovery.Notices[0].Details["processDefinitionKey"])
	require.Equal(t, []string{"pd-a"}, []string(got.DeletePlan.CandidateProcessDefinitionKeys))
	require.EqualValues(t, 2, got.DeletePlan.ActiveProcessInstanceCount)
	require.True(t, got.DeletePlan.RequiresForce)
	require.Equal(t, []string{"pi-a"}, []string(got.DeletePlan.Items[0].CancellationPlan.Roots))
	require.Equal(t, WorkflowStepStatusSubmitted, got.Deletion.Status)
	require.True(t, got.Deletion.NoWait)
	require.Equal(t, "batch-a", got.Deletion.Items[0].BatchOperationKey)
	require.Equal(t, "candidate_duplicates", got.Notices[0].Code)
}

// TestClientPurgeAllProcessDefinitionsMapsDiscoveryFields protects public discovery output conversion.
func TestClientPurgeAllProcessDefinitionsMapsDiscoveryFields(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		allProcessDefinitionsPurge: func(_ context.Context, request d.AllProcessDefinitionsPurgeRequest, _ ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error) {
			discovery := d.ProcessDefinitionDiscoveryResult{
				Status:                         d.OpsWorkflowStepStatusPlanned,
				Filters:                        request.Selection,
				CandidateProcessDefinitionKeys: typex.Keys{"pd-a", "pd-b"},
				CandidateProcessDefinitions: []d.ProcessDefinition{
					{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 2},
					{Key: "pd-b", BpmnProcessId: "payment", ProcessVersion: 1, ProcessVersionTag: "stable"},
				},
				DuplicateCandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
				CandidateProcessDefinitionCount:         2,
				LatestOnly:                              true,
				Notices:                                 []d.AllProcessDefinitionsPurgeWorkflowNotice{{Code: "latest_only_scope", Severity: "info", Message: "latest scope", Details: map[string]string{"scope": "latest"}}},
			}
			return d.AllProcessDefinitionsPurgeResult{
				Request:   request,
				Discovery: discovery,
				Report:    d.AllProcessDefinitionsPurgeReport{Discovery: discovery},
				Outcome:   d.AllProcessDefinitionsPurgeOutcomePlanned,
				Notices:   discovery.Notices,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).PurgeAllProcessDefinitions(context.Background(), AllProcessDefinitionsPurgeRequest{
		Selection: ProcessDefinitionSelection{BpmnProcessId: "invoice", LatestOnly: true},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"pd-a", "pd-b"}, []string(got.Discovery.CandidateProcessDefinitionKeys))
	require.Equal(t, []string{"pd-a"}, []string(got.Discovery.DuplicateCandidateProcessDefinitionKeys))
	require.Len(t, got.Discovery.CandidateProcessDefinitions, 2)
	require.Equal(t, "payment", got.Discovery.CandidateProcessDefinitions[1].BpmnProcessId)
	require.Equal(t, "stable", got.Discovery.CandidateProcessDefinitions[1].ProcessVersionTag)
	require.True(t, got.Discovery.LatestOnly)
	require.Equal(t, "latest_only_scope", got.Discovery.Notices[0].Code)
	require.Equal(t, "latest", got.Discovery.Notices[0].Details["scope"])
	require.Equal(t, got.Discovery, got.Report.Discovery)
	require.Equal(t, "latest_only_scope", got.Notices[0].Code)
}

type stubOpsService struct {
	purge                      func(context.Context, d.OrphanPurgeRequest, ...services.CallOption) (d.OrphanPurgeResult, error)
	retention                  func(context.Context, d.RetentionPolicyRequest, ...services.CallOption) (d.RetentionPolicyResult, error)
	incidentPurge              func(context.Context, d.IncidentPurgeRequest, ...services.CallOption) (d.IncidentPurgeResult, error)
	allProcessDefinitionsPurge func(context.Context, d.AllProcessDefinitionsPurgeRequest, ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error)
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

func (s stubOpsService) PurgeProcessInstancesWithIncidents(ctx context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error) {
	if s.incidentPurge == nil {
		panic("unexpected call")
	}
	return s.incidentPurge(ctx, request, opts...)
}

func (s stubOpsService) PurgeAllProcessDefinitions(ctx context.Context, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error) {
	if s.allProcessDefinitionsPurge == nil {
		panic("unexpected call")
	}
	return s.allProcessDefinitionsPurge(ctx, request, opts...)
}

var _ opsvc.API = stubOpsService{}
