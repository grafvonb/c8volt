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

const incidentPurgeReportSchemaVersion = "ops.process-instances-with-incidents.v1"

// PurgeProcessInstancesWithIncidents prepares the incident-purge workflow result shape.
func (s *Service) PurgeProcessInstancesWithIncidents(ctx context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error) {
	_ = ctx
	request = withIncidentPurgeOptionControls(request, opts...)
	if request.StartedAt.IsZero() {
		request.StartedAt = time.Now().UTC()
	}
	result := newIncidentPurgeResult(request)

	if err := validateIncidentPurgeServiceReady(s); err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.Errors = []string{err.Error()}
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomeFailed, err)
	}

	result.Discovery = d.IncidentDiscoveryResult{
		Status:  d.OpsWorkflowStepStatusPlanned,
		Filters: request.Selection,
	}
	result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
	result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
	return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomePlanned, nil)
}

// withIncidentPurgeOptionControls records call-option controls in the durable request model.
func withIncidentPurgeOptionControls(request d.IncidentPurgeRequest, opts ...services.CallOption) d.IncidentPurgeRequest {
	cfg := services.ApplyCallOptions(opts)
	request.NoWait = request.NoWait || cfg.NoWait
	request.Force = request.Force || cfg.Force
	request.FailFast = request.FailFast || cfg.FailFast
	request.NoWorkerLimit = request.NoWorkerLimit || cfg.NoWorkerLimit
	return request
}

// validateIncidentPurgeServiceReady protects the foundational service boundary before discovery exists.
func validateIncidentPurgeServiceReady(s *Service) error {
	if s == nil || s.piAPI == nil {
		return fmt.Errorf("%w: incident purge requires process-instance service", d.ErrValidation)
	}
	if s.incAPI == nil {
		return fmt.Errorf("%w: incident purge requires incident service", d.ErrValidation)
	}
	return nil
}

// newIncidentPurgeResult initializes report metadata that is available before remote work.
func newIncidentPurgeResult(request d.IncidentPurgeRequest) d.IncidentPurgeResult {
	return d.IncidentPurgeResult{
		Request: request,
		Report: d.IncidentPurgeReport{
			SchemaVersion:    incidentPurgeReportSchemaVersion,
			CommandName:      request.CommandName,
			StartedAt:        request.StartedAt,
			DryRun:           request.DryRun,
			AutoConfirm:      request.AutoConfirm,
			Automation:       request.Automation,
			NoWait:           request.NoWait,
			Force:            request.Force,
			FailFast:         request.FailFast,
			NoWorkerLimit:    request.NoWorkerLimit,
			SelectionFilters: request.Selection,
			Outcome:          d.IncidentPurgeOutcomeFailed,
		},
		Outcome: d.IncidentPurgeOutcomeFailed,
	}
}

// finishIncidentPurgeResult copies step state into the audit report and records terminal errors.
func finishIncidentPurgeResult(result d.IncidentPurgeResult, outcome d.IncidentPurgeOutcome, err error) (d.IncidentPurgeResult, error) {
	finished := time.Now().UTC()
	result.Outcome = outcome
	result.Report.Outcome = outcome
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Discovery = result.Discovery
	result.Report.DeletePlan = result.DeletePlan
	result.Report.Deletion = result.Deletion
	result.Report.Errors = append([]string(nil), result.Errors...)
	result.Report.Notices = append([]d.IncidentPurgeWorkflowNotice(nil), result.Notices...)
	if err != nil {
		msg := err.Error()
		result.Errors = appendIfMissing(result.Errors, msg)
		result.Report.Errors = appendIfMissing(result.Report.Errors, msg)
	}
	return result, err
}
