// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
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
					Status:        d.OpsWorkflowStepStatusPlanned,
					RequestedKeys: []string{"2251799813685249"},
					AffectedKeys:  []string{"2251799813685249", "2251799813685250"},
					RootKeys:      []string{"2251799813685248"},
					TenantEvidence: d.TenantEvidence{
						ResolvedTenantIDs: []string{"tenant-a"},
						TargetCount:       1,
						Targets:           []d.TenantEvidenceTarget{{Key: "2251799813685249", TenantID: "tenant-a"}},
					},
					RequiresConfirmation: true,
					DryRunPreview: d.DryRunPIKeyExpansion{
						Roots:     []string{"2251799813685248"},
						Collected: []string{"2251799813685249", "2251799813685250"},
						TenantEvidence: d.TenantEvidence{
							ResolvedTenantIDs: []string{"tenant-a"},
							TargetCount:       1,
							Targets:           []d.TenantEvidenceTarget{{Key: "2251799813685249", TenantID: "tenant-a"}},
						},
						Outcome: d.TraversalOutcomeComplete,
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
	require.Equal(t, process.TenantEvidence{
		ResolvedTenantIDs: []string{"tenant-a"},
		TargetCount:       1,
		Targets:           []process.TenantEvidenceTarget{{Key: "2251799813685249", TenantID: "tenant-a"}},
	}, got.DeletionPlan.TenantEvidence)
	require.Equal(t, got.DeletionPlan.TenantEvidence, got.DeletionPlan.DryRunPreview.TenantEvidence)
	require.Equal(t, process.TraversalOutcomeComplete, got.DeletionPlan.DryRunPreview.Outcome)
	require.True(t, got.DeleteRequested)
	require.Equal(t, WorkflowStepStatusSubmitted, got.Deletion.Status)
	require.Equal(t, []process.DeleteReport{{Key: "2251799813685248", Ok: true, StatusCode: 202, Status: "accepted"}}, got.Deletion.Items)
}

// TestClientExecuteSmokeTestMapsServiceBoundary verifies the new smoke-test facade remains a thin mapping layer.
func TestClientExecuteSmokeTestMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 17, 9, 45, 0, 0, time.UTC)
	finished := started.Add(30 * time.Second)
	api := stubOpsService{
		smokeTest: func(_ context.Context, request d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
			require.Equal(t, d.SmokeTestRequest{
				CommandName:   "ops execute smoke-test",
				DryRun:        true,
				Count:         2,
				Workers:       3,
				FailFast:      true,
				NoWorkerLimit: true,
				NoCleanup:     true,
				AutoConfirm:   true,
				Automation:    true,
				NoWait:        true,
				OutputMode:    "json",
				ReportFile:    "smoke-test.json",
				ReportFormat:  "json",
				StartedAt:     started,
			}, request)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.NoWait)
			require.True(t, cfg.FailFast)
			require.True(t, cfg.NoWorkerLimit)
			return d.SmokeTestResult{
				Request: request,
				Plan: d.SmokeTestPlan{
					Status:           d.OpsWorkflowStepStatusPlanned,
					CamundaVersion:   "8.9",
					CleanupRequested: false,
					Fixture: d.EmbeddedSmokeTestFixture{
						CamundaVersion: "8.9",
						File:           "embedded/processdefinitions/C89_MultipleSubProcessesParent.bpmn",
						BpmnProcessID:  "C89_MultipleSubProcessesParent",
						Available:      true,
					},
					PlannedSteps: []d.WorkflowStepResult{{Name: "deploy", Status: d.OpsWorkflowStepStatusPlanned, Message: "deploy fixture"}},
				},
				Fixture: d.EmbeddedSmokeTestFixture{
					CamundaVersion: "8.9",
					File:           "embedded/processdefinitions/C89_MultipleSubProcessesParent.bpmn",
					BpmnProcessID:  "C89_MultipleSubProcessesParent",
					Available:      true,
				},
				Deployment: d.SmokeTestDeploymentResult{
					Status:                   d.OpsWorkflowStepStatusSubmitted,
					FixtureFile:              "embedded/processdefinitions/C89_MultipleSubProcessesParent.bpmn",
					BpmnProcessID:            "C89_MultipleSubProcessesParent",
					ProcessDefinitionKey:     "pd-1",
					ProcessDefinitionVersion: 7,
					TenantID:                 "tenant-a",
				},
				Run: d.SmokeTestRunResult{
					Status:              d.OpsWorkflowStepStatusConfirmed,
					RequestedCount:      2,
					CreatedCount:        2,
					ProcessInstanceKeys: typex.Keys{"pi-1", "pi-2"},
					Items: []d.SmokeTestRunItem{
						{ProcessInstanceKey: "pi-1", Status: d.OpsWorkflowStepStatusConfirmed},
						{ProcessInstanceKey: "pi-2", Status: d.OpsWorkflowStepStatusConfirmed},
					},
				},
				Walk: d.SmokeTestWalkResult{
					Status: d.OpsWorkflowStepStatusConfirmed,
					Items: []d.SmokeTestWalkItem{{
						ProcessInstanceKey: "pi-1",
						Status:             d.OpsWorkflowStepStatusConfirmed,
						Summary: d.SmokeTestTraversalSummary{
							ProcessInstanceKey:     "pi-1",
							RootProcessInstanceKey: "root-1",
							FamilyKeys:             typex.Keys{"root-1", "pi-1"},
							MissingAncestors:       []d.MissingAncestor{{Key: "missing", StartKey: "pi-1"}},
							Outcome:                d.TraversalOutcomePartial,
						},
					}},
				},
				Cleanup: d.SmokeTestCleanupResult{
					NoCleanup:                    true,
					RetainedProcessInstanceKeys:  typex.Keys{"pi-1", "pi-2"},
					RetainedProcessDefinitionKey: "pd-1",
					RetainedBpmnProcessID:        "C89_MultipleSubProcessesParent",
					RetainedTenantID:             "tenant-a",
					ProcessInstanceCleanup: d.SmokeTestProcessInstanceCleanupResult{
						Status:        d.OpsWorkflowStepStatusSkipped,
						SubmittedKeys: typex.Keys{"pi-1", "pi-2"},
						Items:         []d.Reporter{{Key: "pi-1", Ok: true, StatusCode: 202, Status: "accepted"}},
						NoWait:        true,
					},
					ProcessDefinitionEligibility: d.SmokeTestCleanupEligibility{
						Status:   d.OpsWorkflowStepStatusSkipped,
						Eligible: true,
					},
					ProcessDefinitionCleanup: d.SmokeTestProcessDefinitionCleanupResult{
						Status:                        d.OpsWorkflowStepStatusSkipped,
						SubmittedProcessDefinitionKey: "pd-1",
						Items: []d.ResourceDeleteResponse{{
							Key:        "pd-1",
							Ok:         true,
							StatusCode: 202,
							Status:     "accepted",
						}},
						NoWait: true,
					},
				},
				Report: d.SmokeTestAuditReport{
					SchemaVersion:    "ops.smoke-test.v1",
					CommandName:      "ops execute smoke-test",
					StartedAt:        started,
					FinishedAt:       finished,
					Duration:         "30s",
					DryRun:           true,
					CamundaVersion:   "8.9",
					ProfileIdentity:  "profile-a",
					TenantID:         "tenant-a",
					CleanupRequested: false,
					NoCleanup:        true,
					NoWait:           true,
					Outcome:          d.SmokeTestOutcomePassedCleanupSkipped,
				},
				Outcome: d.SmokeTestOutcomePassedCleanupSkipped,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).ExecuteSmokeTest(context.Background(), SmokeTestRequest{
		CommandName:   "ops execute smoke-test",
		DryRun:        true,
		Count:         2,
		Workers:       3,
		FailFast:      true,
		NoWorkerLimit: true,
		NoCleanup:     true,
		AutoConfirm:   true,
		Automation:    true,
		NoWait:        true,
		OutputMode:    "json",
		ReportFile:    "smoke-test.json",
		ReportFormat:  "json",
		StartedAt:     started,
	}, foptions.WithNoWait(), foptions.WithFailFast(), foptions.WithNoWorkerLimit())

	require.NoError(t, err)
	require.Equal(t, SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
	require.Equal(t, "C89_MultipleSubProcessesParent", got.Fixture.BpmnProcessID)
	require.Equal(t, []string{"pi-1", "pi-2"}, []string(got.Run.ProcessInstanceKeys))
	require.Equal(t, WorkflowStepStatusConfirmed, got.Walk.Items[0].Status)
	require.Equal(t, process.TraversalOutcomePartial, got.Walk.Items[0].Summary.Outcome)
	require.Equal(t, []process.MissingAncestor{{Key: "missing", StartKey: "pi-1"}}, got.Walk.Items[0].Summary.MissingAncestors)
	require.Equal(t, []process.DeleteReport{{Key: "pi-1", Ok: true, StatusCode: 202, Status: "accepted"}}, got.Cleanup.ProcessInstanceCleanup.Items)
	require.Equal(t, []resource.DeleteReport{{Key: "pd-1", Ok: true, StatusCode: 202, Status: "accepted"}}, got.Cleanup.ProcessDefinitionCleanup.Items)
	require.Equal(t, []string{"pi-1", "pi-2"}, []string(got.Cleanup.RetainedProcessInstanceKeys))
	require.Equal(t, "pd-1", got.Cleanup.RetainedProcessDefinitionKey)
	require.Equal(t, "C89_MultipleSubProcessesParent", got.Cleanup.RetainedBpmnProcessID)
	require.Equal(t, "tenant-a", got.Cleanup.RetainedTenantID)
	require.Equal(t, "profile-a", got.Report.ProfileIdentity)
	require.True(t, got.Report.NoCleanup)
}

// TestClientExecuteSmokeTestMapsProgressOption verifies smoke-test callers can install structured progress through facade options.
func TestClientExecuteSmokeTestMapsProgressOption(t *testing.T) {
	t.Parallel()

	var gotEvent foptions.ProgressEvent
	api := stubOpsService{
		smokeTest: func(_ context.Context, _ d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
			progress := services.ApplyCallOptions(opts).Progress
			require.NotNil(t, progress)
			progress(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindFrozenScope,
				FrozenScope: &d.OpsFrozenScopeProgress{
					Phase:        "starting process instances",
					CoreResource: "process instance(s)",
					Done:         1,
					Total:        2,
				},
			})
			return d.SmokeTestResult{Request: d.SmokeTestRequest{CommandName: "ops execute smoke-test", Count: 2}}, nil
		},
	}

	_, err := New(api, slog.Default()).ExecuteSmokeTest(context.Background(), SmokeTestRequest{CommandName: "ops execute smoke-test", Count: 2}, foptions.WithProgress(func(event foptions.ProgressEvent) {
		gotEvent = event
	}))

	require.NoError(t, err)
	require.Equal(t, foptions.ProgressEventKindFrozenScope, gotEvent.Kind)
	require.Equal(t, &foptions.FrozenScopeProgress{Phase: "starting process instances", CoreResource: "process instance(s)", Done: 1, Total: 2}, gotEvent.FrozenScope)
}

