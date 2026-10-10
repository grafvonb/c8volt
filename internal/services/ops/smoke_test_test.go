// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
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
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn", got.Fixture.File)
	require.Equal(t, "C88_MultipleSubProcessesParent", got.Fixture.BpmnProcessID)
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

// TestExecuteSmokeTestSelectsVersionMatchedFixtures verifies each supported compatibility line resolves to its own smoke fixture.
func TestExecuteSmokeTestSelectsVersionMatchedFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version toolx.CamundaVersion
		file    string
		process string
	}{
		{
			version: toolx.V88,
			file:    "embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn",
			process: "C88_MultipleSubProcessesParent",
		},
		{
			version: toolx.V89,
			file:    "embedded/processdefinitions/C89_MultipleSubProcessesParent.bpmn",
			process: "C89_MultipleSubProcessesParent",
		},
		{
			version: toolx.V810,
			file:    "embedded/processdefinitions/C810_MultipleSubProcessesParent.bpmn",
			process: "C810_MultipleSubProcessesParent",
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

// TestSmokeTestDeploymentUnitsUsesV810Closure verifies the native 8.10 parent fixture deploys with its C810 child definitions.
func TestSmokeTestDeploymentUnitsUsesV810Closure(t *testing.T) {
	t.Parallel()

	fixture, err := smokeTestFixtureForVersion(toolx.V810)
	require.NoError(t, err)

	units, err := smokeTestDeploymentUnits(fixture)

	require.NoError(t, err)
	require.Len(t, units, 3)
	require.Equal(t, "processdefinitions/C810_SimpleUserTask.bpmn", units[0].Name)
	require.Equal(t, "processdefinitions/C810_SimpleParent.bpmn", units[1].Name)
	require.Equal(t, "processdefinitions/C810_MultipleSubProcessesParent.bpmn", units[2].Name)
	for _, unit := range units {
		require.Equal(t, "application/xml", unit.ContentType)
		require.Contains(t, string(unit.Data), "C810_")
		require.NotContains(t, string(unit.Data), "C89_")
	}
}

// TestSmokeTestDeploymentResultCapturesEverySupportedFixtureDefinition verifies collection evidence and exact main selection for each compatibility family.
func TestSmokeTestDeploymentResultCapturesEverySupportedFixtureDefinition(t *testing.T) {
	t.Parallel()

	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			t.Parallel()
			fixture, err := smokeTestFixtureForVersion(version)
			require.NoError(t, err)
			deployment := completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a")
			deployment.Units = append(deployment.Units, deployment.Units[0])

			got := smokeTestDeploymentResult(fixture, deployment, d.OpsWorkflowStepStatusConfirmed)

			require.True(t, got.DefinitionScopeComplete)
			require.Empty(t, got.Errors)
			require.Len(t, got.ProcessDefinitions, 3)
			require.Equal(t, []string{"pd-main-child", "pd-main-middle", "pd-main"}, []string{
				got.ProcessDefinitions[0].ProcessDefinitionKey,
				got.ProcessDefinitions[1].ProcessDefinitionKey,
				got.ProcessDefinitions[2].ProcessDefinitionKey,
			})
			require.Equal(t, fixture.BpmnProcessID, got.BpmnProcessID)
			require.Equal(t, "pd-main", got.ProcessDefinitionKey)
		})
	}
}

// TestSmokeTestDeploymentResultRejectsConflictingAndMissingIdentityEvidence preserves evidence without authorizing unsafe keys.
func TestSmokeTestDeploymentResultRejectsConflictingAndMissingIdentityEvidence(t *testing.T) {
	t.Parallel()

	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a")
	deployment.Units = append(deployment.Units,
		d.DeploymentUnit{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: "C89_Other", ProcessDefinitionKey: "pd-main-child", TenantId: "tenant-b"}},
		d.DeploymentUnit{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: "C89_Unkeyed"}},
	)

	got := smokeTestDeploymentResult(fixture, deployment, d.OpsWorkflowStepStatusConfirmed)

	require.False(t, got.DefinitionScopeComplete)
	require.Len(t, got.ProcessDefinitions, 4)
	require.Contains(t, strings.Join(got.Errors, "\n"), "conflicting deployment identity evidence")
	require.Contains(t, strings.Join(got.Errors, "\n"), "missing a process-definition key")
	require.Equal(t, "pd-main", got.ProcessDefinitionKey)
}

// TestCleanupSmokeTestResourcesDeletesAllDefinitionsForEveryFixtureFamily proves each distinct returned key is deleted once in deployment order.
func TestCleanupSmokeTestResourcesDeletesAllDefinitionsForEveryFixtureFamily(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			fixture, err := smokeTestFixtureForVersion(version)
			require.NoError(t, err)
			deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
			var deleted []string
			resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
				deleted = append(deleted, key)
				return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "200 OK"}, nil
			}}
			piAPI := stubProcessInstanceAPI{search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
				return nil, nil
			}}
			pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
				if !services.ApplyCallOptions(opts).WithStat {
					return d.ProcessDefinition{}, d.ErrNotFound
				}
				return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
			}}
			service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, version).(*Service)

			got, err := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{
				Request:    d.SmokeTestRequest{Count: 1},
				Deployment: deployment,
			})

			require.NoError(t, err)
			require.Equal(t, []string{"pd-main-child", "pd-main-middle", "pd-main"}, deleted)
			require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusConfirmed, Total: 3, Confirmed: 3}, got.DefinitionSummary)
			for _, item := range got.DefinitionResults {
				require.Equal(t, d.SmokeTestDefinitionCleanupDispositionConfirmed, item.Disposition)
			}
		})
	}
}

