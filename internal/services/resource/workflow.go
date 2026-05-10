// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/poller"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

func DeleteProcessDefinition(ctx context.Context, api API, pdApi pdsvc.API, piApi pisvc.API, log *slog.Logger, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	cfg := services.ApplyCallOptions(opts)
	log.Info(fmt.Sprintf("checking delete impact for process definition %s", key))
	previewPlan, err := PreviewDeleteProcessDefinitions(ctx, pdApi, piApi, log, types.Keys{key}, opts...)
	if err != nil {
		return d.ResourceDeleteResponse{}, err
	}
	if len(previewPlan.Items) == 0 {
		return d.ResourceDeleteResponse{}, fmt.Errorf("process definition %s was not included in delete impact check", key)
	}
	plan := previewPlan.Items[0]
	log.Info(fmt.Sprintf("delete impact for process definition %s: %d active process instance(s)", key, plan.ActiveProcessInstances()))
	if !cfg.NoStateCheck && plan.ActiveProcessInstances() > 0 {
		if !cfg.Force {
			return d.ResourceDeleteResponse{}, fmt.Errorf("cannot delete process definition %s with %d active process instance(s); use --force to cancel them automatically", key, plan.ActiveProcessInstances())
		}
		log.Info(fmt.Sprintf("force enabled for process definition %s; cancelling active process instances before deletion", key))
		if err := cancelProcessDefinitionActiveInstances(ctx, piApi, log, key, plan, opts...); err != nil {
			return d.ResourceDeleteResponse{}, fmt.Errorf("delete process definition cancel active instances: %w", err)
		}
		if err := waitForActiveProcessDefinitionInstancesDrained(ctx, pdApi, log, key, opts...); err != nil {
			return d.ResourceDeleteResponse{}, fmt.Errorf("delete process definition wait for active instances to drain: %w", err)
		}
		if err := deleteProcessDefinitionProcessInstances(ctx, piApi, log, key, plan, opts...); err != nil {
			return d.ResourceDeleteResponse{}, fmt.Errorf("delete process definition process-instance history: %w", err)
		}
	}
	log.Info(fmt.Sprintf("submitting process-definition resource deletion for %s with associated history", key))
	resp, err := api.Delete(ctx, key, opts...)
	if resp.BatchOperationKey != "" {
		log.Info(fmt.Sprintf("process-definition resource deletion for %s accepted; history deletion batch %s state %s", key, resp.BatchOperationKey, resp.BatchState))
	} else if resp.Status != "" {
		log.Info(fmt.Sprintf("process-definition resource deletion for %s completed with status %s", key, resp.Status))
	}
	return resp, err
}

func PreviewDeleteProcessDefinitions(ctx context.Context, pdApi pdsvc.API, piApi pisvc.API, log *slog.Logger, keys types.Keys, opts ...services.CallOption) (d.DeleteProcessDefinitionPlan, error) {
	ukeys := keys.Unique()
	cfg := services.ApplyCallOptions(opts)
	plan := d.DeleteProcessDefinitionPlan{
		Items:                 make([]d.DeleteProcessDefinitionPlanItem, 0, len(ukeys)),
		StateCheckSkipped:     cfg.NoStateCheck,
		ProcessDefinitionKeys: append([]string(nil), ukeys...),
	}
	if cfg.NoStateCheck {
		stopActivity := logging.StartActivity(ctx, fmt.Sprintf("checking delete impact for %d process definition(s); process-instance state check is skipped; no changes are being made", len(ukeys)))
		defer stopActivity()
		for _, key := range ukeys {
			plan.Items = append(plan.Items, d.DeleteProcessDefinitionPlanItem{Key: key})
		}
		return plan, nil
	}

	activityMsg := fmt.Sprintf("checking active process instances for %d process definition(s); no changes are being made", len(ukeys))
	if cfg.Force {
		activityMsg = fmt.Sprintf("checking active process instances and cancellation roots for %d process definition(s); no changes are being made", len(ukeys))
	}
	stopActivity := logging.StartActivity(ctx, activityMsg)
	defer stopActivity()
	for _, key := range ukeys {
		item, err := previewDeleteProcessDefinitionImpact(ctx, pdApi, piApi, key, cfg.Force, cfg.Verbose, opts...)
		if err != nil {
			return plan, err
		}
		plan.Items = append(plan.Items, item)
	}
	return plan, nil
}

