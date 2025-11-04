package v88

import (
	"bytes"
	"context"
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
)

type Service struct {
	c   GenResourceClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

//nolint:unused
func WithClient(c GenResourceClientCamunda) Option { return func(s *Service) { s.c = c } }

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	c, err := camundav88.NewClientWithResponses(
		cfg.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(httpClient),
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

func (s *Service) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.c.DeleteResourceWithResponse(ctx, resourceKey, camundav88.DeleteResourceJSONRequestBody{})
	if err != nil {
		return err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return err
	}
	return nil
}

func (s *Service) Deploy(ctx context.Context, tenantId string, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error) {
	_ = services.ApplyCallOptions(opts)

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
	s.log.Debug(fmt.Sprintf("deployment of %d resources to tenant %q successful (confirmed, as the api returned 200 OK and is strongly consistent and atomic)", len(units), tenantId))
	return fromDeploymentResult(*resp.JSON200), nil
}