// TestCleanupSmokeTestResourcesNoWaitRecordsSubmittedDefinitions verifies accepted requests are never labeled confirmed.
func TestCleanupSmokeTestResourcesNoWaitRecordsSubmittedDefinitions(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
		require.True(t, services.ApplyCallOptions(opts).NoWait)
		return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}, nil
	}}
	piAPI := stubProcessInstanceAPI{search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
		return nil, nil
	}}
	pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
		require.True(t, services.ApplyCallOptions(opts).WithStat)
		return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
	}}
	service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V89).(*Service)

	got, err := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{Request: d.SmokeTestRequest{Count: 1, NoWait: true}, Deployment: deployment}, services.WithNoWait())

	require.NoError(t, err)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusSubmitted, Total: 3, Submitted: 3}, got.DefinitionSummary)
	for _, item := range got.DefinitionResults {
		require.Equal(t, d.SmokeTestDefinitionCleanupDispositionSubmitted, item.Disposition)
		require.True(t, item.Cleanup.Submitted)
		require.False(t, item.Cleanup.Confirmed)
	}
}

// TestCleanupSmokeTestResourcesFailFastPreservesUnattemptedDefinitions verifies collection accounting after the first eligibility error.
func TestCleanupSmokeTestResourcesFailFastPreservesUnattemptedDefinitions(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	piAPI := stubProcessInstanceAPI{search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
		return nil, errors.New("eligibility unavailable")
	}}
	resource := &stubSmokeTestResourceAPI{delete: func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error) {
		return d.ResourceDeleteResponse{}, errors.New("unexpected delete")
	}}
	service := NewWithWorkflowDependencies(nil, piAPI, nil, stubProcessDefinitionAPI{}, resource, toolx.V89).(*Service)

	got, err := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{Request: d.SmokeTestRequest{Count: 1, FailFast: true}, Deployment: deployment}, services.WithFailFast())

	require.Error(t, err)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionFailed, got.DefinitionResults[0].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, got.DefinitionResults[1].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, got.DefinitionResults[2].Disposition)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Failed: 1, Unattempted: 2}, got.DefinitionSummary)
	require.Zero(t, resource.deleteCalls)
}

// TestCleanupSmokeTestResourcesCancellationPreservesUnattemptedDefinitions verifies cancellation does not erase known deployment resources.
func TestCleanupSmokeTestResourcesCancellationPreservesUnattemptedDefinitions(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewWithWorkflowDependencies(nil, stubProcessInstanceAPI{}, nil, stubProcessDefinitionAPI{}, &stubSmokeTestResourceAPI{}, toolx.V89).(*Service)

	got, err := service.cleanupSmokeTestResources(ctx, d.SmokeTestResult{Request: d.SmokeTestRequest{Count: 1}, Deployment: deployment})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Unattempted: 3}, got.DefinitionSummary)
	for _, item := range got.DefinitionResults {
		require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, item.Disposition)
		require.Contains(t, item.Reason, "context canceled")
	}
}

// TestCleanupSmokeTestResourcesFaultsPreserveSiblingEvidence verifies independent definition faults do not erase or stop safe sibling submissions.
func TestCleanupSmokeTestResourcesFaultsPreserveSiblingEvidence(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)

	tests := []struct {
		name        string
		eligibility func(string) error
		lookup      func(string) error
		delete      func(string) d.ResourceDeleteResponse
		wantReason  string
	}{
		{
			name: "eligibility error",
			eligibility: func(key string) error {
				if key == "pd-main-child" {
					return errors.New("eligibility unavailable")
				}
				return nil
			},
			wantReason: "eligibility unavailable",
		},
		{
			name: "rejected submission",
			delete: func(key string) d.ResourceDeleteResponse {
				if key == "pd-main-child" {
					return d.ResourceDeleteResponse{Ok: false, StatusCode: http.StatusConflict, Status: "409 Conflict"}
				}
				return d.ResourceDeleteResponse{Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}
			},
			wantReason: "cleanup rejected: 409 Conflict",
		},
		{
			name: "empty response",
			delete: func(key string) d.ResourceDeleteResponse {
				if key == "pd-main-child" {
					return d.ResourceDeleteResponse{}
				}
				return d.ResourceDeleteResponse{Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}
			},
			wantReason: "cleanup returned no accepted deletion response",
		},
		{
			name: "pre-delete not found",
			lookup: func(key string) error {
				if key == "pd-main-child" {
					return d.ErrNotFound
				}
				return nil
			},
			wantReason: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var deleted []string
			piAPI := stubProcessInstanceAPI{search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
				if tt.eligibility != nil {
					if err := tt.eligibility(filter.ProcessDefinitionKey); err != nil {
						return nil, err
					}
				}
				return nil, nil
			}}
			pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
				if tt.lookup != nil {
					if err := tt.lookup(key); err != nil {
						return d.ProcessDefinition{}, err
					}
				}
				return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
			}}
			resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
				deleted = append(deleted, key)
				if tt.delete != nil {
					return tt.delete(key), nil
				}
				return d.ResourceDeleteResponse{Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}, nil
			}}
			service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V89).(*Service)

			got, cleanupErr := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{
				Request: d.SmokeTestRequest{Count: 1, NoWait: true}, Deployment: deployment,
			}, services.WithNoWait())

			require.Error(t, cleanupErr)
			require.Equal(t, d.SmokeTestDefinitionCleanupDispositionFailed, got.DefinitionResults[0].Disposition)
			require.Contains(t, got.DefinitionResults[0].Reason, tt.wantReason)
			require.Equal(t, d.SmokeTestDefinitionCleanupDispositionSubmitted, got.DefinitionResults[1].Disposition)
			require.Equal(t, d.SmokeTestDefinitionCleanupDispositionSubmitted, got.DefinitionResults[2].Disposition)
			require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Submitted: 2, Failed: 1}, got.DefinitionSummary)
			require.NotContains(t, deleted, "")
		})
	}
}

