// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"fmt"
	"slices"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	d "github.com/grafvonb/c8volt/internal/domain"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/toolx"
)

// FreezeCompletionPayload copies the approved scalar key, mode and payload before confirmation.
// Later request payload changes cannot replace approval; edited plan attachments are rejected.
func (p *UpdatePlan) FreezeCompletionPayload(r WorkerOutcomeRequest) error {
	frozen, err := jsvc.FreezeCompletionPayload(p.Key, d.JobWorkerOutcomeMode(p.Mode), mapWorkerOutcomeRequest(r))
	if err != nil {
		return ferrors.FromDomain(err)
	}
	p.Variables, _ = toolx.CopyJSONMap(frozen.Variables)
	p.SetBusinessID = toolx.CopyPtr(frozen.SetBusinessID)
	p.approvedCompletion = &frozen
	return nil
}

// fromDomainSubmissionPtr preserves optional submission details with independent HTTP storage.
func fromDomainSubmissionPtr(s *d.CompletionSubmission) *CompletionSubmission {
	if s == nil {
		return nil
	}
	return &CompletionSubmission{Status: CompletionSubmissionStatus(s.Status), HTTPStatus: toolx.CopyPtr(s.HTTPStatus), Diagnostic: s.Diagnostic}
}

// fromDomainObservationPtr mechanically maps independent completion observation.
func fromDomainObservationPtr(o *d.CompletionObservation) *CompletionObservation {
	if o == nil {
		return nil
	}
	return &CompletionObservation{Status: CompletionObservationStatus(o.Status), State: o.State, Phase: CompletionObservationPhase(o.Phase), Diagnostic: o.Diagnostic}
}

// snapshotCompletionPlan keeps returned approval evidence independent of caller storage during dispatch.
func snapshotCompletionPlan(p UpdatePlan) (UpdatePlan, error) {
	p.SetBusinessID = toolx.CopyPtr(p.SetBusinessID)
	p.Current.BusinessId = toolx.CopyPtr(p.Current.BusinessId)
	p.Items = slices.Clone(p.Items)
	variables, err := toolx.CopyJSONMap(p.Variables)
	if err != nil {
		return p, fmt.Errorf("%w: invalid approval variables", d.ErrValidation)
	}
	p.Variables = variables
	return p, nil
}

// ValidateCompletionBusinessID validates optional assignment intent without backend I/O.
// Accepted bytes are preserved; nil retains ordinary completion semantics.
func ValidateCompletionBusinessID(value *string) error {
	return ferrors.FromDomain(d.ValidateBusinessIDAssignment(value))
}
