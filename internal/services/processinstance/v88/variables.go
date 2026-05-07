// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
)

// SearchProcessInstanceVariables requests untruncated process-scope values so human limits stay a display choice.
func (s *Service) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching variables for process instance with key %s using generated camunda client", key))
	processInstanceKeyFilter, err := common.NewProcessInstanceKeyEqFilterPtr(key)
	if err != nil {
		return nil, fmt.Errorf("building process-instance variable filter: %w", err)
	}
	scopeKeyFilter, err := common.NewScopeKeyEqFilterPtr(key)
	if err != nil {
		return nil, fmt.Errorf("building process-instance variable scope filter: %w", err)
	}
	var tenantID *camundav88.TenantId
	if s.cfg.App.Tenant != "" {
		tenant := camundav88.TenantId(s.cfg.App.Tenant)
		tenantID = &tenant
	}
	page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{Size: 1000})
	order := camundav88.ASC
	sortByName := []camundav88.VariableSearchQuerySortRequest{{
		Field: camundav88.VariableSearchQuerySortRequestFieldName,
		Order: &order,
	}}
	body := camundav88.SearchVariablesJSONRequestBody{
		Filter: &camundav88.VariableFilter{
			ProcessInstanceKey: processInstanceKeyFilter,
			ScopeKey:           scopeKeyFilter,
			TenantId:           tenantID,
		},
		Page: &page,
		Sort: &sortByName,
	}
	truncateValues := false
	resp, err := s.cc.SearchVariablesWithResponse(ctx, &camundav88.SearchVariablesParams{TruncateValues: &truncateValues}, body)
	if err != nil {
		return nil, err
	}
	if _, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200); err != nil {
		return nil, err
	}
	payload, err := decodeSearchVariablesResponse(resp.Body, resp.JSON200)
	if err != nil {
		return nil, fmt.Errorf("decode process-instance variables: %w", err)
	}
	variables := toolx.MapSlice(payload.Items, fromVariableSearchResult)
	sort.SliceStable(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})
	return variables, nil
}

func (s *Service) UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
	_ = services.ApplyCallOptions(opts)
	if _, err := common.NewProcessInstanceKeyEqFilterPtr(key); err != nil {
		return d.ProcessInstanceVariableUpdateResponse{Key: key}, err
	}
	resp, err := s.cc.CreateElementInstanceVariablesWithResponse(ctx, camundav88.ElementInstanceKey(key), camundav88.CreateElementInstanceVariablesJSONRequestBody{
		Variables: variables,
	})
	if err != nil {
		return d.ProcessInstanceVariableUpdateResponse{Key: key}, err
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.ProcessInstanceVariableUpdateResponse{
			Key:        key,
			Ok:         false,
			StatusCode: resp.StatusCode(),
			Status:     resp.Status(),
		}, err
	}
	status := resp.Status()
	if status == "" {
		status = http.StatusText(resp.StatusCode())
	}
	return d.ProcessInstanceVariableUpdateResponse{
		Key:        key,
		Ok:         true,
		StatusCode: resp.StatusCode(),
		Status:     status,
	}, nil
}
