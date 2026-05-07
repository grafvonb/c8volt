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

type Service struct {
	co  GenVariableClientOperate
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func WithClientOperate(c GenVariableClientOperate) Option {
	return func(s *Service) {
		if c != nil {
			s.co = c
		}
	}
}

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