// TestClientExecuteSmokeTestMapsProgressTenantContext verifies preflight
// callbacks expose the common tenant context through the public facade.
func TestClientExecuteSmokeTestMapsProgressTenantContext(t *testing.T) {
	t.Parallel()

	domainCtx := d.TenantContext{
		Mode:              d.TenantContextModeCreation,
		Filter:            d.TenantContextFilterNotApplicable,
		TargetTenantID:    "<default>",
		ResolvedTenantIDs: []string{"<default>"},
	}
	var gotEvent foptions.ProgressEvent
	api := stubOpsService{
		smokeTest: func(_ context.Context, _ d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
			progress := services.ApplyCallOptions(opts).Progress
			require.NotNil(t, progress)
			progress(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindPreflight,
				Preflight: &d.OpsPreflightScope{
					Phase:         "preflight",
					TenantContext: &domainCtx,
				},
			})
			return d.SmokeTestResult{Request: d.SmokeTestRequest{CommandName: "ops execute smoke-test", Count: 1}}, nil
		},
	}

	_, err := New(api, slog.Default()).ExecuteSmokeTest(context.Background(), SmokeTestRequest{CommandName: "ops execute smoke-test", Count: 1}, foptions.WithProgress(func(event foptions.ProgressEvent) {
		gotEvent = event
	}))
	require.NoError(t, err)
	domainCtx.ResolvedTenantIDs[0] = "changed"

	require.Equal(t, foptions.ProgressEventKindPreflight, gotEvent.Kind)
	require.NotNil(t, gotEvent.Preflight.TenantContext)
	require.Equal(t, foptions.TenantContextModeCreation, gotEvent.Preflight.TenantContext.Mode)
	require.Equal(t, foptions.TenantContextFilterNotApplicable, gotEvent.Preflight.TenantContext.Filter)
	require.Equal(t, "<default>", gotEvent.Preflight.TenantContext.TargetTenantID)
	require.Equal(t, []string{"<default>"}, gotEvent.Preflight.TenantContext.ResolvedTenantIDs)
}

// TestClientAnalyseAPILatencyMapsServiceBoundary verifies API-latency analysis stays a thin facade conversion.
func TestClientAnalyseAPILatencyMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)
	stagePlans := []d.APILatencyStagePlan{{Index: 1, WorkerCount: 1, PrimarySamples: 3, DerivedRequestLimit: 6}}
	unhealthy := []int{2}
	leaderless := []int{3}
	notices := []string{"notice-a"}
	limitations := []string{"limit-a"}
	api := stubOpsService{
		analyseAPILatency: func(_ context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
			require.Equal(t, "ops analyse api-latency", request.CommandName)
			require.Equal(t, d.APILatencyModeReadOnly, request.Mode)
			require.Equal(t, 7, request.Count)
			require.Equal(t, 4, request.Workers)
			require.Equal(t, "tenant-a", request.TenantID)
			require.Equal(t, 15*time.Second, request.HTTPTimeout)
			require.Equal(t, d.APILatencyBackoffExponential, request.Backoff.Strategy)
			require.Equal(t, "json", request.OutputMode)
			require.Equal(t, started, request.StartedAt)
			require.NotNil(t, request.Progress)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.APILatencyResult{
				SchemaVersion: d.APILatencySchemaVersion,
				Context: d.APILatencyRunContext{
					CommandName:    request.CommandName,
					SchemaVersion:  d.APILatencySchemaVersion,
					CamundaVersion: "8.9",
					Profile:        "profile-a",
					Tenant:         "tenant-a",
					StartedAt:      started,
					FinishedAt:     finished,
					Duration:       "2s",
				},
				Request: request,
				Plan: d.APILatencyPlan{
					Mode:                    request.Mode,
					Stages:                  stagePlans,
					PrimarySampleLimit:      7,
					PrimarySampleAllocation: 7,
					DerivedRequestLimit:     14,
					Notices:                 notices,
					Limitations:             limitations,
				},
				Topology: d.APILatencyTopologyEvidence{
					BrokerCount:          1,
					PartitionCount:       3,
					UnhealthyPartitions:  unhealthy,
					LeaderlessPartitions: leaderless,
					HealthKnown:          true,
				},
				Stages: []d.APILatencyStageResult{{
					Plan:                 stagePlans[0],
					Status:               d.APILatencyStageStatusCompleted,
					ActualMaxConcurrency: 1,
					PrimaryAttempts:      3,
					Categories: []d.APILatencyCategorySummary{{
						Category:  d.APILatencyCategoryTopologyRead,
						Attempts:  3,
						Successes: 3,
					}},
				}},
				Findings: []d.APILatencyFinding{{
					Code:              "no_abnormal_evidence",
					Evidence:          []string{"bounded sample complete"},
					LikelyArea:        "no abnormal evidence",
					Confidence:        d.APILatencyFindingConfidenceLow,
					Limitation:        "bounded",
					NextInvestigation: "rerun later",
				}},
				Notices:     notices,
				Limitations: limitations,
				Outcome:     d.APILatencyOutcomeCompleted,
			}, nil
		},
	}
	var publicEvent ProgressEvent

	got, err := New(api, slog.Default()).AnalyseAPILatency(context.Background(), APILatencyRequest{
		CommandName: "ops analyse api-latency",
		Mode:        APILatencyModeReadOnly,
		Count:       7,
		Workers:     4,
		TenantID:    "tenant-a",
		HTTPTimeout: 15 * time.Second,
		Backoff: APILatencyBackoff{
			Strategy:     APILatencyBackoffExponential,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     time.Second,
			Multiplier:   2,
			Timeout:      5 * time.Second,
			MaxRetries:   3,
		},
		OutputMode: "json",
		StartedAt:  started,
		Progress: func(event ProgressEvent) {
			publicEvent = event
		},
	}, foptions.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, APILatencyOutcomeCompleted, got.Outcome)
	require.Equal(t, APILatencyModeReadOnly, got.Plan.Mode)
	require.Equal(t, []APILatencyStagePlan{{Index: 1, WorkerCount: 1, PrimarySamples: 3, DerivedRequestLimit: 6}}, got.Plan.Stages)
	require.Equal(t, []int{2}, got.Topology.UnhealthyPartitions)
	require.Equal(t, []int{3}, got.Topology.LeaderlessPartitions)
	require.Equal(t, []string{"notice-a"}, got.Notices)
	require.Equal(t, []string{"bounded sample complete"}, got.Findings[0].Evidence)
	stagePlans[0].WorkerCount = 99
	unhealthy[0] = 99
	leaderless[0] = 99
	notices[0] = "mutated"
	limitations[0] = "mutated"
	require.Equal(t, 1, got.Plan.Stages[0].WorkerCount)
	require.Equal(t, []int{2}, got.Topology.UnhealthyPartitions)
	require.Equal(t, []int{3}, got.Topology.LeaderlessPartitions)
	require.Equal(t, []string{"notice-a"}, got.Notices)
	require.Equal(t, []string{"limit-a"}, got.Limitations)
	require.Zero(t, publicEvent)
}

