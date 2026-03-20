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
	if err := ensureStatsSupported(cCfg); err != nil {
		return nil, err
	}

	body := searchProcessDefinitionsRequest(filter, size)
	common.VerboseLog(ctx, cCfg, s.log, "searching process definitions", "baseURL", s.cfg.APIs.Operate.BaseURL, "body", body)
	resp, err := s.co.SearchProcessDefinitionsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	out := toolx.DerefSlicePtr(payload.Items, fromProcessDefinitionResponse)
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)
	common.VerboseLog(ctx, cCfg, s.log, "found process definitions", "count", len(out))
	return out, nil
}

func (s *Service) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	pds, err := s.SearchProcessDefinitions(ctx, filter, 1000, opts...)
	if err != nil {
		return nil, err
	}
	return latestProcessDefinitions(pds), nil
}

func (s *Service) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	if err := ensureStatsSupported(cCfg); err != nil {
		return d.ProcessDefinition{}, err
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
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessDefinition{}, err
	}
	common.VerboseLog(ctx, cCfg, s.log, "process definition retrieved", "bpmnProcessId", payload.BpmnProcessId, "version", payload.Version)
	return fromProcessDefinitionResponse(*payload), nil
}

func ensureStatsSupported(cCfg *services.CallCfg) error {
	if cCfg != nil && cCfg.WithStat {
		return fmt.Errorf("process definition stats not supported in v8.7 API")
	}
	return nil
}

func searchProcessDefinitionsRequest(filter d.ProcessDefinitionFilter, size int32) operatev87.QueryProcessDefinition {
	return operatev87.QueryProcessDefinition{
		Filter: &operatev87.ProcessDefinition{
			BpmnProcessId: toolx.PtrIf(filter.BpmnProcessId, ""),
			Version:       toolx.PtrIfNonZero(filter.ProcessVersion),
			VersionTag:    toolx.PtrIf(filter.ProcessVersionTag, ""),
		},
		Size: &size,
	}
}

func latestProcessDefinitions(pds []d.ProcessDefinition) []d.ProcessDefinition {
	latest := make(map[string]d.ProcessDefinition)
	for _, pd := range pds {
		if cur, ok := latest[pd.BpmnProcessId]; !ok || pd.ProcessVersion > cur.ProcessVersion {
			latest[pd.BpmnProcessId] = pd
		}
	}
	out := make([]d.ProcessDefinition, 0, len(latest))
	for _, pd := range latest {
		out = append(out, pd)
	}
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)
	return out
}
