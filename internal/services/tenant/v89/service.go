// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

type Service struct {
	c   GenTenantClient
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) Client() GenTenantClient { return s.c }
func (s *Service) Config() *config.Config  { return s.cfg }
func (s *Service) Logger() *slog.Logger    { return s.log }

type Option func(*Service)

func WithClient(c GenTenantClient) Option {
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

func (s *Service) SearchTenants(ctx context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
	cCfg := services.ApplyCallOptions(opts)
	body := searchTenantsRequest(size)
	common.VerboseLog(ctx, cCfg, s.log, "searching tenants", "baseURL", s.cfg.APIs.Camunda.BaseURL, "body", body)
	resp, err := s.c.SearchTenantsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	out := toolx.MapSlice(payload.Items, fromTenantResult)
	common.VerboseLog(ctx, cCfg, s.log, "found tenants", "count", len(out))
	return out, nil
}

func (s *Service) GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
	cCfg := services.ApplyCallOptions(opts)
	common.VerboseLog(ctx, cCfg, s.log, "getting tenant", "baseURL", s.cfg.APIs.Camunda.BaseURL, "tenantID", tenantID)
	resp, err := s.c.GetTenantWithResponse(ctx, camundav89.TenantId(tenantID))
	if err != nil {
		return d.Tenant{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Tenant{}, err
	}
	out := fromTenantResult(*payload)
	common.VerboseLog(ctx, cCfg, s.log, "got tenant", "tenantID", out.TenantId)
	return out, nil
}

func searchTenantsRequest(size int32) camundav89.SearchTenantsJSONRequestBody {
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromLimitPagination(camundav89.LimitPagination{Limit: &size})
	order := camundav89.ASC
	sort := []camundav89.TenantSearchQuerySortRequest{
		{Field: camundav89.TenantSearchQuerySortRequestFieldName, Order: &order},
		{Field: camundav89.TenantSearchQuerySortRequestFieldTenantId, Order: &order},
	}
	return camundav89.SearchTenantsJSONRequestBody{
		Page: &page,
		Sort: &sort,
	}
}
