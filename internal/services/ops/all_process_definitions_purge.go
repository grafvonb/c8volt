// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
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
	if err := validateAllProcessDefinitionsPurgeSupportedVersion(s.version); err != nil {
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

	plan, err := buildAllProcessDefinitionsPurgeDeletePlan(ctx, s.pdAPI, s.piAPI, s.log, discovery, request.Workers, !request.DryRun, request.Force, opts...)
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

	if request.DryRun || len(plan.CandidateProcessDefinitionKeys) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishAllProcessDefinitionsPurgeResult(result, d.AllProcessDefinitionsPurgeOutcomePlanned, nil)
	}

	deleteOpts := compactOpsExecutionOptions(opts...)
	if request.Force {
		deleteOpts = append(deleteOpts, services.WithForce())
	}
	if request.NoWait {
		deleteOpts = append(deleteOpts, services.WithNoWait())
	}
	if request.FailFast {
		deleteOpts = append(deleteOpts, services.WithFailFast())
	}
	if request.NoWorkerLimit {
		deleteOpts = append(deleteOpts, services.WithNoWorkerLimit())
	}
	reports, err := pdsvc.DeleteProcessDefinitions(ctx, s.resourceAPI, s.pdAPI, s.piAPI, s.log, plan.CandidateProcessDefinitionKeys, request.Workers, deleteOpts...)
	result.Deletion = d.AllProcessDefinitionsPurgeDeletionResult{
		Status:                         deletionStatusForResourceDeleteResponses(reports, request.NoWait, err),
		SubmittedProcessDefinitionKeys: append(typex.Keys{}, plan.CandidateProcessDefinitionKeys...),
		Items:                          reports,
		Submitted:                      len(reports) > 0,
		Confirmed:                      err == nil && !request.NoWait && allResourceDeleteResponsesOK(reports),
		NoWait:                         request.NoWait,
		Errors:                         deletionErrors(err),
	}
	if err != nil {
		return finishAllProcessDefinitionsPurgeResult(result, allProcessDefinitionsPurgeDeletionOutcomeForResponses(reports), fmt.Errorf("delete all-process-definitions purge process definitions: %w", err))
	}
	return finishAllProcessDefinitionsPurgeResult(result, allProcessDefinitionsPurgeDeletionOutcomeForResponses(reports), nil)
}

// buildAllProcessDefinitionsPurgeDeletePlan adapts frozen process-definition candidates into the shared delete-pd preflight.
func buildAllProcessDefinitionsPurgeDeletePlan(ctx context.Context, pdAPI pdsvc.API, piAPI pisvc.API, log *slog.Logger, discovery d.ProcessDefinitionDiscoveryResult, wantedWorkers int, requiresConfirmation bool, force bool, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeDeletePlan, error) {
	candidates := discovery.CandidateProcessDefinitionKeys.Unique()
	planOpts := append([]services.CallOption{}, opts...)
	if force {
		planOpts = append(planOpts, services.WithForce())
	}
	preview, err := pdsvc.PreviewDeleteProcessDefinitionsWithWorkers(ctx, pdAPI, piAPI, log, candidates, wantedWorkers, planOpts...)
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
	var fallback int64
	var affectedKeys typex.Keys
	for _, item := range items {
		collected := item.CancellationPlan.Collected.Unique()
		if len(collected) == 0 {
			fallback += item.ActiveProcessInstances()
			continue
		}
		affectedKeys = append(affectedKeys, collected...)
	}
	total := int64(len(affectedKeys.Unique())) + fallback
	if active := activeProcessInstanceCountForProcessDefinitionPlan(items); total < active {
		total = active
	}
	return total
}

// deletionStatusForResourceDeleteResponses maps resource delete reports into workflow step status.
func deletionStatusForResourceDeleteResponses(reports []d.ResourceDeleteResponse, noWait bool, err error) d.OpsWorkflowStepStatus {
	if err != nil || !allResourceDeleteResponsesOK(reports) {
		return d.OpsWorkflowStepStatusFailed
	}
	if noWait {
		return d.OpsWorkflowStepStatusSubmitted
	}
	return d.OpsWorkflowStepStatusConfirmed
}

