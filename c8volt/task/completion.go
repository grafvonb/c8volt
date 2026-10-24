// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	utsvc "github.com/grafvonb/c8volt/internal/services/usertask"
	"github.com/grafvonb/c8volt/typex"
)

// CompleteUserTasks submits one or more explicit keys with optional shared variables.
// Nil variables omit data; an empty object is preserved. Zero workers requests the
// automatic bound. Dry-run previews locally; no-wait is unsupported. Results retain
// every unique key in selection order, including partial evidence on failure, and
// acceptance does not imply independently verified task completion.
func (c *client) CompleteUserTasks(ctx context.Context, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...options.FacadeOption) (UserTaskCompletionResults, error) {
	// The coordinator validates and snapshots caller-owned inputs before scheduling.
	got, err := utsvc.CompleteUserTasks(ctx, c.utApi, keys, variables, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainUserTaskCompletionResults(got), ferr.FromDomain(err)
}

// UserTaskCompletionPlan exposes only advisory counts; frozen execution data stays private.
// It is an in-memory approval artifact and is never rendered as the command payload.
type UserTaskCompletionPlan struct {
	ReadyCount      int
	CompletedCount  int
	UnassignedCount int
	UncheckedCount  int
	plan            utsvc.CompletionPlan
}

// PrepareUserTaskCompletions performs a single advisory lookup per unique key before approval.
func (c *client) PrepareUserTaskCompletions(ctx context.Context, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...options.FacadeOption) (UserTaskCompletionPlan, error) {
	plan, err := utsvc.PrepareUserTaskCompletions(ctx, c.utApi, keys, variables, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return UserTaskCompletionPlan{ReadyCount: plan.ReadyCount, CompletedCount: plan.CompletedCount, UnassignedCount: plan.UnassignedCount, UncheckedCount: plan.UncheckedCount, plan: plan}, ferr.FromDomain(err)
}

// ExecuteUserTaskCompletions uses the prepared observations without any additional task lookup.
func (c *client) ExecuteUserTaskCompletions(ctx context.Context, plan UserTaskCompletionPlan) (UserTaskCompletionResults, error) {
	got, err := utsvc.ExecuteUserTaskCompletions(ctx, c.utApi, plan.plan)
	return fromDomainUserTaskCompletionResults(got), ferr.FromDomain(err)
}
