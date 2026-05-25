// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
)

type client struct {
	api jsvc.API
	log *slog.Logger
}

func New(api jsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

func (c *client) GetJob(ctx context.Context, key string, opts ...foptions.FacadeOption) (Job, error) {
	result, err := c.api.GetJob(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Job{}, ferrors.FromDomain(err)
	}
	return fromDomainJob(result), nil
}

func (c *client) SearchJobs(ctx context.Context, request SearchRequest, opts ...foptions.FacadeOption) (SearchResult, error) {
	result, err := c.api.SearchJobs(ctx, toDomainSearchRequest(request), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainSearchResult(result)
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}

func (c *client) SearchJobsPage(ctx context.Context, request SearchRequest, page PageRequest, opts ...foptions.FacadeOption) (Page, error) {
	result, err := c.api.SearchJobsPage(ctx, toDomainSearchRequest(request), toDomainPageRequest(page), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainPage(result)
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}

func (c *client) UpdateJob(ctx context.Context, request UpdateRequest, opts ...foptions.FacadeOption) (UpdateResult, error) {
	result, err := c.api.UpdateJob(ctx, toDomainUpdateRequest(request), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainUpdateResult(result)
	if request.UpdatePlan != nil {
		plan := *request.UpdatePlan
		plan.MutationSubmitted = out.MutationAccepted
		out.Plan = &plan
	}
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}

func (c *client) SubmitJobWorkerOutcome(ctx context.Context, request WorkerOutcomeRequest, opts ...foptions.FacadeOption) (WorkerOutcomeResult, error) {
	result, err := c.api.SubmitJobWorkerOutcome(ctx, toDomainWorkerOutcomeRequest(request), foptions.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainWorkerOutcomeResult(result)
	if request.OutcomePlan != nil {
		plan := *request.OutcomePlan
		plan.MutationSubmitted = out.MutationAccepted
		out.Plan = &plan
	}
	if err != nil {
		return out, ferrors.FromDomain(err)
	}
	return out, nil
}

func fromDomainJob(result d.Job) Job {
	return Job{
		Key:                result.Key,
		State:              result.State,
		Retries:            result.Retries,
		Deadline:           result.Deadline,
		Type:               result.Type,
		Worker:             result.Worker,
		Kind:               result.Kind,
		ListenerEventType:  result.ListenerEventType,
		ProcessInstanceKey: result.ProcessInstanceKey,
		ElementInstanceKey: result.ElementInstanceKey,
		ElementId:          result.ElementId,
		ErrorCode:          result.ErrorCode,
		ErrorMessage:       result.ErrorMessage,
		TenantId:           result.TenantId,
	}
}

func toDomainSearchRequest(request SearchRequest) d.JobSearchQuery {
	return d.JobSearchQuery{
		Key:                request.Key,
		State:              request.State,
		Type:               request.Type,
		ProcessInstanceKey: request.ProcessInstanceKey,
		ElementInstanceKey: request.ElementInstanceKey,
		ElementId:          request.ElementId,
		Worker:             request.Worker,
		Retries:            request.Retries,
		Kind:               request.Kind,
		ListenerEventType:  request.ListenerEventType,
		BatchSize:          request.BatchSize,
		Limit:              request.Limit,
	}
}

func fromDomainSearchResult(result d.JobSearchResult) SearchResult {
	return SearchResult{
		Items: mapDomainJobs(result.Items),
		Limit: result.Limit,
	}
}

func toDomainPageRequest(request PageRequest) d.JobPageRequest {
	return d.JobPageRequest{
		From: request.From,
		Size: request.Size,
	}
}

func fromDomainPage(result d.JobSearchPage) Page {
	return Page{
		Items: mapDomainJobs(result.Items),
		Request: PageRequest{
			From: result.Request.From,
			Size: result.Request.Size,
		},
		OverflowState: fromDomainOverflowState(result.OverflowState),
	}
}

func fromDomainOverflowState(value d.ProcessInstanceOverflowState) OverflowState {
	switch value {
	case d.ProcessInstanceOverflowStateHasMore:
		return OverflowStateHasMore
	default:
		return OverflowStateNoMore
	}
}

func toDomainUpdateRequest(request UpdateRequest) d.JobUpdateRequest {
	return d.JobUpdateRequest{
		Key:               request.Key,
		Retries:           request.Retries,
		TimeoutMillis:     request.TimeoutMillis,
		SkipConfirmation:  request.SkipConfirmation || request.NoWait,
		ConfirmRetries:    request.ConfirmRetries,
		RequestedTimeout:  request.TimeoutRaw,
		RequestedDuration: durationValue(request.Timeout),
	}
}

func durationValue(value *time.Duration) time.Duration {
	if value == nil {
		return 0
	}
	return *value
}

func toDomainWorkerOutcomeRequest(request WorkerOutcomeRequest) d.JobWorkerOutcomeRequest {
	return d.JobWorkerOutcomeRequest{
		Key:                request.Key,
		Mode:               d.JobWorkerOutcomeMode(request.Mode),
		Message:            request.Message,
		Variables:          request.Variables,
		Retries:            request.Retries,
		RetryBackoffMillis: request.RetryBackoffMillis,
		ErrorCode:          request.ErrorCode,
		SkipConfirmation:   request.NoWait,
	}
}

func fromDomainUpdateResult(result d.JobUpdateResult) UpdateResult {
	status := "submitted"
	if result.ConfirmedRetries != nil {
		status = "confirmed"
	}
	if result.MutationError != "" {
		status = "mutation_failed"
	}
	if result.ConfirmationError != "" {
		status = "confirmation_failed"
	}
	return UpdateResult{
		Key:                result.Key,
		Status:             status,
		MutationAccepted:   result.MutationAccepted,
		ConfirmationStatus: result.ConfirmationStatus,
		SubmittedRetries:   result.SubmittedRetries,
		SubmittedTimeoutMS: result.SubmittedTimeoutMS,
		ConfirmedRetries:   result.ConfirmedRetries,
		Error:              firstNonEmpty(result.MutationError, result.ConfirmationError),
	}
}

func fromDomainWorkerOutcomeResult(result d.JobWorkerOutcomeResult) WorkerOutcomeResult {
	status := "submitted"
	if result.MutationError != "" {
		status = "mutation_failed"
	}
	return WorkerOutcomeResult{
		Key:                result.Key,
		Mode:               WorkerOutcomeMode(result.Mode),
		Status:             status,
		MutationAccepted:   result.MutationAccepted,
		ConfirmationStatus: result.ConfirmationStatus,
		SubmittedRetries:   result.SubmittedRetries,
		SubmittedBackoffMS: result.SubmittedBackoffMS,
		SubmittedErrorCode: result.SubmittedErrorCode,
		Error:              result.MutationError,
	}
}

func mapDomainJobs(items []d.Job) []Job {
	out := make([]Job, 0, len(items))
	for _, item := range items {
		out = append(out, fromDomainJob(item))
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
