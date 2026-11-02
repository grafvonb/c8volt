// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// PlanProcessInstanceBusinessIDAssignments prepares copied exact-key scope and prior observations.
// Empty explicit scope is valid; preview does not reserve backend eligibility.
func (c *client) PlanProcessInstanceBusinessIDAssignments(ctx context.Context, keys typex.Keys, value *string, opts ...options.FacadeOption) (BusinessIDAssignmentPlan, error) {
	plan, err := pisvc.PlanProcessInstanceBusinessIDAssignments(ctx, c.piApi, append(typex.Keys(nil), keys...), toolx.CopyPtr(value), options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainBusinessIDAssignmentPlan(plan), ferr.FromDomain(err)
}

// ExecuteProcessInstanceBusinessIDAssignments delegates one-shot execution and bounded observation.
// Partial results remain available alongside the original converted failure; NoWait skips confirmation.
func (c *client) ExecuteProcessInstanceBusinessIDAssignments(ctx context.Context, plan BusinessIDAssignmentPlan, wantedWorkers int, opts ...options.FacadeOption) (BusinessIDAssignmentResults, error) {
	result, err := pisvc.ExecuteProcessInstanceBusinessIDAssignments(ctx, c.piApi, toDomainBusinessIDAssignmentPlan(plan), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainBusinessIDAssignmentResults(result), ferr.FromDomain(err)
}
