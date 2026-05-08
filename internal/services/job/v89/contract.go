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
	LookupJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
	UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error)
}

type GenJobClient interface {
	SearchJobsWithResponse(ctx context.Context, body camundav89.SearchJobsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error)
	UpdateJobWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.UpdateJobJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error)
}

var _ API = (*Service)(nil)
var _ GenJobClient = (*camundav89.ClientWithResponses)(nil)