// TestCleanupSmokeTestResourcesWaitErrorRetainsAcceptedEvidence verifies a confirmation failure remains submitted while later siblings complete.
func TestCleanupSmokeTestResourcesWaitErrorRetainsAcceptedEvidence(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	piAPI := stubProcessInstanceAPI{search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
		return nil, nil
	}}
	pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
		if services.ApplyCallOptions(opts).WithStat {
			return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
		}
		if key == "pd-main-child" {
			return d.ProcessDefinition{}, errors.New("confirmation unavailable")
		}
		return d.ProcessDefinition{}, d.ErrNotFound
	}}
	resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
		return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}, nil
	}}
	service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V89).(*Service)

	got, cleanupErr := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{Request: d.SmokeTestRequest{Count: 1}, Deployment: deployment})

	require.Error(t, cleanupErr)
	require.True(t, got.DefinitionResults[0].Cleanup.Submitted)
	require.False(t, got.DefinitionResults[0].Cleanup.Confirmed)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionFailed, got.DefinitionResults[0].Disposition)
	require.Contains(t, got.DefinitionResults[0].Reason, "confirmation unavailable")
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionConfirmed, got.DefinitionResults[1].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionConfirmed, got.DefinitionResults[2].Disposition)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Confirmed: 2, Failed: 1}, got.DefinitionSummary)
}

// TestCleanupSmokeTestResourcesFailFastPreservesPriorResults verifies fail-fast leaves completed evidence unchanged and explains later unscheduled work.
func TestCleanupSmokeTestResourcesFailFastPreservesPriorResults(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	piAPI := stubProcessInstanceAPI{search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
		if filter.ProcessDefinitionKey == "pd-main-middle" {
			return nil, errors.New("eligibility unavailable")
		}
		return nil, nil
	}}
	pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
		return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
	}}
	resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
		return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}, nil
	}}
	service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V89).(*Service)

	got, cleanupErr := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{
		Request: d.SmokeTestRequest{Count: 1, NoWait: true, FailFast: true}, Deployment: deployment,
	}, services.WithNoWait(), services.WithFailFast())

	require.Error(t, cleanupErr)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionSubmitted, got.DefinitionResults[0].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionFailed, got.DefinitionResults[1].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, got.DefinitionResults[2].Disposition)
	require.Contains(t, got.DefinitionResults[2].Reason, "--fail-fast")
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Submitted: 1, Failed: 1, Unattempted: 1}, got.DefinitionSummary)
	require.Equal(t, 1, resource.deleteCalls)
}

// TestCleanupSmokeTestResourcesCancellationPreservesPriorResults verifies cancellation stops scheduling without overwriting an accepted sibling.
func TestCleanupSmokeTestResourcesCancellationPreservesPriorResults(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	ctx, cancel := context.WithCancel(context.Background())
	piAPI := stubProcessInstanceAPI{search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
		return nil, nil
	}}
	pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
		return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
	}}
	resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
		cancel()
		return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}, nil
	}}
	service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V89).(*Service)

	got, cleanupErr := service.cleanupSmokeTestResources(ctx, d.SmokeTestResult{
		Request: d.SmokeTestRequest{Count: 1, NoWait: true}, Deployment: deployment,
	}, services.WithNoWait())

	require.ErrorIs(t, cleanupErr, context.Canceled)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionSubmitted, got.DefinitionResults[0].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, got.DefinitionResults[1].Disposition)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, got.DefinitionResults[2].Disposition)
	require.Contains(t, got.DefinitionResults[1].Reason, "context canceled")
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Submitted: 1, Unattempted: 2}, got.DefinitionSummary)
	require.Equal(t, 1, resource.deleteCalls)
}

