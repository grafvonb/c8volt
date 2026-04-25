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
	"github.com/grafvonb/c8volt/internal/services/httpc"
	resourcepayload "github.com/grafvonb/c8volt/internal/services/resource/payload"
	"github.com/grafvonb/c8volt/toolx/poller"
)

type Service struct {
	c   GenResourceClientCamunda
	pdc GenProcessDefinitionClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func WithClient(c GenResourceClientCamunda, pdc GenProcessDefinitionClientCamunda) Option {
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
	c, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
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

func (s *Service) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error {
	cCfg := services.ApplyCallOptions(opts)

	if cCfg.AllowInconsistent {
		resp, err := s.c.DeleteResourceOpWithResponse(ctx, resourceKey, camundav89.DeleteResourceOpJSONRequestBody{})
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
	tenantID, vtenantID := s.cfg.App.Tenant, s.cfg.App.ViewTenant()

	contentType, body, err := common.BuildDeploymentBody(tenantID, units)
	if err != nil {
		return d.Deployment{}, err
	}

	resp, err := s.c.CreateDeploymentWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return d.Deployment{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Deployment{}, err
	}
	if !cCfg.NoWait {
		if err = s.waitForDeploymentConfirmation(ctx, *payload, vtenantID); err != nil {
			return d.Deployment{}, err
		}
	} else {
		s.log.Info(fmt.Sprintf("%d deployment(s) to tenant %q finished, not confirmed as --no-wait is set", len(units), vtenantID))
	}
	return fromDeploymentResult(*payload), nil
}

func (s *Service) waitForDeploymentConfirmation(ctx context.Context, dr camundav89.DeploymentResult, vtenantID string) error {
	s.log.Info(fmt.Sprintf("waiting for %d deployment(s) confirmation...", len(dr.Deployments)))
	poll := s.processDefinitionDeployPoller(dr)
	if err := poller.WaitForCompletion(ctx, s.log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return fmt.Errorf("waiting for process definition deployment confirmation failed: %w", err)
	}
	s.log.Info(fmt.Sprintf("%d deployment(s) to tenant %q confirmed by successful poll", len(dr.Deployments), vtenantID))
	return nil
}

// processDefinitionDeployPoller adapts a v8.9 deployment response into the shared visibility poller.
// It keeps the version-specific response shape local while reusing the common confirmation behavior.
func (s *Service) processDefinitionDeployPoller(dr camundav89.DeploymentResult) func(context.Context) (poller.JobPollStatus, error) {
	keys := resourcepayload.DeploymentProcessDefinitionKeys(dr.Deployments, func(dep camundav89.DeploymentMetadataResult) string {
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