// allProcessDefinitionsPurgeDeletionOutcomeForResponses classifies process-definition delete responses.
func allProcessDefinitionsPurgeDeletionOutcomeForResponses(reports []d.ResourceDeleteResponse) d.AllProcessDefinitionsPurgeOutcome {
	if len(reports) == 0 {
		return d.AllProcessDefinitionsPurgeOutcomeFailed
	}
	ok := 0
	for _, report := range reports {
		if report.Ok {
			ok++
		}
	}
	switch ok {
	case len(reports):
		return d.AllProcessDefinitionsPurgeOutcomeDeleted
	case 0:
		return d.AllProcessDefinitionsPurgeOutcomeFailed
	default:
		return d.AllProcessDefinitionsPurgeOutcomePartiallyFailed
	}
}

// allResourceDeleteResponsesOK reports whether every process-definition delete completed successfully.
func allResourceDeleteResponsesOK(reports []d.ResourceDeleteResponse) bool {
	if len(reports) == 0 {
		return false
	}
	for _, report := range reports {
		if !report.Ok {
			return false
		}
	}
	return true
}

// allProcessDefinitionsPurgeDiscovery either reuses a frozen candidate set or performs paged process-definition discovery.
func allProcessDefinitionsPurgeDiscovery(ctx context.Context, api pdsvc.API, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.ProcessDefinitionDiscoveryResult, error) {
	if request.DiscoveredCandidateProcessDefinitionKeys != nil {
		return frozenAllProcessDefinitionsPurgeDiscovery(request), nil
	}
	return discoverAllProcessDefinitionsPurgeCandidates(ctx, api, request, opts...)
}

// discoverAllProcessDefinitionsPurgeCandidates walks process-definition pages and freezes unique process-definition keys.
func discoverAllProcessDefinitionsPurgeCandidates(ctx context.Context, api pdsvc.API, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.ProcessDefinitionDiscoveryResult, error) {
	definitions, status, err := searchAllProcessDefinitionsPurgeCandidates(ctx, api, request, opts...)
	if err != nil {
		return d.ProcessDefinitionDiscoveryResult{}, err
	}
	discovery := d.ProcessDefinitionDiscoveryResult{
		DiscoveryScopeStatus: status,
		Status:               d.OpsWorkflowStepStatusPlanned,
		Filters:              request.Selection,
		LatestOnly:           request.Selection.IsLatestVersion,
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
	discovery.CandidatesFrozen = discovery.CandidateProcessDefinitionCount
	discovery.Notices = allProcessDefinitionsPurgeDiscoveryNotices(discovery)
	return discovery, nil
}

// searchAllProcessDefinitionsPurgeCandidates mirrors get-pd key/latest/all-version branching with paged search.
func searchAllProcessDefinitionsPurgeCandidates(ctx context.Context, api pdsvc.API, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) ([]d.ProcessDefinition, d.DiscoveryScopeStatus, error) {
	status := d.DiscoveryScopeStatus{
		BatchSize: allProcessDefinitionsPurgeDiscoverySize(request),
		Limit:     request.Limit,
	}
	if request.Selection.Key != "" {
		definition, err := api.GetProcessDefinition(ctx, request.Selection.Key, opts...)
		if err != nil {
			return nil, d.DiscoveryScopeStatus{}, err
		}
		status.Complete = true
		status.CandidatesSeen = 1
		status.CandidatesFrozen = 1
		return []d.ProcessDefinition{definition}, status, nil
	}

	pageReq := d.ProcessDefinitionPageRequest{Size: status.BatchSize}
	var definitions []d.ProcessDefinition
	limited := false
	for {
		page, err := api.SearchProcessDefinitionsPage(ctx, request.Selection, pageReq, opts...)
		if err != nil {
			return nil, d.DiscoveryScopeStatus{}, err
		}
		status.Pages++
		status.CandidatesSeen += len(page.Items)
		items := limitAllProcessDefinitionsPurgePageItems(page.Items, request.Limit, len(definitions))
		definitions = append(definitions, items...)
		if request.Limit > 0 && len(definitions) >= int(request.Limit) {
			limited = true
		}
		reportAllProcessDefinitionsPurgeDiscoveryProgress(request, page, status, len(items), len(definitions), limited)
		if limited || allProcessDefinitionsPurgeDiscoveryPageComplete(page) {
			status.Complete = !limited
			status.Limited = limited
			break
		}
		pageReq = nextAllProcessDefinitionsPurgeDiscoveryPageRequest(pageReq, page)
	}
	return definitions, status, nil
}

func reportAllProcessDefinitionsPurgeDiscoveryProgress(request d.AllProcessDefinitionsPurgeRequest, page d.ProcessDefinitionPage, status d.DiscoveryScopeStatus, currentSelected int, selected int, limited bool) {
	if request.Progress == nil {
		return
	}
	if status.Pages == 1 {
		preflight := newAllProcessDefinitionsPurgePreflight(request, page)
		request.Progress(d.OpsProgressEvent{
			Kind:      d.OpsProgressEventKindPreflight,
			Preflight: &preflight,
		})
	}
	pageCount, pageKind := allProcessDefinitionsPurgePageCount(page.ReportedTotal, status.BatchSize)
	progress := d.OpsPageProgress{
		Phase:            "discovering process definitions",
		CurrentPage:      status.Pages,
		PageCount:        pageCount,
		PageCountKind:    pageKind,
		PageSize:         status.BatchSize,
		CurrentPageCount: currentSelected,
		Seen:             status.CandidatesSeen,
		Selected:         selected,
		OverflowState:    allProcessDefinitionsPurgeOpsOverflowState(page.OverflowState),
		LimitReached:     limited,
	}
	request.Progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindPage,
		Page: &progress,
	})
}

