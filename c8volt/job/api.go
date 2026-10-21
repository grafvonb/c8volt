// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

// API exposes job reads, mutations, and read-only batch preparation.
type API interface {
	// PrepareJobCompletionBatch returns complete membership or incomplete diagnostic evidence and an error.
	PrepareJobCompletionBatch(ctx context.Context, request CompletionRequest, opts ...options.FacadeOption) (PreparedCompletionBatch, error)
	// ExecuteJobCompletionBatch returns every frozen outcome alongside any aggregate error.
	ExecuteJobCompletionBatch(ctx context.Context, batch PreparedCompletionBatch, wantedWorkers int, opts ...options.FacadeOption) (CompletionBatchReport, error)
	GetJob(ctx context.Context, key string, opts ...options.FacadeOption) (Job, error)
	SearchJobs(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (SearchResult, error)
	SearchJobsPages(ctx context.Context, request SearchRequest, visitor SearchPageVisitor, opts ...options.FacadeOption) (SearchPagesResult, error)
	SearchJobsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...options.FacadeOption) (Page, error)
	SearchJobsTotal(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (int64, error)
	UpdateJob(ctx context.Context, request UpdateRequest, opts ...options.FacadeOption) (UpdateResult, error)
	SubmitJobWorkerOutcome(ctx context.Context, request WorkerOutcomeRequest, opts ...options.FacadeOption) (WorkerOutcomeResult, error)
}

var _ API = (*client)(nil)
