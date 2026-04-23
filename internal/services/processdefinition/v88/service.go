package v88

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
)

type Service struct {
	cc  GenProcessDefinitionClientCamunda
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) ClientCamunda() GenProcessDefinitionClientCamunda { return s.cc }
func (s *Service) Config() *config.Config                           { return s.cfg }
func (s *Service) Logger() *slog.Logger                             { return s.log }

type Option func(*Service)

func WithClientCamunda(c GenProcessDefinitionClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
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
	c, err := camundav88.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cc: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.cc)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	body, err := searchProcessDefinitionsRequest(common.EffectiveTenant(s.cfg), filter, size)
	if err != nil {
		return nil, fmt.Errorf("building process definition search request: %w", err)
	}
	common.VerboseLog(ctx, cCfg, s.log, "searching process definitions", "baseURL", s.cfg.APIs.Camunda.BaseURL, "body", body)
	resp, err := s.cc.SearchProcessDefinitionsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	out := toolx.MapSlice(payload.Items, fromProcessDefinitionResult)
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)

	if cCfg.WithStat {
		for i := range out {
			if out[i].Key == "" {
				continue
			}
			if err = s.retrieveProcessDefinitionStats(ctx, &out[i]); err != nil {
				return nil, err
			}
		}
	}
	common.VerboseLog(ctx, cCfg, s.log, "found process definitions", "count", len(out))
	return out, nil
}

func (s *Service) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	filter.IsLatestVersion = true
	return s.SearchProcessDefinitions(ctx, filter, 1000, opts...)
}

func (s *Service) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	common.VerboseLog(ctx, cCfg, s.log, "retrieving process definition", "key", key)
	resp, err := s.cc.GetProcessDefinitionWithResponse(ctx, key)
	if err != nil {
		return d.ProcessDefinition{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessDefinition{}, err
	}
	pd := fromProcessDefinitionResult(*payload)
	if cCfg.WithStat {
		if err := s.retrieveProcessDefinitionStats(ctx, &pd); err != nil {
			return d.ProcessDefinition{}, err
		}
	}
	common.VerboseLog(ctx, cCfg, s.log, "process definition retrieved", "bpmnProcessId", pd.BpmnProcessId, "version", pd.ProcessVersion)
	return pd, nil
}

func (s *Service) GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error) {
	cCfg := services.ApplyCallOptions(opts)
	common.VerboseLog(ctx, cCfg, s.log, "retrieving process definition xml", "key", key)
	resp, err := s.cc.GetProcessDefinitionXMLWithResponse(ctx, key)
	if err != nil {
		return "", err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.XML200)
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace([]byte(*payload))) == 0 && len(bytes.TrimSpace(resp.Body)) > 0 {
		return string(resp.Body), nil
	}
	common.VerboseLog(ctx, cCfg, s.log, "process definition xml retrieved", "key", key)
	return *payload, nil
}

func (s *Service) retrieveProcessDefinitionStats(ctx context.Context, pd *d.ProcessDefinition) error {
	s.log.Debug(fmt.Sprintf("retrieving process definition stats for key %q", pd.Key))
	active, err := s.countProcessInstancesForProcessDefinitionState(ctx, *pd, camundav88.ProcessInstanceStateEnumACTIVE)
	if err != nil {
		return err
	}
	completed, err := s.countProcessInstancesForProcessDefinitionState(ctx, *pd, camundav88.ProcessInstanceStateEnumCOMPLETED)
	if err != nil {
		return err
	}
	canceled, err := s.countProcessInstancesForProcessDefinitionState(ctx, *pd, camundav88.ProcessInstanceStateEnum(d.StateTerminated))
	if err != nil {
		return err
	}
	incidents, err := s.countProcessInstancesWithIncidentsForProcessDefinition(ctx, *pd)
	if err != nil {
		return err
	}
	ret := d.ProcessDefinitionStatistics{
		Active:    active,
		Completed: completed,
		Canceled:  canceled,
		Incidents: incidents,
	}
	ret.IncidentCountSupported = true
	pd.Statistics = &ret
	return nil
}

func (s *Service) countProcessInstancesForProcessDefinitionState(ctx context.Context, pd d.ProcessDefinition, state camundav88.ProcessInstanceStateEnum) (int64, error) {
	if pd.Key == "" {
		return 0, nil
	}
	req, err := searchProcessInstancesForDefinitionStateRequest(common.EffectiveTenant(s.cfg), pd.Key, state)
	if err != nil {
		return 0, err
	}
	resp, err := s.cc.SearchProcessInstancesWithResponse(ctx, req)
	if err != nil {
		return 0, err
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return 0, err
	}
	if resp.JSON200 == nil {
		return 0, d.ErrMalformedResponse
	}
	return resp.JSON200.Page.TotalItems, nil
}

