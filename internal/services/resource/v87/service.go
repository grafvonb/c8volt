// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

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
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	resourcepayload "github.com/grafvonb/c8volt/internal/services/resource/payload"
)

type Service struct {
	c   GenResourceClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

// WithClient replaces the generated Camunda resource client, primarily for tests.
// A nil client is ignored so option composition cannot accidentally erase the default client.
//
//nolint:unused
func WithClient(c GenResourceClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

// WithLogger replaces the service logger when logger is non-nil.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New constructs the v8.7 resource service using the Camunda base URL from cfg.
// httpClient and log may be nil; shared dependency preparation supplies defaults before version-specific clients are built.
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

// Delete removes a resource only when AllowInconsistent is set in opts.
// resourceKey is passed directly to the v8.7 resource deletion endpoint; without AllowInconsistent the method is a no-op
// to preserve the facade's safety boundary around eventually consistent deletion.
func (s *Service) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error {
	cCfg := services.ApplyCallOptions(opts)

	if cCfg.AllowInconsistent {
		resp, err := s.c.PostResourcesResourceKeyDeletionWithResponse(ctx, resourceKey, camundav87.PostResourcesResourceKeyDeletionJSONRequestBody{})
		if err != nil {
			return err
		}
		if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Get fetches a resource by Camunda resource key and maps the generated response to the internal domain model.
func (s *Service) Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error) {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.c.GetResourcesResourceKeyWithResponse(ctx, resourceKey)
	if err != nil {
		return d.Resource{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Resource{}, err
	}
	resource, err := resourcepayload.RequireSingleResource(fromResourceResult(*payload), resp.Body)
	if err != nil {
		return d.Resource{}, err
	}
	return resource, nil
}

// Deploy uploads deployment units to the v8.7 deployment endpoint.
// units are multipart "resources" parts; v8.7 is treated as strongly consistent after a 200 OK response.
func (s *Service) Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error) {
	_ = services.ApplyCallOptions(opts)
	tenantId, vtenantId := s.cfg.App.TargetTenant(), s.cfg.App.ViewTenant()

	contentType, body, err := common.BuildDeploymentBody(tenantId, units)
	if err != nil {
		return d.Deployment{}, err
	}

	resp, err := s.c.PostDeploymentsWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return d.Deployment{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Deployment{}, err
	}
	s.log.Debug(fmt.Sprintf("deployment of %d resources to tenant %q successful (confirmed, as the api returned 200 OK and is strongly consistent and atomic)", len(units), vtenantId))
	return fromDeploymentResult(*payload), nil
}
