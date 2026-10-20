// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/job/completion"
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

// PrepareJobCompletionBatch delegates read-only traversal to the shared coordinator
// while retaining this adapter's key-only authorization and normalized page evidence.
func (s *Service) PrepareJobCompletionBatch(ctx context.Context, request d.CompletionRequest, opts ...services.CallOption) (d.PreparedCompletionBatch, error) {
	return completion.Prepare(ctx, request, common.EffectiveTenant(s.cfg), s.cfg.App.Backoff, completion.ReadPorts{Get: s.GetJob, Search: s.SearchJobsPage}, opts...)
}
