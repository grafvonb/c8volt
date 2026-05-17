// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/poller"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

// ResourceDeleteAPI submits the final resource delete request for a process definition.
type ResourceDeleteAPI interface {
	Delete(ctx context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error)
}

// DeleteProcessDefinition deletes one process definition after optional active-instance cleanup.
func DeleteProcessDefinition(ctx context.Context, api ResourceDeleteAPI, pdApi API, piApi pisvc.API, log *slog.Logger, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	cfg := services.ApplyCallOptions(opts)
	plan := d.DeleteProcessDefinitionPlanItem{Key: key}
	if cfg.NoStateCheck {
		previewPlan, err := PreviewDeleteProcessDefinitions(ctx, pdApi, piApi, log, types.Keys{key}, opts...)
		if err != nil {
			return d.ResourceDeleteResponse{}, err
		}
		if len(previewPlan.Items) == 0 {
			return d.ResourceDeleteResponse{}, fmt.Errorf("process definition %s was not included in delete impact check", key)
		}
		plan = previewPlan.Items[0]
	} else {
		var err error
		plan, err = getProcessDefinitionDeletePlanBase(ctx, pdApi, key, opts...)
		if err != nil {
			return d.ResourceDeleteResponse{}, err
		}
	}
	pdLabel := processDefinitionDeleteLogSubject(plan)
	if !cfg.NoStateCheck {
		var err error
		plan, err = completeProcessDefinitionDeleteImpact(ctx, piApi, plan, cfg.Force, cfg.Verbose, 0, opts...)
		if err != nil {
			return d.ResourceDeleteResponse{}, err
		}
	}
	log.Info(fmt.Sprintf("%s; delete impact checked; active pi %d", pdLabel, plan.ActiveProcessInstances()))
	if !cfg.NoStateCheck && plan.ActiveProcessInstances() > 0 {
		if !cfg.Force {
			return d.ResourceDeleteResponse{}, fmt.Errorf("cannot delete process definition %s with %d active process instance(s); use --force to cancel them automatically", key, plan.ActiveProcessInstances())
		}
		if err := cancelProcessDefinitionActiveInstances(ctx, piApi, log, plan, 0, opts...); err != nil {
			return d.ResourceDeleteResponse{}, fmt.Errorf("delete process definition cancel active instances: %w", err)
		}
		if err := waitForActiveProcessDefinitionInstancesDrained(ctx, pdApi, log, plan, opts...); err != nil {
			return d.ResourceDeleteResponse{}, fmt.Errorf("delete process definition wait for active instances to drain: %w", err)
		}
		if err := deleteProcessDefinitionProcessInstances(ctx, piApi, log, plan, 0, opts...); err != nil {
			return d.ResourceDeleteResponse{}, fmt.Errorf("delete process definition process-instance history: %w", err)
		}
	}
	return DeleteProcessDefinitionResource(ctx, api, log, plan, opts...)
}

// logProcessDefinitionDeleteResult emits the final resource deletion lifecycle line.
func logProcessDefinitionDeleteResult(log *slog.Logger, pdLabel string, resp d.ResourceDeleteResponse) {
	if resp.BatchOperationKey != "" {
		if resp.BatchState != "" {
			log.Info(fmt.Sprintf("%s; delete confirmed; batch %s, state %s", pdLabel, resp.BatchOperationKey, resp.BatchState))
		} else {
			log.Info(fmt.Sprintf("%s; delete accepted; batch %s", pdLabel, resp.BatchOperationKey))
		}
	} else if resp.Status != "" {
		log.Info(fmt.Sprintf("%s; delete done; status %s", pdLabel, resp.Status))
	}
}

// DeleteProcessDefinitionResource submits a process-definition resource delete from an already-validated plan item.
func DeleteProcessDefinitionResource(ctx context.Context, api ResourceDeleteAPI, log *slog.Logger, plan d.DeleteProcessDefinitionPlanItem, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	pdLabel := processDefinitionDeleteLogSubject(plan)
	log.Info(fmt.Sprintf("%s; delete request sent; history included", pdLabel))
	resp, err := api.Delete(ctx, plan.Key, opts...)
	logProcessDefinitionDeleteResult(log, pdLabel, resp)
	resp.Key = plan.Key
	return resp, err
}

