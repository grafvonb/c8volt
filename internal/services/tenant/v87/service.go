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

// Config returns the normalized service configuration used by the v8.7 tenant service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the service logger used by the v8.7 tenant service.
func (s *Service) Logger() *slog.Logger { return s.log }

type Option func(*Service)

// WithLogger overrides the default logger for tests and callers that need custom logging.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New prepares a v8.7 tenant service that reports tenant discovery as unsupported.
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

// SearchTenants reports unsupported because Camunda 8.7 lacks the native tenant search endpoint used by c8volt.
func (s *Service) SearchTenants(ctx context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
	_ = ctx
	_ = filter
	_ = size
	_ = services.ApplyCallOptions(opts)
	return nil, unsupportedTenantOperation("tenant search")
}

// GetTenant reports unsupported because Camunda 8.7 lacks the native tenant lookup endpoint used by c8volt.
func (s *Service) GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
	_ = ctx
	_ = tenantID
	_ = services.ApplyCallOptions(opts)
	return d.Tenant{}, unsupportedTenantOperation("tenant lookup")
}

// unsupportedTenantOperation formats the shared unsupported-version message for tenant operations.
func unsupportedTenantOperation(operation string) error {
	return fmt.Errorf("%w: %s requires Camunda 8.8 or newer", d.ErrUnsupported, operation)
}
