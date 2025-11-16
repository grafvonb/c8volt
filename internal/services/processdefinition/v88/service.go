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
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
)

type Service struct {
	cc  GenProcessDefinitionClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	c, err := camundav88.NewClientWithResponses(
		cfg.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cc: c, cfg: cfg, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)

	bodyFilter := &camundav88.ProcessDefinitionFilter{
		ProcessDefinitionId: common.NewStringEqFilterPtr(filter.BpmnProcessId),
		Version:             toolx.PtrIfNonZero(filter.ProcessVersion),
		VersionTag:          toolx.PtrIf(filter.ProcessVersionTag, ""),
		IsLatestVersion:     toolx.PtrIf(filter.IsLatestVersion, false),
	}
	page := camundav88.SearchQueryPageRequest{}
	from := int32(0)
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &from,
		Limit: &size,
	})
	orderDesc := camundav88.DESC
	orderAsc := camundav88.ASC
	sort := []camundav88.ProcessDefinitionSearchQuerySortRequest{
		{
			Field: camundav88.ProcessDefinitionSearchQuerySortRequestFieldVersion,
			Order: &orderDesc,
		},
		{
			Field: camundav88.ProcessDefinitionSearchQuerySortRequestFieldName,
			Order: &orderAsc,
		},
	}
	body := camundav88.SearchProcessDefinitionsJSONRequestBody{
		Filter: bodyFilter,
		Page:   &page,
		Sort:   &sort,
	}
	resp, err := s.cc.SearchProcessDefinitionsWithResponse(ctx, body)
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
	out := toolx.DerefSlicePtr(resp.JSON200.Items, fromProcessDefinitionResult)
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
	return out, nil
}

func (s *Service) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	filter.IsLatestVersion = true
	return s.SearchProcessDefinitions(ctx, filter, 1000, opts...)
}

func (s *Service) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	resp, err := s.cc.GetProcessDefinitionWithResponse(ctx, key)
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
	pd := fromProcessDefinitionResult(*resp.JSON200)
	if cCfg.WithStat {
		if err := s.retrieveProcessDefinitionStats(ctx, &pd); err != nil {
			return d.ProcessDefinition{}, err
		}
	}
	return pd, nil
}

func (s *Service) retrieveProcessDefinitionStats(ctx context.Context, pd *d.ProcessDefinition) error {
	s.log.Debug(fmt.Sprintf("retrieving process definition stats for key %q", pd.Key))
	stats, err := s.cc.GetProcessDefinitionStatisticsWithResponse(ctx, pd.Key, camundav88.GetProcessDefinitionStatisticsJSONRequestBody{
		Filter: &camundav88.ProcessDefinitionStatisticsFilter{},
	})
	if err != nil {
		return err
	}
	if err = httpc.HttpStatusErr(stats.HTTPResponse, stats.Body); err != nil {
		return err
	}
	if stats.JSON200 == nil || stats.JSON200.Items == nil {
		s.log.Warn(fmt.Sprintf("no process definition stats found for key %s", pd.Key))
		return nil
	}
	items := stats.JSON200.Items
	var ret d.ProcessDefinitionStatistics
	if len(*items) > 0 {
		ret = fromProcessElementStatisticsResult((*items)[len(*items)-1])
	} else {
		ret = d.ProcessDefinitionStatistics{}
	}
	pd.Statistics = &ret
	return nil
}
