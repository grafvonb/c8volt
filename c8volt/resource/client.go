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
	previewPlan, err := c.PreviewDeleteProcessDefinitions(ctx, types.Keys{key}, opts...)
	if err != nil {
		return DeleteReport{Key: key, Ok: false}, err
	}
	if len(previewPlan.Items) == 0 {
		return DeleteReport{Key: key, Ok: false}, fmt.Errorf("process definition %s was not included in delete impact check", key)
	}
	plan := previewPlan.Items[0]
	if !cCfg.NoStateCheck && plan.ActiveProcessInstances() > 0 {
		if !cCfg.Force {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cannot delete process definition %s with %d active process instance(s); use --force to cancel them automatically", key, plan.ActiveProcessInstances())
		}
		if err := c.cancelActiveProcessDefinitionInstances(ctx, key, plan.ActiveProcessInstances(), opts...); err != nil {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("delete process definition cancel active instances: %w", err)
		}
	}
	resp, err := c.api.Delete(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return deleteReportFromResponse(key, resp, false), ferr.FromDomain(err)
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

	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("checking active process instances for %d process definition(s); no changes are being made", len(ukeys)))
	for _, key := range ukeys {
		item, err := c.previewDeleteProcessDefinitionImpactCount(ctx, key, opts...)
		if err != nil {
			stopActivity()
			return plan, err
		}
		plan.Items = append(plan.Items, item)
	}
	stopActivity()

	if !cCfg.Force || plan.Totals().ActiveProcessInstances == 0 {
		return plan, nil
	}
	for i := range plan.Items {
		plan.Items[i].CancellationByFilter = plan.Items[i].ActiveProcessInstanceCount > 0
	}
	return plan, nil
}

func (c *client) previewDeleteProcessDefinitionImpactCount(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteProcessDefinitionPlanItem, error) {
	item := DeleteProcessDefinitionPlanItem{Key: key}
	active, err := c.countActiveProcessInstancesForDefinition(ctx, key, opts...)
	if err != nil {
		return item, err
	}
	item.ActiveProcessInstanceCount = active
	return item, nil
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

func (c *client) cancelActiveProcessDefinitionInstances(ctx context.Context, key string, activeCount int64, opts ...options.FacadeOption) error {
	if c.batchAPI == nil {
		return fmt.Errorf("batch-operation API is not available")
	}
	filter := process.ProcessInstanceFilter{ProcessDefinitionKey: key, State: process.StateActive}
	c.log.Info(fmt.Sprintf("submitting cancellation batch for %d active process instance(s) matching process definition %s before deletion", activeCount, key))
	op, err := c.batchAPI.CancelProcessInstancesBatch(ctx, filter, opts...)
	if err != nil {
		return err
	}
	c.log.Info(fmt.Sprintf("cancellation batch %s submitted for process definition %s", op.Key, key))
	completed, err := c.batchAPI.WaitBatchOperation(ctx, op.Key, opts...)
	if err != nil {
		return err
	}
	c.log.Info(fmt.Sprintf("cancellation batch %s completed with state %s for process definition %s", op.Key, completed.State, key))
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
