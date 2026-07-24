// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

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

var _ API = (*Service)(nil)
