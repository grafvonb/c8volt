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
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestExecuteSmokeTestValidatesRequestShape protects local validation before any workflow step can mutate state.
func TestExecuteSmokeTestValidatesRequestShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request d.SmokeTestRequest
		want    string
	}{
		{
			name:    "zero count",
			request: d.SmokeTestRequest{Count: 0},
			want:    "count must be a positive integer",
		},
		{
			name:    "negative count",
			request: d.SmokeTestRequest{Count: -3},
			want:    "count must be a positive integer",
		},
		{
			name: "unsupported report format",
			request: d.SmokeTestRequest{
				Count:        1,
				ReportFormat: "xml",
				ReportFile:   "smoke-test.xml",
			},
			want: "report-format must be markdown or json",
		},
		{
			name: "report format without file",
			request: d.SmokeTestRequest{
				Count:        1,
				ReportFormat: "json",
			},
			want: "report-format requires report-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(nil, nil).ExecuteSmokeTest(context.Background(), tt.request)

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.SmokeTestOutcomeFailed, got.Outcome)
			require.Equal(t, d.SmokeTestOutcomeFailed, got.Report.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deployment.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Run.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Walk.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessInstanceCleanup.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionEligibility.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Plan.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
			require.True(t, strings.Contains(got.Report.Errors[0], tt.want))
		})
	}
}

// TestExecuteSmokeTestRecordsFoundationalControls verifies the boundary records reusable call controls in result metadata.
func TestExecuteSmokeTestRecordsFoundationalControls(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 17, 10, 30, 0, 0, time.UTC)
	request := d.SmokeTestRequest{
		CommandName:  "ops execute smoke-test",
		Count:        1,
		DryRun:       true,
		NoCleanup:    true,
		AutoConfirm:  true,
		Automation:   true,
		OutputMode:   "json",
		ReportFile:   "smoke-test.md",
		ReportFormat: "markdown",
		StartedAt:    started,
	}

	got, err := New(nil, nil).ExecuteSmokeTest(
		context.Background(),
		request,
		services.WithNoWait(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Deployment.Status)
	require.Equal(t, started, got.Request.StartedAt)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.True(t, got.Report.NoWait)
	require.True(t, got.Report.NoCleanup)
	require.False(t, got.Report.CleanupRequested)
	require.Equal(t, got.Plan, got.Report.Plan)
}

func TestExecuteSmokeTestDryRunPlansReadOnlyWorkflow(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{
		topology: d.Topology{GatewayVersion: "8.8.2"},
	}
	request := d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       3,
		DryRun:      true,
		NoCleanup:   true,
		ReportFile:  "smoke-test.md",
	}

	got, err := NewWithWorkflowDependencies(cluster, nil, nil, nil, nil, toolx.V88).ExecuteSmokeTest(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, 1, cluster.topologyCalls)
	require.Equal(t, d.SmokeTestOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan.Status)
	require.Equal(t, "8.8", got.Plan.CamundaVersion)
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn", got.Fixture.File)
	require.Equal(t, "C88_MultipleSubProcessesParentProcess", got.Fixture.BpmnProcessID)
	require.True(t, got.Fixture.Available)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Deployment.Status)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Run.Status)
	require.Equal(t, 3, got.Run.RequestedCount)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Walk.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessInstanceCleanup.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionEligibility.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
	require.False(t, got.Plan.CleanupRequested)
	require.Len(t, got.Plan.PlannedSteps, 7)
	require.Equal(t, "connectivity", got.Plan.PlannedSteps[0].Name)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Plan.PlannedSteps[0].Status)
	require.Equal(t, "cleanup", got.Plan.PlannedSteps[5].Name)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Plan.PlannedSteps[5].Status)
	require.Equal(t, "report", got.Plan.PlannedSteps[6].Name)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Plan.PlannedSteps[6].Status)
	require.Equal(t, got.Plan, got.Report.Plan)
	require.Equal(t, got.Fixture, got.Report.Fixture)
	require.Equal(t, got.Deployment, got.Report.Deployment)
}

