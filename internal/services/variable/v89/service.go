// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	"github.com/grafvonb/c8volt/internal/services/common"
)

type Service struct {
	cc  GenVariableClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func WithClientCamunda(c GenVariableClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
		}
	}
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	cc, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cc: cc, cfg: deps.Config, log: deps.Logger}
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
