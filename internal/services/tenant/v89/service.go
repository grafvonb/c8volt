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

// Client returns the generated Camunda tenant client used by the v8.9 service.
func (s *Service) Client() GenTenantClient { return s.c }

// Config returns the normalized service configuration used by the v8.9 tenant service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the service logger used by the v8.9 tenant service.
func (s *Service) Logger() *slog.Logger { return s.log }

type Option func(*Service)

// WithClient overrides the generated Camunda tenant client, primarily for service tests.
func WithClient(c GenTenantClient) Option {
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

// New prepares a v8.9 tenant service with the generated Camunda v2 tenant client.
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

// SearchTenants lists native Camunda tenants, then applies the local literal tenant name or ID filter.
func (s *Service) SearchTenants(ctx context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
	cCfg := services.ApplyCallOptions(opts)
	body := searchTenantsRequest(size)
	common.VerboseLog(ctx, cCfg, s.log, "searching tenants", "baseURL", s.cfg.APIs.Camunda.BaseURL, "filter", filter, "body", body)
	resp, err := s.c.SearchTenantsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	out := toolx.MapSlice(payload.Items, fromTenantResult)
	out = d.FilterTenantsByNameOrIDContains(out, filter.NameContains)
	common.VerboseLog(ctx, cCfg, s.log, "found tenants", "count", len(out))
	return out, nil
}

// GetTenant fetches one native Camunda tenant by tenant ID.
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

// searchTenantsRequest builds the stable sorted tenant search request used for discovery output.
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