func TestExecuteSmokeTestSelectsVersionMatchedFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version toolx.CamundaVersion
		file    string
		process string
	}{
		{
			version: toolx.V87,
			file:    "embedded/processdefinitions/C87_MultipleSubProcessesParentProcess.bpmn",
			process: "C87_MultipleSubProcessesParentProcess",
		},
		{
			version: toolx.V88,
			file:    "embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn",
			process: "C88_MultipleSubProcessesParentProcess",
		},
		{
			version: toolx.V89,
			file:    "embedded/processdefinitions/C89_MultipleSubProcessesParentProcess.bpmn",
			process: "C89_MultipleSubProcessesParentProcess",
		},
	}

	for _, tt := range tests {
		t.Run(tt.version.String(), func(t *testing.T) {
			t.Parallel()

			got, err := smokeTestFixtureForVersion(tt.version)

			require.NoError(t, err)
			require.Equal(t, tt.version.String(), got.CamundaVersion)
			require.Equal(t, tt.file, got.File)
			require.Equal(t, tt.process, got.BpmnProcessID)
			require.True(t, got.Available)
		})
	}
}

func TestExecuteSmokeTestMissingFixtureFailsBeforeMutation(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{}
	got, err := NewWithWorkflowDependencies(nil, nil, nil, nil, resource, toolx.CamundaVersion("8.10")).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported smoke-test fixture version")
	require.Zero(t, resource.deployCalls)
	require.Equal(t, d.SmokeTestOutcomeFailed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deployment.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Run.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Walk.Status)
}

func TestExecuteSmokeTestDeploysSelectedFixtureThroughResourceAPI(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{
		topology: d.Topology{GatewayVersion: "8.8.2"},
	}
	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, units []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			require.Len(t, units, 1)
			require.Equal(t, "processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn", units[0].Name)
			require.Equal(t, "application/xml", units[0].ContentType)
			require.Contains(t, string(units[0].Data), "C88_MultipleSubProcessesParentProcess")
			return d.Deployment{
				Key:      "deployment-1",
				TenantId: "tenant-a",
				Units: []d.DeploymentUnit{{
					ProcessDefinition: d.ProcessDefinitionDeployment{
						ProcessDefinitionId:      "C88_MultipleSubProcessesParentProcess",
						ProcessDefinitionKey:     "pd-88",
						ProcessDefinitionVersion: 4,
						ResourceName:             "processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn",
						TenantId:                 "tenant-a",
					},
				}},
			}, nil
		},
	}

	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.Equal(t, "pd-88", data.ProcessDefinitionSpecificId)
			require.Empty(t, data.BpmnProcessId)
			return d.ProcessInstanceCreation{
				Key:                  "pi-1",
				BpmnProcessId:        "C88_MultipleSubProcessesParentProcess",
				ProcessDefinitionKey: "pd-88",
				TenantId:             "tenant-a",
			}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "pi-1", startKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeFamily,
				StartKey: "pi-1",
				RootKey:  "pi-1",
				Keys:     []string{"pi-1"},
				Outcome:  pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(cluster, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		NoCleanup:   true,
	})

	require.NoError(t, err)
	require.Equal(t, 1, cluster.topologyCalls)
	require.Equal(t, 1, resource.deployCalls)
	require.Equal(t, d.SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Deployment.Status)
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn", got.Deployment.FixtureFile)
	require.Equal(t, "C88_MultipleSubProcessesParentProcess", got.Deployment.BpmnProcessID)
	require.Equal(t, "pd-88", got.Deployment.ProcessDefinitionKey)
	require.Equal(t, int32(4), got.Deployment.ProcessDefinitionVersion)
	require.Equal(t, "tenant-a", got.Deployment.TenantID)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Run.Status)
	require.Equal(t, 1, got.Run.RequestedCount)
	require.Equal(t, 1, got.Run.CreatedCount)
	require.Equal(t, typexKeys("pi-1"), got.Run.ProcessInstanceKeys)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Walk.Status)
	require.Len(t, got.Walk.Items, 1)
	require.Equal(t, "pi-1", got.Walk.Items[0].ProcessInstanceKey)
	require.Equal(t, d.TraversalOutcomeComplete, got.Walk.Items[0].Summary.Outcome)
	require.Equal(t, got.Deployment, got.Report.Deployment)
	require.Equal(t, got.Run, got.Report.Run)
	require.Equal(t, got.Walk, got.Report.Walk)
}

