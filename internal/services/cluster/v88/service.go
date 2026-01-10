package v88

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
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
	deps, err := common.PrepareCamundaDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav88.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(deps.HTTPClient),
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
	cCfg := services.ApplyCallOptions(opts)
	common.VerboseLog(ctx, cCfg, s.log, "requesting cluster topology", "baseURL", s.cfg.APIs.Camunda.BaseURL)
	resp, err := s.c.GetTopologyWithResponse(ctx)
	if err != nil {
		return d.Topology{}, fmt.Errorf("fetch cluster topology: %w", err)
	}
	if resp == nil {
		return d.Topology{}, fmt.Errorf("%w: topology response is nil", d.ErrMalformedResponse)
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.Topology{}, fmt.Errorf("fetch cluster topology: %w", err)
	}
	if resp.JSON200 == nil {
		return d.Topology{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	topology := fromTopologyResponse(*resp.JSON200)
	common.VerboseLog(ctx, cCfg, s.log, "cluster topology retrieved", "brokers", len(topology.Brokers), "clusterSize", topology.ClusterSize)
	return topology, nil
}
