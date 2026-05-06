// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"sort"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// SearchProcessInstanceVariables requests untruncated process-scope values so human limits stay a display choice.
func (s *Service) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching variables for process instance with key %s using generated camunda client", key))
	processInstanceKeyFilter, err := newProcessInstanceKeyEqFilterPtr(key)
	if err != nil {
		return nil, fmt.Errorf("building process-instance variable filter: %w", err)
	}
	scopeKeyFilter, err := newScopeKeyEqFilterPtr(key)
	if err != nil {
		return nil, fmt.Errorf("building process-instance variable scope filter: %w", err)
	}
	var tenantID *camundav89.TenantId
	if s.cfg.App.Tenant != "" {
		tenant := camundav89.TenantId(s.cfg.App.Tenant)
		tenantID = &tenant
	}
	page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{Size: 1000})
	order := camundav89.ASC
	sortByName := []camundav89.VariableSearchQuerySortRequest{{
		Field: camundav89.VariableSearchQuerySortRequestFieldName,
		Order: &order,
	}}
	body := camundav89.SearchVariablesJSONRequestBody{
		Filter: &camundav89.VariableFilter{
			ProcessInstanceKey: processInstanceKeyFilter,
			ScopeKey:           scopeKeyFilter,
			TenantId:           tenantID,
		},
		Page: &page,
		Sort: &sortByName,
	}
	truncateValues := false
	resp, err := s.cc.SearchVariablesWithResponse(ctx, &camundav89.SearchVariablesParams{TruncateValues: &truncateValues}, body)
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