// PreviewDeleteProcessDefinitions builds a non-mutating delete impact plan for process-definition keys.
func PreviewDeleteProcessDefinitions(ctx context.Context, pdApi API, piApi pisvc.API, log *slog.Logger, keys types.Keys, opts ...services.CallOption) (d.DeleteProcessDefinitionPlan, error) {
	return PreviewDeleteProcessDefinitionsWithWorkers(ctx, pdApi, piApi, log, keys, 0, opts...)
}

// PreviewDeleteProcessDefinitionsWithWorkers builds a delete impact plan while applying caller worker limits to PI dependency expansion.
func PreviewDeleteProcessDefinitionsWithWorkers(ctx context.Context, pdApi API, piApi pisvc.API, log *slog.Logger, keys types.Keys, wantedWorkers int, opts ...services.CallOption) (d.DeleteProcessDefinitionPlan, error) {
	ukeys := keys.Unique()
	cfg := services.ApplyCallOptions(opts)
	plan := d.DeleteProcessDefinitionPlan{
		Items:                 make([]d.DeleteProcessDefinitionPlanItem, 0, len(ukeys)),
		StateCheckSkipped:     cfg.NoStateCheck,
		ProcessDefinitionKeys: append([]string(nil), ukeys...),
	}
	if cfg.NoStateCheck {
		stopActivity := logging.StartActivity(ctx, fmt.Sprintf("checking %d pd delete impact; pi state skipped, dry run", len(ukeys)))
		defer stopActivity()
		for _, key := range ukeys {
			plan.Items = append(plan.Items, d.DeleteProcessDefinitionPlanItem{Key: key})
		}
		return plan, nil
	}

	activityMsg := fmt.Sprintf("checking active pi for %d pd; dry run", len(ukeys))
	if cfg.Force {
		activityMsg = fmt.Sprintf("checking active pi and roots for %d pd; dry run", len(ukeys))
	}
	stopActivity := logging.StartActivity(ctx, activityMsg)
	defer stopActivity()
	for _, key := range ukeys {
		item, err := previewDeleteProcessDefinitionImpact(ctx, pdApi, piApi, key, cfg.Force, cfg.Verbose, wantedWorkers, opts...)
		if err != nil {
			return plan, err
		}
		plan.Items = append(plan.Items, item)
	}
	return plan, nil
}

// DeleteProcessDefinitions deletes process definitions, sharing force cleanup across overlapping process-instance trees.
func DeleteProcessDefinitions(ctx context.Context, api ResourceDeleteAPI, pdApi API, piApi pisvc.API, log *slog.Logger, keys types.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ResourceDeleteResponse, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	if cfg.Force && !cfg.NoStateCheck {
		plan, err := PreviewDeleteProcessDefinitionsWithWorkers(ctx, pdApi, piApi, log, ukeys, wantedWorkers, opts...)
		if err != nil {
			return nil, err
		}
		if err := cleanupProcessDefinitionDeletePlanForceScope(ctx, pdApi, piApi, log, plan.Items, wantedWorkers, opts...); err != nil {
			return nil, err
		}
		return DeleteProcessDefinitionResources(ctx, api, log, plan.Items, wantedWorkers, opts...)
	}
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("deleting pd: requested %d, workers %d", lk, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("deleting %d pd", lk))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.ResourceDeleteResponse](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ResourceDeleteResponse, error) {
		return DeleteProcessDefinition(ctx, api, pdApi, piApi, log, key, opts...)
	})
	if !cfg.NoWait {
		total, oks, noks := resourceDeleteTotals(rs)
		log.Info(fmt.Sprintf("pd delete done; requested %d, ok %d, failed %d", total, oks, noks))
	}
	return rs, err
}

