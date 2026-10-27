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

// PlanProcessInstanceResumption discovers selected subtrees on Camunda 8.10 and
// freezes their suspended targets without mutation. It requires read/search permission,
// retains the up-to-1,000-direct-children lookup limit, and never escalates to ancestors.
// Independently suspended descendants are eligible. Later descendants are outside
// this non-atomic frozen scope; execution has no automatic rollback.
// Pass a copy of TargetKeys to ResumeProcessInstances after checking err; discovery
// observations cannot establish acceptance or confirmation of a subsequent mutation.
func (c *client) PlanProcessInstanceResumption(ctx context.Context, keys typex.Keys, opts ...options.FacadeOption) (ProcessInstanceResumptionPlan, error) {
	plan, err := pisvc.PlanProcessInstanceResumption(ctx, c.piApi, slices.Clone(keys), options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceResumptionPlan(plan), ferr.FromDomain(err)
}
