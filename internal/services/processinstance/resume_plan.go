// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceResumption freezes selected-plus-descendant observations without
// submitting mutations. Discovery retains the walker's direct-child lookup limit;
// this is neither an atomic snapshot nor evidence of later mutation acceptance.
func PlanProcessInstanceResumption(ctx context.Context, api API, keys typex.Keys, opts ...services.CallOption) (d.ProcessInstanceResumptionPlan, error) {
	if err := api.ValidateProcessInstanceResumption(); err != nil {
		return d.ProcessInstanceResumptionPlan{}, err
	}
	seeds, err := common.NormalizeProcessInstanceKeys(keys)
	if err != nil {
		return d.ProcessInstanceResumptionPlan{}, err
	}
	if len(seeds) == 0 {
		return d.ProcessInstanceResumptionPlan{}, fmt.Errorf("%w: resumption planning requires at least one process-instance key", d.ErrValidation)
	}
	if err := ctx.Err(); err != nil {
		return d.ProcessInstanceResumptionPlan{}, err
	}

	// The adapter owns timing validation, including NoWait's exemption. Its local
	// preview establishes no state observation and cannot change execution options.
	previewOpts := append([]services.CallOption(nil), opts...)
	previewOpts = append(previewOpts, services.WithDryRun())
	if _, err := api.ResumeProcessInstance(ctx, seeds[0], previewOpts...); err != nil {
		return d.ProcessInstanceResumptionPlan{}, err
	}
	discoveryOpts := append([]services.CallOption(nil), opts...)
	discoveryOpts = append(discoveryOpts, services.WithIgnoreTenant(), services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs())
	plan := d.ProcessInstanceResumptionPlan{RequestedKeys: seeds, Instances: make([]d.ProcessInstanceResumptionPlanItem, 0), TargetKeys: make(typex.Keys, 0)}
	seen := make(map[string]bool)
	for _, seed := range seeds {
		if err := ctx.Err(); err != nil {
			return d.ProcessInstanceResumptionPlan{}, err
		}
		if seen[seed] {
			continue
		}
		result, err := api.DescendantsResult(ctx, seed, discoveryOpts...)
		if err != nil {
			return d.ProcessInstanceResumptionPlan{}, fmt.Errorf("plan process-instance resumption: %w", err)
		}
		if err := validateDescendantPlanTraversal(seed, result, "resumption"); err != nil {
			return d.ProcessInstanceResumptionPlan{}, err
		}
		for _, key := range result.Keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			item := classifyResumptionPlanItem(result.Chain[key])
			plan.Instances = append(plan.Instances, item)
			if item.Classification == d.ProcessInstanceResumptionPlanEligible {
				plan.TargetKeys = append(plan.TargetKeys, key)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return d.ProcessInstanceResumptionPlan{}, err
	}
	return plan, nil
}

// classifyResumptionPlanItem keeps known observations and normalizes unknown states;
// only SUSPENDED is eligible, independently of any previous suspension operation.
func classifyResumptionPlanItem(instance d.ProcessInstance) d.ProcessInstanceResumptionPlanItem {
	item := d.ProcessInstanceResumptionPlanItem{Key: instance.Key, ObservedState: instance.State, Classification: d.ProcessInstanceResumptionPlanExcluded}
	switch instance.State {
	case d.StateSuspended:
		item.Classification = d.ProcessInstanceResumptionPlanEligible
	case d.StateActive, d.StateCompleted, d.StateCanceled, d.StateTerminated, d.StateAbsent, d.StateUnknown:
	default:
		item.ObservedState = d.StateUnknown
	}
	return item
}
