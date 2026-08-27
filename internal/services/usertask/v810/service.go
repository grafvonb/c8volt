// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

type Service struct {
	cc  GenUserTaskClientCamunda
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) ClientCamunda() GenUserTaskClientCamunda { return s.cc }
func (s *Service) Config() *config.Config                  { return s.cfg }
func (s *Service) Logger() *slog.Logger                    { return s.log }

type Option func(*Service)

func WithClientCamunda(c GenUserTaskClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
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
	c, err := camundav810.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav810.WithHTTPClient(deps.HTTPClient),
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

// GetUserTask resolves a V810 user task through the unified Camunda client only.
func (s *Service) GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("getting user task %s", key))
	resp, err := s.cc.GetUserTaskWithResponse(ctx, camundav810.UserTaskKey(key))
	if err != nil {
		return d.UserTask{}, fmt.Errorf("get user task: %w", err)
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		if errors.Is(err, d.ErrNotFound) {
			return d.UserTask{}, userTaskNotFound(key)
		}
		return d.UserTask{}, fmt.Errorf("get user task: %w", err)
	}
	task := fromUserTaskResult(*payload)
	if task.Key != key {
		return d.UserTask{}, fmt.Errorf("%w: user task %s returned mismatched task %s", d.ErrMalformedResponse, key, task.Key)
	}
	tenantID := common.EffectiveTenant(s.cfg)
	if tenantID != "" && task.TenantId != tenantID {
		return d.UserTask{}, userTaskNotFound(key)
	}
	if task.ProcessInstanceKey == "" {
		return d.UserTask{}, fmt.Errorf("%w: user task %s has no process instance key", d.ErrMalformedResponse, key)
	}
	return task, nil
}

func userTaskNotFound(key string) error {
	return fmt.Errorf("%w: user task %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
}