// TestCleanupSmokeTestResourcesInvalidIdentityNeverMutatesWithoutExactKey verifies incomplete/conflicting evidence is reported without name fallback.
func TestCleanupSmokeTestResourcesInvalidIdentityNeverMutatesWithoutExactKey(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deploymentEvidence := completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a")
	deploymentEvidence.Units = append(deploymentEvidence.Units,
		d.DeploymentUnit{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: "C89_Other", ProcessDefinitionKey: "pd-main-child", TenantId: "tenant-b"}},
		d.DeploymentUnit{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: "C89_Unkeyed", ResourceName: "unkeyed.bpmn"}},
	)
	deployment := smokeTestDeploymentResult(fixture, deploymentEvidence, d.OpsWorkflowStepStatusConfirmed)
	piAPI := stubProcessInstanceAPI{search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
		return nil, nil
	}}
	pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
		return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
	}}
	var deleted []string
	resource := &stubSmokeTestResourceAPI{delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
		deleted = append(deleted, key)
		return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusAccepted, Status: "202 Accepted"}, nil
	}}
	service := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V89).(*Service)

	got, cleanupErr := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{
		Request: d.SmokeTestRequest{Count: 1, NoWait: true}, Deployment: deployment,
	}, services.WithNoWait())

	require.Error(t, cleanupErr)
	require.False(t, deployment.DefinitionScopeComplete)
	require.Equal(t, []string{"pd-main-child", "pd-main-middle", "pd-main"}, deleted)
	require.Equal(t, d.SmokeTestDefinitionCleanupDispositionFailed, got.DefinitionResults[3].Disposition)
	require.Equal(t, "missing process-definition key", got.DefinitionResults[3].Reason)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 4, Submitted: 3, Failed: 1}, got.DefinitionSummary)
	require.Contains(t, strings.Join(got.Errors, "\n"), "conflicting deployment identity evidence")
}

// TestSmokeTestDefinitionEligibilityPagesPastOwnedEvidence proves the real paged safety helper reaches a later blocker page.
func TestSmokeTestDefinitionEligibilityPagesPastOwnedEvidence(t *testing.T) {
	var pages []d.ProcessInstancePageRequest
	piAPI := stubProcessInstanceAPI{searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
		pages = append(pages, page)
		require.Equal(t, "pd-child", filter.ProcessDefinitionKey)
		if page.After == "" {
			return d.ProcessInstancePage{Items: []d.ProcessInstance{{Key: "owned-1"}}, EndCursor: "next", OverflowState: d.ProcessInstanceOverflowStateHasMore}, nil
		}
		return d.ProcessInstancePage{Items: []d.ProcessInstance{{Key: "unrelated-1"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}}

	got, err := smokeTestProcessDefinitionCleanupEligibility(context.Background(), piAPI, d.SmokeTestDeploymentResult{
		ProcessDefinitionKey: "pd-child", BpmnProcessID: "C89_SimpleUserTask",
	}, typexKeys("owned-1"))

	require.NoError(t, err)
	require.False(t, got.Eligible)
	require.Equal(t, []string{"unrelated-1"}, got.Blockers)
	require.Len(t, pages, 2)
	require.Equal(t, "next", pages[1].After)
}

// TestExecuteSmokeTestStartFailureAccountsForKnownDefinitions verifies earlier-stage failures retain an actionable unattempted collection.
func TestExecuteSmokeTestStartFailureAccountsForKnownDefinitions(t *testing.T) {
	resource := &stubSmokeTestResourceAPI{deploy: func(context.Context, []d.DeploymentUnitData, ...services.CallOption) (d.Deployment, error) {
		return completeSmokeTestDeployment("pd-88", ""), nil
	}}
	piAPI := stubProcessInstanceAPI{createProcessInstance: func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
		return d.ProcessInstanceCreation{}, errors.New("start unavailable")
	}}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{CommandName: "ops execute smoke-test", Count: 1})

	require.Error(t, err)
	require.Equal(t, d.SmokeTestOutcomePartiallyFailed, got.Outcome)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusFailed, Total: 3, Unattempted: 3}, got.Cleanup.DefinitionSummary)
	for _, item := range got.Cleanup.DefinitionResults {
		require.Equal(t, d.SmokeTestDefinitionCleanupDispositionUnattempted, item.Disposition)
		require.Contains(t, item.Reason, "process-instance creation failed")
	}
	require.Equal(t, got.Cleanup, got.Report.Cleanup)
}

