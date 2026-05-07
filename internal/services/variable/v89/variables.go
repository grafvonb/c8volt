// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/variable/waiter"
	"github.com/grafvonb/c8volt/toolx"
)

// SearchProcessInstanceVariables returns untruncated process-scope variables for a v8.9 process instance.
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
	from := int32(0)
	size := int32(1000)
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{From: &from, Limit: &size})
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

// UpdateProcessInstanceVariables writes variables to the process instance element scope and confirms visibility unless no-wait is set.
func (s *Service) UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	if _, err := newProcessInstanceKeyEqFilterPtr(key); err != nil {
		return d.ProcessInstanceVariableUpdateResponse{Key: key}, err
	}
	resp, err := s.cc.CreateElementInstanceVariablesWithResponse(ctx, camundav89.ElementInstanceKey(key), camundav89.CreateElementInstanceVariablesJSONRequestBody{
		Variables: variables,
	})
	if err != nil {
		return d.ProcessInstanceVariableUpdateResponse{Key: key}, err
	}
	result := d.ProcessInstanceVariableUpdateResponse{
		Key:        key,
		Ok:         true,
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.Ok = false
		return result, err
	}
	if result.Status == "" {
		result.Status = http.StatusText(resp.StatusCode())
	}
	if cCfg.NoWait {
		return result, nil
	}
	if missing, err := waiter.WaitForProcessInstanceVariables(ctx, s, s.cfg, key, variables, opts...); err != nil {
		return result, err
	} else if len(missing) > 0 {
		return result, fmt.Errorf("requested variable value(s) not visible for process instance %s: %s", key, strings.Join(missing, ", "))
	}
	return result, nil
}

var _ waiter.VariableWaiter = (*Service)(nil)
