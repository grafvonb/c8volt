// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// Service implements variable operations through the Camunda 8.7 Operate API.
type Service struct {
	co  GenVariableClientOperate
	cfg *config.Config
	log *slog.Logger
}

// Option configures a v8.7 variable service.
type Option func(*Service)

// WithClientOperate replaces the generated Operate client when c is non-nil.
func WithClientOperate(c GenVariableClientOperate) Option {
	return func(s *Service) {
		if c != nil {
			s.co = c
		}
	}
}

// New creates a v8.7 variable service with generated Operate client dependencies.
func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	co, err := operatev87.NewClientWithResponses(
		deps.Config.APIs.Operate.BaseURL,
		operatev87.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{co: co, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.co)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}
