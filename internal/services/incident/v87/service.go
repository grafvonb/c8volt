// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// Service implements incident operations supported by Camunda 8.7.
type Service struct {
	cfg *config.Config
	log *slog.Logger
}

// Option configures a v8.7 incident service.
type Option func(*Service)

// New creates a v8.7 incident service with shared service dependencies.
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
