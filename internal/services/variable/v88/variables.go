// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	variablesearch "github.com/grafvonb/c8volt/internal/services/variable/search"
	"github.com/grafvonb/c8volt/internal/services/variable/waiter"
)

// SearchProcessInstanceVariables returns every untruncated root-scope variable.
func (s *Service) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return variablesearch.ProcessInstanceVariables(ctx, s, key, opts...)
}

// SearchProcessInstanceVariablesPage reads one normalized v8.8 variable page.
func (s *Service) SearchProcessInstanceVariablesPage(ctx context.Context, key string, pageReq d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
	return s.searchVariablesPage(ctx, d.ScopeVariableQuery{ProcessInstanceKey: key, ScopeKey: key}, pageReq, false, opts...)
}

// SearchScopeVariables reads all variables owned directly by the selected scope.
func (s *Service) SearchScopeVariables(ctx context.Context, query d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return variablesearch.ScopeVariables(ctx, s, query, opts...)
}

// SearchScopeVariablesPage validates identity before requesting a local page.
func (s *Service) SearchScopeVariablesPage(ctx context.Context, query d.ScopeVariableQuery, pageReq d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
	if err := query.Validate(); err != nil {
		return d.ProcessInstanceVariablePage{}, err
	}
	return s.searchVariablesPage(ctx, query, pageReq, true, opts...)
}

// searchVariablesPage shares wire requests while preserving legacy root errors
// and filtering; only explicit local reads enforce selected-scope identity evidence.
func (s *Service) searchVariablesPage(ctx context.Context, query d.ScopeVariableQuery, pageReq d.ProcessInstanceVariablePageRequest, strictScope bool, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
	if err := pageReq.Validate(); err != nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("validate process-instance variable page: %w", err)
	}
	callCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching pi %s scope %s variables", query.ProcessInstanceKey, query.ScopeKey))
	processInstanceKeyFilter, err := common.NewProcessInstanceKeyEqFilterPtr(query.ProcessInstanceKey)
	if err != nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("building process-instance variable filter: %w", err)
	}
	scopeKeyFilter, err := common.NewScopeKeyEqFilterPtr(query.ScopeKey)
	if err != nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("building process-instance variable scope filter: %w", err)
	}
	var tenantID *camundav88.TenantId
	if !callCfg.IgnoreTenant && s.cfg.App.Tenant != "" {
		tenant := camundav88.TenantId(s.cfg.App.Tenant)
		tenantID = &tenant
	}
	page := camundav88.SearchQueryPageRequest{}
	if pageReq.After != "" {
		_ = page.FromCursorForwardPagination(camundav88.CursorForwardPagination{After: camundav88.EndCursor(pageReq.After), Limit: &pageReq.Size})
	} else {
		_ = page.FromOffsetPagination(camundav88.OffsetPagination{From: &pageReq.From, Limit: &pageReq.Size})
	}
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
		return d.ProcessInstanceVariablePage{}, err
	}
	if resp == nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: empty variable search response", d.ErrMalformedResponse)
	}
	if _, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200); err != nil {
		return d.ProcessInstanceVariablePage{}, err
	}
	payload, err := decodeSearchVariablesResponse(resp.Body, pageReq, query, strictScope)
	if err != nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("decode process-instance variables: %w", err)
	}
	return payload, nil
}

// UpdateProcessInstanceVariables writes variables to the process instance element scope and confirms visibility unless no-wait is set.
func (s *Service) UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	if _, err := common.NewProcessInstanceKeyEqFilterPtr(key); err != nil {
		return d.ProcessInstanceVariableUpdateResponse{Key: key}, err
	}
	body := camundav88.CreateElementInstanceVariablesJSONRequestBody{
		Variables: variables,
	}
	resp, err := services.RetryCamundaMutation(ctx, s.log, "update pi variables", func(ctx context.Context) (*camundav88.CreateElementInstanceVariablesResponse, *http.Response, []byte, error) {
		resp, err := s.cc.CreateElementInstanceVariablesWithResponse(ctx, camundav88.ElementInstanceKey(key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
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

// UpdateScopeVariables writes variables strictly to the requested element scope without confirmation reads.
func (s *Service) UpdateScopeVariables(ctx context.Context, scopeKey string, variables map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
	_ = services.ApplyCallOptions(opts)
	result := d.ScopeVariableUpdateResponse{ScopeKey: scopeKey}
	if strings.TrimSpace(scopeKey) == "" {
		return result, fmt.Errorf("%w: scope key must not be blank", d.ErrValidation)
	}
	if len(variables) == 0 {
		return result, fmt.Errorf("%w: scope variables must not be empty", d.ErrValidation)
	}
	if _, err := common.NewScopeKeyEqFilterPtr(scopeKey); err != nil {
		return result, err
	}
	local := true
	body := camundav88.CreateElementInstanceVariablesJSONRequestBody{
		Local:     &local,
		Variables: variables,
	}
	resp, err := services.RetryCamundaMutation(ctx, s.log, "update scope variables", func(ctx context.Context) (*camundav88.CreateElementInstanceVariablesResponse, *http.Response, []byte, error) {
		resp, err := s.cc.CreateElementInstanceVariablesWithResponse(ctx, camundav88.ElementInstanceKey(scopeKey), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		return result, err
	}
	if resp == nil {
		return result, fmt.Errorf("%w: empty scope variable update response", d.ErrMalformedResponse)
	}
	result.StatusCode = resp.StatusCode()
	result.Status = resp.Status()
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return result, err
	}
	result.Accepted = true
	if result.Status == "" {
		result.Status = http.StatusText(result.StatusCode)
	}
	return result, nil
}

var _ waiter.VariableWaiter = (*Service)(nil)