func TestExecuteSmokeTestStartsCreatedInstancesByDeployedProcessDefinitionKey(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{Units: []d.DeploymentUnit{{
				ProcessDefinition: d.ProcessDefinitionDeployment{
					ProcessDefinitionId:  "C88_MultipleSubProcessesParentProcess",
					ProcessDefinitionKey: "pd-exact",
					TenantId:             "tenant-a",
				},
			}}}, nil
		},
	}
	var created []d.ProcessInstanceData
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
			created = append(created, data)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.FailFast)
			require.True(t, cfg.NoWorkerLimit)
			return d.ProcessInstanceCreation{
				Key:                  "pi-" + string(rune('0'+len(created))),
				BpmnProcessId:        "C88_MultipleSubProcessesParentProcess",
				ProcessDefinitionKey: data.ProcessDefinitionSpecificId,
				TenantId:             data.TenantId,
			}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				Mode:     pitraversal.ModeFamily,
				StartKey: startKey,
				RootKey:  startKey,
				Keys:     []string{startKey, startKey + "-child"},
				Outcome:  pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(
		context.Background(),
		d.SmokeTestRequest{CommandName: "ops execute smoke-test", Count: 2, Workers: 1, FailFast: true, NoWorkerLimit: true, NoCleanup: true},
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Len(t, created, 2)
	for _, data := range created {
		require.Equal(t, "pd-exact", data.ProcessDefinitionSpecificId)
		require.Empty(t, data.BpmnProcessId)
		require.Equal(t, "tenant-a", data.TenantId)
	}
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Run.Status)
	require.Equal(t, 2, got.Run.RequestedCount)
	require.Equal(t, 2, got.Run.CreatedCount)
	require.Equal(t, typexKeys("pi-1", "pi-2"), got.Run.ProcessInstanceKeys)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Walk.Status)
	require.Len(t, got.Walk.Items, 2)
	require.Equal(t, typexKeys("pi-1", "pi-1-child"), got.Walk.Items[0].Summary.FamilyKeys)
	require.Equal(t, d.SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
}

func TestExecuteSmokeTestFallsBackToBPMNProcessIDWhenDeploymentKeyMissing(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.Empty(t, data.ProcessDefinitionSpecificId)
			require.Equal(t, "C88_MultipleSubProcessesParentProcess", data.BpmnProcessId)
			return d.ProcessInstanceCreation{Key: "pi-fallback", BpmnProcessId: data.BpmnProcessId}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Outcome: pitraversal.OutcomeComplete}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		NoCleanup:   true,
	})

	require.NoError(t, err)
	require.Equal(t, typexKeys("pi-fallback"), got.Run.ProcessInstanceKeys)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Walk.Status)
}

func TestExecuteSmokeTestTraversalSummariesCapturePartialFamilyWalks(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{Units: []d.DeploymentUnit{{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionKey: "pd-88"}}}}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{Key: "child", ProcessDefinitionKey: data.ProcessDefinitionSpecificId}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				Mode:             pitraversal.ModeFamily,
				StartKey:         startKey,
				RootKey:          "root",
				Keys:             []string{"root", "child"},
				MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing-parent", StartKey: startKey}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          pitraversal.OutcomePartial,
			}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		NoCleanup:   true,
	})

	require.NoError(t, err)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Walk.Status)
	require.Len(t, got.Walk.Items, 1)
	summary := got.Walk.Items[0].Summary
	require.Equal(t, "child", summary.ProcessInstanceKey)
	require.Equal(t, "root", summary.RootProcessInstanceKey)
	require.Equal(t, typexKeys("root", "child"), summary.FamilyKeys)
	require.Equal(t, []d.MissingAncestor{{Key: "missing-parent", StartKey: "child"}}, summary.MissingAncestors)
	require.Equal(t, d.TraversalOutcomePartial, summary.Outcome)
}

