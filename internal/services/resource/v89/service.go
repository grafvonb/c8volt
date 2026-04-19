package v89

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"

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

	contentType, body, err := buildDeploymentBody(tenantID, units)
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

func (s *Service) processDefinitionDeployPoller(dr camundav89.DeploymentResult) func(context.Context) (poller.JobPollStatus, error) {
	keys := make([]string, 0, len(dr.Deployments))
	for _, dep := range dr.Deployments {
		if dep.ProcessDefinition == nil {
			continue
		}
		k := dep.ProcessDefinition.ProcessDefinitionKey
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}

	return func(ctx context.Context) (poller.JobPollStatus, error) {
		if len(keys) == 0 {
			return poller.JobPollStatus{
				Success: true,
				Message: "no process definitions in deployment; nothing to wait for",
			}, nil
		}
		missing := make([]string, 0)
		for _, k := range keys {
			resp, err := s.pdc.GetProcessDefinitionWithResponse(ctx, k)
			if err != nil {
				if errors.Is(err, d.ErrNotFound) {
					missing = append(missing, k)
					continue
				}
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: %w", k, err)
			}
			if resp == nil || resp.HTTPResponse == nil {
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: empty response", k)
			}
			if resp.HTTPResponse.StatusCode == http.StatusNotFound {
				missing = append(missing, k)
				continue
			}
			if resp.HTTPResponse.StatusCode != http.StatusOK {
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: unexpected status %d", k, resp.HTTPResponse.StatusCode)
			}
		}
		if len(missing) > 0 {
			return poller.JobPollStatus{
				Success: false,
				Message: fmt.Sprintf("process definitions not visible yet, waiting: %v", missing),
			}, nil
		}
		return poller.JobPollStatus{
			Success: true,
			Message: fmt.Sprintf("process definitions visible: %v", keys),
		}, nil
	}
}

func buildDeploymentBody(tenantID string, units []d.DeploymentUnitData) (string, *bytes.Reader, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if tenantID != "" {
		if err := w.WriteField("tenantId", tenantID); err != nil {
			return "", nil, err
		}
	}
	for _, u := range units {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="resources"; filename="`+u.Name+`"`)
		part, err := w.CreatePart(h)
		if err != nil {
			return "", nil, err
		}
		if _, err = part.Write(u.Data); err != nil {
			return "", nil, err
		}
	}
	if err := w.Close(); err != nil {
		return "", nil, err
	}
	return w.FormDataContentType(), bytes.NewReader(buf.Bytes()), nil
}