// TestClientAnalyseAPILatencyMapsProgressOption verifies read-only progress options cross the facade boundary.
func TestClientAnalyseAPILatencyMapsProgressOption(t *testing.T) {
	t.Parallel()

	var gotEvent ProgressEvent
	api := stubOpsService{
		analyseAPILatency: func(_ context.Context, _ d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
			progress := services.ApplyCallOptions(opts).Progress
			require.NotNil(t, progress)
			progress(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindFrozenScope,
				FrozenScope: &d.OpsFrozenScopeProgress{
					Phase:        "measuring read-only API latency",
					CoreResource: "stage 1 sample cycle(s)",
					Done:         1,
					Total:        3,
				},
			})
			return d.APILatencyResult{SchemaVersion: d.APILatencySchemaVersion, Outcome: d.APILatencyOutcomeCompleted}, nil
		},
	}

	_, err := New(api, slog.Default()).AnalyseAPILatency(context.Background(), APILatencyRequest{Count: 3, Workers: 1}, foptions.WithProgress(func(event foptions.ProgressEvent) {
		gotEvent = ProgressEvent{
			Kind:        ProgressEventKind(event.Kind),
			FrozenScope: (*FrozenScopeProgress)(event.FrozenScope),
		}
	}))

	require.NoError(t, err)
	require.Equal(t, ProgressEventKindFrozenScope, gotEvent.Kind)
	require.NotNil(t, gotEvent.FrozenScope)
	require.Equal(t, "measuring read-only API latency", gotEvent.FrozenScope.Phase)
	require.Equal(t, 1, gotEvent.FrozenScope.Done)
	require.Equal(t, 3, gotEvent.FrozenScope.Total)
}

// TestClientAnalyseAPILatencyPreservesPartialUnavailableResults verifies partial read-only evidence survives error conversion.
func TestClientAnalyseAPILatencyPreservesPartialUnavailableResults(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		analyseAPILatency: func(_ context.Context, request d.APILatencyRequest, _ ...services.CallOption) (d.APILatencyResult, error) {
			return d.APILatencyResult{
				SchemaVersion: d.APILatencySchemaVersion,
				Request:       request,
				Plan: d.APILatencyPlan{
					Mode:                    d.APILatencyModeReadOnly,
					Stages:                  []d.APILatencyStagePlan{{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 2}},
					PrimarySampleLimit:      1,
					PrimarySampleAllocation: 1,
					DerivedRequestLimit:     2,
				},
				Stages: []d.APILatencyStageResult{{
					Plan:            d.APILatencyStagePlan{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 2},
					Status:          d.APILatencyStageStatusIncomplete,
					PrimaryAttempts: 2,
					DerivedAttempts: 1,
					Categories: []d.APILatencyCategorySummary{{
						Category:    d.APILatencyCategoryProcessInstanceRead,
						Attempts:    1,
						Unavailable: 1,
					}},
					Classifications: []d.APILatencyClassificationCount{{Classification: d.APILatencyClassificationUnsupported, Count: 1}},
				}},
				Findings: []d.APILatencyFinding{{
					Code:              "timeouts_observed",
					Evidence:          []string{"bounded partial evidence"},
					LikelyArea:        "gateway/connectivity/authentication",
					Confidence:        d.APILatencyFindingConfidenceMedium,
					Limitation:        "partial",
					NextInvestigation: "retry read-only analysis",
				}},
				Limitations: []string{"read-only evidence cannot prove write-path health"},
				Outcome:     d.APILatencyOutcomePartial,
			}, d.ErrUnavailable
		},
	}

	got, err := New(api, slog.Default()).AnalyseAPILatency(context.Background(), APILatencyRequest{Count: 1, Workers: 1})

	require.ErrorIs(t, err, ferr.ErrUnavailable)
	require.Equal(t, APILatencyOutcomePartial, got.Outcome)
	require.Equal(t, APILatencyStageStatusIncomplete, got.Stages[0].Status)
	require.Equal(t, APILatencyCategoryProcessInstanceRead, got.Stages[0].Categories[0].Category)
	require.Equal(t, APILatencyClassificationUnsupported, got.Stages[0].Classifications[0].Classification)
	require.Equal(t, []string{"bounded partial evidence"}, got.Findings[0].Evidence)
	require.Equal(t, []string{"read-only evidence cannot prove write-path health"}, got.Limitations)
}

