package v87

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
)

type Service struct {
	c   GenProcessDefinitionClientOperate
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	c, err := operatev87.NewClientWithResponses(
		cfg.APIs.Operate.BaseURL,
		operatev87.WithHTTPClient(httpClient),
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

func (s *Service) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	_ = services.ApplyCallOptions(opts)
	body := operatev87.QueryProcessDefinition{
		Filter: &operatev87.ProcessDefinition{
			BpmnProcessId: toolx.PtrIf(filter.BpmnProcessId, ""),
			Version:       toolx.PtrIfNonZero(filter.ProcessVersion),
			VersionTag:    toolx.PtrIf(filter.ProcessVersionTag, ""),
		},
		Size: &size,
	}
	resp, err := s.c.SearchProcessDefinitionsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	out := toolx.DerefSlicePtr(resp.JSON200.Items, fromProcessDefinitionResponse)
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)
	return out, nil
}

func (s *Service) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	pds, err := s.SearchProcessDefinitions(ctx, filter, 1000, opts...)
	if err != nil {
		return nil, err
	}
	m := make(map[string]d.ProcessDefinition)
	for _, pd := range pds {
		if cur, ok := m[pd.BpmnProcessId]; !ok || pd.ProcessVersion > cur.ProcessVersion {
			m[pd.BpmnProcessId] = pd
		}
	}
	out := make([]d.ProcessDefinition, 0, len(m))
	for _, pd := range m {
		out = append(out, pd)
	}
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)
	return out, nil
}

func (s *Service) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	_ = services.ApplyCallOptions(opts)
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return d.ProcessDefinition{}, fmt.Errorf("converting process definition key %q to int64: %w", key, err)
	}
	resp, err := s.c.GetProcessDefinitionByKeyWithResponse(ctx, oldKey)
	if err != nil {
		return d.ProcessDefinition{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.ProcessDefinition{}, err
	}
	if resp.JSON200 == nil {
		return d.ProcessDefinition{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	return fromProcessDefinitionResponse(*resp.JSON200), nil
}
