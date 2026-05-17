// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/typex"
)

const incidentPurgeReportSchemaVersion = "ops.process-instances-with-incidents.v1"

// PurgeProcessInstancesWithIncidents prepares the incident-purge workflow result shape.
func (s *Service) PurgeProcessInstancesWithIncidents(ctx context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error) {
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

	discovery, err := incidentPurgeDiscovery(ctx, s.incAPI, request, opts...)
	if err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.Filters = request.Selection
		result.Discovery.Errors = []string{err.Error()}
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomeFailed, err)
	}

	result.Discovery = discovery
	result.Notices = append(result.Notices, discovery.Notices...)

	if len(discovery.CandidateProcessInstanceKeys) == 0 {
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomePlanned, nil)
	}

	plan, err := buildIncidentPurgeDeletePlan(ctx, s.piAPI, discovery, request.Workers, !request.DryRun, opts...)
	result.DeletePlan = plan
	if err != nil {
		result.DeletePlan.Status = d.OpsWorkflowStepStatusFailed
		result.DeletePlan.Errors = []string{err.Error()}
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomeFailed, fmt.Errorf("incident purge delete-plan validation: %w", err))
	}

	if !request.DryRun && !request.Force && len(plan.NonFinalAffectedItems) > 0 {
		err = fmt.Errorf("%w: refusing to delete incident purge process-instance scope: %s; no delete request was submitted; use --force to cancel the non-final affected scope before delete", d.ErrPrecondition, formatIncidentPurgeBlockedScope(plan))
		result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
		result.Deletion.Errors = []string{err.Error()}
		return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomeFailed, err)
	}

	if request.DryRun || len(plan.ResolvedRootKeys) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishIncidentPurgeResult(result, d.IncidentPurgeOutcomePlanned, nil)
	}

	reports, err := pisvc.DeleteProcessInstances(ctx, s.piAPI, s.log, plan.ResolvedRootKeys, request.Workers, len(plan.AffectedKeys), opts...)
	result.Deletion = d.IncidentPurgeDeletionResult{
		Status:            deletionStatusForReports(reports, request.NoWait, err),
		SubmittedRootKeys: plan.ResolvedRootKeys,
		Items:             reports,
		Submitted:         len(reports) > 0,
		Confirmed:         err == nil && !request.NoWait && allReportsOK(reports),
		NoWait:            request.NoWait,
		Errors:            deletionErrors(err),
	}
	if err != nil {
		return finishIncidentPurgeResult(result, incidentPurgeDeletionOutcomeForReports(reports), fmt.Errorf("delete incident purge process instances: %w", err))
	}
	return finishIncidentPurgeResult(result, incidentPurgeDeletionOutcomeForReports(reports), nil)
}

// buildIncidentPurgeDeletePlan adapts frozen incident candidates into the shared process-instance delete plan.
func buildIncidentPurgeDeletePlan(ctx context.Context, api pisvc.API, discovery d.IncidentDiscoveryResult, wantedWorkers int, requiresConfirmation bool, opts ...services.CallOption) (d.IncidentPurgeDeletePlan, error) {
	candidates := discovery.CandidateProcessInstanceKeys.Unique()
	preview, err := pisvc.DryRunCancelOrDeletePlan(ctx, api, candidates, wantedWorkers, opts...)
	plan := d.IncidentPurgeDeletePlan{
		Status:                                d.OpsWorkflowStepStatusPlanned,
		CandidateProcessInstanceKeys:          candidates,
		ResolvedRootKeys:                      preview.Roots,
		AffectedKeys:                          preview.Collected,
		DuplicateCandidateProcessInstanceKeys: discovery.DuplicateCandidateProcessInstanceKeys.Unique(),
		DuplicateResolvedRootKeys:             preview.DuplicateRoots,
		FinalStateItems:                       preview.SelectedFinalState,
		NonFinalAffectedItems:                 preview.RequiresCancelBeforeDelete,
		MissingAncestors:                      preview.MissingAncestors,
		RequiresConfirmation:                  requiresConfirmation && len(preview.Roots) > 0,
	}
	if preview.Warning != "" {
		plan.TraversalWarnings = []string{preview.Warning}
	}
	return plan, err
}

func formatIncidentPurgeBlockedScope(plan d.IncidentPurgeDeletePlan) string {
	if len(plan.NonFinalAffectedItems) == 0 {
		return "no non-final affected process instances"
	}
	return fmt.Sprintf("%d affected process instance(s) are not in a final state", len(plan.NonFinalAffectedItems))
}

// incidentPurgeDeletionOutcomeForReports classifies delete reports into the incident-purge outcome vocabulary.
func incidentPurgeDeletionOutcomeForReports(reports []d.Reporter) d.IncidentPurgeOutcome {
	if len(reports) == 0 {
		return d.IncidentPurgeOutcomeFailed
	}
	ok := 0
	for _, report := range reports {
		if report.Ok {
			ok++
		}
	}
	switch ok {
	case len(reports):
		return d.IncidentPurgeOutcomeDeleted
	case 0:
		return d.IncidentPurgeOutcomeFailed
	default:
		return d.IncidentPurgeOutcomePartiallyFailed
	}
}

