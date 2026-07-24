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

// Service reports runtime element operations as unsupported for Camunda 8.7.
type Service struct {
	cfg *config.Config
	log *slog.Logger
}

// Config returns the normalized service configuration used by the v8.7 element service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the service logger used by the v8.7 element service.
func (s *Service) Logger() *slog.Logger { return s.log }

// Option customizes the v8.7 element service during construction.
type Option func(*Service)

// WithLogger overrides the default logger for tests and callers that need custom logging.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New prepares a v8.7 element service that reports runtime element inspection as unsupported.
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

// GetElement reports unsupported because Camunda 8.7 lacks runtime element lookup endpoints.
func (s *Service) GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error) {
	_ = ctx
	_ = key
	_ = services.ApplyCallOptions(opts)
	return d.Element{}, unsupportedElementOperation("element lookup")
}

// SearchElements reports unsupported because Camunda 8.7 lacks runtime element search endpoints.
func (s *Service) SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	_ = ctx
	_ = query
	_ = services.ApplyCallOptions(opts)
	return d.ElementSearchResult{}, unsupportedElementOperation("element search")
}

// SearchElementsPages reports unsupported because Camunda 8.7 lacks runtime element search endpoints.
func (s *Service) SearchElementsPages(ctx context.Context, query d.ElementSearchQuery, visitor d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error) {
	_ = ctx
	_ = query
	_ = visitor
	_ = services.ApplyCallOptions(opts)
	return d.ElementSearchPagesResult{}, unsupportedElementOperation("element search")
}

// SearchElementsPage reports unsupported because Camunda 8.7 lacks runtime element search endpoints.
func (s *Service) SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error) {
	_ = ctx
	_ = query
	_ = page
	_ = services.ApplyCallOptions(opts)
	return d.ElementSearchPage{}, unsupportedElementOperation("element search")
}

// SearchElementsTotal reports unsupported because Camunda 8.7 lacks runtime element search endpoints.
func (s *Service) SearchElementsTotal(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (int64, error) {
	_ = ctx
	_ = query
	_ = services.ApplyCallOptions(opts)
	return 0, unsupportedElementOperation("element search")
}

// unsupportedElementOperation formats the shared unsupported-version message for element operations.
func unsupportedElementOperation(operation string) error {
	return fmt.Errorf("%w: %s requires Camunda 8.8 or newer", d.ErrUnsupported, operation)
}