// TestClientExecuteAPILatencyTestMapsActiveServiceBoundary verifies active API-latency evidence stays a thin facade conversion.
func TestClientExecuteAPILatencyTestMapsActiveServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 9, 2, 8, 20, 0, 0, time.UTC)
	finished := started.Add(4 * time.Second)
	stagePlans := []d.APILatencyStagePlan{{Index: 1, WorkerCount: 1, PrimarySamples: 3, DerivedRequestLimit: 12}}
	setupOps := []string{"topology preflight", "fixture deploy"}
	planNotices := []string{"cleanup supported"}
	planLimitations := []string{"bounded active sample"}
	unhealthy := []int{4}
	leaderless := []int{5}
	ownerKeys := []string{"pi-a", "pi-b"}
	findingEvidence := []string{"visibility p95 increased"}
	visibility := []d.APILatencyVisibilityResult{{
		ProcessInstanceKey:  "pi-a",
		Attempts:            2,
		AttemptLimit:        5,
		Visible:             true,
		Duration:            750 * time.Millisecond,
		FinalClassification: d.APILatencyClassificationSuccess,
	}}
	cleanup := []d.APILatencyCleanupRecord{
		{ResourceType: d.APILatencyCleanupResourceProcessInstance, Key: "pi-a", Status: d.APILatencyCleanupStatusSubmitted, Classification: d.APILatencyClassificationSuccess},
		{ResourceType: d.APILatencyCleanupResourceProcessInstance, Key: "pi-b", Status: d.APILatencyCleanupStatusFailed, Classification: d.APILatencyClassificationBackpressure, RecoveryCommand: "c8volt delete process-instance --key pi-b --force --auto-confirm"},
		{ResourceType: d.APILatencyCleanupResourceProcessDefinition, Key: "pd-a", Status: d.APILatencyCleanupStatusUnknown, Classification: d.APILatencyClassificationUnavailable, RecoveryCommand: "c8volt delete process-definition --key pd-a --auto-confirm"},
	}
	api := stubOpsService{
		executeAPILatencyTest: func(_ context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
			require.Equal(t, "ops execute api-latency-test", request.CommandName)
			require.Equal(t, d.APILatencyModeActive, request.Mode)
			require.Equal(t, 7, request.Count)
			require.Equal(t, 4, request.Workers)
			require.False(t, request.DryRun)
			require.True(t, request.NoCleanup)
			require.Equal(t, "tenant-a", request.TenantID)
			require.Equal(t, 15*time.Second, request.HTTPTimeout)
			require.Equal(t, d.APILatencyBackoffFixed, request.Backoff.Strategy)
			require.Equal(t, 250*time.Millisecond, request.Backoff.InitialDelay)
			require.Equal(t, "human", request.OutputMode)
			require.Equal(t, started, request.StartedAt)
			require.NotNil(t, request.Progress)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.Verbose)
			require.True(t, cfg.NoWorkerLimit)
			return d.APILatencyResult{
				SchemaVersion: d.APILatencySchemaVersion,
				Context: d.APILatencyRunContext{
					CommandName:    request.CommandName,
					SchemaVersion:  d.APILatencySchemaVersion,
					CamundaVersion: "8.9",
					Profile:        "profile-a",
					Tenant:         "tenant-a",
					StartedAt:      started,
					FinishedAt:     finished,
					Duration:       "4s",
				},
				Request: request,
				Plan: d.APILatencyPlan{
					RunID:                   "run-a",
					Mode:                    request.Mode,
					Stages:                  stagePlans,
					PrimarySampleLimit:      7,
					PrimarySampleAllocation: 7,
					DerivedRequestLimit:     42,
					VisibilityAttemptLimit:  5,
					SetupOperations:         setupOps,
					Fixture: &d.APILatencyFixturePlan{
						CamundaVersion: "8.9",
						File:           "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
						BpmnProcessID:  "C89_SimpleUserTask",
						Available:      true,
					},
					Cleanup: &d.APILatencyCleanupPlan{
						Requested:            true,
						Supported:            true,
						IntentionalRetention: true,
						IndependentBudget:    30 * time.Second,
					},
					Notices:     planNotices,
					Limitations: planLimitations,
				},
				Topology: d.APILatencyTopologyEvidence{
					BrokerCount:          2,
					PartitionCount:       6,
					UnhealthyPartitions:  unhealthy,
					LeaderlessPartitions: leaderless,
					HealthKnown:          true,
				},
				Stages: []d.APILatencyStageResult{{
					Plan:                 stagePlans[0],
					Status:               d.APILatencyStageStatusCompleted,
					ActualMaxConcurrency: 1,
					PrimaryAttempts:      3,
					DerivedAttempts:      6,
					Categories: []d.APILatencyCategorySummary{
						{Category: d.APILatencyCategoryProcessInstanceCreate, Attempts: 3, Successes: 3},
						{Category: d.APILatencyCategoryConcurrentRead, Attempts: 3, Successes: 2, Errors: 1},
						{Category: d.APILatencyCategorySearchVisibility, Attempts: 6, Successes: 5, Timeouts: 1},
					},
					Classifications: []d.APILatencyClassificationCount{{Classification: d.APILatencyClassificationBackpressure, Count: 1}},
				}},
				Findings: []d.APILatencyFinding{{
					Code:              "delayed_visibility",
					Evidence:          findingEvidence,
					LikelyArea:        "exporter visibility",
					Confidence:        d.APILatencyFindingConfidenceMedium,
					Limitation:        "bounded active sample",
					NextInvestigation: "compare exporter lag",
				}},
				Notices:     []string{"active notice"},
				Limitations: []string{"active limitation"},
				Ownership: &d.APILatencyOwnership{
					RunID:                "run-a",
					FixtureName:          "C89_SimpleUserTask.bpmn",
					BpmnProcessID:        "C89_SimpleUserTask",
					DeploymentSubmitted:  true,
					ProcessDefinitionKey: "pd-a",
					ProcessInstanceKeys:  ownerKeys,
				},
				Visibility: visibility,
				Cleanup:    cleanup,
				Outcome:    d.APILatencyOutcomeCompletedRetained,
			}, nil
		},
	}

	var publicEvent ProgressEvent
	got, err := New(api, slog.Default()).ExecuteAPILatencyTest(context.Background(), APILatencyRequest{
		CommandName: "ops execute api-latency-test",
		Mode:        APILatencyModeActive,
		Count:       7,
		Workers:     4,
		NoCleanup:   true,
		TenantID:    "tenant-a",
		HTTPTimeout: 15 * time.Second,
		Backoff: APILatencyBackoff{
			Strategy:     APILatencyBackoffFixed,
			InitialDelay: 250 * time.Millisecond,
			MaxDelay:     time.Second,
			Multiplier:   1,
			Timeout:      5 * time.Second,
			MaxRetries:   4,
		},
		OutputMode: "human",
		StartedAt:  started,
		Progress: func(event ProgressEvent) {
			publicEvent = event
		},
	}, foptions.WithVerbose(), foptions.WithNoWorkerLimit())

	require.NoError(t, err)
	require.Equal(t, APILatencyOutcomeCompletedRetained, got.Outcome)
	require.Equal(t, APILatencyModeActive, got.Plan.Mode)
	require.Equal(t, "run-a", got.Plan.RunID)
	require.Equal(t, 42, got.Plan.DerivedRequestLimit)
	require.Equal(t, 5, got.Plan.VisibilityAttemptLimit)
	require.Equal(t, []APILatencyStagePlan{{Index: 1, WorkerCount: 1, PrimarySamples: 3, DerivedRequestLimit: 12}}, got.Plan.Stages)
	require.Equal(t, []string{"topology preflight", "fixture deploy"}, got.Plan.SetupOperations)
	require.Equal(t, "C89_SimpleUserTask", got.Plan.Fixture.BpmnProcessID)
	require.True(t, got.Plan.Cleanup.IntentionalRetention)
	require.Equal(t, []int{4}, got.Topology.UnhealthyPartitions)
	require.Equal(t, []int{5}, got.Topology.LeaderlessPartitions)
	require.Equal(t, APILatencyCategoryProcessInstanceCreate, got.Stages[0].Categories[0].Category)
	require.Equal(t, APILatencyCategoryConcurrentRead, got.Stages[0].Categories[1].Category)
	require.Equal(t, APILatencyCategorySearchVisibility, got.Stages[0].Categories[2].Category)
	require.Equal(t, []string{"visibility p95 increased"}, got.Findings[0].Evidence)
	require.Equal(t, "pd-a", got.Ownership.ProcessDefinitionKey)
	require.Equal(t, []string{"pi-a", "pi-b"}, got.Ownership.ProcessInstanceKeys)
	require.Equal(t, 750*time.Millisecond, got.Visibility[0].Duration)
	require.Equal(t, APILatencyClassificationSuccess, got.Visibility[0].FinalClassification)
	require.Equal(t, APILatencyCleanupStatusSubmitted, got.Cleanup[0].Status)
	require.Equal(t, APILatencyCleanupStatusFailed, got.Cleanup[1].Status)
	require.Equal(t, APILatencyCleanupStatusUnknown, got.Cleanup[2].Status)
	require.Equal(t, "c8volt delete process-definition --key pd-a --auto-confirm", got.Cleanup[2].RecoveryCommand)
	require.Zero(t, publicEvent)

	stagePlans[0].WorkerCount = 99
	setupOps[0] = "mutated"
	planNotices[0] = "mutated"
	planLimitations[0] = "mutated"
	unhealthy[0] = 99
	leaderless[0] = 99
	ownerKeys[0] = "mutated"
	findingEvidence[0] = "mutated"
	visibility[0].ProcessInstanceKey = "mutated"
	cleanup[0].Key = "mutated"
	require.Equal(t, 1, got.Plan.Stages[0].WorkerCount)
	require.Equal(t, []string{"topology preflight", "fixture deploy"}, got.Plan.SetupOperations)
	require.Equal(t, []string{"cleanup supported"}, got.Plan.Notices)
	require.Equal(t, []string{"bounded active sample"}, got.Plan.Limitations)
	require.Equal(t, []int{4}, got.Topology.UnhealthyPartitions)
	require.Equal(t, []int{5}, got.Topology.LeaderlessPartitions)
	require.Equal(t, []string{"pi-a", "pi-b"}, got.Ownership.ProcessInstanceKeys)
	require.Equal(t, []string{"visibility p95 increased"}, got.Findings[0].Evidence)
	require.Equal(t, "pi-a", got.Visibility[0].ProcessInstanceKey)
	require.Equal(t, "pi-a", got.Cleanup[0].Key)
}

// TestClientExecuteAPILatencyTestMapsProgressOption verifies active progress options cross the facade boundary.
func TestClientExecuteAPILatencyTestMapsProgressOption(t *testing.T) {
	t.Parallel()

	var gotEvent foptions.ProgressEvent
	api := stubOpsService{
		executeAPILatencyTest: func(_ context.Context, _ d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
			progress := services.ApplyCallOptions(opts).Progress
			require.NotNil(t, progress)
			progress(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindFrozenScope,
				FrozenScope: &d.OpsFrozenScopeProgress{
					Phase:        "executing active API latency test",
					CoreResource: "stage 1 sample cycle(s)",
					Done:         2,
					Total:        7,
					Errors:       1,
				},
			})
			return d.APILatencyResult{SchemaVersion: d.APILatencySchemaVersion, Outcome: d.APILatencyOutcomeCompleted}, nil
		},
	}

	_, err := New(api, slog.Default()).ExecuteAPILatencyTest(context.Background(), APILatencyRequest{Count: 7, Workers: 4}, foptions.WithProgress(func(event foptions.ProgressEvent) {
		gotEvent = event
	}))

	require.NoError(t, err)
	require.Equal(t, foptions.ProgressEventKindFrozenScope, gotEvent.Kind)
	require.NotNil(t, gotEvent.FrozenScope)
	require.Equal(t, "executing active API latency test", gotEvent.FrozenScope.Phase)
	require.Equal(t, 2, gotEvent.FrozenScope.Done)
	require.Equal(t, 7, gotEvent.FrozenScope.Total)
	require.Equal(t, 1, gotEvent.FrozenScope.Errors)
}

