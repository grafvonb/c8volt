// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	GetJob(ctx context.Context, key string, opts ...options.FacadeOption) (Job, error)
	SearchJobs(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (SearchResult, error)
	SearchJobsPages(ctx context.Context, request SearchRequest, visitor SearchPageVisitor, opts ...options.FacadeOption) (SearchPagesResult, error)
	SearchJobsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...options.FacadeOption) (Page, error)
	SearchJobsTotal(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (int64, error)
	UpdateJob(ctx context.Context, request UpdateRequest, opts ...options.FacadeOption) (UpdateResult, error)
	SubmitJobWorkerOutcome(ctx context.Context, request WorkerOutcomeRequest, opts ...options.FacadeOption) (WorkerOutcomeResult, error)
}

var _ API = (*client)(nil)
