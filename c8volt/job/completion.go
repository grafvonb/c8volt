// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
)

// PrepareJobCompletionBatch maps values and options without owning discovery or
// discarding the service's partial preparation evidence when an error occurs.
func (c *client) PrepareJobCompletionBatch(ctx context.Context, request CompletionRequest, opts ...foptions.FacadeOption) (PreparedCompletionBatch, error) {
	input, err := toDomainCompletionRequest(request)
	if err != nil {
		return PreparedCompletionBatch{Status: CompletionPreparationIncomplete}, ferrors.FromDomain(err)
	}
	result, err := c.api.PrepareJobCompletionBatch(ctx, input, foptions.MapFacadeOptionsToCallOptions(opts)...)
	out, mapErr := fromDomainPreparedCompletionBatch(result)
	return out, ferrors.FromDomain(errors.Join(err, mapErr))
}

// MarshalJSON ensures the applicable collection is [] even for an empty report;
// preview has no execution outcomes and execution has no preview candidates.
func (r CompletionBatchReport) MarshalJSON() ([]byte, error) {
	type wire CompletionBatchReport
	out := wire(r)
	if r.Status == CompletionBatchPreview || r.Status == CompletionBatchPreparationFailed {
		out.Outcomes = nil
		if out.Candidates == nil {
			out.Candidates = []CompletionCandidate{}
		}
	} else {
		out.Candidates = nil
		if out.Outcomes == nil {
			out.Outcomes = []CompletionOutcome{}
		}
	}
	return json.Marshal(out)
}
