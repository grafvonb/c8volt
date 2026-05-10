// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

type legacyDryRunTraversalOnly interface {
	LegacyDryRunTraversalOnly() bool
}

func CreateNProcessInstances(ctx context.Context, api API, log *slog.Logger, data d.ProcessInstanceData, n int, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstanceCreation, error) {
	cfg := services.ApplyCallOptions(opts)
	nw := toolx.DetermineNoOfWorkers(n, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("creating %d process instances using %d workers", n, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("creating %d process instance(s)", n))
	defer stopActivity()
	pics, err := pool.ExecuteNTimes[d.ProcessInstanceCreation](ctx, n, nw, cfg.FailFast, func(ctx context.Context, _ int) (d.ProcessInstanceCreation, error) {
		return api.CreateProcessInstance(ctx, data, opts...)
	})
	if !cfg.NoWait {
		log.Info(fmt.Sprintf("creation of %d process instances completed", n))
	}
	return pics, err
}

func CancelProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, affectedCount int, opts ...services.CallOption) ([]d.Reporter, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	if affectedCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling process instances requested for %d affected instance(s) across %d root key(s) using %d worker(s)", affectedCount, lk, nw), log, cfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling process instances requested for %d unique key(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	}
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("cancelling", lk, affectedCount))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		resp, _, err := api.CancelProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait {
		t, oks, noks := reporterTotals(rs)
		if affectedCount > t {
			log.Info(fmt.Sprintf("cancelling %d process instance(s) completed via %d root request(s): %d root request(s) succeeded or already cancelled/terminated, %d failed", affectedCount, t, oks, noks))
		} else {
			log.Info(fmt.Sprintf("cancelling %d process instance(s) completed: %d succeeded or already cancelled/terminated, %d failed", t, oks, noks))
		}
	}
	return rs, err
}

func DeleteProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, affectedCount int, opts ...services.CallOption) ([]d.Reporter, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	if affectedCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("deleting process instances requested for %d affected instance(s) across %d root key(s) using %d worker(s)", affectedCount, lk, nw), log, cfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("deleting process instances requested for %d unique key(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	}
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("deleting", lk, affectedCount))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		resp, err := api.DeleteProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait {
		t, oks, noks := reporterTotals(rs)
		if hasStatusCode(rs, http.StatusConflict) {
			affected := affectedCount
			if affected < t {
				affected = t
			}
			log.Info(fmt.Sprintf("cannot delete expanded process-instance scope of %d process instance(s): one or more affected process instances are not in a terminated state; use --force flag to cancel and then delete them", affected))
		}
		if affectedCount > t {
			log.Info(fmt.Sprintf("deleting %d process instance(s) completed via %d root request(s): %d root request(s) succeeded, %d failed", affectedCount, t, oks, noks))
		} else {
			log.Info(fmt.Sprintf("deleting %d process instances completed: %d succeeded, %d failed", t, oks, noks))
		}
	}
	return rs, err
}

func UpdateProcessInstancesVariables(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResults, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("updating variables for %d unique process instance(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("updating variables for", lk, 0))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.ProcessInstanceVariableUpdateResult](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceVariableUpdateResult, error) {
		return UpdateProcessInstanceVariables(ctx, api, d.ProcessInstanceVariableUpdateRequest{Key: key, Variables: variables}, opts...)
	})
	return d.ProcessInstanceVariableUpdateResults{Items: rs}, err
}

func WaitForProcessInstancesState(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("waiting for %d unique process instance(s) to reach desired state(s) %v using %d worker(s)", lk, desired, nw), log, cfg.Verbose)
	return api.WaitForProcessInstancesState(ctx, ukeys, desired, nw, opts...)
}

func WaitForProcessInstancesExpectation(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, request d.ProcessInstanceExpectationRequest, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("waiting for %d unique process instance(s) to satisfy expectation(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	return api.WaitForProcessInstancesExpectation(ctx, ukeys, request, nw, opts...)
}

func GetProcessInstances(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	ukeys := keys.Unique()
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("getting %d process instance(s)", len(ukeys)))
	defer stopActivity()
	return api.GetProcessInstances(ctx, ukeys, wantedWorkers, opts...)
}

