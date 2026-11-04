// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// completionExecution distinguishes a native result from an unscheduled pool slot.
type completionExecution struct {
	started bool
	result  d.UserTaskCompletionResult
	err     error
}

// CompletionPlan holds frozen input and one preflight observation per unique key.
// Its private execution data cannot be altered by facade callers between check and approval.
type CompletionPlan struct {
	ReadyCount      int
	CompletedCount  int
	UnassignedCount int
	UncheckedCount  int
	keys            typex.Keys
	variables       map[string]any
	workers         int
	cfg             *services.CallCfg
	opts            []services.CallOption
	results         d.UserTaskCompletionResults
}

// CompleteUserTasks is the direct-call convenience path through the same preparation and execution.
func CompleteUserTasks(ctx context.Context, api API, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...services.CallOption) (d.UserTaskCompletionResults, error) {
	plan, err := PrepareUserTaskCompletions(ctx, api, keys, variables, wantedWorkers, opts...)
	if err != nil {
		return plan.results, err
	}
	return ExecuteUserTaskCompletions(ctx, api, plan)
}

// PrepareUserTaskCompletions validates and snapshots inputs, then reads each unique task once.
// Read failures are advisory; cancellation aborts preparation. Dry-run remains entirely local.
func PrepareUserTaskCompletions(ctx context.Context, api API, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...services.CallOption) (CompletionPlan, error) {
	plan := CompletionPlan{results: d.UserTaskCompletionResults{Items: make([]d.UserTaskCompletionResult, 0)}}
	cfg := services.ApplyCallOptions(opts)
	if len(keys) == 0 {
		return plan, fmt.Errorf("%w: user-task completion requires at least one key", d.ErrValidation)
	}
	for i, key := range keys {
		if !validUserTaskKey(key) {
			return plan, fmt.Errorf("%w: user-task completion key at index %d must contain exactly 16 decimal digits", d.ErrValidation, i)
		}
	}
	if wantedWorkers < 0 {
		return plan, fmt.Errorf("%w: user-task completion workers must not be negative", d.ErrValidation)
	}
	if cfg.NoWait {
		return plan, fmt.Errorf("%w: user-task completion does not support no-wait", d.ErrValidation)
	}
	snapshot, err := toolx.CopyJSONMap(variables)
	if err != nil {
		return plan, fmt.Errorf("%w: user-task completion variables must be JSON serializable: %w", d.ErrValidation, err)
	}
	if api == nil {
		return plan, fmt.Errorf("%w: user-task completion requires a user-task service", d.ErrPrecondition)
	}
	unique := keys.Unique()
	plan.results.Items = make([]d.UserTaskCompletionResult, len(unique))
	for i, key := range unique {
		plan.results.Items[i] = d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusNotAttempted, Submission: d.UserTaskCompletionSubmissionNotSubmitted}
	}
	plan.keys, plan.variables = unique, snapshot
	plan.workers = toolx.DetermineNoOfWorkers(len(unique), wantedWorkers, cfg.NoWorkerLimit)
	plan.cfg, plan.opts = cfg, append([]services.CallOption(nil), opts...)
	if err = ctx.Err(); err != nil {
		return plan, err
	}
	if cfg.DryRun {
		for i := range plan.results.Items {
			plan.results.Items[i].Status = d.UserTaskCompletionStatusPreview
		}
		return plan, nil
	}

	// Fail-fast applies to completion, not advisory lookups. Preserve successful
	// observations even if other keys cannot be read; never retry or poll here.
	observations, _ := pool.ExecuteSlice(ctx, unique, plan.workers, false, func(ctx context.Context, key string, _ int) (d.UserTask, error) {
		observed, readErr := api.GetNativeUserTask(httpc.WithoutReadRetry(ctx), key, opts...)
		if readErr != nil || observed.Key != key || observed.State == "" {
			return d.UserTask{}, nil
		}
		return observed, nil
	})
	if err := ctx.Err(); err != nil {
		return plan, err
	}
	for i, observed := range observations {
		if observed.State == "COMPLETED" {
			plan.CompletedCount++
			plan.results.Items[i].Status = d.UserTaskCompletionStatusFailed
			plan.results.Items[i].Error = "user task " + unique[i] + " is already completed; completion skipped; supplied variables not applied"
			continue
		}
		plan.ReadyCount++
		if observed.State == "" {
			plan.UncheckedCount++
		} else if observed.Assignee == "" {
			plan.UnassignedCount++
		}
	}
	if cfg.FailFast && plan.ReadyCount < len(unique) {
		plan.ReadyCount = 0
	}
	return plan, nil
}

// ExecuteUserTaskCompletions submits the frozen eligible selection without reading task state again.
func ExecuteUserTaskCompletions(ctx context.Context, api API, plan CompletionPlan) (d.UserTaskCompletionResults, error) {
	out := d.UserTaskCompletionResults{Items: append([]d.UserTaskCompletionResult{}, plan.results.Items...)}
	if plan.cfg == nil || api == nil {
		return out, fmt.Errorf("%w: user-task completion requires a prepared plan and service", d.ErrPrecondition)
	}
	cfg := plan.cfg
	if cfg.DryRun {
		return out, nil
	}
	var err error
	for _, item := range out.Items {
		if item.Status == d.UserTaskCompletionStatusFailed {
			err = errors.Join(err, fmt.Errorf("%w: %s", d.ErrConflict, item.Error))
		}
	}
	if ctx.Err() != nil {
		return out, errors.Join(err, ctx.Err())
	}
	if cfg.FailFast && err != nil {
		return out, err
	}
	var stopped atomic.Bool
	// Scheduling cancellation never erases evidence from requests already submitted.
	results, _ := pool.ExecuteSlice(ctx, plan.keys, plan.workers, cfg.FailFast, func(scheduleCtx context.Context, key string, i int) (completionExecution, error) {
		if out.Items[i].Status == d.UserTaskCompletionStatusFailed || scheduleCtx.Err() != nil || ctx.Err() != nil || stopped.Load() {
			return completionExecution{}, nil
		}
		result, callErr := api.CompleteUserTask(ctx, key, plan.variables, plan.opts...)
		if callErr != nil && cfg.FailFast {
			stopped.Store(true)
		}
		return completionExecution{started: true, result: result, err: callErr}, callErr
	})
	for i, result := range results {
		if result.started {
			out.Items[i] = result.result
			out.Items[i].Key = plan.keys[i]
			err = errors.Join(err, result.err)
		}
	}
	if callerErr := ctx.Err(); callerErr != nil {
		err = errors.Join(err, callerErr)
	}
	return out, err
}

// validUserTaskKey shares the exact ASCII key grammar across native task mutations without trimming input.
func validUserTaskKey(key string) bool {
	if len(key) != 16 {
		return false
	}
	for i := range len(key) {
		if key[i] < '0' || key[i] > '9' {
			return false
		}
	}
	return true
}
