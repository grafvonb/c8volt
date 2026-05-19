// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

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
	pds "github.com/grafvonb/c8volt/internal/services/processdefinition/v88"
	resourcepayload "github.com/grafvonb/c8volt/internal/services/resource/payload"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/poller"
)

type Service struct {
	c   GenResourceClientCamunda
	pdc pds.GenProcessDefinitionClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

//nolint:unused
func WithClient(c GenResourceClientCamunda, pdc pds.GenProcessDefinitionClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
		if pdc != nil {
			s.pdc = pdc
		}
	}
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
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
	s := &Service{c: c, pdc: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.c, s.pdc)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	_ = services.ApplyCallOptions(opts)
	return d.ResourceDeleteResponse{}, fmt.Errorf("%w: process-definition history-safe deletion requires Camunda 8.9 or newer", d.ErrUnsupported)
}

func (s *Service) SupportsProcessDefinitionHistoryDeletion() bool { return false }

func (s *Service) Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error) {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.c.GetResourceWithResponse(ctx, resourceKey)
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

func (s *Service) Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error) {
	cCfg := services.ApplyCallOptions(opts)
	tenantId, vtenantId := s.cfg.App.TargetTenant(), s.cfg.App.ViewTenant()

	contentType, body, err := common.BuildDeploymentBody(tenantId, units)
	if err != nil {
		return d.Deployment{}, err
	}

	resp, err := services.RetryCamundaMutation(ctx, s.log, "deploy pd", func(ctx context.Context) (*camundav88.CreateDeploymentResponse, *http.Response, []byte, error) {
		_, _ = body.Seek(0, 0)
		resp, err := s.c.CreateDeploymentWithBodyWithResponse(ctx, contentType, body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		return d.Deployment{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Deployment{}, err
	}
	if !cCfg.NoWait {
		if err = s.waitForDeploymentConfirmation(ctx, *payload, vtenantId, cCfg.SuppressWorkflowDetailLogs); err != nil {
			return d.Deployment{}, err
		}
	} else if !cCfg.SuppressWorkflowDetailLogs {
		s.log.Info(fmt.Sprintf("pd deploy submitted; count %d, tenant %s, no-wait", len(units), vtenantId))
	}
	return fromDeploymentResult(*payload), nil
}

func (s *Service) waitForDeploymentConfirmation(ctx context.Context, dr camundav88.DeploymentResult, vtenantId string, suppressDetailLogs bool) error {
	if !suppressDetailLogs {
		s.log.Info(fmt.Sprintf("pd deploy wait; count %d", len(dr.Deployments)))
	}
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for %d deployments", len(dr.Deployments)))
	defer stopActivity()
	poll := s.processDefinitionDeployPoller(dr)
	if err := poller.WaitForCompletion(ctx, s.log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return fmt.Errorf("waiting for process definition deployment confirmation failed: %w", err)
	}
	if !suppressDetailLogs {
		s.log.Info(fmt.Sprintf("pd deploy confirmed; count %d, tenant %s", len(dr.Deployments), vtenantId))
	}
	return nil
}

// processDefinitionDeployPoller adapts a v8.8 deployment response into the shared visibility poller.
// It waits only for deployed process definitions; deployments containing only other resource types complete immediately.
func (s *Service) processDefinitionDeployPoller(dr camundav88.DeploymentResult) func(ctx context.Context) (poller.JobPollStatus, error) {
	keys := resourcepayload.DeploymentProcessDefinitionKeys(dr.Deployments, func(dep camundav88.DeploymentMetadataResult) string {
		if dep.ProcessDefinition == nil {
			return ""
		}
		return dep.ProcessDefinition.ProcessDefinitionKey
	})
	return resourcepayload.NewProcessDefinitionVisibilityPoller(keys, func(ctx context.Context, key string) (*http.Response, error) {
		resp, err := s.pdc.GetProcessDefinitionWithResponse(ctx, key)
		if resp == nil {
			return nil, err
		}
		return resp.HTTPResponse, err
	})
}
