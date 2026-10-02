// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

type Service struct {
	cfg *config.Config
	log *slog.Logger
}

// Config returns the configuration used by the user-task service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the logger used by the user-task service.
func (s *Service) Logger() *slog.Logger { return s.log }

// GetUserTask returns an explicit unsupported error for Camunda 8.7 without issuing a request.
func (s *Service) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{}, fmt.Errorf("%w: has-user-tasks lookup is unsupported in Camunda 8.7; requires Camunda 8.8 or newer", d.ErrUnsupported)
}

// GetNativeUserTask rejects direct native reads because Camunda 8.7 does not expose the required API contract.
func (s *Service) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{}, fmt.Errorf("%w: native user-task lookup is unsupported in Camunda 8.7; requires Camunda 8.8 or newer", d.ErrUnsupported)
}

// SearchUserTasksPage rejects native discovery because Camunda 8.7 does not expose the required API contract.
func (s *Service) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	return d.UserTaskSearchPage{}, fmt.Errorf("%w: native user-task search is unsupported in Camunda 8.7; requires Camunda 8.8 or newer", d.ErrUnsupported)
}

type Option func(*Service)

// WithLogger replaces the service logger when a non-nil override is supplied.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New prepares configuration and logging for the adapter that reports unsupported Camunda 8.7
// user-task operations.
func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	return s, nil
}
