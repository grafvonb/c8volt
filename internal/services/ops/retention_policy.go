// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
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
	discovery := pisvc.RetentionDiscovery{
		Filter: filter,
		Keys:   request.DiscoveredKeys.Unique(),
	}
	if request.DiscoveredKeys == nil {
		var err error
		discovery, err = pisvc.DiscoverRetentionProcessInstances(ctx, s.piAPI, pisvc.RetentionDiscoveryRequest{
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
	}

	result.Discovery = d.RetentionDiscoveryResult{
		Status:                 d.OpsWorkflowStepStatusPlanned,
		RetentionDays:          request.RetentionDays,
		DerivedEndDateBoundary: request.DerivedEndDateBoundary,
		Filters:                discovery.Filter,
		SeedKeys:               discovery.Keys,
		Count:                  len(discovery.Keys),
	}
	if len(discovery.Keys) == 0 {
		result.DeletePlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomePlanned, nil)
	}

	plan, err := buildRetentionDeletePlan(ctx, s.piAPI, discovery.Keys, request.Workers, !request.DryRun, opts...)
	result.DeletePlan = plan
	if err != nil {
		result.DeletePlan.Status = d.OpsWorkflowStepStatusFailed
		result.DeletePlan.Errors = []string{err.Error()}
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomeFailed, fmt.Errorf("retention policy delete-plan validation: %w", err))
	}

	if request.DryRun {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomePlanned, nil)
	}

	if len(plan.ResolvedRootKeys) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomePlanned, nil)
	}

	if !request.Force && len(plan.NonFinalAffectedItems) > 0 {
		err = fmt.Errorf("%w: refusing to delete retention process-instance scope: %s; no delete request was submitted; use --force to cancel the non-final affected scope before delete", d.ErrPrecondition, formatRetentionBlockedScope(plan))
		result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
		result.Deletion.Errors = []string{err.Error()}
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomeFailed, err)
	}

	reports, err := pisvc.DeleteProcessInstances(ctx, s.piAPI, s.log, plan.ResolvedRootKeys, request.Workers, len(plan.AffectedKeys), opts...)
	result.Deletion = d.RetentionDeletionResult{
		Status:            deletionStatusForReports(reports, request.NoWait, err),
		SubmittedRootKeys: plan.ResolvedRootKeys,
		Items:             reports,
		Submitted:         len(reports) > 0,
		Confirmed:         err == nil && !request.NoWait && allReportsOK(reports),
		NoWait:            request.NoWait,
		Errors:            deletionErrors(err),
	}
	if err != nil {
		return finishRetentionPolicyResult(result, retentionDeletionOutcomeForReports(reports), fmt.Errorf("delete retention process instances: %w", err))
	}
	return finishRetentionPolicyResult(result, retentionDeletionOutcomeForReports(reports), nil)
}

