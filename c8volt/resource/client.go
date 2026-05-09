// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/batchoperation"
	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/poller"
	types "github.com/grafvonb/c8volt/typex"
)

type client struct {
	api      rsvc.API
	papi     process.API
	batchAPI batchoperation.API
	log      *slog.Logger
}

func New(api rsvc.API, papi process.API, batchAPI batchoperation.API, log *slog.Logger) API {
	return &client{api: api, papi: papi, batchAPI: batchAPI, log: log}
}

func (c *client) GetResource(ctx context.Context, key string, opts ...options.FacadeOption) (Resource, error) {
	resource, err := c.api.Get(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Resource{}, ferr.FromDomain(err)
	}
	return fromResource(resource), nil
}

func (c *client) DeployProcessDefinition(ctx context.Context, units []DeploymentUnitData, opts ...options.FacadeOption) ([]ProcessDefinitionDeployment, error) {
	pdd, err := c.api.Deploy(ctx, toDeploymentUnitDatas(units), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromProcessDefinitionDeployment(pdd), nil
}

func (c *client) DeleteProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	c.log.Info(fmt.Sprintf("checking delete impact for process definition %s", key))
	previewPlan, err := c.PreviewDeleteProcessDefinitions(ctx, types.Keys{key}, opts...)
	if err != nil {
		return DeleteReport{Key: key, Ok: false}, err
	}
	if len(previewPlan.Items) == 0 {
		return DeleteReport{Key: key, Ok: false}, fmt.Errorf("process definition %s was not included in delete impact check", key)
	}
	plan := previewPlan.Items[0]
	c.log.Info(fmt.Sprintf("delete impact for process definition %s: %d active process instance(s)", key, plan.ActiveProcessInstances()))
	if !cCfg.NoStateCheck && plan.ActiveProcessInstances() > 0 {
		if !cCfg.Force {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cannot delete process definition %s with %d active process instance(s); use --force to cancel them automatically", key, plan.ActiveProcessInstances())
		}
		c.log.Info(fmt.Sprintf("force enabled for process definition %s; cancelling active process instances before deletion", key))
		if err := c.cancelProcessDefinitionActiveInstances(ctx, key, plan, opts...); err != nil {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("delete process definition cancel active instances: %w", err)
		}
		if err := c.waitForActiveProcessDefinitionInstancesDrained(ctx, key, opts...); err != nil {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("delete process definition wait for active instances to drain: %w", err)
		}
		if err := c.deleteProcessDefinitionProcessInstances(ctx, key, plan, opts...); err != nil {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("delete process definition process-instance history: %w", err)
		}
	}
	c.log.Info(fmt.Sprintf("submitting process-definition resource deletion for %s with associated history", key))
	resp, err := c.api.Delete(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return deleteReportFromResponse(key, resp, false), ferr.FromDomain(err)
	}
	if resp.BatchOperationKey != "" {
		c.log.Info(fmt.Sprintf("process-definition resource deletion for %s accepted; history deletion batch %s state %s", key, resp.BatchOperationKey, resp.BatchState))
	} else {
		c.log.Info(fmt.Sprintf("process-definition resource deletion for %s completed with status %s", key, resp.Status))
	}
	return deleteReportFromResponse(key, resp, resp.Ok), nil
}

func deleteReportFromResponse(key string, resp d.ResourceDeleteResponse, ok bool) DeleteReport {
	return DeleteReport{
		Key:               key,
		Ok:                ok,
		StatusCode:        resp.StatusCode,
		Status:            resp.Status,
		DeleteHistory:     resp.DeleteHistory,
		BatchOperationKey: resp.BatchOperationKey,
		BatchState:        resp.BatchState,
	}
}

