package v88

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
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	pds "github.com/grafvonb/c8volt/internal/services/processdefinition/v88"
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
	return func(s *Service) { s.c, s.pdc = c, pdc }
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	c, err := camundav88.NewClientWithResponses(
		cfg.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{c: c, pdc: c, cfg: cfg, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error {
	cCfg := services.ApplyCallOptions(opts)

	if cCfg.AllowInconsistent {
		resp, err := s.c.DeleteResourceWithResponse(ctx, resourceKey, camundav88.DeleteResourceJSONRequestBody{})
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

func (s *Service) Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error) {
	cCfg := services.ApplyCallOptions(opts)
	tenantId, vtenantId := s.cfg.App.Tenant, s.cfg.App.ViewTenant()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if tenantId != "" {
		if err := w.WriteField("tenantId", tenantId); err != nil {
			return d.Deployment{}, err
		}
	}
	for _, u := range units {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="resources"; filename="`+u.Name+`"`)
		part, err := w.CreatePart(h)
		if err != nil {
			return d.Deployment{}, err
		}
		if _, err = part.Write(u.Data); err != nil {
			return d.Deployment{}, err
		}
	}
	if err := w.Close(); err != nil {
		return d.Deployment{}, err
	}
	ct := w.FormDataContentType()

	resp, err := s.c.CreateDeploymentWithBodyWithResponse(ctx, ct, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return d.Deployment{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.Deployment{}, err
	}
	if resp.JSON200 == nil {
		return d.Deployment{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	if !cCfg.NoWait {
		dr := *resp.JSON200
		s.log.Info(fmt.Sprintf("waiting for %d deployment(s) confirmation...", len(dr.Deployments)))
		poll := s.processDefinitionDeployPoller(dr)
		if err = poller.WaitForCompletion(ctx, s.log, poller.DefaultCompletionTimeout, true, poll); err != nil {
			return d.Deployment{}, fmt.Errorf("waiting for process definition deployment confirmation failed: %w", err)
		}
		s.log.Info(fmt.Sprintf("%d deployment(s) to tenant %q confirmed by successful poll", len(dr.Deployments), vtenantId))
	} else {
		s.log.Info(fmt.Sprintf("%d deployment(s) to tenant %q finished, not confirmed as --no-wait is set", len(units), vtenantId))
	}
	return fromDeploymentResult(*resp.JSON200), nil
}

func (s *Service) processDefinitionDeployPoller(dr camundav88.DeploymentResult) func(ctx context.Context) (poller.JobPollStatus, error) {
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
