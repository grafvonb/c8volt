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
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
)

type Service struct {
	co  GenProcessDefinitionClientOperate
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) ClientOperate() GenProcessDefinitionClientOperate { return s.co }
func (s *Service) Config() *config.Config                           { return s.cfg }
func (s *Service) Logger() *slog.Logger                             { return s.log }

type Option func(*Service)

func WithClientOperate(c GenProcessDefinitionClientOperate) Option {
	return func(s *Service) {
		if c != nil {
			s.co = c
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
	c, err := operatev87.NewClientWithResponses(
		deps.Config.APIs.Operate.BaseURL,
		operatev87.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{co: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.co)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	if cCfg.WithStat {
		return nil, fmt.Errorf("process definition stats not supported in v8.7 API")
	}

	body := operatev87.QueryProcessDefinition{
		Filter: &operatev87.ProcessDefinition{
			BpmnProcessId: toolx.PtrIf(filter.BpmnProcessId, ""),
			Version:       toolx.PtrIfNonZero(filter.ProcessVersion),
			VersionTag:    toolx.PtrIf(filter.ProcessVersionTag, ""),
		},
		Size: &size,
	}
	common.VerboseLog(ctx, cCfg, s.log, "searching process definitions", "baseURL", s.cfg.APIs.Operate.BaseURL, "body", body)
	resp, err := s.co.SearchProcessDefinitionsWithResponse(ctx, body)
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
	common.VerboseLog(ctx, cCfg, s.log, "found process definitions", "count", len(out))
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
	cCfg := services.ApplyCallOptions(opts)
	if cCfg.WithStat {
		return d.ProcessDefinition{}, fmt.Errorf("process definition stats not supported in v8.7 API")
	}

	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return d.ProcessDefinition{}, fmt.Errorf("converting process definition key %q to int64: %w", key, err)
	}
	common.VerboseLog(ctx, cCfg, s.log, "retrieving process definition", "key", key)
	resp, err := s.co.GetProcessDefinitionByKeyWithResponse(ctx, oldKey)
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
	common.VerboseLog(ctx, cCfg, s.log, "process definition retrieved", "bpmnProcessId", resp.JSON200.BpmnProcessId, "version", resp.JSON200.Version)
	return fromProcessDefinitionResponse(*resp.JSON200), nil
}