func DeleteProcessDefinitions(ctx context.Context, api API, pdApi pdsvc.API, piApi pisvc.API, log *slog.Logger, keys types.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ResourceDeleteResponse, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("deleting process definitions requested for %d unique key(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("deleting %d process definition(s)", lk))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.ResourceDeleteResponse](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ResourceDeleteResponse, error) {
		return DeleteProcessDefinition(ctx, api, pdApi, piApi, log, key, opts...)
	})
	if !cfg.NoWait {
		total, oks, noks := resourceDeleteTotals(rs)
		log.Info(fmt.Sprintf("deleting %d process definitions completed: %d succeeded, %d failed", total, oks, noks))
	}
	return rs, err
}

func previewDeleteProcessDefinitionImpact(ctx context.Context, pdApi pdsvc.API, piApi pisvc.API, key string, force bool, verbose bool, opts ...services.CallOption) (d.DeleteProcessDefinitionPlanItem, error) {
	item := d.DeleteProcessDefinitionPlanItem{Key: key}
	active, err := countActiveProcessInstancesForDefinition(ctx, pdApi, key, opts...)
	if err != nil {
		return item, err
	}
	item.ActiveProcessInstanceCount = active
	if !force || active == 0 {
		return item, nil
	}
	activeInstances, err := listActiveProcessInstancesForDefinition(ctx, piApi, key, opts...)
	if err != nil {
		return item, err
	}
	activeKeys := processInstanceKeys(activeInstances)
	item.ActiveProcessInstanceKeys = activeKeys
	planKeys := activeKeys
	if roots, ok := processInstanceRootKeys(activeInstances); ok {
		planKeys = roots
	}
	cancellationPlan, err := pisvc.DryRunCancelOrDeletePlan(ctx, piApi, planKeys, 0, opts...)
	if err != nil {
		return item, err
	}
	item.CancellationPlan = cancellationPlan
	if cancellationPlan.Warning != "" {
		item.Warnings = append(item.Warnings, formatPartialCancellationImpactWarning(key, cancellationPlan, verbose))
	}
	return item, nil
}

func listActiveProcessInstancesForDefinition(ctx context.Context, piApi pisvc.API, key string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	const pageSize int32 = 500
	filter := d.ProcessInstanceFilter{ProcessDefinitionKey: key, State: d.StateActive}
	pageReq := d.ProcessInstancePageRequest{Size: pageSize}
	var items []d.ProcessInstance
	for {
		page, err := piApi.SearchForProcessInstancesPage(ctx, filter, pageReq, opts...)
		if err != nil {
			return items, err
		}
		items = append(items, page.Items...)
		if page.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			return items, nil
		}
		if len(page.Items) == 0 {
			return items, fmt.Errorf("active process-instance search for process definition %s reported more pages but returned no items", key)
		}
		if page.EndCursor != "" {
			pageReq.After = page.EndCursor
			pageReq.From = 0
			continue
		}
		pageReq.From += int32(len(page.Items))
	}
}

func countActiveProcessInstancesForDefinition(ctx context.Context, pdApi pdsvc.API, key string, opts ...services.CallOption) (int64, error) {
	statOpts := append([]services.CallOption{}, opts...)
	statOpts = append(statOpts, services.WithStat())
	pd, err := pdApi.GetProcessDefinition(ctx, key, statOpts...)
	if err != nil {
		return 0, err
	}
	if pd.Statistics == nil {
		return 0, fmt.Errorf("active process-instance impact check for process definition %s did not return statistics", key)
	}
	return pd.Statistics.Active, nil
}