func TestExecuteSmokeTestMissingFixtureFailsBeforeMutation(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{}
	got, err := NewWithWorkflowDependencies(nil, nil, nil, nil, resource, toolx.CamundaVersion("8.11")).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
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
			require.Len(t, units, 3)
			require.Equal(t, "processdefinitions/C88_SimpleUserTask.bpmn", units[0].Name)
			require.Equal(t, "processdefinitions/C88_SimpleParent.bpmn", units[1].Name)
			require.Equal(t, "processdefinitions/C88_MultipleSubProcessesParent.bpmn", units[2].Name)
			for _, unit := range units {
				require.Equal(t, "application/xml", unit.ContentType)
			}
			require.Contains(t, string(units[2].Data), "C88_MultipleSubProcessesParent")
			return d.Deployment{
				Key:      "deployment-1",
				TenantId: "tenant-a",
				Units: []d.DeploymentUnit{{
					ProcessDefinition: d.ProcessDefinitionDeployment{
						ProcessDefinitionId:      "C88_SimpleUserTask",
						ProcessDefinitionKey:     "pd-88-child",
						ProcessDefinitionVersion: 4,
						ResourceName:             "processdefinitions/C88_SimpleUserTask.bpmn",
						TenantId:                 "tenant-a",
					},
				}, {
					ProcessDefinition: d.ProcessDefinitionDeployment{
						ProcessDefinitionId:      "C88_SimpleParent",
						ProcessDefinitionKey:     "pd-88-middle",
						ProcessDefinitionVersion: 4,
						ResourceName:             "processdefinitions/C88_SimpleParent.bpmn",
						TenantId:                 "tenant-a",
					},
				}, {
					ProcessDefinition: d.ProcessDefinitionDeployment{
						ProcessDefinitionId:      "C88_MultipleSubProcessesParent",
						ProcessDefinitionKey:     "pd-88",
						ProcessDefinitionVersion: 4,
						ResourceName:             "processdefinitions/C88_MultipleSubProcessesParent.bpmn",
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
				BpmnProcessId:        "C88_MultipleSubProcessesParent",
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
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn", got.Deployment.FixtureFile)
	require.Equal(t, "C88_MultipleSubProcessesParent", got.Deployment.BpmnProcessID)
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

// TestExecuteSmokeTestAggregatesTenantEvidenceFromCreatedResources proves
// smoke-test tenant evidence comes from deployment and run responses already used.
func TestExecuteSmokeTestAggregatesTenantEvidenceFromCreatedResources(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-smoke", "tenant-b"), nil
		},
	}
	created := 0
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			created++
			tenantID := data.TenantId
			key := "pi-a"
			if created == 1 {
				tenantID = "tenant-a"
				key = "pi-b"
			}
			return d.ProcessInstanceCreation{
				Key:                  key,
				BpmnProcessId:        "C88_MultipleSubProcessesParent",
				ProcessDefinitionKey: data.ProcessDefinitionSpecificId,
				TenantId:             tenantID,
			}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				Mode:     pitraversal.ModeFamily,
				StartKey: startKey,
				RootKey:  startKey,
				Keys:     []string{startKey},
				Outcome:  pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       2,
		Workers:     1,
		NoCleanup:   true,
	})

	require.NoError(t, err)
	require.Equal(t, d.TenantEvidence{
		ResolvedTenantIDs: []string{"tenant-b"},
		TargetCount:       1,
		Targets: []d.TenantEvidenceTarget{
			{Key: "pd-smoke", TenantID: "tenant-b"},
		},
	}, got.Deployment.TenantEvidence)
	require.Equal(t, d.TenantEvidence{
		ResolvedTenantIDs: []string{"tenant-a", "tenant-b"},
		TargetCount:       2,
		Targets: []d.TenantEvidenceTarget{
			{Key: "pi-b", TenantID: "tenant-a"},
			{Key: "pi-a", TenantID: "tenant-b"},
		},
	}, got.Run.TenantEvidence)
	require.Equal(t, got.Deployment.TenantEvidence, got.Report.Deployment.TenantEvidence)
	require.Equal(t, got.Run.TenantEvidence, got.Report.Run.TenantEvidence)
}

// TestExecuteSmokeTestStartsCreatedInstancesByDeployedProcessDefinitionKey keeps the runnable main fixture as the singular projection.
func TestExecuteSmokeTestStartsCreatedInstancesByDeployedProcessDefinitionKey(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-exact", "tenant-a"), nil
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
				BpmnProcessId:        "C88_MultipleSubProcessesParent",
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

// TestExecuteSmokeTestRejectsMissingMainDeploymentIdentity prevents a name fallback from authorizing a run or cleanup.
func TestExecuteSmokeTestRejectsMissingMainDeploymentIdentity(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{}, nil
		},
	}
	got, err := NewWithWorkflowDependencies(nil, nil, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		NoCleanup:   true,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "does not contain a usable main fixture definition")
	require.Equal(t, d.SmokeTestOutcomePartiallyFailed, got.Outcome)
	require.Empty(t, got.Run.ProcessInstanceKeys)
}

// TestExecuteSmokeTestTraversalSummariesCapturePartialFamilyWalks preserves traversal evidence with complete deployment fixtures.
func TestExecuteSmokeTestTraversalSummariesCapturePartialFamilyWalks(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-88", ""), nil
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

// TestExecuteSmokeTestCleansUpCreatedResources verifies all three exact deployment keys are confirmed once.
func TestExecuteSmokeTestCleansUpCreatedResources(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-88", ""), nil
		},
		delete: func(_ context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
			require.False(t, services.ApplyCallOptions(opts).NoWait)
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 200, Status: "200 OK", DeleteHistory: true, BatchOperationKey: "batch-" + key, BatchState: "COMPLETED"}, nil
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
			require.Contains(t, []string{"pd-88-child", "pd-88-middle", "pd-88"}, filter.ProcessDefinitionKey)
			require.Equal(t, int32(1000), size)
			return nil, nil
		},
	}
	pdAPI := stubProcessDefinitionAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.WithStat {
				return d.ProcessDefinition{}, d.ErrNotFound
			}
			require.True(t, cfg.WithStat)
			return d.ProcessDefinition{Key: key, BpmnProcessId: smokeTestBPMNIDForKey(key), Statistics: &d.ProcessDefinitionStatistics{}}, nil
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
	require.Equal(t, 3, resource.deleteCalls)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusConfirmed, Total: 3, Confirmed: 3}, got.Cleanup.DefinitionSummary)
	require.Len(t, got.Cleanup.DefinitionResults, 3)
	require.Equal(t, got.Cleanup, got.Report.Cleanup)
}