// TestClientExecuteAPILatencyTestMapsPartialErrors verifies partial active evidence survives domain error conversion.
func TestClientExecuteAPILatencyTestMapsPartialErrors(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		executeAPILatencyTest: func(_ context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
			require.Equal(t, d.APILatencyModeActive, request.Mode)
			require.True(t, request.DryRun)
			require.True(t, services.ApplyCallOptions(opts).DryRun)
			return d.APILatencyResult{
				SchemaVersion: d.APILatencySchemaVersion,
				Request:       request,
				Plan: d.APILatencyPlan{
					RunID:                   "run-a",
					Mode:                    d.APILatencyModeActive,
					PrimarySampleLimit:      7,
					PrimarySampleAllocation: 7,
					DerivedRequestLimit:     28,
					VisibilityAttemptLimit:  3,
					SetupOperations:         []string{"preflight"},
					Cleanup:                 &d.APILatencyCleanupPlan{Requested: true, Supported: false, BlockReason: "complete cleanup unsupported"},
				},
				Ownership: &d.APILatencyOwnership{
					RunID:               "run-a",
					FixtureName:         "C89_SimpleUserTask.bpmn",
					BpmnProcessID:       "C89_SimpleUserTask",
					DeploymentSubmitted: false,
					ProcessInstanceKeys: []string{"pi-a"},
				},
				Visibility: []d.APILatencyVisibilityResult{{
					ProcessInstanceKey:  "pi-a",
					Attempts:            2,
					AttemptLimit:        3,
					Visible:             false,
					FinalClassification: d.APILatencyClassificationNotFound,
				}},
				Cleanup: []d.APILatencyCleanupRecord{{
					ResourceType:   d.APILatencyCleanupResourceProcessInstance,
					Key:            "pi-a",
					Status:         d.APILatencyCleanupStatusRetained,
					Classification: d.APILatencyClassificationUnavailable,
				}},
				Outcome: d.APILatencyOutcomePartial,
			}, d.ErrValidation
		},
	}

	got, err := New(api, slog.Default()).ExecuteAPILatencyTest(context.Background(), APILatencyRequest{
		Mode:    APILatencyModeActive,
		Count:   7,
		Workers: 4,
		DryRun:  true,
	}, foptions.WithDryRun())

	require.ErrorIs(t, err, ferr.ErrInvalidInput)
	require.Equal(t, APILatencyOutcomePartial, got.Outcome)
	require.Equal(t, "run-a", got.Plan.RunID)
	require.Equal(t, "complete cleanup unsupported", got.Plan.Cleanup.BlockReason)
	require.Equal(t, []string{"pi-a"}, got.Ownership.ProcessInstanceKeys)
	require.Equal(t, APILatencyClassificationNotFound, got.Visibility[0].FinalClassification)
	require.Equal(t, APILatencyCleanupStatusRetained, got.Cleanup[0].Status)
}

// TestClientExecuteAPILatencyTestPreservesPartialCleanupEvidence verifies cleanup remainders survive facade error conversion.
func TestClientExecuteAPILatencyTestPreservesPartialCleanupEvidence(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		executeAPILatencyTest: func(_ context.Context, request d.APILatencyRequest, _ ...services.CallOption) (d.APILatencyResult, error) {
			require.Equal(t, d.APILatencyModeActive, request.Mode)
			return d.APILatencyResult{
				SchemaVersion: d.APILatencySchemaVersion,
				Request:       request,
				Plan: d.APILatencyPlan{
					RunID: "run-cleanup",
					Mode:  d.APILatencyModeActive,
					Cleanup: &d.APILatencyCleanupPlan{
						Requested:         true,
						Supported:         true,
						IndependentBudget: 30 * time.Second,
					},
				},
				Ownership: &d.APILatencyOwnership{
					RunID:                "run-cleanup",
					FixtureName:          "C89_SimpleUserTask.bpmn",
					BpmnProcessID:        "C89_SimpleUserTask",
					DeploymentSubmitted:  true,
					ProcessDefinitionKey: "2251799813685250",
					ProcessInstanceKeys:  []string{"2251799813685248", "2251799813685249"},
				},
				Cleanup: []d.APILatencyCleanupRecord{
					{
						ResourceType:   d.APILatencyCleanupResourceProcessInstance,
						Key:            "2251799813685248",
						Status:         d.APILatencyCleanupStatusDeleted,
						Classification: d.APILatencyClassificationSuccess,
					},
					{
						ResourceType:    d.APILatencyCleanupResourceProcessInstance,
						Key:             "2251799813685249",
						Status:          d.APILatencyCleanupStatusUnknown,
						Classification:  d.APILatencyClassificationTimeout,
						RecoveryCommand: "c8volt delete process-instance --key 2251799813685249 --force --auto-confirm",
					},
					{
						ResourceType:    d.APILatencyCleanupResourceProcessDefinition,
						Key:             "2251799813685250",
						Status:          d.APILatencyCleanupStatusFailed,
						Classification:  d.APILatencyClassificationBackpressure,
						RecoveryCommand: "c8volt delete process-definition --key 2251799813685250 --auto-confirm",
					},
				},
				Outcome: d.APILatencyOutcomePartial,
			}, d.ErrGatewayTimeout
		},
	}

	got, err := New(api, slog.Default()).ExecuteAPILatencyTest(context.Background(), APILatencyRequest{
		Mode:    APILatencyModeActive,
		Count:   7,
		Workers: 4,
	})

	require.ErrorIs(t, err, ferr.ErrTimeout)
	require.Equal(t, APILatencyOutcomePartial, got.Outcome)
	require.Equal(t, "run-cleanup", got.Plan.RunID)
	require.Equal(t, 30*time.Second, got.Plan.Cleanup.IndependentBudget)
	require.True(t, got.Ownership.DeploymentSubmitted)
	require.Equal(t, "2251799813685250", got.Ownership.ProcessDefinitionKey)
	require.Equal(t, []string{"2251799813685248", "2251799813685249"}, got.Ownership.ProcessInstanceKeys)
	require.Len(t, got.Cleanup, 3)
	require.Equal(t, APILatencyCleanupStatusDeleted, got.Cleanup[0].Status)
	require.Empty(t, got.Cleanup[0].RecoveryCommand)
	require.Equal(t, APILatencyCleanupResourceProcessInstance, got.Cleanup[1].ResourceType)
	require.Equal(t, APILatencyCleanupStatusUnknown, got.Cleanup[1].Status)
	require.Equal(t, APILatencyClassificationTimeout, got.Cleanup[1].Classification)
	require.Equal(t, "c8volt delete process-instance --key 2251799813685249 --force --auto-confirm", got.Cleanup[1].RecoveryCommand)
	require.Equal(t, APILatencyCleanupResourceProcessDefinition, got.Cleanup[2].ResourceType)
	require.Equal(t, APILatencyCleanupStatusFailed, got.Cleanup[2].Status)
	require.Equal(t, APILatencyClassificationBackpressure, got.Cleanup[2].Classification)
	require.Equal(t, "c8volt delete process-definition --key 2251799813685250 --auto-confirm", got.Cleanup[2].RecoveryCommand)
}

