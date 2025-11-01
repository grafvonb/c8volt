package v87

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

type Service struct {
	c   GenClusterClient
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func WithClient(c GenClusterClient) Option { return func(s *Service) { s.c = c } }

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	c, err := camundav87.NewClientWithResponses(
		cfg.APIs.Camunda.BaseURL,
		camundav87.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{c: c, cfg: cfg, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) GetClusterTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error) {
	_ = services.ApplyCallOptions(opts)
	resp, err := s.c.GetTopologyWithResponse(ctx)
	if err != nil {
		return d.Topology{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.Topology{}, err
	}
	if resp.JSON200 == nil {
		return d.Topology{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	return fromTopologyResponse(*resp.JSON200), nil
}
