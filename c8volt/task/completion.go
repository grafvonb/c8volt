// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"fmt"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	utsvc "github.com/grafvonb/c8volt/internal/services/usertask"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// CompleteUserTasks submits one or more explicit keys with optional shared variables.
// Nil variables omit data; an empty object is preserved. Zero workers requests the
// automatic bound. Dry-run previews locally; no-wait is unsupported. Results retain
// every unique key in selection order, including partial evidence on failure, and
// acceptance does not imply independently verified task completion.
func (c *client) CompleteUserTasks(ctx context.Context, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...options.FacadeOption) (UserTaskCompletionResults, error) {
	snapshot, err := toolx.CopyJSONMap(variables)
	if err != nil {
		return fromDomainUserTaskCompletionResults(d.UserTaskCompletionResults{}), ferr.FromDomain(fmt.Errorf("%w: user-task completion variables must be JSON serializable: %w", d.ErrValidation, err))
	}
	got, err := utsvc.CompleteUserTasks(ctx, c.utApi, append(typex.Keys(nil), keys...), snapshot, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainUserTaskCompletionResults(got), ferr.FromDomain(err)
}
