// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"
	"errors"
	"fmt"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstancesIncidents completes all reads before exposing executable incident work.
func PlanProcessInstancesIncidents(ctx context.Context, api API, keys typex.Keys, opts ...services.CallOption) (d.ProcessInstancesIncidentPlan, error) {
	plan := d.ProcessInstancesIncidentPlan{Keys: slices.Clone(keys.Unique())}
	seen := map[string]bool{}
	for _, key := range plan.Keys {
		if key == "" {
			return plan, fmt.Errorf("%w: empty process instance key", d.ErrValidation)
		}
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		incidents, err := api.SearchProcessInstanceIncidents(ctx, key, opts...)
		for _, item := range domainIncidentsForProcessInstance(key, incidents) {
			if item.IncidentKey == "" {
				return plan, fmt.Errorf("%w: incident without key", d.ErrMalformedResponse)
			}
			if !seen[item.IncidentKey] {
				plan.Incidents = append(plan.Incidents, item)
				seen[item.IncidentKey] = true
			}
		}
		if err != nil {
			return plan, err
		}
		plan.CompletedKeys = append(plan.CompletedKeys, key)
	}
	if err := ctx.Err(); err != nil {
		plan.CompletedKeys = nil
		return plan, err
	}
	return plan, nil
}

// ValidateProcessInstancesIncidentPlan rejects incomplete reads, duplicate work and foreign observations.
func ValidateProcessInstancesIncidentPlan(plan d.ProcessInstancesIncidentPlan) error {
	invalid := func() error {
		return fmt.Errorf("%w: incident resolution plan is incomplete or inconsistent", d.ErrPrecondition)
	}
	if !slices.Equal(plan.Keys, plan.CompletedKeys) {
		return invalid()
	}
	keys := map[string]bool{}
	for _, key := range plan.Keys {
		if key == "" || keys[key] {
			return invalid()
		}
		keys[key] = true
	}
	incidents := map[string]bool{}
	for _, item := range plan.Incidents {
		if !keys[item.ProcessInstanceKey] || item.IncidentKey == "" || incidents[item.IncidentKey] {
			return invalid()
		}
		incidents[item.IncidentKey] = true
	}
	return nil
}

// ExecuteProcessInstancesIncidentPlan consumes reviewed observations without searching for new mutation targets.
func ExecuteProcessInstancesIncidentPlan(ctx context.Context, api API, plan d.ProcessInstancesIncidentPlan, workers int, opts ...services.CallOption) (d.ProcessInstanceResolutionResults, error) {
	if err := ValidateProcessInstancesIncidentPlan(plan); err != nil {
		return d.ProcessInstanceResolutionResults{}, err
	}
	if err := ctx.Err(); err != nil {
		return d.ProcessInstanceResolutionResults{}, err
	}
	cfg := services.ApplyCallOptions(opts)
	observations := map[string][]d.ProcessInstanceIncidentDetail{}
	for _, item := range plan.Incidents {
		observations[item.ProcessInstanceKey] = append(observations[item.ProcessInstanceKey], item)
	}
	nw := toolx.DetermineNoOfWorkers(len(plan.Keys), workers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice[string, d.ProcessInstanceResolutionResult](ctx, slices.Clone(plan.Keys), nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceResolutionResult, error) {
		return executePlannedProcessInstanceIncidents(ctx, api, key, observations[key], opts...)
	})
	// Fail-fast slots retain reviewed identities and explicitly remain unattempted.
	for i := range items {
		if items[i].ProcessInstanceKey == "" {
			items[i] = d.ProcessInstanceResolutionResult{ProcessInstanceKey: plan.Keys[i], Status: d.ProcessInstanceResolutionStatusSkipped, ConfirmationStatus: "not_attempted", DryRun: cfg.DryRun, Incidents: slices.Clone(observations[plan.Keys[i]])}
		}
	}
	out := summarizeProcessInstanceResolutionResults(items)
	out.DryRun = cfg.DryRun
	return out, errors.Join(err, ctx.Err())
}

// executePlannedProcessInstanceIncidents preserves original error classes and accepted submissions on partial failure.
func executePlannedProcessInstanceIncidents(ctx context.Context, api API, key string, incidents []d.ProcessInstanceIncidentDetail, opts ...services.CallOption) (d.ProcessInstanceResolutionResult, error) {
	cfg := services.ApplyCallOptions(opts)
	result := d.ProcessInstanceResolutionResult{ProcessInstanceKey: key, DryRun: cfg.DryRun, Incidents: slices.Clone(incidents)}
	planned := activeIncidentKeys(incidents)
	if len(planned) == 0 {
		result.Status = d.ProcessInstanceResolutionStatusSkipped
		result.ConfirmationStatus = "no_active_incidents"
		return result, nil
	}
	if cfg.DryRun {
		result.Status = d.ProcessInstanceResolutionStatusPlanned
		result.AttemptedIncidentKeys = planned
		return result, nil
	}
	var errs []error
	for _, key := range planned {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		result.AttemptedIncidentKeys = append(result.AttemptedIncidentKeys, key)
		resp, err := api.ResolveIncident(ctx, key, opts...)
		if resp.Ok {
			result.MutationSubmitted = true
			result.ResolvedIncidentKeys = append(result.ResolvedIncidentKeys, key)
		}
		if err == nil && !resp.Ok {
			err = fmt.Errorf("%w: %s", d.ErrPrecondition, mutationNotAcceptedMessage(resp.Status))
		}
		if err != nil {
			errs = append(errs, err)
			result.FailedIncidentKeys = append(result.FailedIncidentKeys, key)
		}
	}
	result.SkippedIncidentKeys = slices.Clone(planned[len(result.AttemptedIncidentKeys):])

	if len(result.ResolvedIncidentKeys) > 0 {
		if cfg.NoWait {
			result.ConfirmationStatus = "skipped"
		} else {
			_, err := api.WaitForProcessInstanceIncidentsResolved(ctx, key, slices.Clone(result.ResolvedIncidentKeys), opts...)
			if err != nil {
				errs = append(errs, err)
				result.ConfirmationStatus = "failed"
				result.FailedIncidentKeys = appendUniqueStrings(result.FailedIncidentKeys, result.ResolvedIncidentKeys...)
				result.ResolvedIncidentKeys = nil
			} else {
				result.ConfirmationStatus = "resolved"
			}
		}
	}
	err := errors.Join(errs...)
	result.Status = processInstanceResolutionStatusForFailures(result)
	if err != nil {
		result.Error = err.Error()
		if result.MutationSubmitted {
			result.Status = d.ProcessInstanceResolutionStatusPartialFailed
		} else {
			result.Status = d.ProcessInstanceResolutionStatusFailed
		}
	}
	return result, err
}