// TestExecuteSmokeTestSkipsProcessDefinitionCleanupForUnrelatedInstances retains a blocked definition while removing eligible siblings.
func TestExecuteSmokeTestSkipsProcessDefinitionCleanupForUnrelatedInstances(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-88", ""), nil
		},
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "200 OK"}, nil
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
			if filter.ProcessDefinitionKey == "pd-88" {
				return []d.ProcessInstance{{Key: "unrelated-1", ProcessDefinitionKey: "pd-88"}}, nil
			}
			return nil, nil
		},
	}

	pdAPI := stubProcessDefinitionAPI{getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
		if !services.ApplyCallOptions(opts).WithStat {
			return d.ProcessDefinition{}, d.ErrNotFound
		}
		return d.ProcessDefinition{Key: key, BpmnProcessId: smokeTestBPMNIDForKey(key), Statistics: &d.ProcessDefinitionStatistics{}}, nil
	}}
	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
	})

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusConfirmed, got.Cleanup.ProcessInstanceCleanup.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionEligibility.Status)
	require.Equal(t, []string{"unrelated-1"}, got.Cleanup.ProcessDefinitionEligibility.Blockers)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Cleanup.ProcessDefinitionCleanup.Status)
	require.Equal(t, "pd-88", got.Cleanup.RetainedProcessDefinitionKey)
	require.Equal(t, "C88_MultipleSubProcessesParent", got.Cleanup.RetainedBpmnProcessID)
	require.Equal(t, 2, resource.deleteCalls)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusSkipped, Total: 3, Confirmed: 2, Retained: 1}, got.Cleanup.DefinitionSummary)
}

// TestExecuteSmokeTestNoCleanupRetainsCreatedResources accounts for every deployed definition without mutation.
func TestExecuteSmokeTestNoCleanupRetainsCreatedResources(t *testing.T) {
	t.Parallel()

	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-retained", "tenant-a"), nil
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
	require.Equal(t, "C88_MultipleSubProcessesParent", got.Cleanup.RetainedBpmnProcessID)
	require.Equal(t, "tenant-a", got.Cleanup.RetainedTenantID)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusSkipped, Total: 3, Retained: 3}, got.Cleanup.DefinitionSummary)
	require.Equal(t, got.Cleanup, got.Report.Cleanup)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Plan.PlannedSteps[5].Status)
	require.Zero(t, resource.deleteCalls)
}

// TestExecuteSmokeTestEmitsExplicitWorkProgress verifies smoke-test stages preserve progress callbacks through nested service options.
func TestExecuteSmokeTestEmitsExplicitWorkProgress(t *testing.T) {
	var events []d.OpsProgressEvent
	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-88", ""), nil
		},
	}
	created := 0
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, _ d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.NotNil(t, services.ApplyCallOptions(opts).Progress)
			created++
			return d.ProcessInstanceCreation{Key: string(rune('0' + created))}, nil
		},
		familyResult: func(_ context.Context, key string, opts ...services.CallOption) (pitraversal.Result, error) {
			require.Nil(t, services.ApplyCallOptions(opts).Progress)
			return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: key, RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: {Key: key}}, Outcome: pitraversal.OutcomeComplete}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       2,
		Workers:     1,
		NoCleanup:   true,
		Progress: func(event d.OpsProgressEvent) {
			events = append(events, event)
		},
	})

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
	requireSmokeTestProgressContains(t, events, "deploying smoke-test fixture", 0, 1)
	requireSmokeTestProgressContains(t, events, "deploying smoke-test fixture", 1, 1)
	requireSmokeTestProgressContains(t, events, "starting process instances", 0, 2)
	requireSmokeTestProgressContains(t, events, "starting process instances", 2, 2)
	requireSmokeTestProgressContains(t, events, "walking process-instance families", 0, 2)
	requireSmokeTestProgressContains(t, events, "walking process-instance families", 2, 2)
}

