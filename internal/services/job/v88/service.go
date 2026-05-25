// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/job/waiter"
)

type Service struct {
	c   GenJobClient
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) Client() GenJobClient   { return s.c }
func (s *Service) Config() *config.Config { return s.cfg }
func (s *Service) Logger() *slog.Logger   { return s.log }

type Option func(*Service)

func WithClient(c GenJobClient) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav88.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{c: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.c)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error) {
	_ = services.ApplyCallOptions(opts)

	jobKeyFilter, err := newJobKeyEqFilterPtr(key)
	if err != nil {
		return d.Job{}, fmt.Errorf("building job key filter: %w", err)
	}
	page := newSearchQueryPageRequest(2)
	resp, err := s.c.SearchJobsWithResponse(ctx, camundav88.SearchJobsJSONRequestBody{
		Filter: &camundav88.JobFilter{
			JobKey: jobKeyFilter,
		},
		Page: &page,
	})
	if err != nil {
		return d.Job{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Job{}, err
	}
	return requireSingleJob(payload.Items, key)
}

func (s *Service) SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
	_ = services.ApplyCallOptions(opts)

	filter, err := newJobSearchFilter(query)
	if err != nil {
		return d.JobSearchResult{}, err
	}
	limit := query.Limit
	if limit <= 0 {
		limit = consts.MaxPISearchSize
	}
	page := newSearchQueryPageRequest(limit)
	resp, err := s.c.SearchJobsWithResponse(ctx, camundav88.SearchJobsJSONRequestBody{
		Filter: filter,
		Page:   &page,
	})
	if err != nil {
		return d.JobSearchResult{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.JobSearchResult{}, err
	}
	return d.JobSearchResult{
		Items: fromJobSearchResults(payload.Items),
		Limit: limit,
	}, nil
}

func (s *Service) UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error) {
	cCfg := services.ApplyCallOptions(opts)
	result := d.JobUpdateResult{
		Key:                  request.Key,
		SubmittedRetries:     request.Retries,
		SubmittedTimeoutMS:   request.TimeoutMillis,
		ConfirmationStatus:   "not_applicable",
		UnsupportedOperation: false,
	}
	body := camundav88.UpdateJobJSONRequestBody{
		Changeset: camundav88.JobChangeset{
			Retries: request.Retries,
			Timeout: request.TimeoutMillis,
		},
	}
	resp, err := services.RetryCamundaMutation(ctx, s.log, "update job", func(ctx context.Context) (*camundav88.UpdateJobResponse, *http.Response, []byte, error) {
		resp, err := s.c.UpdateJobWithResponse(ctx, camundav88.JobKey(request.Key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	result.MutationAccepted = true
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.MutationAccepted = false
		result.MutationError = err.Error()
		return result, err
	}
	if cCfg.NoWait || request.SkipConfirmation || !request.ConfirmRetries || request.Retries == nil {
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	confirmed, err := waiter.WaitForRetries(ctx, s, s.cfg, s.log, request.Key, *request.Retries, opts...)
	if err != nil {
		result.ConfirmationStatus = "failed"
		result.ConfirmationError = err.Error()
		return result, nil
	}
	result.ConfirmationStatus = "confirmed"
	result.ConfirmedRetries = &confirmed.Retries
	return result, nil
}

func (s *Service) SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
	_ = services.ApplyCallOptions(opts)
	if request.Mode != d.JobWorkerOutcomeTechnicalFailure {
		return d.JobWorkerOutcomeResult{}, fmt.Errorf("%w: job worker outcome service implementation is pending", d.ErrUnsupported)
	}
	result := d.JobWorkerOutcomeResult{
		Key:                request.Key,
		Mode:               request.Mode,
		SubmittedRetries:   request.Retries,
		SubmittedBackoffMS: request.RetryBackoffMillis,
		ConfirmationStatus: "not_applicable",
	}
	body := newFailJobRequestBody(request)
	resp, err := services.RetryCamundaMutation(ctx, s.log, "fail job", func(ctx context.Context) (*camundav88.FailJobResponse, *http.Response, []byte, error) {
		resp, err := s.c.FailJobWithResponse(ctx, camundav88.JobKey(request.Key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	result.MutationAccepted = true
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.MutationAccepted = false
		result.MutationError = err.Error()
		return result, err
	}
	result.ConfirmationStatus = "skipped"
	return result, nil
}

// newFailJobRequestBody keeps generated v8.8 failure request details inside the
// service adapter while preserving explicit zero retries.
func newFailJobRequestBody(request d.JobWorkerOutcomeRequest) camundav88.FailJobJSONRequestBody {
	body := camundav88.FailJobJSONRequestBody{
		Retries:      request.Retries,
		RetryBackOff: request.RetryBackoffMillis,
	}
	if request.Message != "" {
		body.ErrorMessage = &request.Message
	}
	if request.Variables != nil {
		body.Variables = &request.Variables
	}
	return body
}

var _ waiter.JobGetter = (*Service)(nil)
