// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	"strings"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/incident/waiter"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
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
	resp, err := s.cc.ResolveIncidentWithResponse(ctx, key, camundav88.ResolveIncidentJSONRequestBody{})
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

// SearchProcessInstanceIncidents uses the scoped process-instance incident
// endpoint for v8.8 but does not send an incident filter. Some v8.8 clusters
// expose the endpoint but reject the filter object at runtime, so c8volt keeps
// the request tenant-safe through the path and applies direct/state/tenant
// filtering locally.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	callCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching incidents for process instance with key %s using generated camunda client", key))
	return s.searchProcessInstanceIncidentsPages(ctx, key, common.EffectiveTenant(s.cfg), callCfg)
}

func filterIncidentResults(key string, tenant string, state string, errorType string, errorMessage string, items []camundav88.IncidentResult) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if item.ProcessInstanceKey != key {
			continue
		}
		if tenant != "" && item.TenantId != tenant {
			continue
		}
		if !incidentStateMatches(state, item.State) {
			continue
		}
		if !incidentfilter.ErrorTypeMatches(errorType, string(item.ErrorType)) {
			continue
		}
		if !incidentfilter.ErrorMessageContains(errorMessage, item.ErrorMessage) {
			continue
		}
		out = append(out, fromIncidentResult(item))
	}
	return out
}

func incidentStateMatches(want string, got camundav88.IncidentStateEnum) bool {
	switch strings.ToLower(strings.TrimSpace(want)) {
	case "", "active":
		return got == camundav88.IncidentStateEnumACTIVE
	case "all":
		return true
	default:
		return string(got) == strings.ToUpper(strings.TrimSpace(want))
	}
}

func (s *Service) searchProcessInstanceIncidentsPages(ctx context.Context, key string, tenant string, callCfg *services.CallCfg) ([]d.ProcessInstanceIncidentDetail, error) {
	const pageSize int32 = 1000
	var out []d.ProcessInstanceIncidentDetail
	for from := int32(0); ; {
		page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{From: from, Size: pageSize})
		body := camundav88.SearchProcessInstanceIncidentsJSONRequestBody{
			Page: &page,
		}
		resp, err := s.cc.SearchProcessInstanceIncidentsWithResponse(ctx, key, body)
		if err != nil {
			return nil, err
		}
		payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
		if err != nil {
			return nil, err
		}
		out = append(out, filterIncidentResults(key, tenant, callCfg.IncidentState, callCfg.IncidentErrorType, callCfg.IncidentErrorMessage, payload.Items)...)
		if !incidentSearchHasMore(payload.Page, from, len(payload.Items), pageSize) {
			return out, nil
		}
		from += int32(len(payload.Items))
	}
}

func incidentSearchHasMore(page camundav88.SearchQueryPageResponse, from int32, itemCount int, pageSize int32) bool {
	if itemCount == 0 {
		return false
	}
	visibleCount := int64(from) + int64(itemCount)
	if page.TotalItems > visibleCount {
		return true
	}
	return page.HasMoreTotalItems && itemCount >= int(pageSize)
}

func newIncidentSearchStateFilter(state string) (*camundav88.IncidentStateFilterProperty, error) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "active":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumACTIVE)
	case "pending":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumPENDING)
	case "resolved":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumRESOLVED)
	case "migrated":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumMIGRATED)
	case "unknown":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumUNKNOWN)
	case "all":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported incident state %q", state)
	}
}

// WaitForIncidentResolved polls direct incident lookup until the selected incident is no longer active.
func (s *Service) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForIncidentResolved(ctx, s, s.cfg, s.log, key, opts...)
}

// WaitForProcessInstanceIncidentsResolved polls process-instance incident lookup until the initial incident set is gone.
func (s *Service) WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForProcessInstanceIncidentsResolved(ctx, s, s.cfg, s.log, processInstanceKey, incidentKeys, opts...)
}

// newSearchQueryPageRequest builds the v8.8 page request for incident lookups.
func newSearchQueryPageRequest(pageReq d.ProcessInstancePageRequest) camundav88.SearchQueryPageRequest {
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}
