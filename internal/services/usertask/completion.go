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

// CompleteUserTasks validates and snapshots a nonempty selection, then submits
// stable-unique keys with bounded workers. Native operations must treat the shared
// snapshot as read-only. Results retain every selected key, including unattempted
// work, and report acceptance only; no reads or confirmation polling are performed.
func CompleteUserTasks(ctx context.Context, api API, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...services.CallOption) (d.UserTaskCompletionResults, error) {
	out := d.UserTaskCompletionResults{Items: make([]d.UserTaskCompletionResult, 0)}
	cfg := services.ApplyCallOptions(opts)
	if len(keys) == 0 {
		return out, fmt.Errorf("%w: user-task completion requires at least one key", d.ErrValidation)
	}
	for i, key := range keys {
		if !validCompletionKey(key) {
			return out, fmt.Errorf("%w: user-task completion key at index %d must contain exactly 16 decimal digits", d.ErrValidation, i)
		}
	}
	if wantedWorkers < 0 {
		return out, fmt.Errorf("%w: user-task completion workers must not be negative", d.ErrValidation)
	}
	if cfg.NoWait {
		return out, fmt.Errorf("%w: user-task completion does not support no-wait", d.ErrValidation)
	}
	snapshot, err := toolx.CopyJSONMap(variables)
	if err != nil {
		return out, fmt.Errorf("%w: user-task completion variables must be JSON serializable: %w", d.ErrValidation, err)
	}
	if api == nil {
		return out, fmt.Errorf("%w: user-task completion requires a user-task service", d.ErrPrecondition)
	}
	unique := keys.Unique()
	out.Items = make([]d.UserTaskCompletionResult, len(unique))
	for i, key := range unique {
		out.Items[i] = d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusNotAttempted, Submission: d.UserTaskCompletionSubmissionNotSubmitted}
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if cfg.DryRun {
		for i := range out.Items {
			out.Items[i].Status = d.UserTaskCompletionStatusPreview
		}
		return out, nil
	}
	workers := toolx.DetermineNoOfWorkers(len(unique), wantedWorkers, cfg.NoWorkerLimit)
	var stopped atomic.Bool
	// Pool cancellation stops scheduling only. Already-started native requests keep
	// the caller context, so another key's rejection cannot erase accepted evidence.
	results, _ := pool.ExecuteSlice(ctx, unique, workers, cfg.FailFast, func(scheduleCtx context.Context, key string, _ int) (completionExecution, error) {
		if scheduleCtx.Err() != nil || ctx.Err() != nil || stopped.Load() {
			return completionExecution{}, nil
		}
		result, callErr := api.CompleteUserTask(ctx, key, snapshot, opts...)
		if callErr != nil && cfg.FailFast {
			stopped.Store(true)
		}
		return completionExecution{started: true, result: result, err: callErr}, callErr
	})
	// Join real callback failures, never the pool's synthetic fail-fast cancellation.
	for i, result := range results {
		if result.started {
			out.Items[i] = result.result
			out.Items[i].Key = unique[i]
			err = errors.Join(err, result.err)
		}
	}
	if callerErr := ctx.Err(); callerErr != nil {
		err = errors.Join(err, callerErr)
	}
	return out, err
}

// validCompletionKey applies the existing user-task ASCII key grammar without trimming input.
func validCompletionKey(key string) bool {
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