func UpdateProcessInstanceVariables(ctx context.Context, api API, request d.ProcessInstanceVariableUpdateRequest, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResult, error) {
	cfg := services.ApplyCallOptions(opts)
	resp, err := api.UpdateProcessInstanceVariables(ctx, request.Key, request.Variables, opts...)
	result := domainVariableUpdateResult(resp, request.Variables)
	if result.Key == "" {
		result.Key = request.Key
	}
	if err != nil {
		result.Error = err.Error()
		if result.MutationAccepted {
			result.Status = d.ProcessInstanceVariableUpdateStatusConfirmationFailed
			result.ConfirmationStatus = "failed"
			return result, err
		}
		result.Status = d.ProcessInstanceVariableUpdateStatusMutationFailed
		result.MutationAccepted = false
		if cfg.NoWait {
			result.ConfirmationStatus = "skipped"
			return result, nil
		}
		return result, err
	}
	if cfg.NoWait {
		result.Status = d.ProcessInstanceVariableUpdateStatusSubmitted
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	result.Status = d.ProcessInstanceVariableUpdateStatusConfirmed
	result.ConfirmationStatus = "confirmed"
	return result, nil
}

func DryRunCancelOrDeletePlan(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.DryRunPIKeyExpansion, error) {
	if legacyOnly, ok := api.(legacyDryRunTraversalOnly); ok && legacyOnly.LegacyDryRunTraversalOnly() {
		return dryRunCancelOrDeletePlanLegacy(ctx, api, keys, wantedWorkers, opts...)
	}

	var roots typex.Keys
	var collected typex.Keys
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	ancestryWorkers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	ancestryResults, err := pool.ExecuteSlice[string, pitraversal.Result](ctx, ukeys, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (pitraversal.Result, error) {
		return api.AncestryResult(ctx, key, opts...)
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, result := range ancestryResults {
		if result.RootKey != "" {
			roots = append(roots, result.RootKey)
		}
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantResults, err := pool.ExecuteSlice[string, pitraversal.Result](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (pitraversal.Result, error) {
		return api.DescendantsResult(ctx, root, opts...)
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, result := range descendantResults {
		collected = append(collected, result.Keys...)
	}

	ancestryWarning, ancestryMissing, ancestryOutcome := mapDryRunTraversalWarning(ancestryResults)
	descendantsWarning, descendantsMissing, descendantsOutcome := mapDryRunTraversalWarning(descendantResults)
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

	collected = collected.Unique()
	plan := d.DryRunPIKeyExpansion{
		Roots:                      roots,
		Collected:                  collected,
		SelectedFinalState:         selectedFinalStateProcessInstances(keys, ancestryResults),
		RequiresCancelBeforeDelete: nonFinalProcessInstances(collected, descendantResults),
		MissingAncestors:           uniqueMissingAncestors(append(ancestryMissing, descendantsMissing...)),
		Warning:                    warning,
		Outcome:                    outcome,
	}
	return plan, validateDryRunPIKeyExpansion(plan)
}

func dryRunCancelOrDeletePlanLegacy(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.DryRunPIKeyExpansion, error) {
	var roots typex.Keys
	var collected typex.Keys
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	ancestryWorkers := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	legacyRoots, err := pool.ExecuteSlice[string, string](ctx, ukeys, ancestryWorkers, cfg.FailFast, func(ctx context.Context, key string, _ int) (string, error) {
		rootKey, _, _, err := api.Ancestry(ctx, key, opts...)
		return rootKey, err
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, rootKey := range legacyRoots {
		if rootKey != "" {
			roots = append(roots, rootKey)
		}
	}
	roots = roots.Unique()

	descendantWorkers := toolx.DetermineNoOfWorkers(len(roots), wantedWorkers, cfg.NoWorkerLimit)
	descendantLists, err := pool.ExecuteSlice[string, typex.Keys](ctx, roots, descendantWorkers, cfg.FailFast, func(ctx context.Context, root string, _ int) (typex.Keys, error) {
		desc, _, _, err := api.Descendants(ctx, root, opts...)
		return desc, err
	})
	if err != nil {
		return d.DryRunPIKeyExpansion{}, err
	}
	for _, desc := range descendantLists {
		collected = append(collected, desc...)
	}
	return d.DryRunPIKeyExpansion{
		Roots:     roots,
		Collected: collected.Unique(),
		Outcome:   d.TraversalOutcomeComplete,
	}, nil
}

func processInstanceBulkActivity(verb string, rootCount int, affectedCount int) string {
	if affectedCount > rootCount {
		return fmt.Sprintf("%s %d process instance(s) via %d root request(s)", verb, affectedCount, rootCount)
	}
	return fmt.Sprintf("%s %d process instance(s)", verb, rootCount)
}

func domainVariableUpdateResult(x d.ProcessInstanceVariableUpdateResponse, variables map[string]any) d.ProcessInstanceVariableUpdateResult {
	status := d.ProcessInstanceVariableUpdateStatusSubmitted
	if !x.Ok {
		status = d.ProcessInstanceVariableUpdateStatusMutationFailed
	}
	return d.ProcessInstanceVariableUpdateResult{
		Key:              x.Key,
		Status:           status,
		MutationAccepted: x.Ok,
		StatusCode:       x.StatusCode,
		Message:          x.Status,
		Variables:        toolx.CopyMap(variables),
	}
}

func reporterTotals(items []d.Reporter) (total, oks, noks int) {
	for _, item := range items {
		if item.Ok {
			oks++
		}
	}
	total = len(items)
	noks = total - oks
	return total, oks, noks
}

func hasStatusCode(items []d.Reporter, statusCode int) bool {
	for _, item := range items {
		if item.StatusCode == statusCode {
			return true
		}
	}
	return false
}

func mapDryRunTraversalWarning(results []pitraversal.Result) (warning string, missing []d.MissingAncestor, outcome d.TraversalOutcome) {
	outcome = d.TraversalOutcomeComplete
	for _, result := range results {
		if len(result.MissingAncestors) > 0 {
			missing = append(missing, domainMissingAncestors(result.MissingAncestors)...)
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

func domainMissingAncestors(items []pitraversal.MissingAncestor) []d.MissingAncestor {
	out := make([]d.MissingAncestor, 0, len(items))
	for _, item := range items {
		out = append(out, d.MissingAncestor{Key: item.Key, StartKey: item.StartKey})
	}
	return out
}

func uniqueMissingAncestors(items []d.MissingAncestor) []d.MissingAncestor {
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

func nonFinalProcessInstances(keys typex.Keys, results []pitraversal.Result) []d.ProcessInstance {
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

func selectedFinalStateProcessInstances(keys typex.Keys, results []pitraversal.Result) []d.ProcessInstance {
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

func validateDryRunPIKeyExpansion(plan d.DryRunPIKeyExpansion) error {
	if plan.HasActionableResults() || plan.Outcome != d.TraversalOutcomeUnresolved {
		return nil
	}
	return fmt.Errorf("%w: no process instances resolved during dependency expansion", services.ErrOrphanedInstance)
}
