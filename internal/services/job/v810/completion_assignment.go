// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
)

// submitJobAssignmentCompletion uses the same one-shot evidence workflow for scalar and collection intent.
func (s *Service) submitJobAssignmentCompletion(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
	result := d.JobWorkerOutcomeResult{Key: request.Key, Mode: request.Mode, RequestedBusinessID: toolx.CopyPtr(request.SetBusinessID), Assignment: &d.CompletionAssignment{Status: "not_submitted"}, ConfirmationStatus: "skipped"}
	if request.Mode != d.JobWorkerOutcomeCompletion {
		return result, fmt.Errorf("%w: business-ID assignment requires completion", d.ErrValidation)
	}
	batch, err := s.PrepareJobCompletionBatch(ctx, d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputScalar, Keys: []string{request.Key}}, Variables: request.Variables, SetBusinessID: request.SetBusinessID}, opts...)
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	if request.SkipConfirmation {
		opts = append(opts, services.WithNoWait())
	}
	report, err := s.ExecuteJobCompletionBatch(ctx, batch, 1, opts...)
	if len(report.Outcomes) > 0 {
		out := report.Outcomes[0]
		result.Assignment = out.Assignment
		result.Submission = &out.Submission
		result.Observation = &out.Observation
		result.MutationAccepted = out.Submission.Status == d.CompletionSubmissionAccepted
	}
	if err != nil {
		result.MutationError = err.Error()
	}
	return result, err
}
