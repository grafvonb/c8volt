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

// AssignUserTasks assigns unique explicit keys to an opaque, nonblank assignee.
// Keys must be exact 16-digit ASCII values; duplicates retain their first-input order.
// Existing assignments are protected unless WithUserTaskAssignmentOverride is enabled.
// Zero workers selects the automatic bound; dry-run previews locally and no-wait is invalid.
// Results retain selection order and partial native evidence on failure, without read-back or polling.
func (c *client) AssignUserTasks(ctx context.Context, keys typex.Keys, assignee string, wantedWorkers int, opts ...options.FacadeOption) (UserTaskAssignmentResults, error) {
	got, err := utsvc.AssignUserTasks(ctx, c.utApi, keys, assignee, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainUserTaskAssignmentResults(got), ferr.FromDomain(err)
}

// UnassignUserTasks removes assignments from unique explicit keys without preliminary reads.
// Keys must be exact 16-digit ASCII values; duplicates retain their first-input order.
// Zero workers selects the automatic bound; dry-run previews locally. No-wait and enabled
// assignment override are invalid. Ordered partial native evidence is retained on failure.
func (c *client) UnassignUserTasks(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (UserTaskAssignmentResults, error) {
	got, err := utsvc.UnassignUserTasks(ctx, c.utApi, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainUserTaskAssignmentResults(got), ferr.FromDomain(err)
}
