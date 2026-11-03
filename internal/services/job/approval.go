// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"fmt"
	"reflect"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// ScalarCompletionApproval carries presented plan attachments and the privately
// frozen payload. It is invocation-local evidence, never a backend authorization.
type ScalarCompletionApproval struct {
	Key           string
	Mode          d.JobWorkerOutcomeMode
	SetBusinessID *string
	Variables     map[string]any
	Approved      *d.JobWorkerOutcomeRequest
}

// FreezeCompletionPayload validates scalar completion identity and snapshots exact
// JSON values before approval. Later caller edits cannot substitute approved intent.
func FreezeCompletionPayload(key string, mode d.JobWorkerOutcomeMode, request d.JobWorkerOutcomeRequest) (d.JobWorkerOutcomeRequest, error) {
	if request.Mode != d.JobWorkerOutcomeCompletion || key != request.Key || mode != d.JobWorkerOutcomeCompletion {
		return d.JobWorkerOutcomeRequest{}, approvalInputError("inconsistent completion plan")
	}
	if err := d.ValidateBusinessIDAssignment(request.SetBusinessID); err != nil {
		return d.JobWorkerOutcomeRequest{}, err
	}
	variables, err := toolx.CopyJSONMap(request.Variables)
	if err != nil {
		return d.JobWorkerOutcomeRequest{}, approvalInputError("invalid completion variables")
	}
	request.Variables = variables
	request.SetBusinessID = toolx.CopyPtr(request.SetBusinessID)
	return request, nil
}

// PrepareJobWorkerOutcomeRequest resolves approval policy before service dispatch.
// Approved completion payloads override later request edits; edited plan attachments
// fail closed. Ordinary outcomes retain their existing payload ownership semantics.
func PrepareJobWorkerOutcomeRequest(request d.JobWorkerOutcomeRequest, plan *ScalarCompletionApproval) (d.JobWorkerOutcomeRequest, error) {
	if plan != nil && (request.SetBusinessID != nil || plan.SetBusinessID != nil || plan.Approved != nil) {
		approved := plan.Approved
		if approved == nil {
			return d.JobWorkerOutcomeRequest{}, fmt.Errorf("%w: assignment requires a frozen approval plan", d.ErrValidation)
		}
		if request.Key != approved.Key || request.Mode != approved.Mode || plan.Key != approved.Key || plan.Mode != approved.Mode || !reflect.DeepEqual(plan.SetBusinessID, approved.SetBusinessID) || !reflect.DeepEqual(plan.Variables, approved.Variables) {
			return d.JobWorkerOutcomeRequest{}, fmt.Errorf("%w: completion approval changed", d.ErrValidation)
		}
		request.SetBusinessID = approved.SetBusinessID
		request.Variables = approved.Variables
	}
	if request.SetBusinessID != nil && request.Mode != d.JobWorkerOutcomeCompletion {
		return d.JobWorkerOutcomeRequest{}, fmt.Errorf("%w: business-ID assignment requires completion", d.ErrValidation)
	}
	if err := d.ValidateBusinessIDAssignment(request.SetBusinessID); err != nil {
		return d.JobWorkerOutcomeRequest{}, err
	}
	if request.SetBusinessID != nil {
		variables, err := toolx.CopyJSONMap(request.Variables)
		if err != nil {
			return d.JobWorkerOutcomeRequest{}, fmt.Errorf("%w: invalid completion variables: %v", d.ErrValidation, err)
		}
		request.Variables = variables
	}
	request.SetBusinessID = toolx.CopyPtr(request.SetBusinessID)
	return request, nil
}

// ValidateAdministrativeAssignment keeps assignment eligibility below the facade
// while allowing callers to reject malformed intent before creating a result view.
func ValidateAdministrativeAssignment(assignmentRequested bool) error {
	if assignmentRequested {
		return approvalInputError("business-ID assignment requires worker completion")
	}
	return nil
}

// approvalInputError preserves established facade diagnostic wording while carrying
// domain validation classification for non-facade callers.
type approvalInputError string

// Error returns the bounded local reason without a second classification prefix.
func (e approvalInputError) Error() string { return string(e) }

// Unwrap makes local approval failures inspectable as domain validation errors.
func (e approvalInputError) Unwrap() error { return d.ErrValidation }
