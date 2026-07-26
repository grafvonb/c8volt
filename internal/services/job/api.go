// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/job/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/job/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/job/v89"
)

// API exposes tenant-safe get and update operations for jobs.
type API interface {
	GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
	SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error)
	SearchJobsPages(ctx context.Context, query d.JobSearchQuery, visitor d.JobSearchPageVisitor, opts ...services.CallOption) (d.JobSearchPagesResult, error)
	SearchJobsPage(ctx context.Context, query d.JobSearchQuery, page d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error)
	SearchJobsTotal(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (int64, error)
	UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error)
	SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
