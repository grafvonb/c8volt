// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"slices"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceSuspension discovers selected subtrees on Camunda 8.10 and
// freezes their active targets without mutation. It requires read/search permission,
// retains the up-to-1,000-direct-children lookup limit, and never escalates to ancestors.
// Pass a copy of TargetKeys to SuspendProcessInstances after checking err; discovery
// observations cannot establish acceptance or confirmation of a subsequent mutation.
func (c *client) PlanProcessInstanceSuspension(ctx context.Context, keys typex.Keys, opts ...options.FacadeOption) (ProcessInstanceSuspensionPlan, error) {
	plan, err := pisvc.PlanProcessInstanceSuspension(ctx, c.piApi, slices.Clone(keys), options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceSuspensionPlan(plan), ferr.FromDomain(err)
}
