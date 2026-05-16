// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/typex"
)

const allProcessDefinitionsPurgeReportSchemaVersion = "ops.all-process-definitions.v1"

// PurgeAllProcessDefinitions prepares the all-process-definitions purge workflow result shape.
func (s *Service) PurgeAllProcessDefinitions(ctx context.Context, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error) {
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

	discovery, err := allProcessDefinitionsPurgeDiscovery(ctx, s.pdAPI, request, opts...)
	if err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.Filters = request.Selection
		result.Discovery.LatestOnly = request.Selection.IsLatestVersion
		result.Discovery.Errors = []string{err.Error()}
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomeFailed, err)
	}
	result.Discovery = discovery
	result.Notices = append(result.Notices, discovery.Notices...)

	if len(discovery.CandidateProcessDefinitionKeys) == 0 {
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomePlanned, nil)
	}

	plan, err := buildAllProcessDefinitionsPurgeDeletePlan(ctx, s.pdAPI, s.piAPI, s.log, discovery, !request.DryRun, request.Force, opts...)
	result.DeletePlan = plan
	if err != nil {
		result.DeletePlan.Status = d.OpsWorkflowStepStatusFailed
		result.DeletePlan.Errors = []string{err.Error()}
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomeFailed, fmt.Errorf("all-process-definitions purge delete-plan validation: %w", err))
	}

	if !request.DryRun && plan.RequiresForce {
		err = fmt.Errorf("%w: refusing to delete all-process-definitions purge scope: %d active process instance(s) are affected; no delete request was submitted; use --force to cancel active process instances before delete", d.ErrPrecondition, plan.ActiveProcessInstanceCount)
		result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
		result.Deletion.Errors = []string{err.Error()}
		return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomeFailed, err)
	}

	result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
	return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomePlanned, nil)
}

// buildAllProcessDefinitionsPurgeDeletePlan adapts frozen process-definition candidates into the shared delete-pd preflight.
func buildAllProcessDefinitionsPurgeDeletePlan(ctx context.Context, pdAPI pdsvc.API, piAPI pisvc.API, log *slog.Logger, discovery d.ProcessDefinitionDiscoveryResult, requiresConfirmation bool, force bool, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeDeletePlan, error) {
	candidates := discovery.CandidateProcessDefinitionKeys.Unique()
	preview, err := rsvc.PreviewDeleteProcessDefinitions(ctx, pdAPI, piAPI, log, candidates, opts...)
	plan := d.AllProcessDefinitionsPurgeDeletePlan{
		Status:                                  d.OpsWorkflowStepStatusPlanned,
		CandidateProcessDefinitionKeys:          candidates,
		Items:                                   append([]d.DeleteProcessDefinitionPlanItem(nil), preview.Items...),
		DuplicateCandidateProcessDefinitionKeys: discovery.DuplicateCandidateProcessDefinitionKeys.Unique(),
		RequiresConfirmation:                    requiresConfirmation && len(candidates) > 0,
	}
	plan.ActiveProcessInstanceCount = activeProcessInstanceCountForProcessDefinitionPlan(preview.Items)
	plan.AffectedProcessInstanceCount = affectedProcessInstanceCountForProcessDefinitionPlan(preview.Items)
	plan.RequiresForce = !force && plan.ActiveProcessInstanceCount > 0
	return plan, err
}

func activeProcessInstanceCountForProcessDefinitionPlan(items []d.DeleteProcessDefinitionPlanItem) int64 {
	var total int64
	for _, item := range items {
		total += item.ActiveProcessInstances()
	}
	return total
}

func affectedProcessInstanceCountForProcessDefinitionPlan(items []d.DeleteProcessDefinitionPlanItem) int64 {
	var total int64
	for _, item := range items {
		affected := int64(len(item.CancellationPlan.Collected.Unique()))
		if affected < item.ActiveProcessInstances() {
			affected = item.ActiveProcessInstances()
		}
		total += affected
	}
	return total
}

// allProcessDefinitionsPurgeDiscovery either reuses a frozen candidate set or performs one process-definition lookup.
func allProcessDefinitionsPurgeDiscovery(ctx context.Context, api pdsvc.API, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.ProcessDefinitionDiscoveryResult, error) {
	if request.DiscoveredCandidateProcessDefinitionKeys != nil {
		return frozenAllProcessDefinitionsPurgeDiscovery(request), nil
	}
	return discoverAllProcessDefinitionsPurgeCandidates(ctx, api, request, opts...)
}