// TestExecuteSmokeTestSuppressesLegacyProgressLogsWithStructuredProgress
// verifies semantic callbacks replace smoke-test stage INFO logs while callers
// without callbacks keep the legacy diagnostics.
func TestExecuteSmokeTestSuppressesLegacyProgressLogsWithStructuredProgress(t *testing.T) {
	run := func(t *testing.T, withProgress bool) string {
		t.Helper()
		var logBuf bytes.Buffer
		log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
		var events []d.OpsProgressEvent
		resource := &stubSmokeTestResourceAPI{
			deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
				return completeSmokeTestDeployment("pd-88", ""), nil
			},
		}
		piAPI := stubProcessInstanceAPI{
			createProcessInstance: func(_ context.Context, _ d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
				return d.ProcessInstanceCreation{Key: "pi-1"}, nil
			},
			familyResult: func(_ context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
				return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: key, RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: {Key: key}}, Outcome: pitraversal.OutcomeComplete}, nil
			},
		}
		request := d.SmokeTestRequest{
			CommandName: "ops execute smoke-test",
			Count:       1,
			NoCleanup:   true,
		}
		if withProgress {
			request.Progress = func(event d.OpsProgressEvent) {
				events = append(events, event)
			}
		}

		got, err := NewWithWorkflowDependencies(nil, piAPI, nil, nil, resource, toolx.V88, log).ExecuteSmokeTest(context.Background(), request)

		require.NoError(t, err)
		require.Equal(t, d.SmokeTestOutcomePassedCleanupSkipped, got.Outcome)
		if withProgress {
			require.NotEmpty(t, events)
		}
		return logBuf.String()
	}

	legacyLog := run(t, false)
	require.Contains(t, legacyLog, "deploy: fixture embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn")
	require.Contains(t, legacyLog, "deploy: confirmed process definition pd-88")
	require.Contains(t, legacyLog, "start: 1 process instance")
	require.Contains(t, legacyLog, "start: created 1/1")
	require.Contains(t, legacyLog, "walk: 1 process-instance family")
	require.Contains(t, legacyLog, "cleanup: skipped (--no-cleanup)")

	structuredLog := run(t, true)
	require.NotContains(t, structuredLog, "deploy:")
	require.NotContains(t, structuredLog, "start:")
	require.NotContains(t, structuredLog, "walk:")
	require.NotContains(t, structuredLog, "cleanup:")
}

// TestExecuteSmokeTestEmitsStageCompletionFacts verifies the smoke-test
// workflow exposes high-level deploy, start, walk, and cleanup completions.
func TestExecuteSmokeTestEmitsStageCompletionFacts(t *testing.T) {
	var events testx.SafeSlice[d.OpsProgressEvent]
	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, _ []d.DeploymentUnitData, _ ...services.CallOption) (d.Deployment, error) {
			return completeSmokeTestDeployment("pd-88", ""), nil
		},
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: http.StatusOK, Status: "200 OK"}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{Key: "pi-1", ProcessDefinitionKey: data.ProcessDefinitionSpecificId}, nil
		},
		familyResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{Mode: pitraversal.ModeFamily, StartKey: startKey, RootKey: startKey, Keys: []string{startKey}, Chain: map[string]d.ProcessInstance{
				startKey: {Key: startKey, State: d.StateActive, ProcessDefinitionKey: "pd-88"},
			}, Outcome: pitraversal.OutcomeComplete}, nil
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
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "pi-1", key)
			return d.DeleteResponse{Ok: true, StatusCode: http.StatusOK, Status: "200 OK"}, nil
		},
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Contains(t, []string{"pd-88-child", "pd-88-middle", "pd-88"}, filter.ProcessDefinitionKey)
			return nil, nil
		},
	}
	pdAPI := stubProcessDefinitionAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
			if !services.ApplyCallOptions(opts).WithStat {
				return d.ProcessDefinition{}, d.ErrNotFound
			}
			return d.ProcessDefinition{Key: key, BpmnProcessId: smokeTestBPMNIDForKey(key), Statistics: &d.ProcessDefinitionStatistics{}}, nil
		},
	}

	got, err := NewWithWorkflowDependencies(nil, piAPI, nil, pdAPI, resource, toolx.V88).ExecuteSmokeTest(context.Background(), d.SmokeTestRequest{
		CommandName: "ops execute smoke-test",
		Count:       1,
		Progress:    events.Append,
	})

	require.NoError(t, err)
	require.Equal(t, d.SmokeTestOutcomePassed, got.Outcome)
	snapshot := events.Snapshot()
	require.Equal(t, []d.OpsCompletionProgress{{
		Phase:        "deploying smoke-test fixture",
		CoreResource: "deployment(s)",
		Total:        1,
		Identity:     "pd-88",
		Disposition:  d.OpsCompletionDispositionConfirmed,
	}}, opsCompletionProgressByPhase(snapshot, "deploying smoke-test fixture"))
	require.Equal(t, []d.OpsCompletionProgress{{
		Phase:        "starting process instances",
		CoreResource: "process instance(s)",
		Total:        1,
		Identity:     "pi-1",
		Disposition:  d.OpsCompletionDispositionConfirmed,
	}}, opsCompletionProgressByPhase(snapshot, "starting process instances"))
	require.Equal(t, []d.OpsCompletionProgress{{
		Phase:        "walking process-instance families",
		CoreResource: "process instance(s)",
		Total:        1,
		Identity:     "pi-1",
		Disposition:  d.OpsCompletionDispositionConfirmed,
	}}, opsCompletionProgressByPhase(snapshot, "walking process-instance families"))
	require.Equal(t, []d.OpsCompletionProgress{{
		Phase:        "cleaning up smoke-test process instances",
		CoreResource: "process-instance tree(s)",
		Total:        1,
		Identity:     "pi-1",
		Disposition:  d.OpsCompletionDispositionConfirmed,
	}}, opsCompletionProgressByPhase(snapshot, "cleaning up smoke-test process instances"))
	require.Equal(t, []d.OpsCompletionProgress{
		{Phase: "cleaning up smoke-test process definitions", CoreResource: "process definition(s)", Total: 3, Identity: "pd-88-child", Disposition: d.OpsCompletionDispositionConfirmed},
		{Phase: "cleaning up smoke-test process definitions", CoreResource: "process definition(s)", Total: 3, Identity: "pd-88-middle", Disposition: d.OpsCompletionDispositionConfirmed},
		{Phase: "cleaning up smoke-test process definitions", CoreResource: "process definition(s)", Total: 3, Identity: "pd-88", Disposition: d.OpsCompletionDispositionConfirmed},
	}, opsCompletionProgressByPhase(snapshot, "cleaning up smoke-test process definitions"))
	require.Empty(t, opsCompletionProgressByPhase(snapshot, "delete process definitions"))
}

