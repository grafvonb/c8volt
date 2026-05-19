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

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
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
	if request.DiscoveryMode == d.OpsRepairDiscoveryModeSearch {
		return s.repairFilteredIncidents(ctx, request, opts...)
	}
	return s.repairExplicitIncidents(ctx, request, opts...)
}

// RepairProcessInstances initializes the process-instance repair workflow boundary.
func (s *Service) RepairProcessInstances(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	request.Target = d.OpsRepairTargetProcessInstance
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
	if request.DiscoveryMode == d.OpsRepairDiscoveryModeSearch {
		return s.repairFilteredProcessInstances(ctx, request, opts...)
	}
	return s.repairExplicitProcessInstances(ctx, request, opts...)
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
		return s.finishDryRunIncidentRepair(request, result, incidents)
	}

	var varErr error
	result.VariableUpdates, varErr = s.executeRepairVariableUpdates(ctx, request, incidents, opts...)
	variableUpdates := variableUpdatesByScope(result.VariableUpdates)
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(incidents), request.Workers, cfg.NoWorkerLimit)
	items, runErr := pool.ExecuteSlice(ctx, incidents, workers, cfg.FailFast, func(ctx context.Context, incident d.ProcessInstanceIncidentDetail, _ int) (repairIncidentExecution, error) {
		return s.executeIncidentRepair(ctx, request, incident, variableUpdates, opts...)
	})
	for _, item := range items {
		if item.Plan.IncidentKey == "" {
			continue
		}
		result.Plan = append(result.Plan, item.Plan)
		result.JobApplicability = append(result.JobApplicability, item.JobApplicability)
	}
	result.Remaining = d.OpsRepairRemainingIncidentSummary{Status: d.OpsWorkflowStepStatusConfirmed, Checked: !request.NoWait}
	repairErr := errorsJoin(varErr, runErr)
	outcome := repairOutcomeForPlans(result.Plan, repairErr)
	return finishRepairResult(result, s.version, outcome, repairErr)
}

// repairFilteredIncidents discovers matching incidents once, freezes them, and then reuses the shared incident repair execution path.
func (s *Service) repairFilteredIncidents(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	result := newRepairResult(request)
	discoverySize := repairIncidentDiscoverySize(request)
	incidents, err := incsvc.SearchIncidents(ctx, s.incAPI, request.IncidentSelection, discoverySize, opts...)
	if err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	incidents = limitRepairIncidents(incidents, request.Limit)
	result.FrozenSet = freezeIncidentSearchSet(request, incidents)
	result.Notices = append(result.Notices, repairBoundedSearchNotice(request.Limit, discoverySize, len(incidents), "incidents")...)
	if len(incidents) == 0 {
		result.Remaining.Status = d.OpsWorkflowStepStatusSkipped
		result.Notices = append(result.Notices, d.OpsRepairWorkflowNotice{
			Code:     "no_matching_incidents",
			Severity: "info",
			Message:  "no matching incidents found for repair",
		})
		return finishRepairResult(result, s.version, d.OpsRepairOutcomePlanned, nil)
	}
	if request.DryRun {
		return s.finishDryRunIncidentRepair(request, result, incidents)
	}

	var varErr error
	result.VariableUpdates, varErr = s.executeRepairVariableUpdates(ctx, request, incidents, opts...)
	variableUpdates := variableUpdatesByScope(result.VariableUpdates)
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(incidents), request.Workers, cfg.NoWorkerLimit)
	items, runErr := pool.ExecuteSlice(ctx, incidents, workers, cfg.FailFast, func(ctx context.Context, incident d.ProcessInstanceIncidentDetail, _ int) (repairIncidentExecution, error) {
		return s.executeIncidentRepair(ctx, request, incident, variableUpdates, opts...)
	})
	for _, item := range items {
		if item.Plan.IncidentKey == "" {
			continue
		}
		result.Plan = append(result.Plan, item.Plan)
		result.JobApplicability = append(result.JobApplicability, item.JobApplicability)
	}
	result.Remaining = d.OpsRepairRemainingIncidentSummary{Status: d.OpsWorkflowStepStatusConfirmed, Checked: !request.NoWait}
	repairErr := errorsJoin(varErr, runErr)
	outcome := repairOutcomeForPlans(result.Plan, repairErr)
	return finishRepairResult(result, s.version, outcome, repairErr)
}

