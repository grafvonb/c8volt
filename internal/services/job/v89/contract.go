// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
	SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error)
	SearchJobsPages(ctx context.Context, query d.JobSearchQuery, visitor d.JobSearchPageVisitor, opts ...services.CallOption) (d.JobSearchPagesResult, error)
	SearchJobsPage(ctx context.Context, query d.JobSearchQuery, page d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error)
	SearchJobsTotal(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (int64, error)
	UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error)
	SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error)
}

type GenJobClient interface {
	SearchJobsWithResponse(ctx context.Context, body camundav89.SearchJobsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error)
	UpdateJobWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.UpdateJobJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error)
	CompleteJobWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.CompleteJobJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CompleteJobResponse, error)
	ThrowJobErrorWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.ThrowJobErrorJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.ThrowJobErrorResponse, error)
	FailJobWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.FailJobJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.FailJobResponse, error)
}

var _ API = (*Service)(nil)
var _ GenJobClient = (*camundav89.ClientWithResponses)(nil)
