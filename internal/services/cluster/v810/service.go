// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	clustercommon "github.com/grafvonb/c8volt/internal/services/cluster/common"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// Service implements cluster workflows using the Camunda 8.10 unified client.
type Service struct {
	c   GenClusterClient
	cfg *config.Config
	log *slog.Logger
}

// Client returns the generated client, primarily for package tests.
func (s *Service) Client() GenClusterClient { return s.c }

// Config returns the normalized service config, primarily for package tests.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the configured service logger, primarily for package tests.
func (s *Service) Logger() *slog.Logger { return s.log }

// Option customizes a V810 cluster service for tests and callers.
type Option func(*Service)

// WithClient replaces the generated client while preserving a previously configured client for nil inputs.
func WithClient(c GenClusterClient) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

// WithLogger replaces the service logger while preserving a previously configured logger for nil inputs.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New creates a V810 cluster service with the unified Camunda client.
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

// GetClusterTopology retrieves and converts V810 topology details through the shared cluster helper.
func (s *Service) GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error) {
	return clustercommon.GetClusterTopology(ctx, s.log, s.cfg.APIs.Camunda.BaseURL, opts, func(ctx context.Context) (clustercommon.PayloadResponse[camundav810.TopologyResponse], error) {
		resp, err := s.c.GetTopologyWithResponse(ctx)
		if resp == nil || err != nil {
			return clustercommon.PayloadResponse[camundav810.TopologyResponse]{Received: resp != nil}, err
		}
		return clustercommon.PayloadResponse[camundav810.TopologyResponse]{
			Received:     true,
			HTTPResponse: resp.HTTPResponse,
			Body:         resp.Body,
			Payload:      resp.JSON200,
		}, nil
	}, fromTopologyResponse)
}

// GetClusterLicense retrieves and converts V810 license details through the shared cluster helper.
func (s *Service) GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error) {
	return clustercommon.GetClusterLicense(ctx, s.log, s.cfg.APIs.Camunda.BaseURL, opts, func(ctx context.Context) (clustercommon.PayloadResponse[camundav810.LicenseResponse], error) {
		resp, err := s.c.GetLicenseWithResponse(ctx)
		if resp == nil || err != nil {
			return clustercommon.PayloadResponse[camundav810.LicenseResponse]{Received: resp != nil}, err
		}
		return clustercommon.PayloadResponse[camundav810.LicenseResponse]{
			Received:     true,
			HTTPResponse: resp.HTTPResponse,
			Body:         resp.Body,
			Payload:      resp.JSON200,
		}, nil
	}, fromLicenseResponse)
}
