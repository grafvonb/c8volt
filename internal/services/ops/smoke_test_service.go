// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/embedded"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
)

const smokeTestReportSchemaVersion = "ops.smoke-test.v1"

// ExecuteSmokeTest validates the shared smoke-test request shape and returns the foundational workflow envelope.
func (s *Service) ExecuteSmokeTest(ctx context.Context, request d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
	started := request.StartedAt
	if started.IsZero() {
		started = time.Now().UTC()
		request.StartedAt = started
	}
	request = withSmokeTestOptionControls(request, opts...)
	result := newSmokeTestResult(request)

	if err := validateSmokeTestRequest(request); err != nil {
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.Errors = []string{err.Error()}
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	if request.DryRun {
		return s.executeSmokeTestDryRun(ctx, result, opts...)
	}

	result.Plan.Status = d.OpsWorkflowStepStatusPlanned
	result.Plan.PlannedSteps = smokeTestPlannedSteps(request, d.OpsWorkflowStepStatusSkipped, "", nil)
	result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
	result.Run.Status = d.OpsWorkflowStepStatusSkipped
	result.Walk.Status = d.OpsWorkflowStepStatusSkipped
	result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
	result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
	result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
	return finishSmokeTestResult(result, d.SmokeTestOutcomePlanned, nil)
}

func (s *Service) executeSmokeTestDryRun(ctx context.Context, result d.SmokeTestResult, opts ...services.CallOption) (d.SmokeTestResult, error) {
	version := s.version
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	fixture, err := smokeTestFixtureForVersion(version)
	if err != nil {
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.CamundaVersion = version.String()
		result.Plan.Errors = []string{err.Error()}
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	connectivityStatus := d.OpsWorkflowStepStatusSkipped
	connectivityMessage := "cluster topology check not configured"
	if s.clusterAPI != nil {
		if _, err := s.clusterAPI.GetClusterTopology(ctx, opts...); err != nil {
			result.Plan.Status = d.OpsWorkflowStepStatusFailed
			result.Plan.CamundaVersion = version.String()
			result.Plan.Fixture = fixture
			result.Fixture = fixture
			result.Plan.PlannedSteps = smokeTestPlannedSteps(result.Request, d.OpsWorkflowStepStatusFailed, err.Error(), []string{err.Error()})
			result.Plan.Errors = []string{err.Error()}
			result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
			result.Run.Status = d.OpsWorkflowStepStatusSkipped
			result.Walk.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
			return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, fmt.Errorf("smoke-test connectivity validation: %w", err))
		}
		connectivityStatus = d.OpsWorkflowStepStatusConfirmed
		connectivityMessage = "cluster topology retrieved"
	}

	result.Fixture = fixture
	result.Plan = d.SmokeTestPlan{
		Status:           d.OpsWorkflowStepStatusPlanned,
		CamundaVersion:   version.String(),
		Fixture:          fixture,
		CleanupRequested: !result.Request.NoCleanup,
		PlannedSteps:     smokeTestPlannedSteps(result.Request, connectivityStatus, connectivityMessage, nil),
	}
	result.Deployment = d.SmokeTestDeploymentResult{
		Status:        d.OpsWorkflowStepStatusPlanned,
		FixtureFile:   fixture.File,
		BpmnProcessID: fixture.BpmnProcessID,
	}
	result.Run = d.SmokeTestRunResult{
		Status:         d.OpsWorkflowStepStatusPlanned,
		RequestedCount: result.Request.Count,
	}
	result.Walk.Status = d.OpsWorkflowStepStatusPlanned
	cleanupStatus := d.OpsWorkflowStepStatusPlanned
	if result.Request.NoCleanup {
		cleanupStatus = d.OpsWorkflowStepStatusSkipped
	}
	result.Cleanup.ProcessInstanceCleanup.Status = cleanupStatus
	result.Cleanup.ProcessDefinitionEligibility.Status = cleanupStatus
	result.Cleanup.ProcessDefinitionCleanup.Status = cleanupStatus
	return finishSmokeTestResult(result, d.SmokeTestOutcomePlanned, nil)
}

func smokeTestFixtureForVersion(version toolx.CamundaVersion) (d.EmbeddedSmokeTestFixture, error) {
	normalized, err := toolx.NormalizeCamundaVersion(version.String())
	if err != nil {
		return d.EmbeddedSmokeTestFixture{}, fmt.Errorf("%w: unsupported smoke-test fixture version %q", d.ErrPrecondition, version)
	}
	processID := normalized.FilePrefix() + "MultipleSubProcessesParentProcess"
	fsPath := "processdefinitions/" + processID + ".bpmn"
	if _, err := fs.Stat(embedded.FS, fsPath); err != nil {
		return d.EmbeddedSmokeTestFixture{}, fmt.Errorf("%w: embedded smoke-test fixture not found: %s", d.ErrPrecondition, fsPath)
	}
	return d.EmbeddedSmokeTestFixture{
		CamundaVersion: normalized.String(),
		File:           filepath.ToSlash(filepath.Join("embedded", fsPath)),
		BpmnProcessID:  processID,
		Available:      true,
	}, nil
}

func smokeTestPlannedSteps(request d.SmokeTestRequest, connectivityStatus d.OpsWorkflowStepStatus, connectivityMessage string, connectivityErrors []string) []d.WorkflowStepResult {
	if connectivityStatus == "" {
		connectivityStatus = d.OpsWorkflowStepStatusPlanned
	}
	steps := []d.WorkflowStepResult{
		{Name: "connectivity", Status: connectivityStatus, Message: connectivityMessage, Errors: append([]string(nil), connectivityErrors...)},
		{Name: "fixture", Status: d.OpsWorkflowStepStatusPlanned, Message: "select version-matched embedded fixture"},
		{Name: "deployment", Status: d.OpsWorkflowStepStatusPlanned, Message: "deploy selected fixture"},
		{Name: "run", Status: d.OpsWorkflowStepStatusPlanned, Message: fmt.Sprintf("start %d process instance(s)", request.Count)},
		{Name: "walk", Status: d.OpsWorkflowStepStatusPlanned, Message: "walk each created process-instance family"},
	}
	cleanupStatus := d.OpsWorkflowStepStatusPlanned
	cleanupMessage := "clean up created process instances and eligible process definition"
	if request.NoCleanup {
		cleanupStatus = d.OpsWorkflowStepStatusSkipped
		cleanupMessage = "retain created resources because --no-cleanup is set"
	}
	steps = append(steps, d.WorkflowStepResult{Name: "cleanup", Status: cleanupStatus, Message: cleanupMessage})
	reportStatus := d.OpsWorkflowStepStatusSkipped
	reportMessage := "no audit report requested"
	if strings.TrimSpace(request.ReportFile) != "" {
		reportStatus = d.OpsWorkflowStepStatusPlanned
		reportMessage = "write audit report"
	}
	return append(steps, d.WorkflowStepResult{Name: "report", Status: reportStatus, Message: reportMessage})
}

// validateSmokeTestRequest enforces local request invariants before any later workflow step can mutate state.
func validateSmokeTestRequest(request d.SmokeTestRequest) error {
	if request.Count < 1 {
		return fmt.Errorf("%w: count must be a positive integer", d.ErrValidation)
	}
	switch request.ReportFormat {
	case "", "markdown", "json":
	default:
		return fmt.Errorf("%w: report-format must be markdown or json", d.ErrValidation)
	}
	if request.ReportFormat != "" && request.ReportFile == "" {
		return fmt.Errorf("%w: report-format requires report-file", d.ErrValidation)
	}
	return nil
}

// withSmokeTestOptionControls folds reusable call options into the persisted smoke-test request controls.
func withSmokeTestOptionControls(request d.SmokeTestRequest, opts ...services.CallOption) d.SmokeTestRequest {
	cfg := services.ApplyCallOptions(opts)
	request.NoWait = request.NoWait || cfg.NoWait
	request.FailFast = request.FailFast || cfg.FailFast
	request.NoWorkerLimit = request.NoWorkerLimit || cfg.NoWorkerLimit
	return request
}

// newSmokeTestResult initializes the common report envelope before validation or workflow execution.
func newSmokeTestResult(request d.SmokeTestRequest) d.SmokeTestResult {
	cleanupRequested := !request.NoCleanup
	return d.SmokeTestResult{
		Request: request,
		Plan: d.SmokeTestPlan{
			CleanupRequested: cleanupRequested,
		},
		Cleanup: d.SmokeTestCleanupResult{
			NoCleanup: request.NoCleanup,
		},
		Report: d.SmokeTestAuditReport{
			SchemaVersion:    smokeTestReportSchemaVersion,
			CommandName:      request.CommandName,
			StartedAt:        request.StartedAt,
			DryRun:           request.DryRun,
			CleanupRequested: cleanupRequested,
			NoCleanup:        request.NoCleanup,
			AutoConfirm:      request.AutoConfirm,
			Automation:       request.Automation,
			NoWait:           request.NoWait,
			Outcome:          d.SmokeTestOutcomeFailed,
		},
		Outcome: d.SmokeTestOutcomeFailed,
	}
}

// finishSmokeTestResult snapshots final status and mirrors step data into the audit report.
func finishSmokeTestResult(result d.SmokeTestResult, outcome d.SmokeTestOutcome, err error) (d.SmokeTestResult, error) {
	finished := time.Now().UTC()
	result.Outcome = outcome
	result.Report.Outcome = outcome
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Plan = result.Plan
	result.Report.Fixture = result.Fixture
	result.Report.Deployment = result.Deployment
	result.Report.Run = result.Run
	result.Report.Walk = result.Walk
	result.Report.Cleanup = result.Cleanup
	if err != nil {
		result.Errors = appendIfMissing(result.Errors, err.Error())
	}
	result.Report.Errors = append([]string(nil), result.Errors...)
	return result, err
}
