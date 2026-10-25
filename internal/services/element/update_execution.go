// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"errors"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
)

// ExecuteElementVariableUpdates validates and snapshots the entire approved plan
// before workers run. Submission and confirmation are separate phases; neither
// phase rediscovers destinations or resubmits an accepted mutation.
func (s *VariableUpdateService) ExecuteElementVariableUpdates(ctx context.Context, plan d.ElementVariableUpdatePlan, wantedWorkers int, opts ...services.CallOption) (d.ElementVariableUpdateResults, error) {
	frozen, err := validateAndCopyElementVariableUpdatePlan(plan)
	if err != nil {
		return d.ElementVariableUpdateResults{}, err
	}
	cfg := services.ApplyCallOptions(opts)
	if cfg.DryRun {
		return d.ElementVariableUpdateResults{Items: []d.ElementVariableUpdateResult{}}, nil
	}
	opts = append(append([]services.CallOption(nil), opts...), services.WithIgnoreTenant())
	items := make([]d.ElementVariableUpdateResult, len(frozen.Elements))
	changed := make([]int, 0, frozen.UpdateCount)
	for i, element := range frozen.Elements {
		items[i] = d.ElementVariableUpdateResult{
			Key: element.ElementKey, ProcessInstanceKey: element.ProcessInstanceKey, ScopeKey: element.ScopeKey,
			Status: d.ElementVariableUpdateStatusUnchanged, SubmissionStatus: d.ScopeVariableSubmissionNotSent,
			ConfirmationStatus: "not_required", Variables: copyElementVariableMap(element.MutationVariables),
		}
		if len(element.MutationVariables) > 0 {
			items[i].Status = d.ElementVariableUpdateStatusSkipped
			changed = append(changed, i)
		}
	}
	results := d.ElementVariableUpdateResults{Items: items}
	if len(changed) == 0 {
		return results, nil
	}
	if s.writer == nil || (!cfg.NoWait && s.variables == nil) {
		return d.ElementVariableUpdateResults{}, fmt.Errorf("%w: element variable execution requires scope writer and confirmation reader", d.ErrPrecondition)
	}
	workers := toolx.DetermineNoOfWorkers(len(changed), wantedWorkers, cfg.NoWorkerLimit)
	mutations, mutationErr := pool.ExecuteSlice(ctx, changed, workers, cfg.FailFast, func(ctx context.Context, index int, _ int) (elementVariableExecution, error) {
		element := frozen.Elements[index]
		result := items[index]
		response, err := s.writer.UpdateScopeVariables(ctx, element.ScopeKey, copyElementVariableMap(element.MutationVariables), opts...)
		result.SubmissionStatus = response.SubmissionStatus
		if result.SubmissionStatus == "" {
			result.SubmissionStatus = d.ScopeVariableSubmissionUnknown
		}
		result.MutationAccepted = result.SubmissionStatus == d.ScopeVariableSubmissionAccepted
		result.StatusCode = response.StatusCode
		result.Message = response.Status
		if result.MutationAccepted {
			result.Status = d.ElementVariableUpdateStatusSubmitted
			result.ConfirmationStatus = "not_started"
			if cfg.NoWait {
				result.ConfirmationStatus = "skipped"
			}
		} else {
			result.Status = d.ElementVariableUpdateStatusMutationFailed
			if err == nil {
				err = fmt.Errorf("%w: element %s variable update acceptance is not established", d.ErrUpstream, element.ElementKey)
			}
		}
		if err != nil {
			result.Error = err.Error()
		}
		return elementVariableExecution{attempted: true, result: result}, err
	})
	contextErr := ctx.Err()
	mutationErr = joinElementExecutionContextError(mutationErr, contextErr)
	for i, index := range changed {
		if mutations[i].attempted {
			items[index] = mutations[i].result
		}
	}
	if cfg.NoWait {
		return results, mutationErr
	}
	// A failed fail-fast phase cannot start fresh confirmation work. Accepted
	// in-flight writes retain their evidence and the interruption's original cause.
	if contextErr != nil || (cfg.FailFast && mutationErr != nil) {
		for _, index := range changed {
			if items[index].MutationAccepted && items[index].Error == "" {
				items[index].Error = mutationErr.Error()
			}
		}
		return results, mutationErr
	}
	confirming := make([]int, 0, len(changed))
	for _, index := range changed {
		if items[index].MutationAccepted && items[index].Error == "" {
			confirming = append(confirming, index)
		}
	}
	workers = toolx.DetermineNoOfWorkers(len(confirming), wantedWorkers, cfg.NoWorkerLimit)
	confirmations, confirmationErr := pool.ExecuteSlice(ctx, confirming, workers, cfg.FailFast, func(ctx context.Context, index int, _ int) (elementVariableConfirmation, error) {
		err := s.waitForElementVariables(ctx, frozen.Elements[index], opts...)
		return elementVariableConfirmation{attempted: true, err: err}, err
	})
	confirmationErr = joinElementExecutionContextError(confirmationErr, ctx.Err())
	for i, index := range confirming {
		confirmation := confirmations[i]
		if !confirmation.attempted {
			if confirmationErr != nil {
				items[index].Error = confirmationErr.Error()
			}
			continue
		}
		if confirmation.err != nil {
			items[index].Status = d.ElementVariableUpdateStatusConfirmationFailed
			items[index].ConfirmationStatus = "failed"
			items[index].Error = confirmation.err.Error()
		} else {
			items[index].Status = d.ElementVariableUpdateStatusConfirmed
			items[index].ConfirmationStatus = "confirmed"
		}
	}
	return results, errors.Join(mutationErr, confirmationErr)
}

// elementVariableExecution distinguishes locally started failures from unscheduled work.
type elementVariableExecution struct {
	attempted bool
	result    d.ElementVariableUpdateResult
}

// elementVariableConfirmation distinguishes interrupted scheduling from failed read-back.
type elementVariableConfirmation struct {
	attempted bool
	err       error
}

// joinElementExecutionContextError preserves cancellation even when the pool started no jobs.
func joinElementExecutionContextError(err, contextErr error) error {
	if contextErr == nil || errors.Is(err, contextErr) {
		return err
	}
	return errors.Join(err, contextErr)
}
