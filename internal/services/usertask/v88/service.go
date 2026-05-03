// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	tasklistv88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

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

// WithClientTasklist injects the Tasklist V1 client used only after the primary v2 lookup misses.
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
	c, err := camundav88.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	var t GenUserTaskClientTasklist
	if deps.Config.APIs.Tasklist.BaseURL != "" {
		t, err = tasklistv88.NewClientWithResponses(
			deps.Config.APIs.Tasklist.BaseURL,
			tasklistv88.WithHTTPClient(deps.HTTPClient),
		)
		if err != nil {
			return nil, err
		}
	}
	s := &Service{cc: c, ct: t, cfg: deps.Config, log: deps.Logger}
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

// GetUserTask resolves a Camunda user task through search so tenant scoping and generated v8.8 filter shapes stay explicit.
func (s *Service) GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
	_ = services.ApplyCallOptions(opts)
	task, err := s.searchPrimaryUserTask(ctx, key)
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, d.ErrNotFound) {
		return d.UserTask{}, err
	}
	return s.searchFallbackTask(ctx, key)
}

func (s *Service) searchPrimaryUserTask(ctx context.Context, key string) (d.UserTask, error) {
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

// searchFallbackTask resolves legacy Tasklist URL ids; the fallback endpoint does not accept tenant filters.
func (s *Service) searchFallbackTask(ctx context.Context, key string) (d.UserTask, error) {
	if s.ct == nil {
		return d.UserTask{}, fmt.Errorf("get fallback task: %w", common.ErrNoClientConfigured)
	}
	s.log.Debug(fmt.Sprintf("getting user task with key %s using generated tasklist client fallback", key))
	resp, err := s.ct.GetTaskByIdWithResponse(ctx, key)
	if err != nil {
		return d.UserTask{}, fmt.Errorf("get fallback task: %w", err)
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		if errors.Is(err, d.ErrNotFound) {
			return d.UserTask{}, fmt.Errorf("%w: fallback user task %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
		}
		return d.UserTask{}, fmt.Errorf("get fallback task: %w", err)
	}
	task := fromTaskResponse(*payload)
	tenantID := common.EffectiveTenant(s.cfg)
	if tenantID != "" && task.TenantId != tenantID {
		return d.UserTask{}, fmt.Errorf("%w: fallback user task %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
	}
	if task.Key != key {
		return d.UserTask{}, fmt.Errorf("%w: fallback user task %s returned mismatched task %s", d.ErrMalformedResponse, key, task.Key)
	}
	if task.ProcessInstanceKey == "" {
		return d.UserTask{}, fmt.Errorf("%w: fallback user task %s has no process instance key", d.ErrMalformedResponse, key)
	}
	return task, nil
}

// searchUserTaskRequest builds the v8.8 user-task search body used to find one task within the effective tenant.
func searchUserTaskRequest(tenantID, key string) (camundav88.SearchUserTasksJSONRequestBody, error) {
	tenantIDFilter, err := common.NewStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav88.SearchUserTasksJSONRequestBody{}, err
	}
	userTaskKey := camundav88.UserTaskKey(key)
	return camundav88.SearchUserTasksJSONRequestBody{
		Filter: &camundav88.UserTaskFilter{
			TenantId:    tenantIDFilter,
			UserTaskKey: &userTaskKey,
		},
	}, nil
}

// requireSingleUserTask turns the search response into lookup semantics, preserving missing and duplicate matches as domain errors.
func requireSingleUserTask(items []camundav88.UserTaskResult, key string) (d.UserTask, error) {
	switch len(items) {
	case 0:
		return d.UserTask{}, fmt.Errorf("%w: user task %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
	case 1:
		return fromUserTaskResult(items[0]), nil
	default:
		return d.UserTask{}, fmt.Errorf("%w: user task %s returned %d matches", d.ErrMalformedResponse, key, len(items))
	}
}
