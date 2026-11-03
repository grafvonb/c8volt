// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"fmt"
	"net/http"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
)

// ValidateBusinessIDAssignment checks required intent and bounded observation before any I/O.
func (s *Service) ValidateBusinessIDAssignment(value *string, opts ...services.CallOption) error {
	if value == nil {
		return fmt.Errorf("%w: direct business ID assignment requires a value", d.ErrValidation)
	}
	if err := d.ValidateBusinessIDAssignment(value); err != nil {
		return err
	}
	cfg := services.ApplyCallOptions(opts)
	if !cfg.NoWait {
		if s.cfg == nil {
			return fmt.Errorf("%w: assignment requires backoff configuration", d.ErrValidation)
		}
		if err := s.cfg.App.Backoff.Validate(); err != nil {
			return fmt.Errorf("%w: assignment backoff: %w", d.ErrValidation, err)
		}
	}
	return nil
}

// AssignProcessInstanceBusinessID submits exactly once, never redirects children, and observes only known acceptance.
func (s *Service) AssignProcessInstanceBusinessID(ctx context.Context, key string, value *string, opts ...services.CallOption) (d.BusinessIDAssignmentResult, error) {
	result := d.BusinessIDAssignmentResult{Key: key, RequestedBusinessID: toolx.CopyPtr(value), Status: "not_submitted", Confirmation: "skipped"}
	if err := s.ValidateBusinessIDAssignment(result.RequestedBusinessID, opts...); err != nil {
		return assignmentFailure(result, err)
	}
	keys, err := common.NormalizeProcessInstanceKeys(typex.Keys{key})
	if err != nil {
		return assignmentFailure(result, err)
	}
	key = keys[0]
	result.Key = key
	if err := ctx.Err(); err != nil {
		return assignmentFailure(result, err)
	}
	cfg := services.ApplyCallOptions(opts)
	if cfg.DryRun {
		return result, nil
	}
	// The generated operation has no success JSON. Only its documented 204 acknowledgement establishes acceptance.
	resp, callErr := s.cc.AssignProcessInstanceBusinessIdWithResponse(ctx, key, camunda.AssignProcessInstanceBusinessIdJSONRequestBody{BusinessId: *result.RequestedBusinessID})
	var response *http.Response
	var body []byte
	if resp != nil {
		response = resp.HTTPResponse
		body = resp.Body
	}
	if callErr == nil && response != nil && response.StatusCode >= 200 && response.StatusCode < 300 && (response.StatusCode != http.StatusNoContent || len(body) != 0) {
		callErr = fmt.Errorf("%w: unexpected assignment acknowledgement", d.ErrMalformedResponse)
	}
	submission, err := common.ClassifyJobCompletionSubmission(response, body, callErr)
	result.HTTPStatus = toolx.CopyPtr(submission.HTTPStatus)
	switch submission.Status {
	case d.CompletionSubmissionAccepted:
		result.Status = "accepted"
	case d.CompletionSubmissionRejected:
		result.Status = "rejected"
	default:
		result.Status = "unknown"
	}
	if err != nil {
		return assignmentFailure(result, err)
	}
	if cfg.NoWait {
		return result, nil
	}
	result.Confirmation = "unconfirmed"
	observed, err := waiter.WaitForProcessInstanceBusinessIDAssignment(ctx, s, s.cfg.App.Backoff, key, *result.RequestedBusinessID, opts...)
	result.HasObservation = observed.HasObservation
	result.ObservedBusinessID = toolx.CopyPtr(observed.BusinessID)
	if err != nil {
		return assignmentFailure(result, err)
	}
	result.Status = "confirmed"
	result.Confirmation = "confirmed"
	return result, nil
}

// assignmentFailure retains the original classified cause and machine evidence without logging transport details.
func assignmentFailure(result d.BusinessIDAssignmentResult, err error) (d.BusinessIDAssignmentResult, error) {
	err = fmt.Errorf("assign process instance business ID: %w", err)
	result.Error = err.Error()
	return result, err
}