// TestClientExecuteAPILatencyTestPreservesTerminalOutcomeEvidence verifies active terminal states keep partial service evidence.
func TestClientExecuteAPILatencyTestPreservesTerminalOutcomeEvidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		outcome     d.APILatencyOutcome
		serviceErr  error
		wantErr     error
		cleanupStat d.APILatencyCleanupStatus
	}{
		{
			name:        "interrupted",
			outcome:     d.APILatencyOutcomeInterrupted,
			serviceErr:  context.Canceled,
			wantErr:     context.Canceled,
			cleanupStat: d.APILatencyCleanupStatusDeleted,
		},
		{
			name:        "partial",
			outcome:     d.APILatencyOutcomePartial,
			serviceErr:  d.ErrGatewayTimeout,
			wantErr:     ferr.ErrTimeout,
			cleanupStat: d.APILatencyCleanupStatusUnknown,
		},
		{
			name:        "failed",
			outcome:     d.APILatencyOutcomeFailed,
			serviceErr:  d.ErrPrecondition,
			wantErr:     ferr.ErrLocalPrecondition,
			cleanupStat: d.APILatencyCleanupStatusFailed,
		},
		{
			name:        "completed retained",
			outcome:     d.APILatencyOutcomeCompletedRetained,
			cleanupStat: d.APILatencyCleanupStatusRetained,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := stubOpsService{
				executeAPILatencyTest: func(_ context.Context, request d.APILatencyRequest, _ ...services.CallOption) (d.APILatencyResult, error) {
					return d.APILatencyResult{
						SchemaVersion: d.APILatencySchemaVersion,
						Request:       request,
						Plan: d.APILatencyPlan{
							RunID: "run-terminal",
							Mode:  d.APILatencyModeActive,
							Cleanup: &d.APILatencyCleanupPlan{
								Requested:         tt.cleanupStat != d.APILatencyCleanupStatusRetained,
								Supported:         true,
								IndependentBudget: 45 * time.Second,
							},
						},
						Ownership: &d.APILatencyOwnership{
							RunID:                "run-terminal",
							FixtureName:          "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
							BpmnProcessID:        "C89_SimpleUserTask",
							DeploymentSubmitted:  true,
							ProcessDefinitionKey: "pd-terminal",
							ProcessInstanceKeys:  []string{"pi-terminal"},
						},
						Visibility: []d.APILatencyVisibilityResult{{
							ProcessInstanceKey:  "pi-terminal",
							Attempts:            2,
							AttemptLimit:        3,
							FinalClassification: d.APILatencyClassificationNotFound,
						}},
						Cleanup: []d.APILatencyCleanupRecord{{
							ResourceType:    d.APILatencyCleanupResourceProcessInstance,
							Key:             "pi-terminal",
							Status:          tt.cleanupStat,
							Classification:  d.APILatencyClassificationTimeout,
							RecoveryCommand: "c8volt delete process-instance --key pi-terminal --force --auto-confirm",
						}},
						Outcome: tt.outcome,
					}, tt.serviceErr
				},
			}

			got, err := New(api, slog.Default()).ExecuteAPILatencyTest(context.Background(), APILatencyRequest{
				Mode:    APILatencyModeActive,
				Count:   1,
				Workers: 1,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, APILatencyOutcome(tt.outcome), got.Outcome)
			require.Equal(t, "run-terminal", got.Plan.RunID)
			require.Equal(t, 45*time.Second, got.Plan.Cleanup.IndependentBudget)
			require.True(t, got.Ownership.DeploymentSubmitted)
			require.Equal(t, "pd-terminal", got.Ownership.ProcessDefinitionKey)
			require.Equal(t, []string{"pi-terminal"}, got.Ownership.ProcessInstanceKeys)
			require.Equal(t, APILatencyClassificationNotFound, got.Visibility[0].FinalClassification)
			require.Equal(t, APILatencyCleanupStatus(tt.cleanupStat), got.Cleanup[0].Status)
			require.Equal(t, "c8volt delete process-instance --key pi-terminal --force --auto-confirm", got.Cleanup[0].RecoveryCommand)
		})
	}
}

// TestClientAnalyseSlowProcessInstancesMapsListenerServiceBoundary verifies the slow-analysis facade stays thin.
func TestClientAnalyseSlowProcessInstancesMapsListenerServiceBoundary(t *testing.T) {
	t.Parallel()

	captured := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	rootDurationLonger := 10 * time.Minute
	durationAfter := 5 * time.Second
	progressTotal := int64(800)
	progressPages := int64(8)
	remaining := 3 * time.Minute
	var progressEvents []ProgressEvent
	api := stubOpsService{
		slowProcessAnalysis: func(_ context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
			require.NotNil(t, request.Progress)
			requestForCompare := request
			requestForCompare.Progress = nil
			require.Equal(t, d.SlowProcessAnalysisRequest{
				CommandName:   "ops analyse slow-process-instances",
				SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
				InputKeys:     typex.Keys{"2251799813685249"},
				ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
					BpmnProcessID:        "OrderProcess",
					ProcessDefinitionKey: "2251799813687001",
				},
				ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{
					State:           d.StateCompleted,
					StartDateAfter:  "2026-07-18",
					StartDateBefore: "2026-07-19",
					EndDateAfter:    "2026-07-18T10:00:00Z",
					EndDateBefore:   "2026-07-18T11:00:00Z",
					NoIncidentsOnly: true,
				},
				DetailFilters: d.SlowProcessAnalysisDetailFilters{
					ElementID:     "ReserveStock",
					Type:          "SERVICE_TASK",
					ElementState:  "COMPLETED",
					DurationAfter: durationAfter,
				},
				RootDurationLonger: rootDurationLonger,
				BatchSize:          250,
				Limit:              10,
				CapturedNow:        captured,
				OutputMode:         "json",
				WithListeners:      true,
			}, requestForCompare)
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			request.Progress(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindPreflight,
				Preflight: &d.OpsPreflightScope{
					Phase:           "preflight",
					Command:         request.CommandName,
					CoreResource:    "process_instance",
					SelectorSummary: "OrderProcess",
					Total:           &progressTotal,
					TotalKind:       d.OpsTotalCertaintyLowerBound,
					PageSize:        250,
					PageCount:       &progressPages,
					PageCountKind:   d.OpsPageCountKindEstimated,
					ConsequenceSummary: d.OpsConsequenceSummary{
						ResourceSummary: "at least 800 process instances",
						WorkSummary:     "discover all matches and load runtime element timelines",
						RiskSummary:     "read-only, expensive",
					},
				},
			})
			return d.SlowProcessAnalysisResult{
				Request: request,
				DiscoveredScopeStatus: d.DiscoveryScopeStatus{
					Complete:         false,
					Limited:          true,
					Limit:            10,
					BatchSize:        250,
					Pages:            2,
					CandidatesSeen:   12,
					CandidatesFrozen: 10,
				},
				PreflightScope: &d.OpsPreflightScope{
					Phase:           "preflight",
					Command:         request.CommandName,
					CoreResource:    "process_instance",
					SelectorSummary: "OrderProcess",
					Total:           &progressTotal,
					TotalKind:       d.OpsTotalCertaintyLowerBound,
					PageSize:        250,
					PageCount:       &progressPages,
					PageCountKind:   d.OpsPageCountKindEstimated,
				},
				FrozenScopeProgress: &d.OpsFrozenScopeProgress{
					Phase:        "loading runtime elements",
					CoreResource: "process instance(s)",
					Done:         48,
					Total:        800,
					Elapsed:      2 * time.Minute,
					ETA:          &remaining,
				},
				CapturedAt: captured,
				Items: []d.SlowProcessAnalysisProcessInstance{{
					Key:                    "2251799813685249",
					TenantID:               "tenant-a",
					BpmnProcessID:          "OrderProcess",
					ProcessDefinitionKey:   "2251799813687001",
					ProcessVersion:         7,
					State:                  d.StateCompleted,
					StartDate:              "2026-07-18T10:00:00Z",
					EndDate:                "2026-07-18T10:10:00Z",
					RootProcessInstanceKey: "2251799813685249",
					Duration:               "10m0s",
					DurationMillis:         600000,
					DurationAvailable:      true,
					RelativePercentile:     95,
					ComparisonSampleCount:  12,
					Timeline: []d.SlowProcessAnalysisTimelineEntry{{
						Kind:                  d.SlowProcessAnalysisTimelineEntryKindElement,
						ElementInstanceKey:    "2251799813685250",
						ElementID:             "ReserveStock",
						Type:                  "SERVICE_TASK",
						State:                 "COMPLETED",
						Duration:              "5s",
						DurationMillis:        5000,
						DurationAvailable:     true,
						ProcessDurationShare:  1,
						ComparisonSampleCount: 3,
						Listeners: &[]d.RuntimeListenerJob{{
							JobKey:             "2251799813689101",
							Kind:               d.JobKindTaskListener,
							ListenerEventType:  "COMPLETING",
							Type:               "audit-user-task",
							State:              "CREATED",
							Retries:            3,
							ProcessInstanceKey: "2251799813685249",
							ElementInstanceKey: "2251799813685250",
						}},
					}},
				}},
				Count:    1,
				Warnings: []string{"sample warning"},
			}, nil
		},
	}

	got, err := New(api, slog.Default()).AnalyseSlowProcessInstances(context.Background(), SlowProcessAnalysisRequest{
		CommandName:   "ops analyse slow-process-instances",
		SelectionMode: SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249"},
		ProcessDefinitionSelector: SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID:        "OrderProcess",
			ProcessDefinitionKey: "2251799813687001",
		},
		ProcessInstanceFilters: SlowProcessAnalysisProcessInstanceSearchFilters{
			State:           process.StateCompleted,
			StartDateAfter:  "2026-07-18",
			StartDateBefore: "2026-07-19",
			EndDateAfter:    "2026-07-18T10:00:00Z",
			EndDateBefore:   "2026-07-18T11:00:00Z",
			NoIncidentsOnly: true,
		},
		DetailFilters: SlowProcessAnalysisDetailFilters{
			ElementID:     "ReserveStock",
			Type:          "SERVICE_TASK",
			ElementState:  "COMPLETED",
			DurationAfter: durationAfter,
		},
		RootDurationLonger: rootDurationLonger,
		BatchSize:          250,
		Limit:              10,
		CapturedNow:        captured,
		OutputMode:         "json",
		WithListeners:      true,
		Progress: func(event ProgressEvent) {
			progressEvents = append(progressEvents, event)
		},
	}, foptions.WithVerbose())

	require.NoError(t, err)
	require.Len(t, progressEvents, 1)
	require.Equal(t, ProgressEventKindPreflight, progressEvents[0].Kind)
	require.Equal(t, TotalCertaintyLowerBound, progressEvents[0].Preflight.TotalKind)
	require.Equal(t, captured, got.CapturedAt)
	require.Equal(t, 1, got.Count)
	require.Equal(t, []string{"sample warning"}, got.Warnings)
	require.Equal(t, DiscoveryScopeStatus{
		Complete:         false,
		Limited:          true,
		Limit:            10,
		BatchSize:        250,
		Pages:            2,
		CandidatesSeen:   12,
		CandidatesFrozen: 10,
	}, got.DiscoveredScopeStatus)
	require.Equal(t, TotalCertaintyLowerBound, got.PreflightScope.TotalKind)
	require.Equal(t, PageCountKindEstimated, got.PreflightScope.PageCountKind)
	require.Equal(t, "loading runtime elements", got.FrozenScopeProgress.Phase)
	require.Equal(t, &remaining, got.FrozenScopeProgress.ETA)
	require.Equal(t, process.StateCompleted, got.Items[0].State)
	require.Equal(t, SlowProcessAnalysisTimelineEntryKindElement, got.Items[0].Timeline[0].Kind)
	require.Equal(t, 1, got.Items[0].Timeline[0].ProcessDurationShare)
	require.True(t, got.Request.WithListeners)
	require.NotNil(t, got.Items[0].Timeline[0].Listeners)
	require.Equal(t, []RuntimeListenerJob{{
		JobKey:             "2251799813689101",
		Kind:               d.JobKindTaskListener,
		ListenerEventType:  "COMPLETING",
		Type:               "audit-user-task",
		State:              "CREATED",
		Retries:            3,
		ProcessInstanceKey: "2251799813685249",
		ElementInstanceKey: "2251799813685250",
	}}, *got.Items[0].Timeline[0].Listeners)
}