// discoverAllProcessDefinitionsPurgeCandidates reuses get-pd search behavior and freezes unique process-definition keys.
func discoverAllProcessDefinitionsPurgeCandidates(ctx context.Context, api pdsvc.API, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.ProcessDefinitionDiscoveryResult, error) {
	definitions, err := searchAllProcessDefinitionsPurgeCandidates(ctx, api, request.Selection, opts...)
	if err != nil {
		return d.ProcessDefinitionDiscoveryResult{}, err
	}
	discovery := d.ProcessDefinitionDiscoveryResult{
		Status:     d.OpsWorkflowStepStatusPlanned,
		Filters:    request.Selection,
		LatestOnly: request.Selection.IsLatestVersion,
	}
	seenDefinitions := make(map[string]int, len(definitions))
	var duplicateCandidates typex.Keys
	for _, definition := range definitions {
		if definition.Key == "" {
			continue
		}
		seenDefinitions[definition.Key]++
		if seenDefinitions[definition.Key] == 1 {
			discovery.CandidateProcessDefinitionKeys = append(discovery.CandidateProcessDefinitionKeys, definition.Key)
			discovery.CandidateProcessDefinitions = append(discovery.CandidateProcessDefinitions, definition)
			continue
		}
		duplicateCandidates = append(duplicateCandidates, definition.Key)
	}
	discovery.CandidateProcessDefinitionKeys = discovery.CandidateProcessDefinitionKeys.Unique()
	discovery.DuplicateCandidateProcessDefinitionKeys = duplicateCandidates.Unique()
	discovery.CandidateProcessDefinitionCount = len(discovery.CandidateProcessDefinitionKeys)
	discovery.Notices = allProcessDefinitionsPurgeDiscoveryNotices(discovery)
	return discovery, nil
}

// searchAllProcessDefinitionsPurgeCandidates mirrors get-pd key/latest/all-version branching.
func searchAllProcessDefinitionsPurgeCandidates(ctx context.Context, api pdsvc.API, selection d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if selection.Key != "" {
		definition, err := api.GetProcessDefinition(ctx, selection.Key, opts...)
		if err != nil {
			return nil, err
		}
		return []d.ProcessDefinition{definition}, nil
	}
	if selection.IsLatestVersion {
		return api.SearchProcessDefinitionsLatest(ctx, selection, opts...)
	}
	return api.SearchProcessDefinitions(ctx, selection, pdsvc.MaxResultSize, opts...)
}

func frozenAllProcessDefinitionsPurgeDiscovery(request d.AllProcessDefinitionsPurgeRequest) d.ProcessDefinitionDiscoveryResult {
	candidates := request.DiscoveredCandidateProcessDefinitionKeys.Unique()
	discovery := d.ProcessDefinitionDiscoveryResult{
		Status:                          d.OpsWorkflowStepStatusPlanned,
		Filters:                         request.Selection,
		CandidateProcessDefinitionKeys:  candidates,
		CandidateProcessDefinitionCount: len(candidates),
		LatestOnly:                      request.Selection.IsLatestVersion,
	}
	discovery.Notices = allProcessDefinitionsPurgeDiscoveryNotices(discovery)
	return discovery
}

// allProcessDefinitionsPurgeDiscoveryNotices records semantic discovery facts for reports and machine output.
func allProcessDefinitionsPurgeDiscoveryNotices(discovery d.ProcessDefinitionDiscoveryResult) []d.AllProcessDefinitionsPurgeWorkflowNotice {
	var notices []d.AllProcessDefinitionsPurgeWorkflowNotice
	if discovery.CandidateProcessDefinitionCount == 0 {
		notices = append(notices, d.AllProcessDefinitionsPurgeWorkflowNotice{
			Code:     "no_candidate_process_definitions",
			Severity: "info",
			Message:  "no matching candidate process definitions found",
		})
	}
	if discovery.LatestOnly {
		notices = append(notices, d.AllProcessDefinitionsPurgeWorkflowNotice{
			Code:     "latest_only_scope",
			Severity: "info",
			Message:  "candidate discovery was narrowed to latest matching process definitions",
		})
	}
	if len(discovery.DuplicateCandidateProcessDefinitionKeys) > 0 {
		notices = append(notices, d.AllProcessDefinitionsPurgeWorkflowNotice{
			Code:     "duplicate_candidate_process_definitions",
			Severity: "info",
			Message:  "duplicate candidate process-definition keys detected",
			Details: map[string]string{
				"count": fmt.Sprintf("%d", len(discovery.DuplicateCandidateProcessDefinitionKeys)),
			},
		})
	}
	return notices
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