func buildRetentionDeletePlan(ctx context.Context, api pisvc.API, seedKeys typex.Keys, wantedWorkers int, requiresConfirmation bool, opts ...services.CallOption) (d.RetentionDeletePlan, error) {
	cfg := services.ApplyCallOptions(opts)
	seeds := seedKeys.Unique()
	ancestryWorkers := toolx.DetermineNoOfWorkers(len(seeds), wantedWorkers, cfg.NoWorkerLimit)
	ancestryResults, err := pool.ExecuteSlice[string, pitraversal.Result](ctx, seeds, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (pitraversal.Result, error) {
		return api.AncestryResult(ctx, key, opts...)
	})
	if err != nil {
		return d.RetentionDeletePlan{}, err
	}

	var roots typex.Keys
	var skippedSeedKeys typex.Keys
	var skippedRoots []d.ProcessInstance
	seenRoots := make(map[string]struct{}, len(ancestryResults))
	var duplicateRoots typex.Keys
	for _, result := range ancestryResults {
		rootKey := result.RootKey
		if rootKey == "" {
			continue
		}
		root, ok := result.Chain[rootKey]
		if !ok {
			root = d.ProcessInstance{Key: rootKey}
		}
		if !root.State.IsTerminal() {
			skippedSeedKeys = append(skippedSeedKeys, result.StartKey)
			skippedRoots = appendUniqueProcessInstancesByKey(skippedRoots, root)
			continue
		}
		if _, ok := seenRoots[rootKey]; ok {
			duplicateRoots = append(duplicateRoots, rootKey)
		}
		seenRoots[rootKey] = struct{}{}
		roots = append(roots, rootKey)
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantResults, err := pool.ExecuteSlice[string, pitraversal.Result](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (pitraversal.Result, error) {
		return api.DescendantsResult(ctx, root, opts...)
	})
	if err != nil {
		return d.RetentionDeletePlan{}, err
	}

	var collected typex.Keys
	for _, result := range descendantResults {
		collected = append(collected, result.Keys...)
	}
	collected = collected.Unique()

	ancestryWarning, ancestryMissing, ancestryOutcome := retentionTraversalWarning(ancestryResults)
	descendantsWarning, descendantsMissing, descendantsOutcome := retentionTraversalWarning(descendantResults)
	warning := ancestryWarning
	if warning == "" {
		warning = descendantsWarning
	}
	outcome := ancestryOutcome
	if outcome == d.TraversalOutcomeComplete {
		outcome = descendantsOutcome
	} else if descendantsOutcome == d.TraversalOutcomePartial {
		outcome = d.TraversalOutcomePartial
	}

	plan := d.RetentionDeletePlan{
		Status:                d.OpsWorkflowStepStatusPlanned,
		SeedKeys:              seeds,
		ResolvedRootKeys:      roots,
		AffectedKeys:          collected,
		DuplicateKeys:         duplicateRoots.Unique(),
		FinalStateItems:       retentionSelectedFinalStateProcessInstances(seeds, ancestryResults),
		NonFinalAffectedItems: retentionNonFinalProcessInstances(collected, descendantResults),
		SkippedSeedKeys:       skippedSeedKeys.Unique(),
		SkippedNonFinalRoots:  skippedRoots,
		MissingAncestors:      retentionUniqueMissingAncestors(append(ancestryMissing, descendantsMissing...)),
		RequiresConfirmation:  requiresConfirmation && len(roots) > 0,
	}
	if warning != "" {
		plan.TraversalWarnings = []string{warning}
	}
	if !retentionPlanHasActionableResults(plan) && outcome == d.TraversalOutcomeUnresolved {
		return plan, fmt.Errorf("%w: no process instances resolved during retention dependency expansion", services.ErrOrphanedInstance)
	}
	return plan, nil
}

func retentionTraversalWarning(results []pitraversal.Result) (warning string, missing []d.MissingAncestor, outcome d.TraversalOutcome) {
	outcome = d.TraversalOutcomeComplete
	for _, result := range results {
		if len(result.MissingAncestors) > 0 {
			missing = append(missing, retentionDomainMissingAncestors(result.MissingAncestors)...)
		}
		if result.Warning != "" && warning == "" {
			warning = result.Warning
		}
		switch result.Outcome {
		case pitraversal.OutcomeUnresolved:
			if outcome == d.TraversalOutcomeComplete {
				outcome = d.TraversalOutcomeUnresolved
			}
		case pitraversal.OutcomePartial:
			outcome = d.TraversalOutcomePartial
		}
	}
	if len(missing) > 0 && warning == "" {
		warning = "one or more parent process instances were not found"
	}
	return warning, missing, outcome
}

func retentionDomainMissingAncestors(items []pitraversal.MissingAncestor) []d.MissingAncestor {
	out := make([]d.MissingAncestor, 0, len(items))
	for _, item := range items {
		out = append(out, d.MissingAncestor{Key: item.Key, StartKey: item.StartKey})
	}
	return out
}

func retentionUniqueMissingAncestors(items []d.MissingAncestor) []d.MissingAncestor {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]d.MissingAncestor, 0, len(items))
	for _, item := range items {
		key := item.Key + "\x00" + item.StartKey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func retentionNonFinalProcessInstances(keys typex.Keys, results []pitraversal.Result) []d.ProcessInstance {
	if len(keys) == 0 || len(results) == 0 {
		return nil
	}
	byKey := make(map[string]d.ProcessInstance)
	for _, result := range results {
		for key, pi := range result.Chain {
			if _, ok := byKey[key]; !ok {
				byKey[key] = pi
			}
		}
	}
	out := make([]d.ProcessInstance, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		pi, ok := byKey[key]
		if !ok || pi.State == "" || pi.State.IsTerminal() {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pi)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func retentionSelectedFinalStateProcessInstances(keys typex.Keys, results []pitraversal.Result) []d.ProcessInstance {
	if len(keys) == 0 || len(results) == 0 {
		return nil
	}
	byStartKey := make(map[string]pitraversal.Result, len(results))
	for _, result := range results {
		byStartKey[result.StartKey] = result
	}
	out := make([]d.ProcessInstance, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		result, ok := byStartKey[key]
		if !ok || result.Chain == nil {
			continue
		}
		pi, ok := result.Chain[key]
		if !ok || !pi.State.IsTerminal() {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pi)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func appendUniqueProcessInstancesByKey(items []d.ProcessInstance, item d.ProcessInstance) []d.ProcessInstance {
	if item.Key == "" {
		return items
	}
	for _, existing := range items {
		if existing.Key == item.Key {
			return items
		}
	}
	return append(items, item)
}

func retentionPlanHasActionableResults(plan d.RetentionDeletePlan) bool {
	return len(plan.ResolvedRootKeys) > 0 || len(plan.AffectedKeys) > 0 || len(plan.SkippedSeedKeys) > 0 || len(plan.SkippedNonFinalRoots) > 0
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

func formatRetentionBlockedScope(plan d.RetentionDeletePlan) string {
	items := plan.NonFinalAffectedItems
	seen := make(map[d.State]struct{}, len(items))
	states := make([]string, 0, len(items))
	pairs := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.State]; !ok {
			seen[item.State] = struct{}{}
			states = append(states, item.State.String())
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", item.Key, item.State))
	}
	return fmt.Sprintf("retention matched %d ended seed(s); delete planning expanded to %d affected process instance(s) across %d root(s); %d non-final descendant process instance(s) in otherwise final-root retention scope; states: %s; %s", len(plan.SeedKeys), len(plan.AffectedKeys), len(plan.ResolvedRootKeys), len(items), strings.Join(states, ", "), strings.Join(pairs, ", "))
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

func retentionDeletionOutcomeForReports(reports []d.Reporter) d.RetentionPolicyOutcome {
	if len(reports) == 0 {
		return d.RetentionPolicyOutcomeFailed
	}
	ok := 0
	for _, report := range reports {
		if report.Ok {
			ok++
		}
	}
	switch ok {
	case len(reports):
		return d.RetentionPolicyOutcomeDeleted
	case 0:
		return d.RetentionPolicyOutcomeFailed
	default:
		return d.RetentionPolicyOutcomePartiallyFailed
	}
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
