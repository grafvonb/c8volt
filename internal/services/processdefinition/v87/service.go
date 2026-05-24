// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
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

	page, err := s.SearchProcessDefinitionsPage(ctx, filter, d.ProcessDefinitionPageRequest{Size: size}, opts...)
	if err != nil {
		return nil, err
	}
	out := page.Items
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)
	common.VerboseLog(ctx, cCfg, s.log, "found process definitions", "count", len(out))
	return out, nil
}

// SearchProcessDefinitionsPage returns one locally windowed Operate process-definition search page.
func (s *Service) SearchProcessDefinitionsPage(ctx context.Context, filter d.ProcessDefinitionFilter, pageReq d.ProcessDefinitionPageRequest, opts ...services.CallOption) (d.ProcessDefinitionPage, error) {
	cCfg := services.ApplyCallOptions(opts)
	if err := ensureStatsSupported(cCfg); err != nil {
		return d.ProcessDefinitionPage{}, err
	}

	tenantID := common.EffectiveTenant(s.cfg)
	if cCfg.IgnoreTenant {
		tenantID = ""
	}
	fetchSize := pickProcessDefinitionSearchFetchSize(pageReq)
	body := searchProcessDefinitionsRequest(tenantID, filter, fetchSize)
	common.VerboseLog(ctx, cCfg, s.log, "searching process definitions", "baseURL", s.cfg.APIs.Operate.BaseURL, "body", body)
	resp, err := s.co.SearchProcessDefinitionsWithResponse(ctx, body)
	if err != nil {
		return d.ProcessDefinitionPage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessDefinitionPage{}, err
	}
	items := toolx.DerefSlicePtr(payload.Items, fromProcessDefinitionResponse)
	window, overflow := trimProcessDefinitionPageWindow(items, payload.Total, pageReq, fetchSize)
	return d.ProcessDefinitionPage{
		Items:         window,
		Request:       pageReq,
		OverflowState: overflow,
		ReportedTotal: newProcessDefinitionReportedTotal(payload.Total),
	}, nil
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

	oldKey, err := processDefinitionKeyInt64(key)
	if err != nil {
		return d.ProcessDefinition{}, err
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

func (s *Service) GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error) {
	cCfg := services.ApplyCallOptions(opts)
	if err := ensureStatsSupported(cCfg); err != nil {
		return "", err
	}

	oldKey, err := processDefinitionKeyInt64(key)
	if err != nil {
		return "", err
	}
	common.VerboseLog(ctx, cCfg, s.log, "retrieving process definition xml", "key", key)
	resp, err := s.co.GetProcessDefinitionAsXmlByKeyWithResponse(ctx, oldKey)
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

func processDefinitionKeyInt64(key string) (int64, error) {
	oldKey, err := toolx.StringToInt64(key)
	if err != nil {
		return 0, fmt.Errorf("converting process definition key %q to int64: %w", key, err)
	}
	return oldKey, nil
}

func ensureStatsSupported(cCfg *services.CallCfg) error {
	if cCfg != nil && cCfg.WithStat {
		return fmt.Errorf("process definition stats not supported in v8.7 API")
	}
	return nil
}

// newProcessDefinitionReportedTotal converts an Operate total into exact domain metadata.
func newProcessDefinitionReportedTotal(total *int64) *d.ProcessDefinitionReportedTotal {
	if total == nil {
		return nil
	}
	return &d.ProcessDefinitionReportedTotal{
		Count: *total,
		Kind:  d.ProcessDefinitionReportedTotalKindExact,
	}
}

// pickProcessDefinitionSearchFetchSize over-fetches enough records to emulate offset paging.
func pickProcessDefinitionSearchFetchSize(pageReq d.ProcessDefinitionPageRequest) int32 {
	if pageReq.Size <= 0 {
		return 0
	}
	fetchSize := pageReq.From + pageReq.Size
	if fetchSize <= 0 {
		return pageReq.Size
	}
	if fetchSize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return fetchSize
}

// trimProcessDefinitionPageWindow applies the requested local page window to v8.7 search results.
func trimProcessDefinitionPageWindow(items []d.ProcessDefinition, total *int64, pageReq d.ProcessDefinitionPageRequest, fetchSize int32) ([]d.ProcessDefinition, d.ProcessInstanceOverflowState) {
	if pageReq.From < 0 {
		pageReq.From = 0
	}
	start := int(pageReq.From)
	if start >= len(items) {
		return nil, pickProcessDefinitionOverflowState(total, pageReq, 0, len(items), fetchSize)
	}
	end := start + int(pageReq.Size)
	if end > len(items) {
		end = len(items)
	}
	window := items[start:end]
	return window, pickProcessDefinitionOverflowState(total, pageReq, len(window), len(items), fetchSize)
}

// pickProcessDefinitionOverflowState records whether more v8.7 results are visible after the local window.
func pickProcessDefinitionOverflowState(total *int64, pageReq d.ProcessDefinitionPageRequest, windowCount int, fetchedCount int, fetchSize int32) d.ProcessInstanceOverflowState {
	visibleThrough := int64(pageReq.From) + int64(windowCount)
	if total != nil {
		if *total > visibleThrough {
			return d.ProcessInstanceOverflowStateHasMore
		}
		return d.ProcessInstanceOverflowStateNoMore
	}
	if pageReq.From+pageReq.Size > consts.MaxPISearchSize {
		return d.ProcessInstanceOverflowStateIndeterminate
	}
	if int32(fetchedCount) < fetchSize {
		return d.ProcessInstanceOverflowStateNoMore
	}
	return d.ProcessInstanceOverflowStateIndeterminate
}

func searchProcessDefinitionsRequest(tenantID string, filter d.ProcessDefinitionFilter, size int32) operatev87.QueryProcessDefinition {
	return operatev87.QueryProcessDefinition{
		Filter: &operatev87.ProcessDefinition{
			BpmnProcessId: toolx.PtrIf(filter.BpmnProcessId, ""),
			TenantId:      toolx.PtrIf(tenantID, ""),
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
