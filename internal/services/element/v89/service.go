// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// Service adapts Camunda 8.9 runtime element endpoints to the internal API.
type Service struct {
	c   GenElementClient
	cfg *config.Config
	log *slog.Logger
}

// Client returns the generated Camunda element client used by the v8.9 service.
func (s *Service) Client() GenElementClient { return s.c }

// Config returns the normalized service configuration used by the v8.9 element service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the service logger used by the v8.9 element service.
func (s *Service) Logger() *slog.Logger { return s.log }

// Option customizes the v8.9 element service during construction.
type Option func(*Service)

// WithClient overrides the generated Camunda element client, primarily for service tests.
func WithClient(c GenElementClient) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

// WithLogger overrides the default logger for tests and callers that need custom logging.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New prepares a v8.9 element service with the generated Camunda v2 client.
func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{c: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.c)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

// GetElement will use the generated direct lookup endpoint once US1 fills the adapter.
func (s *Service) GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error) {
	_ = ctx
	_ = key
	_ = services.ApplyCallOptions(opts)
	return d.Element{}, pendingElementOperation("element lookup")
}

// SearchElements will collect pages once US2 fills the adapter.
func (s *Service) SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	_ = ctx
	_ = query
	_ = services.ApplyCallOptions(opts)
	return d.ElementSearchResult{}, pendingElementOperation("element search")
}

// SearchElementsPage will call the generated search endpoint once US2 fills the adapter.
func (s *Service) SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error) {
	_ = ctx
	_ = query
	_ = page
	_ = services.ApplyCallOptions(opts)
	return d.ElementSearchPage{}, pendingElementOperation("element search")
}

// pendingElementOperation prevents accidental success before the version adapter is implemented.
func pendingElementOperation(operation string) error {
	return fmt.Errorf("%w: %s service implementation is pending", d.ErrUnsupported, operation)
}
