// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// assignmentExecution distinguishes native evidence from an unscheduled pool slot.
type assignmentExecution struct {
	started bool
	result  d.UserTaskAssignmentResult
	err     error
}

// AssignUserTasks validates and freezes a protected assignment for every unique explicit key.
// Zero workers chooses the automatic bound; dry-run is local and no-wait is rejected.
func AssignUserTasks(ctx context.Context, api API, keys typex.Keys, assignee string, wantedWorkers int, opts ...services.CallOption) (d.UserTaskAssignmentResults, error) {
	return executeUserTaskAssignments(ctx, api, keys, assignee, false, wantedWorkers, opts...)
}

// UnassignUserTasks removes assignments without reads, polling or implicit reassignment.
// Enabled assignment override and no-wait are invalid; ordered partial evidence survives errors.
func UnassignUserTasks(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.UserTaskAssignmentResults, error) {
	return executeUserTaskAssignments(ctx, api, keys, "", true, wantedWorkers, opts...)
}

// executeUserTaskAssignments owns both actions' validation, immutable intent and ordered scheduling.
// Scheduling cancellation stops new work, while the original caller context controls submissions.
func executeUserTaskAssignments(ctx context.Context, api API, keys typex.Keys, assignee string, unassign bool, wantedWorkers int, opts ...services.CallOption) (d.UserTaskAssignmentResults, error) {
	out := d.UserTaskAssignmentResults{Items: make([]d.UserTaskAssignmentResult, 0)}
	// Unique allocates independent storage before caller-provided options are evaluated.
	keys = keys.Unique()
	cfg := *services.ApplyCallOptions(opts)
	if len(keys) == 0 {
		return out, fmt.Errorf("%w: user-task assignment requires at least one key", d.ErrValidation)
	}
	for i, key := range keys {
		if !validUserTaskKey(key) {
			return out, fmt.Errorf("%w: user-task assignment key at index %d must contain exactly 16 decimal digits", d.ErrValidation, i)
		}
	}
	if !unassign && strings.TrimSpace(assignee) == "" {
		return out, fmt.Errorf("%w: user-task assignment requires a nonblank assignee", d.ErrValidation)
	}
	if wantedWorkers < 0 {
		return out, fmt.Errorf("%w: user-task assignment workers must not be negative", d.ErrValidation)
	}
	if cfg.NoWait {
		return out, fmt.Errorf("%w: user-task assignment and unassignment do not support no-wait", d.ErrValidation)
	}
	if unassign && cfg.UserTaskAssignmentOverride {
		return out, fmt.Errorf("%w: user-task unassignment does not support assignment override", d.ErrValidation)
	}
	if api == nil {
		return out, fmt.Errorf("%w: user-task assignment requires a user-task service", d.ErrPrecondition)
	}
	out.Items = make([]d.UserTaskAssignmentResult, len(keys))
	for i, key := range keys {
		out.Items[i] = d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusNotAttempted, Submission: d.UserTaskAssignmentSubmissionNotSubmitted}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if cfg.DryRun {
		for i := range out.Items {
			out.Items[i].Status = d.UserTaskAssignmentStatusPreview
		}
		return out, nil
	}
	// Copy evaluated values into each adapter config; never re-run caller closures or share their config pointer.
	frozenOption := func(dst *services.CallCfg) { *dst = cfg }
	workers := toolx.DetermineNoOfWorkers(len(keys), wantedWorkers, cfg.NoWorkerLimit)
	var stopped atomic.Bool
	executions, _ := pool.ExecuteSlice(ctx, keys, workers, cfg.FailFast, func(scheduleCtx context.Context, key string, _ int) (assignmentExecution, error) {
		if scheduleCtx.Err() != nil || ctx.Err() != nil || stopped.Load() {
			return assignmentExecution{}, nil
		}
		var result d.UserTaskAssignmentResult
		var err error
		if unassign {
			result, err = api.UnassignUserTask(ctx, key, frozenOption)
		} else {
			result, err = api.AssignUserTask(ctx, key, assignee, frozenOption)
		}
		if err != nil && cfg.FailFast {
			stopped.Store(true)
		}
		return assignmentExecution{started: true, result: result, err: err}, err
	})
	// Pool cancellation errors describe scheduling, not native failures. Join only real started results and the caller error.
	var err error
	for i, execution := range executions {
		if execution.started {
			out.Items[i] = execution.result
			out.Items[i].Key = keys[i]
			err = errors.Join(err, execution.err)
		}
	}
	if callerErr := ctx.Err(); callerErr != nil {
		err = errors.Join(err, callerErr)
	}
	return out, err
}
