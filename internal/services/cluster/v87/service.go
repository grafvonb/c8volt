package v87

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	clustercommon "github.com/grafvonb/c8volt/internal/services/cluster/common"
	"github.com/grafvonb/c8volt/internal/services/common"
)

type Service struct {
	c   GenClusterClient
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) Client() GenClusterClient { return s.c }
func (s *Service) Config() *config.Config   { return s.cfg }
func (s *Service) Logger() *slog.Logger     { return s.log }

type Option func(*Service)

func WithClient(c GenClusterClient) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

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
	c, err := camundav87.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav87.WithHTTPClient(deps.HTTPClient),
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

func (s *Service) GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error) {
	return clustercommon.GetClusterTopology(ctx, s.log, s.cfg.APIs.Camunda.BaseURL, opts, func(ctx context.Context) (clustercommon.PayloadResponse[camundav87.TopologyResponse], error) {
		resp, err := s.c.GetTopologyWithResponse(ctx)
		if resp == nil || err != nil {
			return clustercommon.PayloadResponse[camundav87.TopologyResponse]{Received: resp != nil}, err
		}
		return clustercommon.PayloadResponse[camundav87.TopologyResponse]{
			Received:     true,
			HTTPResponse: resp.HTTPResponse,
			Body:         resp.Body,
			Payload:      resp.JSON200,
		}, nil
	}, fromTopologyResponse)
}

func (s *Service) GetClusterLicense(ctx context.Context, opts ...services.CallOption) (d.License, error) {
	return clustercommon.GetClusterLicense(ctx, s.log, s.cfg.APIs.Camunda.BaseURL, opts, func(ctx context.Context) (clustercommon.PayloadResponse[camundav87.LicenseResponse], error) {
		resp, err := s.c.GetLicenseWithResponse(ctx)
		if resp == nil || err != nil {
			return clustercommon.PayloadResponse[camundav87.LicenseResponse]{Received: resp != nil}, err
		}
		return clustercommon.PayloadResponse[camundav87.LicenseResponse]{
			Received:     true,
			HTTPResponse: resp.HTTPResponse,
			Body:         resp.Body,
			Payload:      resp.JSON200,
		}, nil
	}, fromLicenseResponse)
}