func TestExecuteSmokeTestCleansUpCreatedResources(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{Units: []d.DeploymentUnit{{ProcessDefinition: d.ProcessDefinitionDeployment{
				ProcessDefinitionId:  "C88_MultipleSubProcessesParentProcess",
				ProcessDefinitionKey: "pd-88",
			}}}}, nil
		},
		delete: func(_ context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
			require.Equal(t, "pd-88", key)
			require.False(t, services.ApplyCallOptions(opts).NoWait)
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 200, Status: "200 OK", DeleteHistory: true, BatchOperationKey: "batch-1", BatchState: "COMPLETED"}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{Key: "pi-1", ProcessDefinitionKey: data.ProcessDefinitionSpecificId}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Outcome: pitraversal.OutcomeComplete}, nil
		},
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeAncestry, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Chain: map[string]d.ProcessInstance{
				startKey: {Key: startKey, State: d.StateActive, ProcessDefinitionKey: "pd-88"},
			}, Outcome: pitraversal.OutcomeComplete}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeDescendants, StartKey: rootKey, RootKey: rootKey, Keys: []string{rootKey}, Chain: map[string]d.ProcessInstance{
				rootKey: {Key: rootKey, State: d.StateActive, ProcessDefinitionKey: "pd-88"},
			}, Outcome: pitraversal.OutcomeComplete}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "pi-1", key)
			require.True(t, services.ApplyCallOptions(opts).Force)
			return d.DeleteResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, nil
		},
		search: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, "pd-88", filter.ProcessDefinitionKey)
			require.Equal(t, int32(1000), size)
			return nil, nil
		},
	}
	pdAPI := stubProcessDefinitionAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
			require.Equal(t, "pd-88", key)
			cfg := services.ApplyCallOptions(opts)
			if !cfg.WithStat {
				return d.ProcessDefinition{}, d.ErrNotFound
			}
			require.True(t, cfg.WithStat)
			return d.ProcessDefinition{Key: key, BpmnProcessId: "C88_MultipleSubProcessesParentProcess", Statistics: &d.ProcessDefinitionStatistics{}}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
	})

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePassed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Cleanup.ProcessInstanceCleanup.Status)
	require.Equal(t, typexKeys("pi-1"), got.Cleanup.ProcessInstanceCleanup.SubmittedKeys)
	require.True(t, got.Cleanup.ProcessInstanceCleanup.Submitted)
	require.True(t, got.Cleanup.ProcessInstanceCleanup.Confirmed)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Cleanup.ProcessDefinitionEligibility.Status)
	require.True(t, got.Cleanup.ProcessDefinitionEligibility.Eligible)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Cleanup.ProcessDefinitionCleanup.Status)
	require.Equal(t, "pd-88", got.Cleanup.ProcessDefinitionCleanup.SubmittedProcessDefinitionKey)
	require.True(t, got.Cleanup.ProcessDefinitionCleanup.Submitted)
	require.True(t, got.Cleanup.ProcessDefinitionCleanup.Confirmed)
	require.Equal(t, 1, resource.deleteCalls)
	require.Equal(t, got.Cleanup, got.Report.Cleanup)
}

func TestExecuteSmokeTestBlocksProcessDefinitionCleanupForUnrelatedInstances(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{Units: []d.DeploymentUnit{{ProcessDefinition: d.ProcessDefinitionDeployment{
				ProcessDefinitionId:  "C88_MultipleSubProcessesParentProcess",
				ProcessDefinitionKey: "pd-88",
			}}}}, nil
		},
		delete: func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{}, errors.New("unexpected process-definition delete")
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{Key: "pi-1", ProcessDefinitionKey: data.ProcessDefinitionSpecificId}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Outcome: pitraversal.OutcomeComplete}, nil
		},
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeAncestry, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Chain: map[string]d.ProcessInstance{
				startKey: {Key: startKey, State: d.StateCompleted, ProcessDefinitionKey: "pd-88"},
			}, Outcome: pitraversal.OutcomeComplete}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeDescendants, StartKey: rootKey, RootKey: rootKey, Keys: []string{rootKey}, Chain: map[string]d.ProcessInstance{
				rootKey: {Key: rootKey, State: d.StateCompleted, ProcessDefinitionKey: "pd-88"},
			}, Outcome: pitraversal.OutcomeComplete}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "pi-1", key)
			return d.DeleteResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, nil
		},
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, "pd-88", filter.ProcessDefinitionKey)
			return []d.ProcessInstance{{Key: "unrelated-1", ProcessDefinitionKey: "pd-88"}}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, stubProcessDefinitionAPI{}, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrPrecondition), "got %v", err)
	require.Contains(t, err.Error(), "process-definition cleanup blocked")
	require.Equal(t, d.SmokeTestOutcomePartiallyFailed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Cleanup.ProcessInstanceCleanup.Status)
	require.Equal(t, d.OpsWorkflowStepStatusBlocked, got.Cleanup.ProcessDefinitionEligibility.Status)
	require.Equal(t, []string{"unrelated-1"}, got.Cleanup.ProcessDefinitionEligibility.Blockers)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
	require.Zero(t, resource.deleteCalls)
}

