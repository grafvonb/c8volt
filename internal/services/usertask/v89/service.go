// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	tasklistv89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

const fallbackTaskSearchPageSize int32 = 2

type Service struct {
	cc  GenUserTaskClientCamunda
	ct  GenUserTaskClientTasklist
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) ClientCamunda() GenUserTaskClientCamunda { return s.cc }
func (s *Service) ClientTasklist() GenUserTaskClientTasklist {
	return s.ct
}
func (s *Service) Config() *config.Config { return s.cfg }
func (s *Service) Logger() *slog.Logger   { return s.log }

type Option func(*Service)

func WithClientCamunda(c GenUserTaskClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
		}
	}
}

func WithClientTasklist(c GenUserTaskClientTasklist) Option {
	return func(s *Service) {
		if c != nil {
			s.ct = c
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
	c, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cc: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.cc)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

// GetUserTask resolves a Camunda user task through search so tenant scoping and generated v8.9 filter shapes stay explicit.
func (s *Service) GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching user task with key %s using generated camunda client", key))
	body, err := searchUserTaskRequest(common.EffectiveTenant(s.cfg), key)
	if err != nil {
		return d.UserTask{}, fmt.Errorf("build user task search request: %w", err)
	}
	resp, err := s.cc.SearchUserTasksWithResponse(ctx, body)
	if err != nil {
		return d.UserTask{}, fmt.Errorf("search user task: %w", err)
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.UserTask{}, fmt.Errorf("search user task: %w", err)
	}
	task, err := requireSingleUserTask(payload.Items, key)
	if err != nil {
		return d.UserTask{}, err
	}
	if task.ProcessInstanceKey == "" {
		return d.UserTask{}, fmt.Errorf("%w: user task %s has no process instance key", d.ErrMalformedResponse, key)
	}
	return task, nil
}

// searchUserTaskRequest builds the v8.9 user-task search body used to find one task within the effective tenant.
func searchUserTaskRequest(tenantID, key string) (camundav89.SearchUserTasksJSONRequestBody, error) {
	tenantIDFilter, err := newStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav89.SearchUserTasksJSONRequestBody{}, err
	}
	userTaskKey := camundav89.UserTaskKey(key)
	return camundav89.SearchUserTasksJSONRequestBody{
		Filter: &camundav89.UserTaskFilter{
			TenantId:    tenantIDFilter,
			UserTaskKey: &userTaskKey,
		},
	}, nil
}

// searchFallbackTaskRequest builds the Tasklist V1 fallback search body used after the primary lookup misses.
func searchFallbackTaskRequest(tenantID, key string) tasklistv89.SearchTasksJSONRequestBody {
	implementation := tasklistv89.TaskSearchRequestImplementationJOBWORKER
	body := tasklistv89.SearchTasksJSONRequestBody{
		Implementation:     &implementation,
		PageSize:           ptr(fallbackTaskSearchPageSize),
		ProcessInstanceKey: &key,
	}
	if tenantID != "" {
		body.TenantIds = &[]string{tenantID}
	}
	return body
}

// requireSingleUserTask turns the search response into lookup semantics, preserving missing and duplicate matches as domain errors.
func requireSingleUserTask(items []camundav89.UserTaskResult, key string) (d.UserTask, error) {
	switch len(items) {
	case 0:
		return d.UserTask{}, fmt.Errorf("%w: user task %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
	case 1:
		return fromUserTaskResult(items[0]), nil
	default:
		return d.UserTask{}, fmt.Errorf("%w: user task %s returned %d matches", d.ErrMalformedResponse, key, len(items))
	}
}

func ptr[T any](v T) *T {
	return &v
}

func newStringEqFilterPtr(v string) (*camundav89.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}