func (c *client) PreviewDeleteProcessDefinitions(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (DeleteProcessDefinitionPlan, error) {
	ukeys := keys.Unique()
	cCfg := options.ApplyFacadeOptions(opts)

	plan := DeleteProcessDefinitionPlan{
		Items:                 make([]DeleteProcessDefinitionPlanItem, 0, len(ukeys)),
		StateCheckSkipped:     cCfg.NoStateCheck,
		ProcessDefinitionKeys: append([]string(nil), ukeys...),
	}
	if cCfg.NoStateCheck {
		stopActivity := logging.StartActivity(ctx, fmt.Sprintf("checking delete impact for %d process definition(s); process-instance state check is skipped; no changes are being made", len(ukeys)))
		defer stopActivity()
		for _, key := range ukeys {
			plan.Items = append(plan.Items, DeleteProcessDefinitionPlanItem{Key: key})
		}
		return plan, nil
	}

	activityMsg := fmt.Sprintf("checking active process instances for %d process definition(s); no changes are being made", len(ukeys))
	if cCfg.Force {
		activityMsg = fmt.Sprintf("checking active process instances and cancellation roots for %d process definition(s); no changes are being made", len(ukeys))
	}
	stopActivity := logging.StartActivity(ctx, activityMsg)
	for _, key := range ukeys {
		item, err := c.previewDeleteProcessDefinitionImpact(ctx, key, cCfg.Force, opts...)
		if err != nil {
			stopActivity()
			return plan, err
		}
		plan.Items = append(plan.Items, item)
	}
	stopActivity()

	return plan, nil
}

func (c *client) previewDeleteProcessDefinitionImpact(ctx context.Context, key string, force bool, opts ...options.FacadeOption) (DeleteProcessDefinitionPlanItem, error) {
	item := DeleteProcessDefinitionPlanItem{Key: key}
	active, err := c.countActiveProcessInstancesForDefinition(ctx, key, opts...)
	if err != nil {
		return item, err
	}
	item.ActiveProcessInstanceCount = active
	if !force || active == 0 {
		return item, nil
	}
	activeInstances, err := c.listActiveProcessInstancesForDefinition(ctx, key, opts...)
	if err != nil {
		return item, err
	}
	activeKeys := processInstanceKeys(activeInstances)
	item.ActiveProcessInstanceKeys = activeKeys
	planKeys := activeKeys
	if roots, ok := processInstanceRootKeys(activeInstances); ok {
		planKeys = roots
	}
	cancellationPlan, err := c.papi.DryRunCancelOrDeletePlan(ctx, planKeys, 0, opts...)
	if err != nil {
		return item, err
	}
	item.CancellationPlan = cancellationPlan
	if cancellationPlan.Warning != "" {
		item.Warnings = append(item.Warnings, formatPartialCancellationImpactWarning(key, cancellationPlan, options.ApplyFacadeOptions(opts).Verbose))
	}
	return item, nil
}

func (c *client) listActiveProcessInstancesForDefinition(ctx context.Context, key string, opts ...options.FacadeOption) ([]process.ProcessInstance, error) {
	const pageSize int32 = 500

	filter := process.ProcessInstanceFilter{ProcessDefinitionKey: key, State: process.StateActive}
	pageReq := process.ProcessInstancePageRequest{Size: pageSize}
	var items []process.ProcessInstance
	for {
		page, err := c.papi.SearchProcessInstancesPage(ctx, filter, pageReq, opts...)
		if err != nil {
			return items, err
		}
		items = append(items, page.Items...)
		if page.OverflowState != process.ProcessInstanceOverflowStateHasMore {
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

func processInstanceKeys(items []process.ProcessInstance) types.Keys {
	keys := make(types.Keys, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			keys = append(keys, item.Key)
		}
	}
	return keys.Unique()
}

func processInstanceRootKeys(items []process.ProcessInstance) (types.Keys, bool) {
	roots := make(types.Keys, 0, len(items))
	for _, item := range items {
		switch {
		case item.RootProcessInstanceKey != "":
			roots = append(roots, item.RootProcessInstanceKey)
		case item.ParentKey == "" && item.ParentProcessInstanceKey == "" && item.Key != "":
			roots = append(roots, item.Key)
		default:
			return nil, false
		}
	}
	return roots.Unique(), len(roots) > 0
}

func (c *client) countActiveProcessInstancesForDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (int64, error) {
	statOpts := append([]options.FacadeOption{}, opts...)
	statOpts = append(statOpts, options.WithStat())
	pd, err := c.papi.GetProcessDefinition(ctx, key, statOpts...)
	if err != nil {
		return 0, ferr.FromDomain(err)
	}
	if pd.Statistics == nil {
		return 0, fmt.Errorf("active process-instance impact check for process definition %s did not return statistics", key)
	}
	return pd.Statistics.Active, nil
}

func (c *client) cancelProcessDefinitionActiveInstances(ctx context.Context, key string, plan DeleteProcessDefinitionPlanItem, opts ...options.FacadeOption) error {
	roots := plan.CancellationPlan.Roots.Unique()
	if len(roots) == 0 {
		return fmt.Errorf("no root process instances found to cancel for process definition %s", key)
	}
	affected := len(plan.CancellationPlan.Collected.Unique())
	if affected == 0 {
		affected = len(plan.ActiveProcessInstanceKeys)
	}
	c.log.Info(fmt.Sprintf("cancelling %d root process instance(s), affecting %d process instance(s), for process definition %s before deletion", len(roots), affected, key))
	cancelOpts := append([]options.FacadeOption{}, opts...)
	cancelOpts = append(cancelOpts, options.WithAffectedProcessInstanceCount(affected))
	reports, err := c.papi.CancelProcessInstances(ctx, roots, 0, cancelOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reports.Totals()
	if failed > 0 {
		return fmt.Errorf("cancelling root process instances for process definition %s failed for %d root request(s)", key, failed)
	}
	return nil
}

func (c *client) deleteProcessDefinitionProcessInstances(ctx context.Context, key string, plan DeleteProcessDefinitionPlanItem, opts ...options.FacadeOption) error {
	roots := plan.CancellationPlan.Roots.Unique()
	if len(roots) == 0 {
		return fmt.Errorf("no root process instances found to delete for process definition %s", key)
	}
	affected := len(plan.CancellationPlan.Collected.Unique())
	if affected == 0 {
		affected = len(plan.ActiveProcessInstanceKeys)
	}
	c.log.Info(fmt.Sprintf("deleting historical data for %d process instance(s) in %d root tree(s) before deleting process definition %s", affected, len(roots), key))
	deleteOpts := append([]options.FacadeOption{}, opts...)
	deleteOpts = append(deleteOpts, options.WithAffectedProcessInstanceCount(affected))
	reports, err := c.papi.DeleteProcessInstances(ctx, roots, 0, deleteOpts...)
	if err != nil {
		return err
	}
	_, _, failed := reports.Totals()
	if failed > 0 {
		return fmt.Errorf("deleting process-instance tree for process definition %s failed for %d root request(s)", key, failed)
	}
	return nil
}

func (c *client) waitForActiveProcessDefinitionInstancesDrained(ctx context.Context, key string, opts ...options.FacadeOption) error {
	c.log.Info(fmt.Sprintf("waiting until process definition %s has no active process instances before deletion", key))
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		active, err := c.countActiveProcessInstancesForDefinition(ctx, key, opts...)
		if err != nil {
			return poller.JobPollStatus{}, err
		}
		if active == 0 {
			return poller.JobPollStatus{
				Success: true,
				Message: fmt.Sprintf("process definition %s has no active process instances", key),
			}, nil
		}
		c.log.Info(fmt.Sprintf("process definition %s still has %d active process instance(s); waiting before deletion", key, active))
		return poller.JobPollStatus{
			Success: false,
			Message: fmt.Sprintf("process definition %s still has %d active process instance(s)", key, active),
		}, nil
	}
	if err := poller.WaitForCompletion(ctx, c.log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return err
	}
	c.log.Info(fmt.Sprintf("process definition %s has no active process instances; deleting process definition", key))
	return nil
}

func formatPartialCancellationImpactWarning(key string, plan process.DryRunPIKeyExpansion, verbose bool) string {
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

func processMissingAncestorKeys(items []process.MissingAncestor) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

//nolint:unused
func (c *client) waitForProcessDefinitionRemoval(ctx context.Context, key string, opts ...options.FacadeOption) error {
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		_, err := c.papi.GetProcessDefinition(ctx, key, opts...)
		if err != nil {
			if errors.Is(err, ferr.ErrNotFound) {
				return poller.JobPollStatus{
					Success: true,
					Message: fmt.Sprintf("process definition %s no longer listed", key),
				}, nil
			}
			return poller.JobPollStatus{}, err
		}
		return poller.JobPollStatus{
			Success: false,
			Message: fmt.Sprintf("process definition %s still listed", key),
		}, nil
	}
	return poller.WaitForCompletion(ctx, c.log, poller.DefaultCompletionTimeout, true, poll)
}