// incidentPurgeDiscovery either reuses a frozen candidate set or performs a single incident search.
func incidentPurgeDiscovery(ctx context.Context, api incsvc.API, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentDiscoveryResult, error) {
	if request.DiscoveredCandidateProcessInstanceKeys != nil {
		return frozenIncidentPurgeDiscovery(request), nil
	}
	return discoverIncidentPurgeCandidates(ctx, api, request, opts...)
}

// frozenIncidentPurgeDiscovery reconstructs enough discovery state to execute a previously confirmed plan without expanding scope.
func frozenIncidentPurgeDiscovery(request d.IncidentPurgeRequest) d.IncidentDiscoveryResult {
	candidates := request.DiscoveredCandidateProcessInstanceKeys.Unique()
	return d.IncidentDiscoveryResult{
		Status:                        d.OpsWorkflowStepStatusPlanned,
		Filters:                       request.Selection,
		CandidateProcessInstanceKeys:  candidates,
		CandidateProcessInstanceCount: len(candidates),
	}
}

// discoverIncidentPurgeCandidates runs incident search once and freezes the process-instance candidate set.
func discoverIncidentPurgeCandidates(ctx context.Context, api incsvc.API, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentDiscoveryResult, error) {
	incidents, err := incsvc.SearchIncidents(ctx, api, request.Selection, incidentPurgeDiscoverySize(request), opts...)
	if err != nil {
		return d.IncidentDiscoveryResult{}, err
	}
	incidents = limitIncidentPurgeCandidateIncidents(incidents, request.Limit)

	discovery := d.IncidentDiscoveryResult{
		Status:             d.OpsWorkflowStepStatusPlanned,
		Filters:            request.Selection,
		CandidateIncidents: append([]d.ProcessInstanceIncidentDetail(nil), incidents...),
		IncidentCount:      len(incidents),
	}
	seenProcessInstances := make(map[string]int, len(incidents))
	var duplicateCandidates typex.Keys
	for _, incident := range incidents {
		if incident.IncidentKey != "" {
			discovery.IncidentKeys = append(discovery.IncidentKeys, incident.IncidentKey)
		}
		if incident.ProcessInstanceKey == "" {
			discovery.SkippedIncidents = append(discovery.SkippedIncidents, d.IncidentPurgeSkippedIncident{
				Incident: incident,
				Reason:   "missing process-instance key",
			})
			continue
		}
		seenProcessInstances[incident.ProcessInstanceKey]++
		if seenProcessInstances[incident.ProcessInstanceKey] == 1 {
			discovery.CandidateProcessInstanceKeys = append(discovery.CandidateProcessInstanceKeys, incident.ProcessInstanceKey)
			continue
		}
		duplicateCandidates = append(duplicateCandidates, incident.ProcessInstanceKey)
	}
	discovery.CandidateProcessInstanceKeys = discovery.CandidateProcessInstanceKeys.Unique()
	discovery.DuplicateCandidateProcessInstanceKeys = duplicateCandidates.Unique()
	discovery.CandidateProcessInstanceCount = len(discovery.CandidateProcessInstanceKeys)
	discovery.Notices = incidentPurgeDiscoveryNotices(discovery)
	return discovery, nil
}

// incidentPurgeDiscoveryNotices records semantic discovery facts for reports without inflating compact output.
func incidentPurgeDiscoveryNotices(discovery d.IncidentDiscoveryResult) []d.IncidentPurgeWorkflowNotice {
	var notices []d.IncidentPurgeWorkflowNotice
	if discovery.IncidentCount == 0 {
		notices = append(notices, d.IncidentPurgeWorkflowNotice{
			Code:     "no_candidate_incidents",
			Severity: "info",
			Message:  "no matching candidate incidents found",
		})
	}
	if len(discovery.DuplicateCandidateProcessInstanceKeys) > 0 {
		notices = append(notices, d.IncidentPurgeWorkflowNotice{
			Code:     "duplicate_candidate_process_instances",
			Severity: "info",
			Message:  "duplicate candidate process instances detected",
			Details: map[string]string{
				"count": fmt.Sprintf("%d", len(discovery.DuplicateCandidateProcessInstanceKeys)),
			},
		})
	}
	if len(discovery.SkippedIncidents) > 0 {
		notices = append(notices, d.IncidentPurgeWorkflowNotice{
			Code:     "skipped_incidents",
			Severity: "warning",
			Message:  "some candidate incidents could not produce process-instance keys",
			Details: map[string]string{
				"count": fmt.Sprintf("%d", len(discovery.SkippedIncidents)),
			},
		})
	}
	return notices
}

// incidentPurgeDiscoverySize normalizes the search size while allowing --limit to cap candidate incidents first.
func incidentPurgeDiscoverySize(request d.IncidentPurgeRequest) int32 {
	if request.Limit > 0 {
		return request.Limit
	}
	if request.BatchSize > 0 && request.BatchSize <= consts.MaxPISearchSize {
		return request.BatchSize
	}
	return consts.MaxPISearchSize
}

// limitIncidentPurgeCandidateIncidents protects candidate extraction even when a stub or backend over-returns.
func limitIncidentPurgeCandidateIncidents(items []d.ProcessInstanceIncidentDetail, limit int32) []d.ProcessInstanceIncidentDetail {
	if limit <= 0 || len(items) <= int(limit) {
		return items
	}
	return items[:limit]
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