func newAllProcessDefinitionsPurgePreflight(request d.AllProcessDefinitionsPurgeRequest, page d.ProcessDefinitionPage) d.OpsPreflightScope {
	total, totalKind := allProcessDefinitionsPurgeTotal(page.ReportedTotal)
	pageCount, pageKind := allProcessDefinitionsPurgePageCount(page.ReportedTotal, allProcessDefinitionsPurgeDiscoverySize(request))
	return d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         request.CommandName,
		CoreResource:    "process_definition",
		SelectorSummary: "all-process-definitions purge",
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        allProcessDefinitionsPurgeDiscoverySize(request),
		PageCount:       pageCount,
		PageCountKind:   pageKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			WorkSummary: allProcessDefinitionsPurgeWorkSummary(request.DryRun),
			RiskSummary: allProcessDefinitionsPurgeRiskSummary(request.DryRun),
		},
		RequiresConfirmation: !request.DryRun && !request.AutoConfirm && !request.Automation,
	}
}

func allProcessDefinitionsPurgeWorkSummary(dryRun bool) string {
	if dryRun {
		return "process-definition purge dry run will discover matching process definitions and validate delete impact only; no changes will be applied"
	}
	return "process-definition purge will discover matching process definitions, validate delete impact, and delete confirmed definitions"
}

func allProcessDefinitionsPurgeRiskSummary(dryRun bool) string {
	if dryRun {
		return ""
	}
	return "potentially destructive purge"
}

func allProcessDefinitionsPurgeTotal(total *d.ProcessDefinitionReportedTotal) (*int64, d.OpsTotalCertainty) {
	if total == nil {
		return nil, d.OpsTotalCertaintyUnknown
	}
	count := total.Count
	switch total.Kind {
	case d.ProcessDefinitionReportedTotalKindExact:
		return &count, d.OpsTotalCertaintyExact
	case d.ProcessDefinitionReportedTotalKindLowerBound:
		return &count, d.OpsTotalCertaintyLowerBound
	default:
		return &count, d.OpsTotalCertaintyUnknown
	}
}

