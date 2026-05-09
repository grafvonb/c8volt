// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"strings"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/incident/waiter"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/toolx"
)

// GetIncident loads a single incident by key for direct resolution planning and confirmation.
func (s *Service) GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("getting incident with key %s using generated camunda client", key))
	resp, err := s.cc.GetIncidentWithResponse(ctx, key)
	if err != nil {
		return d.ProcessInstanceIncidentDetail{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessInstanceIncidentDetail{}, err
	}
	return fromIncidentResult(*payload), nil
}

// ResolveIncident submits direct incident resolution without doing confirmation polling.
func (s *Service) ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("resolving incident with key %s using generated camunda client", key))
	resp, err := s.cc.ResolveIncidentWithResponse(ctx, key, camundav89.ResolveIncidentJSONRequestBody{})
	if err != nil {
		return d.IncidentResolutionResponse{Key: key}, err
	}
	result := d.IncidentResolutionResponse{
		Key:        key,
		Ok:         resp.StatusCode() >= 200 && resp.StatusCode() < 300,
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.Ok = false
		return result, err
	}
	return result, nil
}

// SearchProcessInstanceIncidents uses the scoped process-instance incident endpoint for active incident enrichment.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	callCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching incidents for process instance with key %s using generated camunda client", key))
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return nil, fmt.Errorf("building tenant incident filter: %w", err)
	}
	stateFilter, err := newIncidentSearchStateFilter(callCfg.IncidentState)
	if err != nil {
		return nil, fmt.Errorf("building incident state filter: %w", err)
	}
	errorTypeFilter, err := newIncidentSearchErrorTypeFilter(callCfg.IncidentErrorType)
	if err != nil {
		return nil, fmt.Errorf("building incident error type filter: %w", err)
	}
	filter := &camundav89.IncidentFilter{
		State:     stateFilter,
		TenantId:  tenantFilter,
		ErrorType: errorTypeFilter,
	}
	return s.searchProcessInstanceIncidentsPages(ctx, key, filter, callCfg.IncidentErrorMessage)
}

func newIncidentSearchStateFilter(state string) (*camundav89.IncidentStateFilterProperty, error) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "active":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumACTIVE)
	case "pending":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumPENDING)
	case "resolved":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumRESOLVED)
	case "migrated":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumMIGRATED)
	case "unknown":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumUNKNOWN)
	case "all":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported incident state %q", state)
	}
}

func newIncidentSearchErrorTypeFilter(errorType string) (*camundav89.IncidentErrorTypeFilterProperty, error) {
	normalized, ok := incidentfilter.NormalizeErrorType(errorType)
	if !ok {
		return nil, fmt.Errorf("unsupported incident error type %q", errorType)
	}
	if normalized == "" {
		return nil, nil
	}
	return newIncidentErrorTypeEqFilterPtr(camundav89.IncidentErrorTypeEnum(normalized))
}

func (s *Service) searchProcessInstanceIncidentsPages(ctx context.Context, key string, filter *camundav89.IncidentFilter, errorMessage string) ([]d.ProcessInstanceIncidentDetail, error) {
	const pageSize int32 = 1000
	var out []d.ProcessInstanceIncidentDetail
	for from := int32(0); ; {
		page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{From: from, Size: pageSize})
		body := camundav89.SearchProcessInstanceIncidentsJSONRequestBody{
			Page:   &page,
			Filter: filter,
		}
		resp, err := s.cc.SearchProcessInstanceIncidentsWithResponse(ctx, key, body)
		if err != nil {
			return nil, err
		}
		payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
		if err != nil {
			return nil, err
		}
		out = append(out, filterIncidentDetailsByMessage(errorMessage, toolx.MapSlice(payload.Items, fromIncidentResult))...)
		if !incidentSearchHasMore(payload.Page, from, len(payload.Items), pageSize) {
			return out, nil
		}
		from += int32(len(payload.Items))
	}
}

func filterIncidentDetailsByMessage(errorMessage string, items []d.ProcessInstanceIncidentDetail) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if !incidentfilter.ErrorMessageContains(errorMessage, item.ErrorMessage) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func incidentSearchHasMore(page camundav89.SearchQueryPageResponse, from int32, itemCount int, pageSize int32) bool {
	if itemCount == 0 {
		return false
	}
	visibleCount := int64(from) + int64(itemCount)
	if page.TotalItems > visibleCount {
		return true
	}
	return page.HasMoreTotalItems && itemCount >= int(pageSize)
}

// WaitForIncidentResolved polls direct incident lookup until the selected incident is no longer active.
func (s *Service) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForIncidentResolved(ctx, s, s.cfg, s.log, key, opts...)
}

// WaitForProcessInstanceIncidentsResolved polls process-instance incident lookup until the initial incident set is gone.
func (s *Service) WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForProcessInstanceIncidentsResolved(ctx, s, s.cfg, s.log, processInstanceKey, incidentKeys, opts...)
}

// newSearchQueryPageRequest builds the v8.9 page request for incident lookups.
func newSearchQueryPageRequest(pageReq d.ProcessInstancePageRequest) camundav89.SearchQueryPageRequest {
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}
