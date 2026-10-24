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
