// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"fmt"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceBusinessIDAssignments validates complete input before reading exact targets.
// Read failures retain incomplete diagnostic scope and cannot authorize execution.
func PlanProcessInstanceBusinessIDAssignments(ctx context.Context, api API, keys typex.Keys, value *string, opts ...services.CallOption) (d.BusinessIDAssignmentPlan, error) {
	plan := d.BusinessIDAssignmentPlan{RequestedBusinessID: toolx.CopyPtr(value), Items: make([]d.BusinessIDAssignmentItem, 0)}
	if err := api.ValidateBusinessIDAssignment(plan.RequestedBusinessID, opts...); err != nil {
		return plan, err
	}
	normalized, err := common.NormalizeProcessInstanceKeys(keys)
	if err != nil {
		return plan, err
	}
	plan.RequestedKeys = append([]string{}, normalized...)
	for _, key := range normalized {
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		pi, err := api.GetProcessInstance(ctx, key, opts...)
		if err != nil {
			return plan, err
		}
		if pi.Key != key {
			return plan, fmt.Errorf("%w: assignment preview identity mismatch", d.ErrMalformedResponse)
		}
		plan.Items = append(plan.Items, d.BusinessIDAssignmentItem{Key: key, PreviouslyObservedBusinessID: toolx.CopyPtr(pi.BusinessId), State: pi.State})
	}
	if err := ctx.Err(); err != nil {
		return plan, err
	}
	plan.Ready = true
	return plan, nil
}

// ExecuteProcessInstanceBusinessIDAssignments snapshots and validates the entire plan before writes.
// Worker results never replace not-started slots with zero values; original causes remain inspectable.
func ExecuteProcessInstanceBusinessIDAssignments(ctx context.Context, api API, plan d.BusinessIDAssignmentPlan, wantedWorkers int, opts ...services.CallOption) (d.BusinessIDAssignmentResults, error) {
	out := d.BusinessIDAssignmentResults{Items: make([]d.BusinessIDAssignmentResult, 0)}
	value := toolx.CopyPtr(plan.RequestedBusinessID)
	if err := api.ValidateBusinessIDAssignment(value, opts...); err != nil {
		return out, err
	}
	keys, err := common.NormalizeProcessInstanceKeys(typex.Keys(plan.RequestedKeys))
	if err != nil {
		return out, err
	}
	if !plan.Ready || !slices.Equal([]string(keys), plan.RequestedKeys) || len(keys) != len(plan.Items) {
		return out, fmt.Errorf("%w: incomplete or inconsistent assignment plan", d.ErrValidation)
	}
	for i, key := range keys {
		if plan.Items[i].Key != key {
			return out, fmt.Errorf("%w: assignment plan target mismatch", d.ErrValidation)
		}
		out.Items = append(out.Items, d.BusinessIDAssignmentResult{Key: key, RequestedBusinessID: toolx.CopyPtr(value), PreviouslyObservedBusinessID: toolx.CopyPtr(plan.Items[i].PreviouslyObservedBusinessID), Status: "not_submitted", Confirmation: "skipped"})
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	cfg := services.ApplyCallOptions(opts)
	if cfg.DryRun || len(keys) == 0 {
		return out, nil
	}
	workers := toolx.DetermineNoOfWorkers(len(keys), wantedWorkers, cfg.NoWorkerLimit)
	actual, err := pool.ExecuteSlice(ctx, keys, workers, cfg.FailFast, func(ctx context.Context, key string, i int) (d.BusinessIDAssignmentResult, error) {
		item, err := api.AssignProcessInstanceBusinessID(ctx, key, toolx.CopyPtr(value), opts...)
		item.Key = key
		item.RequestedBusinessID = toolx.CopyPtr(value)
		item.PreviouslyObservedBusinessID = toolx.CopyPtr(out.Items[i].PreviouslyObservedBusinessID)
		return item, err
	})
	for i, item := range actual {
		if item.Key != "" {
			out.Items[i] = item
		}
	}
	err = errors.Join(err, ctx.Err())
	if err != nil {
		for i := range out.Items {
			if out.Items[i].Status == "not_submitted" {
				out.Items[i].Error = "assignment not started after interruption or failure"
			}
		}
	}
	return out, err
}