// TestClientAnalyseSlowProcessInstancesCopiesKeysAndMapsErrors verifies public slices and domain errors stay boundary-safe.
func TestClientAnalyseSlowProcessInstancesCopiesKeysAndMapsErrors(t *testing.T) {
	t.Parallel()

	inputKeys := typex.Keys{"2251799813685249"}
	api := stubOpsService{
		slowProcessAnalysis: func(_ context.Context, request d.SlowProcessAnalysisRequest, _ ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
			inputKeys[0] = "2251799813685250"
			require.Equal(t, typex.Keys{"2251799813685249"}, request.InputKeys)
			request.InputKeys[0] = "2251799813685251"
			return d.SlowProcessAnalysisResult{
				Request: request,
				Items: []d.SlowProcessAnalysisProcessInstance{{
					Key:               "2251799813685249",
					DurationAvailable: true,
					Timeline: []d.SlowProcessAnalysisTimelineEntry{{
						Kind: d.SlowProcessAnalysisTimelineEntryKindElement,
					}},
				}},
				Count: 1,
			}, fmt.Errorf("%w: blocked", d.ErrForbidden)
		},
	}

	got, err := New(api, slog.Default()).AnalyseSlowProcessInstances(context.Background(), SlowProcessAnalysisRequest{
		SelectionMode: SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     inputKeys,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked")
	require.Equal(t, typex.Keys{"2251799813685251"}, got.Request.InputKeys)
	require.Equal(t, "2251799813685249", got.Items[0].Key)
	require.Equal(t, SlowProcessAnalysisTimelineEntryKindElement, got.Items[0].Timeline[0].Kind)
}

func TestClientExecuteSmokeTestMapsDeploymentResult(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		smokeTest: func(_ context.Context, request d.SmokeTestRequest, _ ...services.CallOption) (d.SmokeTestResult, error) {
			return d.SmokeTestResult{
				Request: request,
				Deployment: d.SmokeTestDeploymentResult{
					Status:                   d.OpsWorkflowStepStatusConfirmed,
					FixtureFile:              "embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn",
					BpmnProcessID:            "C88_MultipleSubProcessesParent",
					ProcessDefinitionKey:     "pd-88",
					ProcessDefinitionVersion: 4,
					TenantID:                 "tenant-a",
				},
				Outcome: d.SmokeTestOutcomePassedCleanupSkipped,
			}, nil
		},
	}

	got, err := New(api, slog.Default()).ExecuteSmokeTest(context.Background(), SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
	})

	require.NoError(t, err)
	require.Equal(t, WorkflowStepStatusConfirmed, got.Deployment.Status)
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn", got.Deployment.FixtureFile)
	require.Equal(t, "C88_MultipleSubProcessesParent", got.Deployment.BpmnProcessID)
	require.Equal(t, "pd-88", got.Deployment.ProcessDefinitionKey)
	require.Equal(t, int32(4), got.Deployment.ProcessDefinitionVersion)
	require.Equal(t, "tenant-a", got.Deployment.TenantID)
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
				BatchSize:     25,
				Limit:         5,
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
				DiscoveredScopeStatus: d.DiscoveryScopeStatus{Limited: true, Limit: 5, BatchSize: 25, Pages: 1, CandidatesSeen: 6, CandidatesFrozen: 5},
				StartedAt:             started,
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
					DiscoveryScopeStatus:           d.DiscoveryScopeStatus{Limited: true, Limit: 5, BatchSize: 25, Pages: 1, CandidatesSeen: 6, CandidatesFrozen: 5},
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
		BatchSize:     25,
		Limit:         5,
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
		DiscoveredScopeStatus: DiscoveryScopeStatus{Limited: true, Limit: 5, BatchSize: 25, Pages: 1, CandidatesSeen: 6, CandidatesFrozen: 5},
		StartedAt:             started,
	}, foptions.WithVerbose(), foptions.WithNoWait(), foptions.WithForce(), foptions.WithFailFast())

	require.NoError(t, err)
	require.Equal(t, AllProcessDefinitionsPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, []string{"pd-a"}, []string(got.Discovery.CandidateProcessDefinitionKeys))
	require.True(t, got.Discovery.Limited)
	require.EqualValues(t, 25, got.Discovery.BatchSize)
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