// TestCleanupSmokeTestResourcesProgressAccountsForRetainedDefinitions verifies
// no-cleanup freezes one collection total and reports each retained definition as skipped.
func TestCleanupSmokeTestResourcesProgressAccountsForRetainedDefinitions(t *testing.T) {
	fixture, err := smokeTestFixtureForVersion(toolx.V89)
	require.NoError(t, err)
	deployment := smokeTestDeploymentResult(fixture, completeSmokeTestDeploymentForFixture(fixture, "pd-main", "tenant-a"), d.OpsWorkflowStepStatusConfirmed)
	var events []d.OpsProgressEvent
	service := NewWithWorkflowDependencies(nil, nil, nil, nil, nil, toolx.V89).(*Service)

	got, cleanupErr := service.cleanupSmokeTestResources(context.Background(), d.SmokeTestResult{
		Request:    d.SmokeTestRequest{Count: 1, NoCleanup: true},
		Deployment: deployment,
	}, services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, cleanupErr)
	require.Equal(t, &d.SmokeTestDefinitionCleanupSummary{Status: d.OpsWorkflowStepStatusSkipped, Total: 3, Retained: 3}, got.DefinitionSummary)
	requireSmokeTestProgressContains(t, events, "cleaning up smoke-test process definitions", 0, 3)
	requireSmokeTestProgressContains(t, events, "cleaning up smoke-test process definitions", 3, 3)
	completions := opsCompletionProgressByPhase(events, "cleaning up smoke-test process definitions")
	require.Len(t, completions, 3)
	for _, completion := range completions {
		require.Equal(t, 3, completion.Total)
		require.Equal(t, d.OpsCompletionDispositionSkipped, completion.Disposition)
		require.Equal(t, "cleanup disabled (--no-cleanup)", completion.FailureDetail)
	}
}

// requireSmokeTestProgressContains asserts a stage counter appeared without depending on concurrent event ordering.
func requireSmokeTestProgressContains(t *testing.T, events []d.OpsProgressEvent, phase string, done int, total int) {
	t.Helper()
	for _, event := range events {
		if event.Kind != d.OpsProgressEventKindFrozenScope || event.FrozenScope == nil {
			continue
		}
		if event.FrozenScope.Phase == phase && event.FrozenScope.Done == done && event.FrozenScope.Total == total {
			return
		}
	}
	require.Failf(t, "missing smoke-test progress", "phase=%s done=%d total=%d events=%v", phase, done, total, events)
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

// completeSmokeTestDeployment returns realistic three-definition evidence for the 8.8 fixture family.
func completeSmokeTestDeployment(mainKey string, tenantID string) d.Deployment {
	fixture, _ := smokeTestFixtureForVersion(toolx.V88)
	return completeSmokeTestDeploymentForFixture(fixture, mainKey, tenantID)
}

// completeSmokeTestDeploymentForFixture builds exact three-definition response evidence for one supported fixture family.
func completeSmokeTestDeploymentForFixture(fixture d.EmbeddedSmokeTestFixture, mainKey string, tenantID string) d.Deployment {
	expected := smokeTestExpectedDefinitionIDs(fixture)
	return d.Deployment{TenantId: tenantID, Units: []d.DeploymentUnit{
		{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: expected[0], ProcessDefinitionKey: mainKey + "-child", ProcessDefinitionVersion: 4, ResourceName: "processdefinitions/" + expected[0] + ".bpmn", TenantId: tenantID}},
		{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: expected[1], ProcessDefinitionKey: mainKey + "-middle", ProcessDefinitionVersion: 4, ResourceName: "processdefinitions/" + expected[1] + ".bpmn", TenantId: tenantID}},
		{ProcessDefinition: d.ProcessDefinitionDeployment{ProcessDefinitionId: expected[2], ProcessDefinitionKey: mainKey, ProcessDefinitionVersion: 4, ResourceName: "processdefinitions/" + expected[2] + ".bpmn", TenantId: tenantID}},
	}}
}

// smokeTestBPMNIDForKey maps deterministic fixture keys back to their exact deployed identities.
func smokeTestBPMNIDForKey(key string) string {
	switch {
	case strings.HasSuffix(key, "-child"):
		return "C88_SimpleUserTask"
	case strings.HasSuffix(key, "-middle"):
		return "C88_SimpleParent"
	default:
		return "C88_MultipleSubProcessesParent"
	}
}