func (s *Service) countProcessInstancesWithIncidentsForProcessDefinition(ctx context.Context, pd d.ProcessDefinition) (int64, error) {
	if pd.Key == "" {
		return 0, nil
	}
	req, err := searchProcessInstancesForDefinitionIncidentRequest(common.EffectiveTenant(s.cfg), pd.Key)
	if err != nil {
		return 0, err
	}
	resp, err := s.cc.SearchProcessInstancesWithResponse(ctx, req)
	if err != nil {
		return 0, err
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return 0, err
	}
	if resp.JSON200 == nil {
		return 0, d.ErrMalformedResponse
	}
	return resp.JSON200.Page.TotalItems, nil
}

func searchProcessDefinitionsRequest(tenantID string, filter d.ProcessDefinitionFilter, size int32) (camundav88.SearchProcessDefinitionsJSONRequestBody, error) {
	processDefinitionIDFilter, err := common.NewStringEqFilterPtr(filter.BpmnProcessId)
	if err != nil {
		return camundav88.SearchProcessDefinitionsJSONRequestBody{}, err
	}
	bodyFilter := &camundav88.ProcessDefinitionFilter{
		ProcessDefinitionId: processDefinitionIDFilter,
		TenantId:            toolx.PtrIf(tenantID, ""),
		Version:             toolx.PtrIfNonZero(filter.ProcessVersion),
		VersionTag:          toolx.PtrIf(filter.ProcessVersionTag, ""),
		IsLatestVersion:     toolx.PtrIf(filter.IsLatestVersion, false),
	}
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  new(int32(0)),
		Limit: &size,
	})
	sort := []camundav88.ProcessDefinitionSearchQuerySortRequest{
		{
			Field: camundav88.ProcessDefinitionSearchQuerySortRequestFieldVersion,
			Order: new(camundav88.DESC),
		},
		{
			Field: camundav88.ProcessDefinitionSearchQuerySortRequestFieldName,
			Order: new(camundav88.ASC),
		},
	}
	return camundav88.SearchProcessDefinitionsJSONRequestBody{
		Filter: bodyFilter,
		Page:   &page,
		Sort:   &sort,
	}, nil
}

func searchProcessInstancesForDefinitionStateRequest(tenantID, processDefinitionKey string, state camundav88.ProcessInstanceStateEnum) (camundav88.SearchProcessInstancesJSONRequestBody, error) {
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(processDefinitionKey)
	if err != nil {
		return camundav88.SearchProcessInstancesJSONRequestBody{}, err
	}
	stateFilter, err := common.NewProcessInstanceStateEqFilterPtr(string(state))
	if err != nil {
		return camundav88.SearchProcessInstancesJSONRequestBody{}, err
	}
	tenantIDFilter, err := common.NewStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav88.SearchProcessInstancesJSONRequestBody{}, err
	}
	var from int32
	var limit int32 = 1
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return camundav88.SearchProcessInstancesJSONRequestBody{
		Filter: &camundav88.ProcessInstanceFilter{
			ProcessDefinitionKey: processDefinitionKeyFilter,
			State:                stateFilter,
			TenantId:             tenantIDFilter,
		},
		Page: &page,
	}, nil
}

func searchProcessInstancesForDefinitionIncidentRequest(tenantID, processDefinitionKey string) (camundav88.SearchProcessInstancesJSONRequestBody, error) {
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(processDefinitionKey)
	if err != nil {
		return camundav88.SearchProcessInstancesJSONRequestBody{}, err
	}
	tenantIDFilter, err := common.NewStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav88.SearchProcessInstancesJSONRequestBody{}, err
	}
	hasIncident := true
	var from int32
	var limit int32 = 1
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return camundav88.SearchProcessInstancesJSONRequestBody{
		Filter: &camundav88.ProcessInstanceFilter{
			ProcessDefinitionKey: processDefinitionKeyFilter,
			TenantId:             tenantIDFilter,
			HasIncident:          &hasIncident,
		},
		Page: &page,
	}, nil
}

func newProcessDefinitionKeyEqFilterPtr(v string) (*camundav88.ProcessDefinitionKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var filter camundav88.ProcessDefinitionKeyFilterProperty
	if err := filter.FromProcessDefinitionKeyFilterProperty0(camundav88.ProcessDefinitionKey(v)); err != nil {
		return nil, err
	}
	return &filter, nil
}
