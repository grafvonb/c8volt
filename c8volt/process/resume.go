// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/typex"
)

// ResumeProcessInstance resumes only key on Camunda 8.10 through the collection
// workflow. Key must contain exactly 16 decimal digits after trimming. DryRun
// previews without I/O; NoWait reports acceptance without confirmation reads.
// Otherwise ACTIVE or COMPLETED confirms within configured bounds. Evidence is
// returned alongside any converted error, including accepted but unconfirmed work.
func (c *client) ResumeProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ResumptionResult, error) {
	result, err := c.ResumeProcessInstances(ctx, typex.Keys{key}, 1, opts...)
	if len(result.Items) == 0 {
		return ResumptionResult{}, err
	}
	return result.Items[0], err
}

// ResumeProcessInstances validates all keys and local timing before dispatching
// independent targets. Duplicate keys are normalized once in first-input order;
// selected parents and children remain independent. Empty input succeeds empty on
// supported versions. DryRun, NoWait, FailFast and NoWorkerLimit pass to services;
// wantedWorkers follows the existing worker policy (zero selects automatic limits).
// Partial results preserve started and unattempted slots, even after fail-fast or
// interruption, alongside the converted error. Acceptance does not imply confirmation.
func (c *client) ResumeProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (ResumptionResults, error) {
	result, err := pisvc.ResumeProcessInstances(ctx, c.piApi, c.log, append(typex.Keys(nil), keys...), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainResumptionResults(result), ferr.FromDomain(err)
}