// TestClientRepairIncidentsMapsServiceBoundary verifies repair requests remain thin facade conversions.
func TestClientRepairIncidentsMapsServiceBoundary(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 17, 14, 0, 0, 0, time.UTC)
	retries := int32(1)
	api := stubOpsService{
		repairIncidents: func(_ context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
			require.Equal(t, d.OpsRepairRequest{
				CommandName:         "ops repair incident",
				Target:              d.OpsRepairTargetIncident,
				DiscoveryMode:       d.OpsRepairDiscoveryModeKeyed,
				InputKeys:           typex.Keys{"inc-1", "inc-2"},
				IncidentSelection:   d.IncidentFilter{State: "ACTIVE", ErrorType: "JOB_NO_RETRIES"},
				BatchSize:           50,
				Limit:               2,
				Workers:             3,
				FailFast:            true,
				NoWorkerLimit:       true,
				DryRun:              true,
				AutoConfirm:         true,
				Automation:          true,
				NoWait:              true,
				OutputMode:          "json",
				Variables:           map[string]any{"approved": true},
				VariablesFile:       "vars.json",
				RequestedRetries:    &retries,
				RequestedJobTimeout: 5 * time.Minute,
				ReportFile:          "repair.json",
				ReportFormat:        "json",
				StartedAt:           started,
			}, request)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.Verbose)
			require.True(t, cfg.NoWait)
			require.True(t, cfg.FailFast)
			return d.OpsRepairResult{
				Request: request,
				FrozenSet: d.OpsRepairFrozenSet{
					Status:            d.OpsWorkflowStepStatusPlanned,
					Target:            d.OpsRepairTargetIncident,
					DiscoveryMode:     d.OpsRepairDiscoveryModeKeyed,
					InputKeys:         typex.Keys{"inc-1", "inc-2"},
					IncidentKeys:      typex.Keys{"inc-1", "inc-2"},
					JobKeys:           typex.Keys{"job-1"},
					VariableScopes:    typex.Keys{"pi-1"},
					IncidentFilters:   request.IncidentSelection,
					OriginalIncidents: []d.ProcessInstanceIncidentDetail{{IncidentKey: "inc-1", ProcessInstanceKey: "pi-1", JobKey: "job-1"}},
				},
				Plan: []d.OpsRepairPlanItem{{
					IncidentKey:            "inc-1",
					ProcessInstanceKey:     "pi-1",
					JobKey:                 "job-1",
					RequestedRetries:       &retries,
					RetryUpdateStatus:      d.OpsWorkflowStepStatusPlanned,
					TimeoutUpdateStatus:    d.OpsWorkflowStepStatusNotApplicable,
					ResolutionStatus:       d.OpsWorkflowStepStatusPlanned,
					ConfirmationStatus:     d.OpsWorkflowStepStatusSkipped,
					RequestedVariableNames: []string{"approved"},
				}},
				VariableUpdates: []d.OpsRepairVariableScopeUpdate{{
					ScopeKey:              "pi-1",
					VariableNames:         []string{"approved"},
					Payload:               map[string]any{"approved": true},
					DependentIncidentKeys: typex.Keys{"inc-1"},
					Status:                d.OpsWorkflowStepStatusPlanned,
				}},
				JobApplicability: []d.OpsRepairJobApplicability{{
					IncidentKey:      "inc-1",
					JobKey:           "job-1",
					RetryStatus:      d.OpsWorkflowStepStatusPlanned,
					TimeoutStatus:    d.OpsWorkflowStepStatusNotApplicable,
					RequestedRetries: &retries,
					Reason:           "no timeout requested",
				}},
				Remaining: d.OpsRepairRemainingIncidentSummary{Status: d.OpsWorkflowStepStatusSkipped},
				Report: d.OpsRepairAuditReport{
					SchemaVersion:    "ops.repair.v1",
					CommandName:      "ops repair incident",
					StartedAt:        started,
					DryRun:           true,
					Request:          request,
					FrozenSet:        d.OpsRepairFrozenSet{Status: d.OpsWorkflowStepStatusPlanned, Target: d.OpsRepairTargetIncident, IncidentKeys: typex.Keys{"inc-1"}},
					JobApplicability: []d.OpsRepairJobApplicability{{IncidentKey: "inc-1", TimeoutStatus: d.OpsWorkflowStepStatusNotApplicable}},
					Outcome:          d.OpsRepairOutcomePlanned,
				},
				Outcome: d.OpsRepairOutcomePlanned,
				Notices: []d.OpsRepairWorkflowNotice{{Code: "job_timeout_not_requested", Severity: "info", Details: map[string]string{"incidentKey": "inc-1"}}},
			}, nil
		},
	}

	got, err := New(api, slog.Default()).RepairIncidents(context.Background(), RepairRequest{
		CommandName:         "ops repair incident",
		Target:              RepairTargetIncident,
		DiscoveryMode:       RepairDiscoveryModeKeyed,
		InputKeys:           typex.Keys{"inc-1", "inc-2"},
		IncidentSelection:   incident.Filter{State: "ACTIVE", ErrorType: "JOB_NO_RETRIES"},
		BatchSize:           50,
		Limit:               2,
		Workers:             3,
		FailFast:            true,
		NoWorkerLimit:       true,
		DryRun:              true,
		AutoConfirm:         true,
		Automation:          true,
		NoWait:              true,
		OutputMode:          "json",
		Variables:           map[string]any{"approved": true},
		VariablesFile:       "vars.json",
		RequestedRetries:    &retries,
		RequestedJobTimeout: 5 * time.Minute,
		ReportFile:          "repair.json",
		ReportFormat:        "json",
		StartedAt:           started,
	}, foptions.WithVerbose(), foptions.WithNoWait(), foptions.WithFailFast())

	require.NoError(t, err)
	require.Equal(t, RepairOutcomePlanned, got.Outcome)
	require.Equal(t, RepairTargetIncident, got.FrozenSet.Target)
	require.Equal(t, []string{"inc-1", "inc-2"}, []string(got.FrozenSet.IncidentKeys))
	require.Equal(t, "job-1", got.FrozenSet.OriginalIncidents[0].JobKey)
	require.Equal(t, WorkflowStepStatusNotApplicable, got.Plan[0].TimeoutUpdateStatus)
	require.Equal(t, []string{"approved"}, got.VariableUpdates[0].VariableNames)
	require.Equal(t, WorkflowStepStatusNotApplicable, got.JobApplicability[0].TimeoutStatus)
	require.Equal(t, "job_timeout_not_requested", got.Notices[0].Code)
	require.Equal(t, RepairOutcomePlanned, got.Report.Outcome)
	require.Equal(t, []string{"inc-1"}, []string(got.Report.FrozenSet.IncidentKeys))
}

// TestClientRepairIncidentsMapsServiceErrors verifies explicit repair returns the partial result with facade-normalized errors.
func TestClientRepairIncidentsMapsServiceErrors(t *testing.T) {
	t.Parallel()

	api := stubOpsService{
		repairIncidents: func(_ context.Context, request d.OpsRepairRequest, _ ...services.CallOption) (d.OpsRepairResult, error) {
			return d.OpsRepairResult{
				Request: request,
				FrozenSet: d.OpsRepairFrozenSet{
					Status:       d.OpsWorkflowStepStatusFailed,
					Target:       d.OpsRepairTargetIncident,
					IncidentKeys: typex.Keys{"2251799813685249"},
					Errors:       []string{"boom"},
				},
				Outcome: d.OpsRepairOutcomeFailed,
				Errors:  []string{"boom"},
			}, fmt.Errorf("%w: boom", d.ErrValidation)
		},
	}

	got, err := New(api, slog.Default()).RepairIncidents(context.Background(), RepairRequest{
		CommandName: "ops repair incident",
		Target:      RepairTargetIncident,
		InputKeys:   typex.Keys{"2251799813685249"},
		RequestedRetries: func() *int32 {
			v := int32(1)
			return &v
		}(),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	require.Equal(t, RepairOutcomeFailed, got.Outcome)
	require.Equal(t, WorkflowStepStatusFailed, got.FrozenSet.Status)
	require.Equal(t, []string{"2251799813685249"}, []string(got.FrozenSet.IncidentKeys))
}

type stubOpsService struct {
	smokeTest                  func(context.Context, d.SmokeTestRequest, ...services.CallOption) (d.SmokeTestResult, error)
	analyseAPILatency          func(context.Context, d.APILatencyRequest, ...services.CallOption) (d.APILatencyResult, error)
	executeAPILatencyTest      func(context.Context, d.APILatencyRequest, ...services.CallOption) (d.APILatencyResult, error)
	purge                      func(context.Context, d.OrphanPurgeRequest, ...services.CallOption) (d.OrphanPurgeResult, error)
	retention                  func(context.Context, d.RetentionPolicyRequest, ...services.CallOption) (d.RetentionPolicyResult, error)
	incidentPurge              func(context.Context, d.IncidentPurgeRequest, ...services.CallOption) (d.IncidentPurgeResult, error)
	allProcessDefinitionsPurge func(context.Context, d.AllProcessDefinitionsPurgeRequest, ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error)
	repairIncidents            func(context.Context, d.OpsRepairRequest, ...services.CallOption) (d.OpsRepairResult, error)
	repairProcessInstances     func(context.Context, d.OpsRepairRequest, ...services.CallOption) (d.OpsRepairResult, error)
	slowProcessAnalysis        func(context.Context, d.SlowProcessAnalysisRequest, ...services.CallOption) (d.SlowProcessAnalysisResult, error)
}

func (s stubOpsService) ExecuteSmokeTest(ctx context.Context, request d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
	if s.smokeTest == nil {
		panic("unexpected call")
	}
	return s.smokeTest(ctx, request, opts...)
}

func (s stubOpsService) AnalyseAPILatency(ctx context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
	if s.analyseAPILatency == nil {
		panic("unexpected call")
	}
	return s.analyseAPILatency(ctx, request, opts...)
}

func (s stubOpsService) ExecuteAPILatencyTest(ctx context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
	if s.executeAPILatencyTest == nil {
		panic("unexpected call")
	}
	return s.executeAPILatencyTest(ctx, request, opts...)
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

func (s stubOpsService) RepairIncidents(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	if s.repairIncidents == nil {
		panic("unexpected call")
	}
	return s.repairIncidents(ctx, request, opts...)
}

func (s stubOpsService) RepairProcessInstances(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	if s.repairProcessInstances == nil {
		panic("unexpected call")
	}
	return s.repairProcessInstances(ctx, request, opts...)
}

func (s stubOpsService) AnalyseSlowProcessInstances(ctx context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
	if s.slowProcessAnalysis == nil {
		panic("unexpected call")
	}
	return s.slowProcessAnalysis(ctx, request, opts...)
}

var _ opsvc.API = stubOpsService{}
