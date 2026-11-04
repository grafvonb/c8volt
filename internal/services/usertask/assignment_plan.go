// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"fmt"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// AssignmentReplacement identifies the observed ownership requiring explicit approval.
type AssignmentReplacement struct {
	Key      string
	Assignee string
}

// AssignmentPlan freezes observations before approval; execution never repeats reads.
type AssignmentPlan struct {
	prepared       bool
	ReadyKeys      typex.Keys
	BlockedCount   int
	UnchangedCount int
	Replacements   []AssignmentReplacement
	replacements   []AssignmentReplacement
	keys           typex.Keys
	assignee       string
	workers        int
	cfg            *services.CallCfg
	results        d.UserTaskAssignmentResults
}

// PrepareUserTaskAssignments reads each protected key once. A different assignee
// blocks that key; the same assignee is an unchanged result with no mutation.
// Lookup errors abort preparation. Dry-run and override never perform reads.
func PrepareUserTaskAssignments(ctx context.Context, api API, keys typex.Keys, assignee string, workers int, opts ...services.CallOption) (AssignmentPlan, error) {
	keys = keys.Unique()
	cfg := *services.ApplyCallOptions(opts)
	// Reuse the mutation coordinator's complete local validation and unique snapshot
	// through its request-free preview path, evaluating caller options only once.
	previewCfg := cfg
	previewCfg.DryRun = true
	results, err := AssignUserTasks(ctx, api, keys, assignee, workers, func(dst *services.CallCfg) { *dst = previewCfg })
	plan := AssignmentPlan{assignee: assignee, workers: workers, cfg: &cfg, results: results}
	if err != nil {
		return plan, err
	}
	for _, item := range results.Items {
		plan.keys = append(plan.keys, item.Key)
	}
	if cfg.DryRun {
		plan.prepared = true
		return plan, nil
	}
	for i := range plan.results.Items {
		plan.results.Items[i].Status = d.UserTaskAssignmentStatusNotAttempted
	}
	if !cfg.UserTaskAssignmentOverride {
		observations, readErr := pool.ExecuteSlice(ctx, plan.keys, toolx.DetermineNoOfWorkers(len(plan.keys), workers, cfg.NoWorkerLimit), false, func(ctx context.Context, key string, _ int) (d.UserTask, error) {
			observed, err := api.GetNativeUserTask(httpc.WithoutReadRetry(ctx), key, func(dst *services.CallCfg) { *dst = cfg })
			if err != nil {
				return d.UserTask{}, err
			}
			if observed.Key != key || observed.State == "" {
				return d.UserTask{}, fmt.Errorf("%w: invalid user-task assignment lookup", d.ErrMalformedResponse)
			}
			return observed, nil
		})
		if err := errors.Join(readErr, ctx.Err()); err != nil {
			return plan, err
		}
		for i, observed := range observations {
			if observed.Assignee == assignee {
				plan.UnchangedCount++
				plan.results.Items[i].Status = d.UserTaskAssignmentStatusUnchanged
			} else if strings.TrimSpace(observed.Assignee) != "" {
				plan.BlockedCount++
				plan.replacements = append(plan.replacements, AssignmentReplacement{Key: plan.keys[i], Assignee: observed.Assignee})
				plan.results.Items[i].Status = d.UserTaskAssignmentStatusFailed
				plan.results.Items[i].ReassignmentDenied = true
				plan.results.Items[i].Error = "user task " + plan.keys[i] + " is already assigned; assignment not submitted"
			}
		}
	}
	if !cfg.FailFast || plan.BlockedCount == 0 {
		for i, item := range plan.results.Items {
			if item.Status == d.UserTaskAssignmentStatusNotAttempted {
				plan.ReadyKeys = append(plan.ReadyKeys, plan.keys[i])
			}
		}
	}
	plan.Replacements = append([]AssignmentReplacement{}, plan.replacements...)
	plan.prepared = true
	return plan, nil
}

// ExecuteUserTaskAssignments submits only the frozen eligible keys and retains
// ordered failed/not-submitted evidence for preflight blocks. Native protection
// remains enabled for tasks not explicitly approved for replacement. Approval
// authorizes replacing the current owner; the backend has no compare-and-set option.
func ExecuteUserTaskAssignments(ctx context.Context, api API, plan AssignmentPlan, allowReassignment bool) (d.UserTaskAssignmentResults, error) {
	out := d.UserTaskAssignmentResults{Items: append([]d.UserTaskAssignmentResult{}, plan.results.Items...)}
	if !plan.prepared || plan.cfg == nil || api == nil {
		return out, fmt.Errorf("%w: user-task assignment requires a prepared plan and service", d.ErrPrecondition)
	}
	if plan.cfg.DryRun {
		return out, nil
	}
	var err error
	var ready typex.Keys
	overrideKeys := map[string]bool{}
	if allowReassignment {
		for i := range out.Items {
			if out.Items[i].ReassignmentDenied {
				overrideKeys[out.Items[i].Key] = true
				out.Items[i] = d.UserTaskAssignmentResult{Key: out.Items[i].Key, Status: d.UserTaskAssignmentStatusNotAttempted, Submission: d.UserTaskAssignmentSubmissionNotSubmitted}
			}
		}
	}
	for _, item := range out.Items {
		if item.Status == d.UserTaskAssignmentStatusFailed {
			err = errors.Join(err, fmt.Errorf("%w: %s", d.ErrConflict, item.Error))
		} else if item.Status == d.UserTaskAssignmentStatusNotAttempted {
			ready = append(ready, item.Key)
		}
	}
	if len(ready) == 0 || (plan.cfg.FailFast && err != nil) {
		return out, errors.Join(err, ctx.Err())
	}
	results, executeErr := executeUserTaskAssignments(ctx, api, ready, plan.assignee, false, plan.workers, overrideKeys, func(dst *services.CallCfg) { *dst = *plan.cfg })
	byKey := make(map[string]d.UserTaskAssignmentResult, len(results.Items))
	for _, item := range results.Items {
		byKey[item.Key] = item
	}
	for i, item := range out.Items {
		if result, ok := byKey[item.Key]; ok {
			out.Items[i] = result
		}
	}
	return out, errors.Join(err, executeErr)
}
