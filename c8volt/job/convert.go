// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"fmt"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// toDomainCompletionSelection mechanically copies selection provenance and optional zero retries.
func toDomainCompletionSelection(s CompletionSelection) d.CompletionSelection {
	filter := d.CompletionFilter(s.Filter)
	filter.Retries = toolx.CopyPtr(s.Filter.Retries)
	return d.CompletionSelection{Mode: d.CompletionSelectionMode(s.Mode), InputForm: d.CompletionInputForm(s.InputForm), Keys: slices.Clone(s.Keys), Filter: filter, BatchSize: s.BatchSize, Limit: s.Limit}
}

// fromDomainCompletionSelection isolates mutable input provenance from service-owned values.
func fromDomainCompletionSelection(s d.CompletionSelection) CompletionSelection {
	filter := CompletionFilter(s.Filter)
	filter.Retries = toolx.CopyPtr(s.Filter.Retries)
	return CompletionSelection{Mode: CompletionSelectionMode(s.Mode), InputForm: CompletionInputForm(s.InputForm), Keys: slices.Clone(s.Keys), Filter: filter, BatchSize: s.BatchSize, Limit: s.Limit}
}

// toDomainCompletionRequest snapshots JSON-compatible payloads without shared nested maps or float coercion.
func toDomainCompletionRequest(r CompletionRequest) (d.CompletionRequest, error) {
	variables, err := toolx.CopyJSONMap(r.Variables)
	if err != nil {
		return d.CompletionRequest{}, fmt.Errorf("%w: invalid completion variables: %v", d.ErrValidation, err)
	}
	return d.CompletionRequest{Selection: toDomainCompletionSelection(r.Selection), Variables: variables}, nil
}

// fromDomainCompletionCandidate preserves observed facts without synthesizing lifecycle data.
func fromDomainCompletionCandidate(c d.CompletionCandidate) CompletionCandidate {
	return CompletionCandidate(c)
}

// fromDomainPreparedCompletionBatch preserves both complete plans and incomplete observations.
func fromDomainPreparedCompletionBatch(b d.PreparedCompletionBatch) (PreparedCompletionBatch, error) {
	variables, err := toolx.CopyJSONMap(b.Variables)
	out := PreparedCompletionBatch{Selection: fromDomainCompletionSelection(b.Selection), DiscoveryTenant: b.DiscoveryTenant, TenantIDs: slices.Clone(b.TenantIDs), Candidates: toolx.MapSlice(b.Candidates, fromDomainCompletionCandidate), Status: CompletionPreparationStatus(b.Status), CompletionReason: CompletionReason(b.CompletionReason), PageCount: b.PageCount, ObservedCount: b.ObservedCount, Limited: b.Limited, Variables: variables}
	if err != nil {
		return out, fmt.Errorf("%w: invalid prepared completion variables: %v", d.ErrMalformedResponse, err)
	}
	return out, nil
}

// toDomainPreparedCompletionBatch snapshots the public in-memory plan before service validation.
func toDomainPreparedCompletionBatch(b PreparedCompletionBatch) (d.PreparedCompletionBatch, error) {
	variables, err := toolx.CopyJSONMap(b.Variables)
	out := d.PreparedCompletionBatch{Selection: toDomainCompletionSelection(b.Selection), DiscoveryTenant: b.DiscoveryTenant, TenantIDs: slices.Clone(b.TenantIDs), Candidates: toolx.MapSlice(b.Candidates, toDomainCompletionCandidate), Status: d.CompletionPreparationStatus(b.Status), CompletionReason: d.CompletionReason(b.CompletionReason), PageCount: b.PageCount, ObservedCount: b.ObservedCount, Limited: b.Limited, Variables: variables}
	if err != nil {
		return out, fmt.Errorf("%w: invalid prepared completion variables: %v", d.ErrValidation, err)
	}
	return out, nil
}

// toDomainCompletionCandidate maps frozen facts without adding eligibility policy.
func toDomainCompletionCandidate(c CompletionCandidate) d.CompletionCandidate {
	return d.CompletionCandidate(c)
}

// fromDomainCompletionOutcome preserves independent evidence axes and copies optional HTTP status.
func fromDomainCompletionOutcome(o d.CompletionOutcome) CompletionOutcome {
	return CompletionOutcome{Candidate: fromDomainCompletionCandidate(o.Candidate), Disposition: CompletionDisposition(o.Disposition), Reason: CompletionOutcomeReason(o.Reason), Submission: CompletionSubmission{Status: CompletionSubmissionStatus(o.Submission.Status), HTTPStatus: toolx.CopyPtr(o.Submission.HTTPStatus), Diagnostic: o.Submission.Diagnostic}, Observation: CompletionObservation{Status: CompletionObservationStatus(o.Observation.Status), State: o.Observation.State, Phase: CompletionObservationPhase(o.Observation.Phase), Diagnostic: o.Observation.Diagnostic}}
}

// fromDomainCompletionBatchReport maps partial and successful reports without interpreting errors or counts.
func fromDomainCompletionBatchReport(r d.CompletionBatchReport) CompletionBatchReport {
	return CompletionBatchReport{Status: CompletionBatchStatus(r.Status), Selection: fromDomainCompletionSelection(r.Selection), DiscoveryTenant: r.DiscoveryTenant, TenantIDs: slices.Clone(r.TenantIDs), CompletionReason: CompletionReason(r.CompletionReason), PageCount: r.PageCount, ObservedCount: r.ObservedCount, Limited: r.Limited, Counts: CompletionCounts(r.Counts), Candidates: toolx.MapSlice(r.Candidates, fromDomainCompletionCandidate), Outcomes: toolx.MapSlice(r.Outcomes, fromDomainCompletionOutcome)}
}