// DeleteProcessDefinitionResources submits resource deletes for preplanned process-definition items.
func DeleteProcessDefinitionResources(ctx context.Context, api ResourceDeleteAPI, log *slog.Logger, plans []d.DeleteProcessDefinitionPlanItem, wantedWorkers int, opts ...services.CallOption) ([]d.ResourceDeleteResponse, error) {
	cfg := services.ApplyCallOptions(opts)
	lk := len(plans)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("deleting pd: requested %d, workers %d", lk, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("deleting %d pd", lk))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[d.DeleteProcessDefinitionPlanItem, d.ResourceDeleteResponse](ctx, plans, nw, cfg.FailFast, func(ctx context.Context, plan d.DeleteProcessDefinitionPlanItem, _ int) (d.ResourceDeleteResponse, error) {
		return DeleteProcessDefinitionResource(ctx, api, log, plan, opts...)
	})
	if !cfg.NoWait {
		total, oks, noks := resourceDeleteTotals(rs)
		log.Info(fmt.Sprintf("pd delete done; requested %d, ok %d, failed %d", total, oks, noks))
	}
	return rs, err
}

// FindUnrelatedProcessInstancesForDefinition returns process instances matching
// a process definition that are not part of the caller-owned cleanup scope.
func FindUnrelatedProcessInstancesForDefinition(ctx context.Context, piApi pisvc.API, processDefinitionKey string, bpmnProcessID string, allowedKeys types.Keys, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	filter := d.ProcessInstanceFilter{ProcessDefinitionKey: strings.TrimSpace(processDefinitionKey)}
	if filter.ProcessDefinitionKey == "" {
		filter.BpmnProcessId = strings.TrimSpace(bpmnProcessID)
	}
	if filter.ProcessDefinitionKey == "" && filter.BpmnProcessId == "" {
		return nil, fmt.Errorf("%w: process-definition cleanup eligibility requires process-definition key or BPMN process ID", d.ErrValidation)
	}
	items, err := piApi.SearchForProcessInstances(ctx, filter, MaxResultSize, opts...)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(allowedKeys))
	for _, key := range allowedKeys.Unique() {
		allowed[key] = struct{}{}
	}
	unrelated := make([]d.ProcessInstance, 0, len(items))
	for _, item := range items {
		if item.Key == "" {
			continue
		}
		if _, ok := allowed[item.Key]; ok {
			continue
		}
		unrelated = append(unrelated, item)
	}
	return unrelated, nil
}

// previewDeleteProcessDefinitionImpact loads metadata and completes active-instance impact expansion for one process definition.
func previewDeleteProcessDefinitionImpact(ctx context.Context, pdApi API, piApi pisvc.API, key string, force bool, verbose bool, wantedWorkers int, opts ...services.CallOption) (d.DeleteProcessDefinitionPlanItem, error) {
	item, err := getProcessDefinitionDeletePlanBase(ctx, pdApi, key, opts...)
	if err != nil {
		return item, err
	}
	return completeProcessDefinitionDeleteImpact(ctx, piApi, item, force, verbose, wantedWorkers, opts...)
}

// completeProcessDefinitionDeleteImpact adds force-only process-instance roots and affected keys to a plan item.
func completeProcessDefinitionDeleteImpact(ctx context.Context, piApi pisvc.API, item d.DeleteProcessDefinitionPlanItem, force bool, verbose bool, wantedWorkers int, opts ...services.CallOption) (d.DeleteProcessDefinitionPlanItem, error) {
	if !force || item.ActiveProcessInstances() == 0 {
		return item, nil
	}
	activeInstances, err := listActiveProcessInstancesForDefinition(ctx, piApi, item.Key, opts...)
	if err != nil {
		return item, err
	}
	activeKeys := processInstanceKeys(activeInstances)
	item.ActiveProcessInstanceKeys = activeKeys
	planKeys := activeKeys
	if roots, ok := processInstanceRootKeys(activeInstances); ok {
		planKeys = roots
	}
	cancellationPlan, err := pisvc.DryRunCancelOrDeletePlan(ctx, piApi, planKeys, wantedWorkers, opts...)
	if err != nil {
		return item, err
	}
	item.CancellationPlan = cancellationPlan
	if cancellationPlan.Warning != "" {
		item.Warnings = append(item.Warnings, formatPartialCancellationImpactWarning(item.Key, cancellationPlan, verbose))
	}
	return item, nil
}

