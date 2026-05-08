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

func fromDomainJob(result d.Job) Job {
	return Job{
		Key:                result.Key,
		State:              result.State,
		Retries:            result.Retries,
		Deadline:           result.Deadline,
		ProcessInstanceKey: result.ProcessInstanceKey,
		ElementInstanceKey: result.ElementInstanceKey,
		ErrorCode:          result.ErrorCode,
		ErrorMessage:       result.ErrorMessage,
		TenantId:           result.TenantId,
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
