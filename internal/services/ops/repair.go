// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

const repairReportSchemaVersion = "ops.repair.v1"

// RepairIncidents initializes the incident repair workflow boundary.
func (s *Service) RepairIncidents(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	request.Target = d.OpsRepairTargetIncident
	request = withRepairOptionControls(request, opts...)
	if request.StartedAt.IsZero() {
		request.StartedAt = time.Now().UTC()
	}
	if err := validateRepairServiceReady(s); err != nil {
		result := newRepairResult(request)
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	return s.repairExplicitIncidents(ctx, request, opts...)
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

func (s *Service) repairExplicitIncidents(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	if request.DiscoveryMode == "" {
		request.DiscoveryMode = d.OpsRepairDiscoveryModeKeyed
	}
	result := newRepairResult(request)
	keys := request.InputKeys.Unique()
	if len(keys) == 0 {
		err := fmt.Errorf("%w: no incident keys provided for repair", d.ErrValidation)
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	incidents, err := incsvc.GetIncidents(ctx, s.incAPI, keys, request.Workers, opts...)
	if err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	result.FrozenSet = freezeExplicitIncidentSet(request, incidents)
	if request.DryRun {
		result.Plan, result.JobApplicability = buildRepairPlans(request, incidents)
		result.Remaining.Status = d.OpsWorkflowStepStatusSkipped
		return finishRepairResult(result, s.version, d.OpsRepairOutcomePlanned, nil)
	}

	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(incidents), request.Workers, cfg.NoWorkerLimit)
	items, runErr := pool.ExecuteSlice(ctx, incidents, workers, cfg.FailFast, func(ctx context.Context, incident d.ProcessInstanceIncidentDetail, _ int) (repairIncidentExecution, error) {
		return s.executeIncidentRepair(ctx, request, incident, opts...)
	})
	for _, item := range items {
		if item.Plan.IncidentKey == "" {
			continue
		}
		result.Plan = append(result.Plan, item.Plan)
		result.JobApplicability = append(result.JobApplicability, item.JobApplicability)
	}
	result.Remaining = d.OpsRepairRemainingIncidentSummary{Status: d.OpsWorkflowStepStatusConfirmed, Checked: !request.NoWait}
	outcome := repairOutcomeForPlans(result.Plan, runErr)
	return finishRepairResult(result, s.version, outcome, runErr)
}

type repairIncidentExecution struct {
	Plan             d.OpsRepairPlanItem
	JobApplicability d.OpsRepairJobApplicability
}

func freezeExplicitIncidentSet(request d.OpsRepairRequest, incidents []d.ProcessInstanceIncidentDetail) d.OpsRepairFrozenSet {
	frozen := newRepairResult(request).FrozenSet
	frozen.Status = d.OpsWorkflowStepStatusConfirmed
	frozen.IncidentKeys = incidentKeysFromDetails(incidents)
	frozen.ProcessInstanceKeys = processInstanceKeysFromIncidents(incidents)
	frozen.RootProcessKeys = rootProcessInstanceKeysFromIncidents(incidents)
	frozen.JobKeys = jobKeysFromIncidents(incidents)
	if len(request.Variables) > 0 {
		frozen.VariableScopes = frozen.ProcessInstanceKeys.Unique()
	}
	frozen.OriginalIncidents = append([]d.ProcessInstanceIncidentDetail(nil), incidents...)
	return frozen
}

func buildRepairPlans(request d.OpsRepairRequest, incidents []d.ProcessInstanceIncidentDetail) ([]d.OpsRepairPlanItem, []d.OpsRepairJobApplicability) {
	plans := make([]d.OpsRepairPlanItem, 0, len(incidents))
	applicability := make([]d.OpsRepairJobApplicability, 0, len(incidents))
	for _, incident := range incidents {
		plan, job := newIncidentRepairPlan(request, incident)
		plans = append(plans, plan)
		applicability = append(applicability, job)
	}
	return plans, applicability
}

func (s *Service) executeIncidentRepair(ctx context.Context, request d.OpsRepairRequest, incident d.ProcessInstanceIncidentDetail, opts ...services.CallOption) (repairIncidentExecution, error) {
	plan, jobApplicability := newIncidentRepairPlan(request, incident)
	var errs []error
	if !incidentIsActive(incident) {
		plan.ResolutionStatus = d.OpsWorkflowStepStatusSkipped
		plan.ConfirmationStatus = d.OpsWorkflowStepStatusSkipped
		plan.Notices = append(plan.Notices, d.OpsRepairWorkflowNotice{
			Code:     "incident_not_active",
			Severity: "info",
			Message:  "incident is not active; resolution was skipped",
			Details:  map[string]string{"incidentKey": incident.IncidentKey, "state": incident.State},
		})
		return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, nil
	}
	if incident.JobKey != "" {
		if request.RequestedRetries == nil || *request.RequestedRetries > 0 {
			status, errText, err := s.updateRepairJobRetries(ctx, request, incident.JobKey, opts...)
			plan.RetryUpdateStatus = status
			jobApplicability.RetryStatus = status
			if err != nil {
				plan.Errors = append(plan.Errors, errText)
				errs = append(errs, err)
			}
		}
		if request.RequestedJobTimeout > 0 {
			status, errText, err := s.updateRepairJobTimeout(ctx, request, incident.JobKey, opts...)
			plan.TimeoutUpdateStatus = status
			jobApplicability.TimeoutStatus = status
			if err != nil {
				plan.Errors = append(plan.Errors, errText)
				errs = append(errs, err)
			}
		}
	}

	resp, err := s.incAPI.ResolveIncident(ctx, incident.IncidentKey, opts...)
	if err != nil {
		plan.ResolutionStatus = d.OpsWorkflowStepStatusFailed
		plan.ConfirmationStatus = d.OpsWorkflowStepStatusSkipped
		plan.Errors = append(plan.Errors, err.Error())
		errs = append(errs, err)
		return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, errorsJoin(errs...)
	}
	if !resp.Ok {
		err := fmt.Errorf("%w: incident %s resolution was not accepted: %s", d.ErrUpstream, incident.IncidentKey, resp.Status)
		plan.ResolutionStatus = d.OpsWorkflowStepStatusFailed
		plan.ConfirmationStatus = d.OpsWorkflowStepStatusSkipped
		plan.Errors = append(plan.Errors, err.Error())
		errs = append(errs, err)
		return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, errorsJoin(errs...)
	}
	plan.ResolutionStatus = d.OpsWorkflowStepStatusSubmitted
	if request.NoWait {
		plan.ConfirmationStatus = d.OpsWorkflowStepStatusSkipped
		return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, errorsJoin(errs...)
	}
	if _, err := s.incAPI.WaitForIncidentResolved(ctx, incident.IncidentKey, opts...); err != nil {
		plan.ConfirmationStatus = d.OpsWorkflowStepStatusConfirmationFailed
		plan.Errors = append(plan.Errors, err.Error())
		errs = append(errs, err)
		return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, errorsJoin(errs...)
	}
	plan.ConfirmationStatus = d.OpsWorkflowStepStatusConfirmed
	return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, errorsJoin(errs...)
}

func newIncidentRepairPlan(request d.OpsRepairRequest, incident d.ProcessInstanceIncidentDetail) (d.OpsRepairPlanItem, d.OpsRepairJobApplicability) {
	retries := requestedRepairRetries(request)
	timeout := requestedRepairTimeout(request)
	plan := d.OpsRepairPlanItem{
		IncidentKey:            incident.IncidentKey,
		ProcessInstanceKey:     incident.ProcessInstanceKey,
		RootProcessInstanceKey: incident.RootProcessInstanceKey,
		JobKey:                 incident.JobKey,
		VariableScopeKey:       incident.ProcessInstanceKey,
		RequestedRetries:       retries,
		RequestedTimeout:       timeout,
		VariableUpdateStatus:   d.OpsWorkflowStepStatusSkipped,
		RetryUpdateStatus:      d.OpsWorkflowStepStatusPlanned,
		TimeoutUpdateStatus:    d.OpsWorkflowStepStatusSkipped,
		ResolutionStatus:       d.OpsWorkflowStepStatusPlanned,
		ConfirmationStatus:     d.OpsWorkflowStepStatusSkipped,
	}
	job := d.OpsRepairJobApplicability{
		IncidentKey:      incident.IncidentKey,
		JobKey:           incident.JobKey,
		RetryStatus:      d.OpsWorkflowStepStatusPlanned,
		TimeoutStatus:    d.OpsWorkflowStepStatusSkipped,
		RequestedRetries: retries,
		RequestedTimeout: timeout,
	}
	if len(request.Variables) > 0 {
		plan.RequestedVariableNames = sortedMapKeys(request.Variables)
		plan.VariableUpdateStatus = d.OpsWorkflowStepStatusPlanned
	}
	if retries != nil && *retries == 0 {
		plan.RetryUpdateStatus = d.OpsWorkflowStepStatusSkipped
		job.RetryStatus = d.OpsWorkflowStepStatusSkipped
		job.Reason = "retry update skipped because requested retries is 0"
	}
	if timeout != "" {
		plan.TimeoutUpdateStatus = d.OpsWorkflowStepStatusPlanned
		job.TimeoutStatus = d.OpsWorkflowStepStatusPlanned
	}
	if incident.JobKey == "" {
		plan.RetryUpdateStatus = d.OpsWorkflowStepStatusNotApplicable
		plan.TimeoutUpdateStatus = d.OpsWorkflowStepStatusNotApplicable
		job.RetryStatus = d.OpsWorkflowStepStatusNotApplicable
		job.TimeoutStatus = d.OpsWorkflowStepStatusNotApplicable
		job.Reason = "incident has no related job"
		notice := d.OpsRepairWorkflowNotice{
			Code:     "incident_has_no_related_job",
			Severity: "info",
			Message:  "job repair steps do not apply because the incident has no related job",
			Details:  map[string]string{"incidentKey": incident.IncidentKey},
		}
		plan.Notices = append(plan.Notices, notice)
	}
	return plan, job
}

func requestedRepairRetries(request d.OpsRepairRequest) *int32 {
	if request.RequestedRetries != nil {
		return toolx.CopyPtr(request.RequestedRetries)
	}
	retries := int32(1)
	return &retries
}

func requestedRepairTimeout(request d.OpsRepairRequest) string {
	if request.RequestedJobTimeout <= 0 {
		return ""
	}
	return request.RequestedJobTimeout.String()
}

func (s *Service) updateRepairJobRetries(ctx context.Context, request d.OpsRepairRequest, jobKey string, opts ...services.CallOption) (d.OpsWorkflowStepStatus, string, error) {
	retries := requestedRepairRetries(request)
	result, err := s.jobAPI.UpdateJob(ctx, d.JobUpdateRequest{
		Key:              jobKey,
		Retries:          retries,
		ConfirmRetries:   !request.NoWait,
		SkipConfirmation: request.NoWait,
	}, opts...)
	if err != nil {
		return d.OpsWorkflowStepStatusFailed, err.Error(), err
	}
	if result.ConfirmationError != "" {
		return d.OpsWorkflowStepStatusConfirmationFailed, result.ConfirmationError, fmt.Errorf("%w: %s", d.ErrUpstream, result.ConfirmationError)
	}
	if request.NoWait || result.ConfirmationStatus == "skipped" {
		return d.OpsWorkflowStepStatusSubmitted, "", nil
	}
	return d.OpsWorkflowStepStatusConfirmed, "", nil
}

func (s *Service) updateRepairJobTimeout(ctx context.Context, request d.OpsRepairRequest, jobKey string, opts ...services.CallOption) (d.OpsWorkflowStepStatus, string, error) {
	timeoutMillis := request.RequestedJobTimeout.Milliseconds()
	result, err := s.jobAPI.UpdateJob(ctx, d.JobUpdateRequest{
		Key:               jobKey,
		TimeoutMillis:     &timeoutMillis,
		RequestedTimeout:  request.RequestedJobTimeout.String(),
		RequestedDuration: request.RequestedJobTimeout,
		SkipConfirmation:  true,
	}, opts...)
	if err != nil {
		return d.OpsWorkflowStepStatusFailed, err.Error(), err
	}
	if result.MutationError != "" {
		return d.OpsWorkflowStepStatusFailed, result.MutationError, fmt.Errorf("%w: %s", d.ErrUpstream, result.MutationError)
	}
	return d.OpsWorkflowStepStatusSubmitted, "", nil
}

func repairOutcomeForPlans(plans []d.OpsRepairPlanItem, err error) d.OpsRepairOutcome {
	if err == nil {
		return d.OpsRepairOutcomeRepaired
	}
	for _, plan := range plans {
		if plan.ConfirmationStatus == d.OpsWorkflowStepStatusConfirmed || plan.ResolutionStatus == d.OpsWorkflowStepStatusSubmitted {
			return d.OpsRepairOutcomePartiallyFailed
		}
	}
	return d.OpsRepairOutcomeFailed
}

func incidentKeysFromDetails(incidents []d.ProcessInstanceIncidentDetail) typex.Keys {
	keys := make(typex.Keys, 0, len(incidents))
	for _, incident := range incidents {
		keys = append(keys, incident.IncidentKey)
	}
	return keys.Unique()
}

func processInstanceKeysFromIncidents(incidents []d.ProcessInstanceIncidentDetail) typex.Keys {
	keys := make(typex.Keys, 0, len(incidents))
	for _, incident := range incidents {
		if incident.ProcessInstanceKey != "" {
			keys = append(keys, incident.ProcessInstanceKey)
		}
	}
	return keys.Unique()
}

func rootProcessInstanceKeysFromIncidents(incidents []d.ProcessInstanceIncidentDetail) typex.Keys {
	keys := make(typex.Keys, 0, len(incidents))
	for _, incident := range incidents {
		if incident.RootProcessInstanceKey != "" {
			keys = append(keys, incident.RootProcessInstanceKey)
		}
	}
	return keys.Unique()
}

func jobKeysFromIncidents(incidents []d.ProcessInstanceIncidentDetail) typex.Keys {
	keys := make(typex.Keys, 0, len(incidents))
	for _, incident := range incidents {
		if incident.JobKey != "" {
			keys = append(keys, incident.JobKey)
		}
	}
	return keys.Unique()
}

func incidentIsActive(incident d.ProcessInstanceIncidentDetail) bool {
	return strings.EqualFold(incident.State, "active")
}

func sortedMapKeys(values map[string]any) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func errorsJoin(errs ...error) error {
	return errors.Join(errs...)
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