func cancelProcessDefinitionActiveInstances(ctx context.Context, piApi pisvc.API, log *slog.Logger, key string, plan d.DeleteProcessDefinitionPlanItem, opts ...services.CallOption) error {
	roots := plan.CancellationPlan.Roots.Unique()
	if len(roots) == 0 {
		return fmt.Errorf("no root process instances found to cancel for process definition %s", key)
	}
	affected := len(plan.CancellationPlan.Collected.Unique())
	if affected == 0 {
		affected = len(plan.ActiveProcessInstanceKeys)
	}
	log.Info(fmt.Sprintf("cancelling %d root process instance(s), affecting %d process instance(s), for process definition %s before deletion", len(roots), affected, key))
	cancelOpts := append([]services.CallOption{}, opts...)
	cancelOpts = append(cancelOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err := pisvc.CancelProcessInstances(ctx, piApi, log, roots, 0, affected, cancelOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reporterTotals(reports)
	if failed > 0 {
		return fmt.Errorf("cancelling root process instances for process definition %s failed for %d root request(s)", key, failed)
	}
	return nil
}

func deleteProcessDefinitionProcessInstances(ctx context.Context, piApi pisvc.API, log *slog.Logger, key string, plan d.DeleteProcessDefinitionPlanItem, opts ...services.CallOption) error {
	roots := plan.CancellationPlan.Roots.Unique()
	if len(roots) == 0 {
		return fmt.Errorf("no root process instances found to delete for process definition %s", key)
	}
	affected := len(plan.CancellationPlan.Collected.Unique())
	if affected == 0 {
		affected = len(plan.ActiveProcessInstanceKeys)
	}
	log.Info(fmt.Sprintf("deleting historical data for %d process instance(s) in %d root tree(s) before deleting process definition %s", affected, len(roots), key))
	deleteOpts := append([]services.CallOption{}, opts...)
	deleteOpts = append(deleteOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err := pisvc.DeleteProcessInstances(ctx, piApi, log, roots, 0, affected, deleteOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reporterTotals(reports)
	if failed > 0 {
		return fmt.Errorf("deleting process-instance tree for process definition %s failed for %d root request(s)", key, failed)
	}
	return nil
}

func waitForActiveProcessDefinitionInstancesDrained(ctx context.Context, pdApi pdsvc.API, log *slog.Logger, key string, opts ...services.CallOption) error {
	log.Info(fmt.Sprintf("waiting until process definition %s has no active process instances before deletion", key))
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		active, err := countActiveProcessInstancesForDefinition(ctx, pdApi, key, opts...)
		if err != nil {
			return poller.JobPollStatus{}, err
		}
		if active == 0 {
			return poller.JobPollStatus{Success: true, Message: fmt.Sprintf("process definition %s has no active process instances", key)}, nil
		}
		log.Info(fmt.Sprintf("process definition %s still has %d active process instance(s); waiting before deletion", key, active))
		return poller.JobPollStatus{Success: false, Message: fmt.Sprintf("process definition %s still has %d active process instance(s)", key, active)}, nil
	}
	if err := poller.WaitForCompletion(ctx, log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("process definition %s has no active process instances; deleting process definition", key))
	return nil
}

func processInstanceKeys(items []d.ProcessInstance) types.Keys {
	keys := make(types.Keys, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			keys = append(keys, item.Key)
		}
	}
	return keys.Unique()
}

func processInstanceRootKeys(items []d.ProcessInstance) (types.Keys, bool) {
	roots := make(types.Keys, 0, len(items))
	for _, item := range items {
		switch {
		case item.RootProcessInstanceKey != "":
			roots = append(roots, item.RootProcessInstanceKey)
		case item.ParentKey == "" && item.Key != "":
			roots = append(roots, item.Key)
		default:
			return nil, false
		}
	}
	return roots.Unique(), len(roots) > 0
}

func formatPartialCancellationImpactWarning(key string, plan d.DryRunPIKeyExpansion, verbose bool) string {
	warning := plan.Warning
	if warning == "" {
		warning = "one or more parent process instances were not found"
	}
	if len(plan.MissingAncestors) == 0 {
		return fmt.Sprintf("process definition %s cancellation impact check is partial: %s", key, warning)
	}
	if verbose {
		return fmt.Sprintf("process definition %s cancellation impact check is partial: %s (missing ancestor keys: %s)", key, warning, strings.Join(processMissingAncestorKeys(plan.MissingAncestors), ", "))
	}
	return fmt.Sprintf("process definition %s cancellation impact check is partial: %s (%d missing ancestor key(s); use --verbose to list keys)", key, warning, len(plan.MissingAncestors))
}

func processMissingAncestorKeys(items []d.MissingAncestor) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
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

func resourceDeleteTotals(items []d.ResourceDeleteResponse) (total, oks, noks int) {
	for _, item := range items {
		if item.Ok {
			oks++
		}
	}
	total = len(items)
	noks = total - oks
	return total, oks, noks
}
