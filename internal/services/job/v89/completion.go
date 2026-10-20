// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// submitBulkCompletion is the private coordinator port for a single intentional
// completion attempt. The supplied job must be the coordinator's latest recheck;
// this adapter validates payload compatibility but never re-reads or retries.
func (s *Service) submitBulkCompletion(ctx context.Context, job d.Job, variables map[string]any) (d.CompletionSubmission, error) {
	if err := ctx.Err(); err != nil {
		return d.CompletionSubmission{Status: d.CompletionSubmissionNotSubmitted, Diagnostic: err.Error()}, err
	}
	if err := common.ValidateJobCompletionPayload(job, variables); err != nil {
		return d.CompletionSubmission{Status: d.CompletionSubmissionNotSubmitted, Diagnostic: err.Error()}, err
	}
	body := newCompleteJobRequestBody(d.JobWorkerOutcomeRequest{Key: job.Key, Variables: variables})
	response, err := s.c.CompleteJobWithResponse(ctx, camunda.JobKey(job.Key), body)
	if response == nil {
		return common.ClassifyJobCompletionSubmission(nil, nil, err)
	}
	return common.ClassifyJobCompletionSubmission(response.HTTPResponse, response.Body, err)
}
