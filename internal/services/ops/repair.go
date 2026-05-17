// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
)

const repairReportSchemaVersion = "ops.repair.v1"

// RepairIncidents initializes the incident repair workflow boundary.
func (s *Service) RepairIncidents(_ context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	request.Target = d.OpsRepairTargetIncident
	return s.prepareRepairResult(request, opts...)
}

// RepairProcessInstances initializes the process-instance repair workflow boundary.
func (s *Service) RepairProcessInstances(_ context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	request.Target = d.OpsRepairTargetProcessInstance
	return s.prepareRepairResult(request, opts...)
}

// prepareRepairResult records foundational repair controls without performing target discovery or mutation.
func (s *Service) prepareRepairResult(request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	request = withRepairOptionControls(request, opts...)
	if request.StartedAt.IsZero() {
		request.StartedAt = time.Now().UTC()
	}
	result := newRepairResult(request)
	if err := validateRepairServiceReady(s); err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	return finishRepairResult(result, s.version, d.OpsRepairOutcomePlanned, nil)
}

// withRepairOptionControls folds shared call options into the durable repair request.
func withRepairOptionControls(request d.OpsRepairRequest, opts ...services.CallOption) d.OpsRepairRequest {
	cfg := services.ApplyCallOptions(opts)
	request.NoWait = request.NoWait || cfg.NoWait
	request.FailFast = request.FailFast || cfg.FailFast
	request.NoWorkerLimit = request.NoWorkerLimit || cfg.NoWorkerLimit
	request.DryRun = request.DryRun || cfg.DryRun
	return request
}

// newRepairResult constructs the report-safe repair result skeleton used before concrete target behavior is implemented.
func newRepairResult(request d.OpsRepairRequest) d.OpsRepairResult {
	frozen := d.OpsRepairFrozenSet{
		Status:          d.OpsWorkflowStepStatusPlanned,
		Target:          request.Target,
		DiscoveryMode:   request.DiscoveryMode,
		InputKeys:       request.InputKeys.Unique(),
		IncidentFilters: request.IncidentSelection,
		ProcessFilters:  request.ProcessInstanceSelection,
	}
	switch request.Target {
	case d.OpsRepairTargetIncident:
		frozen.IncidentKeys = request.InputKeys.Unique()
	case d.OpsRepairTargetProcessInstance:
		frozen.ProcessInstanceKeys = request.InputKeys.Unique()
	}
	return d.OpsRepairResult{
		Request:   request,
		FrozenSet: frozen,
		Remaining: d.OpsRepairRemainingIncidentSummary{Status: d.OpsWorkflowStepStatusSkipped},
		Outcome:   d.OpsRepairOutcomePlanned,
	}
}

// validateRepairServiceReady keeps required workflow dependencies explicit at the ops boundary.
func validateRepairServiceReady(s *Service) error {
	switch {
	case s == nil:
		return fmt.Errorf("%w: ops service is required for repair workflow", d.ErrValidation)
	case s.piAPI == nil:
		return fmt.Errorf("%w: process-instance service is required for repair workflow", d.ErrValidation)
	case s.incAPI == nil:
		return fmt.Errorf("%w: incident service is required for repair workflow", d.ErrValidation)
	case s.jobAPI == nil:
		return fmt.Errorf("%w: job service is required for repair workflow", d.ErrValidation)
	default:
		return nil
	}
}

// finishRepairResult stamps final audit metadata and carries errors into result and report fields.
func finishRepairResult(result d.OpsRepairResult, version toolx.CamundaVersion, outcome d.OpsRepairOutcome, err error) (d.OpsRepairResult, error) {
	result.Outcome = outcome
	if err != nil {
		result.Errors = []string{err.Error()}
	}
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	finished := time.Now().UTC()
	result.Report = d.OpsRepairAuditReport{
		SchemaVersion:    repairReportSchemaVersion,
		CommandName:      result.Request.CommandName,
		StartedAt:        result.Request.StartedAt,
		FinishedAt:       finished,
		Duration:         finished.Sub(result.Request.StartedAt).String(),
		DryRun:           result.Request.DryRun,
		CamundaVersion:   string(version),
		Request:          result.Request,
		FrozenSet:        result.FrozenSet,
		Plan:             append([]d.OpsRepairPlanItem(nil), result.Plan...),
		VariableUpdates:  append([]d.OpsRepairVariableScopeUpdate(nil), result.VariableUpdates...),
		JobApplicability: append([]d.OpsRepairJobApplicability(nil), result.JobApplicability...),
		Remaining:        result.Remaining,
		AutoConfirm:      result.Request.AutoConfirm,
		Automation:       result.Request.Automation,
		NoWait:           result.Request.NoWait,
		FailFast:         result.Request.FailFast,
		NoWorkerLimit:    result.Request.NoWorkerLimit,
		Errors:           append([]string(nil), result.Errors...),
		Notices:          append([]d.OpsRepairWorkflowNotice(nil), result.Notices...),
		Outcome:          outcome,
	}
	return result, err
}