func allProcessDefinitionsPurgePageCount(total *d.ProcessDefinitionReportedTotal, pageSize int32) (*int64, d.OpsPageCountKind) {
	if total == nil || pageSize <= 0 {
		return nil, d.OpsPageCountKindUnknown
	}
	pages := (total.Count + int64(pageSize) - 1) / int64(pageSize)
	switch total.Kind {
	case d.ProcessDefinitionReportedTotalKindExact:
		return &pages, d.OpsPageCountKindExact
	case d.ProcessDefinitionReportedTotalKindLowerBound:
		return &pages, d.OpsPageCountKindEstimated
	default:
		return nil, d.OpsPageCountKindUnknown
	}
}

func allProcessDefinitionsPurgeOpsOverflowState(state d.ProcessInstanceOverflowState) d.OpsOverflowState {
	switch state {
	case d.ProcessInstanceOverflowStateHasMore:
		return d.OpsOverflowStateHasMore
	case d.ProcessInstanceOverflowStateIndeterminate:
		return d.OpsOverflowStateIndeterminate
	case d.ProcessInstanceOverflowStateNoMore:
		return d.OpsOverflowStateNoMore
	default:
		return d.OpsOverflowStateUnknown
	}
}

func frozenAllProcessDefinitionsPurgeDiscovery(request d.AllProcessDefinitionsPurgeRequest) d.ProcessDefinitionDiscoveryResult {
	candidates := request.DiscoveredCandidateProcessDefinitionKeys.Unique()
	status := request.DiscoveredScopeStatus
	if status.BatchSize == 0 {
		status.BatchSize = allProcessDefinitionsPurgeDiscoverySize(request)
	}
	if status.CandidatesFrozen == 0 {
		status.CandidatesFrozen = len(candidates)
	}
	if !status.Complete && !status.Limited {
		status.Complete = true
	}
	discovery := d.ProcessDefinitionDiscoveryResult{
		DiscoveryScopeStatus:            status,
		Status:                          d.OpsWorkflowStepStatusPlanned,
		Filters:                         request.Selection,
		CandidateProcessDefinitionKeys:  candidates,
		CandidateProcessDefinitionCount: len(candidates),
		LatestOnly:                      request.Selection.IsLatestVersion,
	}
	discovery.Notices = allProcessDefinitionsPurgeDiscoveryNotices(discovery)
	return discovery
}

// allProcessDefinitionsPurgeDiscoverySize normalizes page size; --limit caps total discovery separately.
func allProcessDefinitionsPurgeDiscoverySize(request d.AllProcessDefinitionsPurgeRequest) int32 {
	if request.BatchSize > 0 && request.BatchSize <= consts.MaxPISearchSize {
		return request.BatchSize
	}
	return consts.MaxPISearchSize
}

// limitAllProcessDefinitionsPurgePageItems applies the total APD target cap after page retrieval.
func limitAllProcessDefinitionsPurgePageItems(items []d.ProcessDefinition, limit int32, cumulative int) []d.ProcessDefinition {
	if limit <= 0 {
		return items
	}
	remaining := int(limit) - cumulative
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

// allProcessDefinitionsPurgeDiscoveryPageComplete stops on backend completion or contradictory empty pages.
func allProcessDefinitionsPurgeDiscoveryPageComplete(page d.ProcessDefinitionPage) bool {
	return len(page.Items) == 0 || page.OverflowState != d.ProcessInstanceOverflowStateHasMore
}

// nextAllProcessDefinitionsPurgeDiscoveryPageRequest advances by cursor when available, otherwise by returned items.
func nextAllProcessDefinitionsPurgeDiscoveryPageRequest(current d.ProcessDefinitionPageRequest, page d.ProcessDefinitionPage) d.ProcessDefinitionPageRequest {
	next := d.ProcessDefinitionPageRequest{From: current.From + int32(len(page.Items)), Size: current.Size}
	if page.EndCursor != "" {
		next.After = page.EndCursor
		return next
	}
	return next
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
	if request.Progress == nil {
		request.Progress = cfg.Progress
	}
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

func validateAllProcessDefinitionsPurgeSupportedVersion(version toolx.CamundaVersion) error {
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	if version != toolx.V89 {
		return fmt.Errorf("%w: all-process-definitions purge requires Camunda 8.9 or newer; configured Camunda version is %s", d.ErrUnsupported, version.String())
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
