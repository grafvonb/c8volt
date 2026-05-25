// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
	SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error)
	SearchJobsPage(ctx context.Context, query d.JobSearchQuery, page d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error)
	UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error)
	SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error)
}

type GenJobClient interface {
	SearchJobsWithResponse(ctx context.Context, body camundav88.SearchJobsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error)
	UpdateJobWithResponse(ctx context.Context, jobKey camundav88.JobKey, body camundav88.UpdateJobJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.UpdateJobResponse, error)
	CompleteJobWithResponse(ctx context.Context, jobKey camundav88.JobKey, body camundav88.CompleteJobJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CompleteJobResponse, error)
	ThrowJobErrorWithResponse(ctx context.Context, jobKey camundav88.JobKey, body camundav88.ThrowJobErrorJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.ThrowJobErrorResponse, error)
	FailJobWithResponse(ctx context.Context, jobKey camundav88.JobKey, body camundav88.FailJobJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.FailJobResponse, error)
}

var _ API = (*Service)(nil)
var _ GenJobClient = (*camundav88.ClientWithResponses)(nil)