// repairExplicitProcessInstances discovers active incidents for the frozen explicit process-instance key set.
func (s *Service) repairExplicitProcessInstances(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	if request.DiscoveryMode == "" {
		request.DiscoveryMode = d.OpsRepairDiscoveryModeKeyed
	}
	result := newRepairResult(request)
	keys := request.InputKeys.Unique()
	if len(keys) == 0 {
		err := fmt.Errorf("%w: no process-instance keys provided for repair", d.ErrValidation)
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	pis, err := s.piAPI.GetProcessInstances(ctx, keys, request.Workers, opts...)
	if err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	processInstanceKeys := processInstanceKeysFromDetails(pis)
	incidents, err := s.discoverProcessInstanceRepairIncidents(ctx, request, processInstanceKeys, opts...)
	if err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	result.FrozenSet = freezeProcessInstanceRepairSet(request, processInstanceKeys, incidents)
	return s.finishProcessInstanceIncidentRepair(ctx, request, result, incidents, opts...)
}

// repairFilteredProcessInstances searches incident-bearing process instances before freezing and repairing their active incidents.
func (s *Service) repairFilteredProcessInstances(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	result := newRepairResult(request)
	discoverySize := repairProcessInstanceDiscoverySize(request)
	pis, err := s.piAPI.SearchForProcessInstances(ctx, request.ProcessInstanceSelection, discoverySize, opts...)
	if err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	processInstanceKeys := processInstanceKeysFromDetails(limitRepairProcessInstances(pis, request.Limit))
	incidents, err := s.discoverProcessInstanceRepairIncidents(ctx, request, processInstanceKeys, opts...)
	if err != nil {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	if request.DirectIncidentsOnly {
		processInstanceKeys = processInstanceKeysFromIncidents(incidents)
	}
	result.FrozenSet = freezeProcessInstanceRepairSet(request, processInstanceKeys, incidents)
	result.FrozenSet.DiscoveryMode = d.OpsRepairDiscoveryModeSearch
	result.FrozenSet.InputKeys = nil
	result.FrozenSet.ProcessFilters = request.ProcessInstanceSelection
	result.Notices = append(result.Notices, repairBoundedSearchNotice(request.Limit, discoverySize, len(pis), "pi")...)
	return s.finishProcessInstanceIncidentRepair(ctx, request, result, incidents, opts...)
}

// finishProcessInstanceIncidentRepair routes process-instance selected incidents through the shared incident execution rules.
func (s *Service) finishProcessInstanceIncidentRepair(ctx context.Context, request d.OpsRepairRequest, result d.OpsRepairResult, incidents []d.ProcessInstanceIncidentDetail, opts ...services.CallOption) (d.OpsRepairResult, error) {
	if len(incidents) == 0 {
		result.Remaining.Status = d.OpsWorkflowStepStatusSkipped
		result.Notices = append(result.Notices, d.OpsRepairWorkflowNotice{
			Code:     "no_active_incidents",
			Severity: "info",
			Message:  "no active incidents found for selected process instances",
		})
		return finishRepairResult(result, s.version, d.OpsRepairOutcomePlanned, nil)
	}
	if request.DryRun {
		return s.finishDryRunIncidentRepair(request, result, incidents)
	}

	var varErr error
	result.VariableUpdates, varErr = s.executeRepairVariableUpdates(ctx, request, incidents, opts...)
	variableUpdates := variableUpdatesByScope(result.VariableUpdates)
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(incidents), request.Workers, cfg.NoWorkerLimit)
	items, runErr := pool.ExecuteSlice(ctx, incidents, workers, cfg.FailFast, func(ctx context.Context, incident d.ProcessInstanceIncidentDetail, _ int) (repairIncidentExecution, error) {
		return s.executeIncidentRepair(ctx, request, incident, variableUpdates, opts...)
	})
	for _, item := range items {
		if item.Plan.IncidentKey == "" {
			continue
		}
		result.Plan = append(result.Plan, item.Plan)
		result.JobApplicability = append(result.JobApplicability, item.JobApplicability)
	}
	result.Remaining = d.OpsRepairRemainingIncidentSummary{Status: d.OpsWorkflowStepStatusConfirmed, Checked: !request.NoWait}
	repairErr := errorsJoin(varErr, runErr)
	outcome := repairOutcomeForPlans(result.Plan, repairErr)
	return finishRepairResult(result, s.version, outcome, repairErr)
}

// finishDryRunIncidentRepair returns the same planned repair shape for every incident discovery mode without submitting mutations.
func (s *Service) finishDryRunIncidentRepair(request d.OpsRepairRequest, result d.OpsRepairResult, incidents []d.ProcessInstanceIncidentDetail) (d.OpsRepairResult, error) {
	result.VariableUpdates = buildRepairVariableScopeUpdates(request, incidents, d.OpsWorkflowStepStatusPlanned)
	result.Plan, result.JobApplicability = buildRepairPlans(request, incidents, variableUpdatesByScope(result.VariableUpdates))
	result.Remaining.Status = d.OpsWorkflowStepStatusSkipped
	return finishRepairResult(result, s.version, d.OpsRepairOutcomePlanned, nil)
}

// discoverProcessInstanceRepairIncidents loads direct active incidents for each selected process instance and dedupes by incident key.
func (s *Service) discoverProcessInstanceRepairIncidents(ctx context.Context, request d.OpsRepairRequest, processInstanceKeys typex.Keys, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	keys := processInstanceKeys.Unique()
	if len(keys) == 0 {
		return nil, nil
	}
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(keys), request.Workers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice(ctx, keys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) ([]d.ProcessInstanceIncidentDetail, error) {
		return s.incAPI.SearchProcessInstanceIncidents(ctx, key, opts...)
	})
	seen := make(map[string]struct{})
	out := make([]d.ProcessInstanceIncidentDetail, 0)
	for _, item := range items {
		for _, incident := range item {
			if !incidentIsActive(incident) {
				continue
			}
			if incident.IncidentKey == "" {
				continue
			}
			if _, ok := seen[incident.IncidentKey]; ok {
				continue
			}
			seen[incident.IncidentKey] = struct{}{}
			out = append(out, incident)
		}
	}
	return out, err
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

// freezeIncidentSearchSet records the filtered discovery result without retaining any later incidents.
func freezeIncidentSearchSet(request d.OpsRepairRequest, incidents []d.ProcessInstanceIncidentDetail) d.OpsRepairFrozenSet {
	frozen := freezeExplicitIncidentSet(request, incidents)
	frozen.DiscoveryMode = d.OpsRepairDiscoveryModeSearch
	frozen.InputKeys = nil
	frozen.IncidentFilters = request.IncidentSelection
	return frozen
}

// freezeProcessInstanceRepairSet records repairable process instances and skipped direct selections.
func freezeProcessInstanceRepairSet(request d.OpsRepairRequest, selectedProcessInstanceKeys typex.Keys, incidents []d.ProcessInstanceIncidentDetail) d.OpsRepairFrozenSet {
	frozen := newRepairResult(request).FrozenSet
	frozen.Status = d.OpsWorkflowStepStatusConfirmed
	frozen.IncidentKeys = incidentKeysFromDetails(incidents)
	frozen.ProcessInstanceKeys = processInstanceKeysFromIncidents(incidents)
	frozen.SkippedProcessInstanceKeys = skippedProcessInstanceKeysWithoutActiveIncidents(selectedProcessInstanceKeys, frozen.ProcessInstanceKeys)
	frozen.RootProcessKeys = rootProcessInstanceKeysFromIncidents(incidents)
	frozen.JobKeys = jobKeysFromIncidents(incidents)
	if len(request.Variables) > 0 {
		frozen.VariableScopes = frozen.ProcessInstanceKeys.Unique()
	}
	frozen.OriginalIncidents = append([]d.ProcessInstanceIncidentDetail(nil), incidents...)
	return frozen
}

// repairIncidentDiscoverySize normalizes the search size while allowing --limit to cap repair targets first.
func repairIncidentDiscoverySize(request d.OpsRepairRequest) int32 {
	if request.Limit > 0 {
		return request.Limit
	}
	if request.BatchSize > 0 && request.BatchSize <= consts.MaxPISearchSize {
		return request.BatchSize
	}
	return consts.MaxPISearchSize
}

// repairProcessInstanceDiscoverySize normalizes process-instance discovery size for bounded repair searches.
func repairProcessInstanceDiscoverySize(request d.OpsRepairRequest) int32 {
	if request.Limit > 0 {
		return request.Limit
	}
	if request.BatchSize > 0 && request.BatchSize <= consts.MaxPISearchSize {
		return request.BatchSize
	}
	return consts.MaxPISearchSize
}

func repairBoundedSearchNotice(limit int32, discoverySize int32, count int, target string) []d.OpsRepairWorkflowNotice {
	if limit > 0 || discoverySize <= 0 || count < int(discoverySize) {
		return nil
	}
	return []d.OpsRepairWorkflowNotice{{
		Code:     "bounded_search_scope",
		Severity: "info",
		Message:  fmt.Sprintf("candidate %s reached batch size %d; more matching %s may exist", target, discoverySize, target),
		Details: map[string]string{
			"batchSize": fmt.Sprintf("%d", discoverySize),
			"target":    target,
		},
	}}
}

// limitRepairIncidents protects repair scope even when a stub or backend over-returns.
func limitRepairIncidents(items []d.ProcessInstanceIncidentDetail, limit int32) []d.ProcessInstanceIncidentDetail {
	if limit <= 0 || len(items) <= int(limit) {
		return items
	}
	return items[:limit]
}

// limitRepairProcessInstances protects repair scope even when a stub or backend over-returns.
func limitRepairProcessInstances(items []d.ProcessInstance, limit int32) []d.ProcessInstance {
	if limit <= 0 || len(items) <= int(limit) {
		return items
	}
	return items[:limit]
}

// skippedProcessInstanceKeysWithoutActiveIncidents reports selected process instances that are not repair targets.
func skippedProcessInstanceKeysWithoutActiveIncidents(selectedProcessInstanceKeys, repairableProcessInstanceKeys typex.Keys) typex.Keys {
	if len(selectedProcessInstanceKeys) == 0 {
		return nil
	}
	repairable := make(map[string]struct{}, len(repairableProcessInstanceKeys))
	for _, key := range repairableProcessInstanceKeys {
		if key == "" {
			continue
		}
		repairable[key] = struct{}{}
	}
	out := make(typex.Keys, 0)
	for _, key := range selectedProcessInstanceKeys.Unique() {
		if _, ok := repairable[key]; ok {
			continue
		}
		out = append(out, key)
	}
	return out
}

func buildRepairPlans(request d.OpsRepairRequest, incidents []d.ProcessInstanceIncidentDetail, variables map[string]d.OpsRepairVariableScopeUpdate) ([]d.OpsRepairPlanItem, []d.OpsRepairJobApplicability) {
	plans := make([]d.OpsRepairPlanItem, 0, len(incidents))
	applicability := make([]d.OpsRepairJobApplicability, 0, len(incidents))
	for _, incident := range incidents {
		plan, job := newIncidentRepairPlan(request, incident)
		applyRepairVariableStatus(&plan, variables[incident.ProcessInstanceKey])
		plans = append(plans, plan)
		applicability = append(applicability, job)
	}
	return plans, applicability
}

// buildRepairVariableScopeUpdates groups incidents by process-instance variable scope while preserving discovery order.
func buildRepairVariableScopeUpdates(request d.OpsRepairRequest, incidents []d.ProcessInstanceIncidentDetail, status d.OpsWorkflowStepStatus) []d.OpsRepairVariableScopeUpdate {
	if len(request.Variables) == 0 {
		return nil
	}
	names := sortedMapKeys(request.Variables)
	updates := make([]d.OpsRepairVariableScopeUpdate, 0)
	byScope := make(map[string]int)
	for _, incident := range incidents {
		if incident.ProcessInstanceKey == "" {
			continue
		}
		idx, ok := byScope[incident.ProcessInstanceKey]
		if !ok {
			idx = len(updates)
			byScope[incident.ProcessInstanceKey] = idx
			updates = append(updates, d.OpsRepairVariableScopeUpdate{
				ScopeKey:      incident.ProcessInstanceKey,
				VariableNames: append([]string(nil), names...),
				Payload:       toolx.CopyMap(request.Variables),
				Status:        status,
			})
		}
		if incident.IncidentKey != "" {
			updates[idx].DependentIncidentKeys = append(updates[idx].DependentIncidentKeys, incident.IncidentKey).Unique()
		}
	}
	return updates
}

// executeRepairVariableUpdates applies one variable payload per unique process-instance scope before incident mutation.
func (s *Service) executeRepairVariableUpdates(ctx context.Context, request d.OpsRepairRequest, incidents []d.ProcessInstanceIncidentDetail, opts ...services.CallOption) ([]d.OpsRepairVariableScopeUpdate, error) {
	planned := buildRepairVariableScopeUpdates(request, incidents, d.OpsWorkflowStepStatusPlanned)
	if len(planned) == 0 {
		return nil, nil
	}
	scopes := make(typex.Keys, 0, len(planned))
	dependents := make(map[string]typex.Keys, len(planned))
	for _, update := range planned {
		scopes = append(scopes, update.ScopeKey)
		dependents[update.ScopeKey] = append(typex.Keys(nil), update.DependentIncidentKeys...)
	}
	results, err := pisvc.UpdateProcessInstancesVariables(ctx, s.piAPI, s.log, scopes, request.Variables, request.Workers, opts...)
	resultsByScope := make(map[string]d.ProcessInstanceVariableUpdateResult, len(results.Items))
	for _, item := range results.Items {
		if item.Key != "" {
			resultsByScope[item.Key] = item
		}
	}
	updates := make([]d.OpsRepairVariableScopeUpdate, 0, len(planned))
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}
	for _, plannedUpdate := range planned {
		item, ok := resultsByScope[plannedUpdate.ScopeKey]
		if !ok {
			item = d.ProcessInstanceVariableUpdateResult{
				Key:       plannedUpdate.ScopeKey,
				Status:    d.ProcessInstanceVariableUpdateStatusMutationFailed,
				Error:     repairVariableMissingResultError(plannedUpdate.ScopeKey, err),
				Variables: plannedUpdate.Payload,
			}
		}
		status := repairVariableStatusFromProcessInstanceStatus(item.Status)
		if repairVariableStatusBlocksResolution(status) {
			if item.Error != "" {
				errs = append(errs, errors.New(item.Error))
			} else {
				errs = append(errs, fmt.Errorf("variable update failed for process-instance scope %s", item.Key))
			}
		}
		updates = append(updates, d.OpsRepairVariableScopeUpdate{
			ScopeKey:              item.Key,
			VariableNames:         sortedMapKeys(item.Variables),
			Payload:               toolx.CopyMap(item.Variables),
			DependentIncidentKeys: append(typex.Keys(nil), dependents[item.Key]...),
			Status:                status,
			Errors:                repairVariableErrors(item),
		})
	}
	return updates, errorsJoin(errs...)
}

