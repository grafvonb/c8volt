// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

const retentionPolicyReportSchemaVersion = "ops.retention-policy.v1"

func (s *Service) ExecuteRetentionPolicy(ctx context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error) {
	started := request.StartedAt
	if started.IsZero() {
		started = time.Now().UTC()
		request.StartedAt = started
	}
	if request.DerivedEndDateBoundary == "" {
		request.DerivedEndDateBoundary = deriveRetentionEndDateBoundary(started, request.RetentionDays)
	}
	request = withRetentionPolicyOptionControls(request, opts...)
	result := newRetentionPolicyResult(request)

	if err := validateRetentionPolicyRequest(request); err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.Errors = []string{err.Error()}
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomeFailed, err)
	}

	filter := request.Selection
	filter.EndDateBefore = request.DerivedEndDateBoundary
	discovery, err := pisvc.DiscoverRetentionProcessInstances(ctx, s.piAPI, pisvc.RetentionDiscoveryRequest{
		Filter:    filter,
		BatchSize: request.BatchSize,
		Limit:     request.Limit,
	}, opts...)
	if err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.RetentionDays = request.RetentionDays
		result.Discovery.DerivedEndDateBoundary = request.DerivedEndDateBoundary
		result.Discovery.Filters = filter
		result.Discovery.Errors = []string{err.Error()}
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomeFailed, err)
	}

	result.Discovery = d.RetentionDiscoveryResult{
		Status:                 d.OpsWorkflowStepStatusPlanned,
		RetentionDays:          request.RetentionDays,
		DerivedEndDateBoundary: request.DerivedEndDateBoundary,
		Filters:                discovery.Filter,
		SeedKeys:               discovery.Keys,
		Count:                  len(discovery.Keys),
	}
	result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
	result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
	return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomePlanned, nil)
}

func deriveRetentionEndDateBoundary(now time.Time, retentionDays int) string {
	return now.UTC().AddDate(0, 0, -retentionDays).Format(time.DateOnly)
}

func validateRetentionPolicyRequest(request d.RetentionPolicyRequest) error {
	if request.RetentionDays < 0 {
		return fmt.Errorf("%w: retention-days must be a non-negative integer", d.ErrValidation)
	}
	if request.Selection.Key != "" {
		return fmt.Errorf("%w: retention policy discovers eligible process instances and does not accept explicit process-instance keys", d.ErrValidation)
	}
	return nil
}

func withRetentionPolicyOptionControls(request d.RetentionPolicyRequest, opts ...services.CallOption) d.RetentionPolicyRequest {
	cfg := services.ApplyCallOptions(opts)
	request.NoWait = request.NoWait || cfg.NoWait
	request.NoStateCheck = request.NoStateCheck || cfg.NoStateCheck
	request.Force = request.Force || cfg.Force
	request.FailFast = request.FailFast || cfg.FailFast
	request.NoWorkerLimit = request.NoWorkerLimit || cfg.NoWorkerLimit
	return request
}

func newRetentionPolicyResult(request d.RetentionPolicyRequest) d.RetentionPolicyResult {
	return d.RetentionPolicyResult{
		Request: request,
		Report: d.RetentionAuditReport{
			SchemaVersion:          retentionPolicyReportSchemaVersion,
			CommandName:            request.CommandName,
			StartedAt:              request.StartedAt,
			DryRun:                 request.DryRun,
			RetentionDays:          request.RetentionDays,
			DerivedEndDateBoundary: request.DerivedEndDateBoundary,
			SelectionFilters:       request.Selection,
			AutoConfirm:            request.AutoConfirm,
			Automation:             request.Automation,
			NoWait:                 request.NoWait,
			NoStateCheck:           request.NoStateCheck,
			Force:                  request.Force,
			FailFast:               request.FailFast,
			NoWorkerLimit:          request.NoWorkerLimit,
			Outcome:                d.RetentionPolicyOutcomeFailed,
		},
		Outcome: d.RetentionPolicyOutcomeFailed,
	}
}

func finishRetentionPolicyResult(result d.RetentionPolicyResult, outcome d.RetentionPolicyOutcome, err error) (d.RetentionPolicyResult, error) {
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
	if err != nil {
		msg := err.Error()
		result.Errors = appendIfMissing(result.Errors, msg)
	}
	result.Report.Errors = append([]string(nil), result.Errors...)
	return result, err
}
