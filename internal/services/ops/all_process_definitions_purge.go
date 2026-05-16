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

const allProcessDefinitionsPurgeReportSchemaVersion = "ops.all-process-definitions.v1"

// PurgeAllProcessDefinitions prepares the all-process-definitions purge workflow result shape.
func (s *Service) PurgeAllProcessDefinitions(ctx context.Context, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error) {
	_ = ctx
	request = withAllProcessDefinitionsPurgeOptionControls(request, opts...)
	if request.StartedAt.IsZero() {
		request.StartedAt = time.Now().UTC()
	}
	result := newAllProcessDefinitionsPurgeResult(request)

	if err := validateAllProcessDefinitionsPurgeServiceReady(s); err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.Errors = []string{err.Error()}
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomeFailed, err)
	}

	if request.DiscoveredCandidateProcessDefinitionKeys != nil {
		result.Discovery = frozenAllProcessDefinitionsPurgeDiscovery(request)
	} else {
		result.Discovery.Status = d.OpsWorkflowStepStatusSkipped
		result.Discovery.Filters = request.Selection
		result.Discovery.LatestOnly = request.Selection.IsLatestVersion
	}
	result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
	result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
	return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomePlanned, nil)
}

func frozenAllProcessDefinitionsPurgeDiscovery(request d.AllProcessDefinitionsPurgeRequest) d.ProcessDefinitionDiscoveryResult {
	candidates := request.DiscoveredCandidateProcessDefinitionKeys.Unique()
	return d.ProcessDefinitionDiscoveryResult{
		Status:                          d.OpsWorkflowStepStatusPlanned,
		Filters:                         request.Selection,
		CandidateProcessDefinitionKeys:  candidates,
		CandidateProcessDefinitionCount: len(candidates),
		LatestOnly:                      request.Selection.IsLatestVersion,
	}
}

func withAllProcessDefinitionsPurgeOptionControls(request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) d.AllProcessDefinitionsPurgeRequest {
	cfg := services.ApplyCallOptions(opts)
	request.NoWait = request.NoWait || cfg.NoWait
	request.Force = request.Force || cfg.Force
	request.FailFast = request.FailFast || cfg.FailFast
	request.NoWorkerLimit = request.NoWorkerLimit || cfg.NoWorkerLimit
	return request
}

func validateAllProcessDefinitionsPurgeServiceReady(s *Service) error {
	if s == nil || s.piAPI == nil {
		return fmt.Errorf("%w: all-process-definitions purge requires process-instance service", d.ErrValidation)
	}
	if s.pdAPI == nil {
		return fmt.Errorf("%w: all-process-definitions purge requires process-definition service", d.ErrValidation)
	}
	if s.resourceAPI == nil {
		return fmt.Errorf("%w: all-process-definitions purge requires resource service", d.ErrValidation)
	}
	return nil
}

func newAllProcessDefinitionsPurgeResult(request d.AllProcessDefinitionsPurgeRequest) d.AllProcessDefinitionsPurgeResult {
	return d.AllProcessDefinitionsPurgeResult{
		Request: request,
		Report: d.AllProcessDefinitionsPurgeReport{
			SchemaVersion:    allProcessDefinitionsPurgeReportSchemaVersion,
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
			Outcome:          d.AllProcessDefinitionsPurgeOutcomeFailed,
		},
		Outcome: d.AllProcessDefinitionsPurgeOutcomeFailed,
	}
}

func finishAllProcessDefinitionsPurgeResult(result d.AllProcessDefinitionsPurgeResult, outcome d.AllProcessDefinitionsPurgeOutcome, err error) (d.AllProcessDefinitionsPurgeResult, error) {
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
	result.Report.Notices = append([]d.AllProcessDefinitionsPurgeWorkflowNotice(nil), result.Notices...)
	if err != nil {
		msg := err.Error()
		result.Errors = appendIfMissing(result.Errors, msg)
		result.Report.Errors = appendIfMissing(result.Report.Errors, msg)
	}
	return result, err
}