// repairVariableMissingResultError explains unprocessed scopes when fail-fast or cancellation stopped bulk updates.
func repairVariableMissingResultError(scope string, err error) string {
	if err != nil {
		return fmt.Sprintf("variable update did not complete for process-instance scope %s: %v", scope, err)
	}
	return fmt.Sprintf("variable update did not complete for process-instance scope %s", scope)
}

// variableUpdatesByScope indexes variable update results for per-incident gating.
func variableUpdatesByScope(updates []d.OpsRepairVariableScopeUpdate) map[string]d.OpsRepairVariableScopeUpdate {
	if len(updates) == 0 {
		return nil
	}
	out := make(map[string]d.OpsRepairVariableScopeUpdate, len(updates))
	for _, update := range updates {
		out[update.ScopeKey] = update
	}
	return out
}

// applyRepairVariableStatus transfers the deduped scope outcome onto one incident plan.
func applyRepairVariableStatus(plan *d.OpsRepairPlanItem, update d.OpsRepairVariableScopeUpdate) bool {
	if plan == nil || plan.VariableUpdateStatus != d.OpsWorkflowStepStatusPlanned || update.ScopeKey == "" {
		return false
	}
	plan.VariableUpdateStatus = update.Status
	if repairVariableStatusBlocksResolution(update.Status) {
		plan.Errors = append(plan.Errors, update.Errors...)
	}
	return true
}

