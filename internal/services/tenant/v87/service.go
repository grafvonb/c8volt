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

func (s *Service) Config() *config.Config { return s.cfg }
func (s *Service) Logger() *slog.Logger   { return s.log }

type Option func(*Service)

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
	s := &Service{cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	return s, nil
}

func (s *Service) SearchTenants(ctx context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
	_ = ctx
	_ = filter
	_ = size
	_ = services.ApplyCallOptions(opts)
	return nil, unsupportedTenantOperation("tenant search")
}

func (s *Service) GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
	_ = ctx
	_ = tenantID
	_ = services.ApplyCallOptions(opts)
	return d.Tenant{}, unsupportedTenantOperation("tenant lookup")
}

func unsupportedTenantOperation(operation string) error {
	return fmt.Errorf("%w: %s requires Camunda 8.8 or newer", d.ErrUnsupported, operation)
}
