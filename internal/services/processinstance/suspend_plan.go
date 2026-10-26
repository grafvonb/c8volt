// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceSuspension freezes selected-plus-descendant observations without
// submitting mutations. Discovery retains the walker's direct-child lookup limit;
// this is neither an atomic snapshot nor evidence of later mutation acceptance.
func PlanProcessInstanceSuspension(ctx context.Context, api API, keys typex.Keys, opts ...services.CallOption) (d.ProcessInstanceSuspensionPlan, error) {
	if err := api.ValidateProcessInstanceSuspension(); err != nil {
		return d.ProcessInstanceSuspensionPlan{}, err
	}
	seeds, err := common.NormalizeProcessInstanceSuspensionKeys(keys)
	if err != nil {
		return d.ProcessInstanceSuspensionPlan{}, err
	}
	if len(seeds) == 0 {
		return d.ProcessInstanceSuspensionPlan{}, fmt.Errorf("%w: suspension planning requires at least one process-instance key", d.ErrValidation)
	}
	if err := ctx.Err(); err != nil {
		return d.ProcessInstanceSuspensionPlan{}, err
	}

	// The adapter owns timing validation, including NoWait's exemption. Its local
	// preview establishes no state observation and cannot change execution options.
	previewOpts := append([]services.CallOption(nil), opts...)
	previewOpts = append(previewOpts, services.WithDryRun())
	if _, err := api.SuspendProcessInstance(ctx, seeds[0], previewOpts...); err != nil {
		return d.ProcessInstanceSuspensionPlan{}, err
	}
	discoveryOpts := append([]services.CallOption(nil), opts...)
	discoveryOpts = append(discoveryOpts, services.WithIgnoreTenant(), services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs())
	plan := d.ProcessInstanceSuspensionPlan{RequestedKeys: seeds, Instances: make([]d.ProcessInstanceSuspensionPlanItem, 0), TargetKeys: make(typex.Keys, 0)}
	seen := make(map[string]bool)
	for _, seed := range seeds {
		if err := ctx.Err(); err != nil {
			return d.ProcessInstanceSuspensionPlan{}, err
		}
		if seen[seed] {
			continue
		}
		result, err := api.DescendantsResult(ctx, seed, discoveryOpts...)
		if err != nil {
			return d.ProcessInstanceSuspensionPlan{}, fmt.Errorf("plan process-instance suspension: %w", err)
		}
		if err := validateSuspensionTraversal(seed, result); err != nil {
			return d.ProcessInstanceSuspensionPlan{}, err
		}
		for _, key := range result.Keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			item := classifySuspensionPlanItem(result.Chain[key])
			plan.Instances = append(plan.Instances, item)
			if item.Classification == d.ProcessInstanceSuspensionPlanEligible {
				plan.TargetKeys = append(plan.TargetKeys, key)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return d.ProcessInstanceSuspensionPlan{}, err
	}
	return plan, nil
}

// validateSuspensionTraversal rejects incomplete scope and missing or mismatched
// observations, including duplicate members already observed by an earlier seed.
func validateSuspensionTraversal(seed string, result traversal.Result) error {
	if result.Outcome == traversal.OutcomePartial || result.Outcome == traversal.OutcomeUnresolved {
		return fmt.Errorf("%w: suspension planning requires complete descendant discovery", d.ErrPrecondition)
	}
	if result.Outcome != traversal.OutcomeComplete || result.Mode != traversal.ModeDescendants || result.StartKey != seed || result.RootKey != seed || len(result.MissingAncestors) != 0 {
		return fmt.Errorf("%w: invalid suspension descendant scope", d.ErrMalformedResponse)
	}
	foundSeed := false
	for _, key := range result.Keys {
		normalized, err := common.NormalizeProcessInstanceSuspensionKeys(typex.Keys{key})
		if err != nil || normalized[0] != key {
			return fmt.Errorf("%w: invalid discovered process-instance key", d.ErrMalformedResponse)
		}
		instance, ok := result.Chain[key]
		if !ok || instance.Key != key {
			return fmt.Errorf("%w: missing or mismatched process-instance observation", d.ErrMalformedResponse)
		}
		if key == seed {
			foundSeed = true
		}
	}
	if !foundSeed {
		return fmt.Errorf("%w: descendant scope omits selected process instance", d.ErrMalformedResponse)
	}
	return nil
}

// classifySuspensionPlanItem keeps known observations and normalizes unknown states;
// only ACTIVE is eligible and a prior SUSPENDED observation proves no new mutation.
func classifySuspensionPlanItem(instance d.ProcessInstance) d.ProcessInstanceSuspensionPlanItem {
	item := d.ProcessInstanceSuspensionPlanItem{Key: instance.Key, ObservedState: instance.State, Classification: d.ProcessInstanceSuspensionPlanExcluded}
	switch instance.State {
	case d.StateActive:
		item.Classification = d.ProcessInstanceSuspensionPlanEligible
	case d.StateSuspended:
		item.Classification = d.ProcessInstanceSuspensionPlanAlreadySuspended
	case d.StateCompleted, d.StateCanceled, d.StateTerminated, d.StateAbsent, d.StateUnknown:
	default:
		item.ObservedState = d.StateUnknown
	}
	return item
}
