// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API describes the job operations implemented by the native v8.10 adapter.
type API interface {
	GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
	SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error)
	SearchJobsPages(ctx context.Context, query d.JobSearchQuery, visitor d.JobSearchPageVisitor, opts ...services.CallOption) (d.JobSearchPagesResult, error)
	SearchJobsPage(ctx context.Context, query d.JobSearchQuery, page d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error)
	SearchJobsTotal(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (int64, error)
	UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error)
	SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error)
}

// GenJobClient captures the generated Camunda calls used by the v8.10 job service.
type GenJobClient interface {
	SearchJobsWithResponse(ctx context.Context, body camundav810.SearchJobsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchJobsResponse, error)
	UpdateJobWithResponse(ctx context.Context, jobKey camundav810.JobKey, body camundav810.UpdateJobJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.UpdateJobResponse, error)
	CompleteJobWithResponse(ctx context.Context, jobKey camundav810.JobKey, body camundav810.CompleteJobJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CompleteJobResponse, error)
	ThrowJobErrorWithResponse(ctx context.Context, jobKey camundav810.JobKey, body camundav810.ThrowJobErrorJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.ThrowJobErrorResponse, error)
	FailJobWithResponse(ctx context.Context, jobKey camundav810.JobKey, body camundav810.FailJobJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.FailJobResponse, error)
}

var _ API = (*Service)(nil)
var _ GenJobClient = (*camundav810.ClientWithResponses)(nil)