// listActiveProcessInstancesForDefinition pages through active instances for one process definition.
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

// countActiveProcessInstancesForDefinition reloads the current active count from process-definition statistics.
func countActiveProcessInstancesForDefinition(ctx context.Context, pdApi API, key string, opts ...services.CallOption) (int64, error) {
	item, err := getProcessDefinitionDeletePlanBase(ctx, pdApi, key, opts...)
	return item.ActiveProcessInstanceCount, err
}

// getProcessDefinitionDeletePlanBase loads process-definition metadata and statistics for delete planning.
func getProcessDefinitionDeletePlanBase(ctx context.Context, pdApi API, key string, opts ...services.CallOption) (d.DeleteProcessDefinitionPlanItem, error) {
	item := d.DeleteProcessDefinitionPlanItem{Key: key}
	statOpts := append([]services.CallOption{}, opts...)
	statOpts = append(statOpts, services.WithStat())
	pd, err := pdApi.GetProcessDefinition(ctx, key, statOpts...)
	if err != nil {
		return item, err
	}
	item.BpmnProcessId = pd.BpmnProcessId
	item.ProcessVersion = pd.ProcessVersion
	item.ProcessVersionTag = pd.ProcessVersionTag
	item.TenantId = pd.TenantId
	if pd.Statistics == nil {
		return item, fmt.Errorf("active process-instance impact check for process definition %s did not return statistics", key)
	}
	item.ActiveProcessInstanceCount = pd.Statistics.Active
	return item, nil
}