func TestExecuteSmokeTestNoCleanupRetainsCreatedResources(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{Units: []d.DeploymentUnit{{ProcessDefinition: d.ProcessDefinitionDeployment{
				ProcessDefinitionId:  "C88_MultipleSubProcessesParentProcess",
				ProcessDefinitionKey: "pd-retained",
				TenantId:             "tenant-a",
			}}}}, nil
		},
		delete: func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{}, errors.New("unexpected process-definition delete")
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{Key: "pi-retained", ProcessDefinitionKey: data.ProcessDefinitionSpecificId, TenantId: data.TenantId}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Outcome: pitraversal.OutcomeComplete}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		NoCleanup:   true,
	})

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
	require.True(t, got.Cleanup.NoCleanup)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessInstanceCleanup.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionEligibility.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
	require.Equal(t, typexKeys("pi-retained"), got.Cleanup.RetainedProcessInstanceKeys)
	require.Equal(t, "pd-retained", got.Cleanup.RetainedProcessDefinitionKey)
	require.Equal(t, "C88_MultipleSubProcessesParentProcess", got.Cleanup.RetainedBpmnProcessID)
	require.Equal(t, "tenant-a", got.Cleanup.RetainedTenantID)
	require.Equal(t, got.Cleanup, got.Report.Cleanup)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Plan.PlannedSteps[5].Status)
	require.Zero(t, resource.deleteCalls)
}

func TestExecuteSmokeTestDryRunConnectivityFailureDoesNotPlanMutation(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{err: errors.New("topology unavailable")}

	got, err := NewWithWorkflowDependencies(cluster, nil, nil, nil, nil, toolx.V89).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		DryRun:      true,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "smoke-test connectivity validation")
	require.Equal(t, 1, cluster.topologyCalls)
	require.Equal(t, d.SmokeTestOutcomeFailed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deployment.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Run.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Walk.Status)
	require.Len(t, got.Plan.PlannedSteps, 7)
	require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Plan.PlannedSteps[0].Status)
	require.NotEmpty(t, got.Errors)
}

type stubSmokeTestClusterAPI struct {
	topology      d.Topology
	err           error
	topologyCalls int
}

func (s *stubSmokeTestClusterAPI) GetClusterTopology(context.Context, ...services.CallOption) (d.Topology, error) {
	s.topologyCalls++
	if s.err != nil {
		return d.Topology{}, s.err
	}
	return s.topology, nil
}

func (*stubSmokeTestClusterAPI) GetClusterLicense(context.Context, ...services.CallOption) (d.License, error) {
	return d.License{}, nil
}

type stubSmokeTestResourceAPI struct {
	deploy      func(context.Context, []d.DeploymentUnitData, ...services.CallOption) (d.Deployment, error)
	delete      func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error)
	deployCalls int
	deleteCalls int
}

func (s *stubSmokeTestResourceAPI) Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error) {
	s.deployCalls++
	if s.deploy == nil {
		return d.Deployment{}, errors.New("unexpected deploy call")
	}
	return s.deploy(ctx, units, opts...)
}

func (s *stubSmokeTestResourceAPI) Delete(ctx context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	s.deleteCalls++
	if s.delete == nil {
		return d.ResourceDeleteResponse{}, errors.New("unexpected delete call")
	}
	return s.delete(ctx, key, opts...)
}

func (*stubSmokeTestResourceAPI) Get(context.Context, string, ...services.CallOption) (d.Resource, error) {
	return d.Resource{}, errors.New("unexpected get call")
}
