// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

const smokeTestReportSchemaVersion = "ops.smoke-test.v1"

// ExecuteSmokeTest validates the shared smoke-test request shape and returns the foundational workflow envelope.
func (s *Service) ExecuteSmokeTest(_ context.Context, request d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
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

	result.Plan.Status = d.OpsWorkflowStepStatusPlanned
	result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
	result.Run.Status = d.OpsWorkflowStepStatusSkipped
	result.Walk.Status = d.OpsWorkflowStepStatusSkipped
	result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
	result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
	result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
	return finishSmokeTestResult(result, d.SmokeTestOutcomePlanned, nil)
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