// cancelProcessDefinitionActiveInstances cancels the root instances required before deleting one process definition.
func cancelProcessDefinitionActiveInstances(ctx context.Context, piApi pisvc.API, log *slog.Logger, plan d.DeleteProcessDefinitionPlanItem, wantedWorkers int, opts ...services.CallOption) error {
	key := plan.Key
	roots := plan.CancellationPlan.Roots.Unique()
	if len(roots) == 0 {
		return fmt.Errorf("no root process instances found to cancel for process definition %s", key)
	}
	affected := len(plan.CancellationPlan.Collected.Unique())
	if affected == 0 {
		affected = len(plan.ActiveProcessInstanceKeys)
	}
	log.Info(fmt.Sprintf("%s; force cancel active pi; roots %d, affected %d", processDefinitionDeleteLogSubject(plan), len(roots), affected))
	cancelOpts := append([]services.CallOption{}, opts...)
	cancelOpts = append(cancelOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err := pisvc.CancelProcessInstances(ctx, piApi, log, roots, wantedWorkers, affected, cancelOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reporterTotals(reports)
	if failed > 0 {
		return fmt.Errorf("cancelling root process instances for process definition %s failed for %d root request(s)", key, failed)
	}
	return nil
}

// deleteProcessDefinitionProcessInstances deletes process-instance history required before deleting one process definition.
func deleteProcessDefinitionProcessInstances(ctx context.Context, piApi pisvc.API, log *slog.Logger, plan d.DeleteProcessDefinitionPlanItem, wantedWorkers int, opts ...services.CallOption) error {
	key := plan.Key
	roots := plan.CancellationPlan.Roots.Unique()
	if len(roots) == 0 {
		return fmt.Errorf("no root process instances found to delete for process definition %s", key)
	}
	affected := len(plan.CancellationPlan.Collected.Unique())
	if affected == 0 {
		affected = len(plan.ActiveProcessInstanceKeys)
	}
	log.Info(fmt.Sprintf("%s; delete pi history; affected %d, roots %d", processDefinitionDeleteLogSubject(plan), affected, len(roots)))
	deleteOpts := append([]services.CallOption{}, opts...)
	deleteOpts = append(deleteOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err := pisvc.DeleteProcessInstances(ctx, piApi, log, roots, wantedWorkers, affected, deleteOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reporterTotals(reports)
	if failed > 0 {
		return fmt.Errorf("deleting process-instance tree for process definition %s failed for %d root request(s)", key, failed)
	}
	return nil
}

type processDefinitionDeleteCleanupScope struct {
	Roots    types.Keys
	Affected types.Keys
}

// cleanupProcessDefinitionDeletePlanForceScope cancels and deletes unique process-instance roots across a bulk delete plan.
func cleanupProcessDefinitionDeletePlanForceScope(ctx context.Context, pdApi API, piApi pisvc.API, log *slog.Logger, items []d.DeleteProcessDefinitionPlanItem, wantedWorkers int, opts ...services.CallOption) error {
	scope := processDefinitionDeleteCleanupScopeForPlan(items)
	if len(scope.Roots) == 0 && len(scope.Affected) == 0 {
		return nil
	}
	if len(scope.Roots) == 0 {
		return fmt.Errorf("no root process instances found to cancel for process-definition delete scope")
	}
	affected := len(scope.Affected)
	if affected == 0 {
		affected = len(scope.Roots)
	}
	log.Info(fmt.Sprintf("pd delete; force cancel active pi; roots %d, affected %d", len(scope.Roots), affected))
	cancelOpts := append([]services.CallOption{}, opts...)
	cancelOpts = append(cancelOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err := pisvc.CancelProcessInstances(ctx, piApi, log, scope.Roots, wantedWorkers, affected, cancelOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reporterTotals(reports)
	if failed > 0 {
		return fmt.Errorf("cancelling root process instances for process-definition delete scope failed for %d root request(s)", failed)
	}
	if err := waitForProcessDefinitionDeletePlanActiveInstancesDrained(ctx, pdApi, log, items, opts...); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("pd delete; delete pi history; affected %d, roots %d", affected, len(scope.Roots)))
	deleteOpts := append([]services.CallOption{}, opts...)
	deleteOpts = append(deleteOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err = pisvc.DeleteProcessInstances(ctx, piApi, log, scope.Roots, wantedWorkers, affected, deleteOpts...)
	if err != nil {
		return err
	}
	_, _, failed = reporterTotals(reports)
	if failed > 0 {
		return fmt.Errorf("deleting process-instance trees for process-definition delete scope failed for %d root request(s)", failed)
	}
	return nil
}

// processDefinitionDeleteCleanupScopeForPlan extracts unique root and affected process-instance keys from plan items.
func processDefinitionDeleteCleanupScopeForPlan(items []d.DeleteProcessDefinitionPlanItem) processDefinitionDeleteCleanupScope {
	var roots types.Keys
	var affected types.Keys
	for _, item := range items {
		roots = append(roots, item.CancellationPlan.Roots...)
		affected = append(affected, item.CancellationPlan.Collected...)
		if len(item.CancellationPlan.Collected) == 0 {
			affected = append(affected, item.ActiveProcessInstanceKeys...)
		}
	}
	return processDefinitionDeleteCleanupScope{
		Roots:    roots.Unique(),
		Affected: affected.Unique(),
	}
}

// waitForProcessDefinitionDeletePlanActiveInstancesDrained waits until all planned process definitions report zero active instances.
func waitForProcessDefinitionDeletePlanActiveInstancesDrained(ctx context.Context, pdApi API, log *slog.Logger, items []d.DeleteProcessDefinitionPlanItem, opts ...services.CallOption) error {
	log.Info("pd delete; waiting until cancelled pi are no longer active")
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		active, err := activeProcessInstanceCountForCurrentProcessDefinitions(ctx, pdApi, items, opts...)
		if err != nil {
			return poller.JobPollStatus{}, err
		}
		if active == 0 {
			return poller.JobPollStatus{Success: true, Message: "process-definition delete scope has no active process instances"}, nil
		}
		log.Info(fmt.Sprintf("pd delete; active pi %d; waiting", active))
		return poller.JobPollStatus{Success: false, Message: fmt.Sprintf("pd delete; active pi %d", active)}, nil
	}
	if err := poller.WaitForCompletion(ctx, log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return err
	}
	log.Info("pd delete; cancelled pi no longer active")
	return nil
}

// activeProcessInstanceCountForCurrentProcessDefinitions sums current active counts for planned process definitions.
func activeProcessInstanceCountForCurrentProcessDefinitions(ctx context.Context, pdApi API, items []d.DeleteProcessDefinitionPlanItem, opts ...services.CallOption) (int64, error) {
	var total int64
	for _, item := range items {
		active, err := countActiveProcessInstancesForDefinition(ctx, pdApi, item.Key, opts...)
		if err != nil {
			return 0, err
		}
		total += active
	}
	return total, nil
}

// waitForActiveProcessDefinitionInstancesDrained waits until one process definition reports zero active instances.
func waitForActiveProcessDefinitionInstancesDrained(ctx context.Context, pdApi API, log *slog.Logger, plan d.DeleteProcessDefinitionPlanItem, opts ...services.CallOption) error {
	key := plan.Key
	pdLabel := processDefinitionDeleteLogSubject(plan)
	log.Info(fmt.Sprintf("%s; waiting until cancelled pi are no longer active", pdLabel))
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		active, err := countActiveProcessInstancesForDefinition(ctx, pdApi, key, opts...)
		if err != nil {
			return poller.JobPollStatus{}, err
		}
		if active == 0 {
			return poller.JobPollStatus{Success: true, Message: fmt.Sprintf("process definition %s has no active process instances", key)}, nil
		}
		log.Info(fmt.Sprintf("%s; active pi %d; waiting", pdLabel, active))
		return poller.JobPollStatus{Success: false, Message: fmt.Sprintf("%s; active pi %d", pdLabel, active)}, nil
	}
	if err := poller.WaitForCompletion(ctx, log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("%s; cancelled pi no longer active", pdLabel))
	return nil
}

// processDefinitionDeleteLogSubject formats the stable process-definition label used in delete logs.
func processDefinitionDeleteLogSubject(plan d.DeleteProcessDefinitionPlanItem) string {
	if plan.BpmnProcessId == "" {
		return fmt.Sprintf("pd %s", plan.Key)
	}
	parts := []string{"pd", plan.Key, plan.BpmnProcessId}
	if version := processDefinitionDeleteVersionText(plan); version != "" {
		parts = append(parts, version)
	}
	if plan.TenantId != "" {
		parts = append(parts, plan.TenantId)
	}
	return strings.Join(parts, " ")
}

// processDefinitionDeleteVersionText formats process version and version tag for delete logs.
func processDefinitionDeleteVersionText(plan d.DeleteProcessDefinitionPlanItem) string {
	if plan.ProcessVersion <= 0 && plan.ProcessVersionTag == "" {
		return ""
	}
	version := "v?"
	if plan.ProcessVersion > 0 {
		version = fmt.Sprintf("v%d", plan.ProcessVersion)
	}
	if plan.ProcessVersionTag != "" {
		version += "/" + plan.ProcessVersionTag
	}
	return version
}

// processInstanceKeys extracts unique non-empty process-instance keys.
func processInstanceKeys(items []d.ProcessInstance) types.Keys {
	keys := make(types.Keys, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			keys = append(keys, item.Key)
		}
	}
	return keys.Unique()
}

// processInstanceRootKeys extracts root keys when every active instance has resolvable root context.
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

// formatPartialCancellationImpactWarning formats partial traversal warnings for delete-plan output.
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

// processMissingAncestorKeys extracts missing ancestor keys for verbose warnings.
func processMissingAncestorKeys(items []d.MissingAncestor) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

// reporterTotals counts successful and failed process-instance reports.
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

// resourceDeleteTotals counts successful and failed process-definition resource delete responses.
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