// repairVariableStatusBlocksResolution identifies variable outcomes that must stop dependent incident resolution.
func repairVariableStatusBlocksResolution(status d.OpsWorkflowStepStatus) bool {
	return status == d.OpsWorkflowStepStatusFailed || status == d.OpsWorkflowStepStatusConfirmationFailed
}

// repairVariableStatusFromProcessInstanceStatus maps process-instance update vocabulary into repair step vocabulary.
func repairVariableStatusFromProcessInstanceStatus(status d.ProcessInstanceVariableUpdateStatus) d.OpsWorkflowStepStatus {
	switch status {
	case d.ProcessInstanceVariableUpdateStatusConfirmed:
		return d.OpsWorkflowStepStatusConfirmed
	case d.ProcessInstanceVariableUpdateStatusSubmitted:
		return d.OpsWorkflowStepStatusSubmitted
	case d.ProcessInstanceVariableUpdateStatusConfirmationFailed:
		return d.OpsWorkflowStepStatusConfirmationFailed
	default:
		return d.OpsWorkflowStepStatusFailed
	}
}

// repairVariableErrors preserves per-scope update failure text without fabricating success messages.
func repairVariableErrors(item d.ProcessInstanceVariableUpdateResult) []string {
	if item.Error == "" {
		return nil
	}
	return []string{item.Error}
}

func (s *Service) executeIncidentRepair(ctx context.Context, request d.OpsRepairRequest, incident d.ProcessInstanceIncidentDetail, variables map[string]d.OpsRepairVariableScopeUpdate, opts ...services.CallOption) (repairIncidentExecution, error) {
	plan, jobApplicability := newIncidentRepairPlan(request, incident)
	if applyRepairVariableStatus(&plan, variables[incident.ProcessInstanceKey]) && repairVariableStatusBlocksResolution(plan.VariableUpdateStatus) {
		plan.ResolutionStatus = d.OpsWorkflowStepStatusBlocked
		plan.ConfirmationStatus = d.OpsWorkflowStepStatusSkipped
		if len(plan.Errors) == 0 {
			plan.Errors = append(plan.Errors, fmt.Sprintf("variable update failed for process-instance scope %s", incident.ProcessInstanceKey))
		}
		return repairIncidentExecution{Plan: plan, JobApplicability: jobApplicability}, nil
	}
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

// processInstanceKeysFromDetails extracts stable process-instance keys from selected process-instance records.
func processInstanceKeysFromDetails(items []d.ProcessInstance) typex.Keys {
	keys := make(typex.Keys, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			keys = append(keys, item.Key)
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
