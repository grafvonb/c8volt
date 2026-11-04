// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// variableSearchQueryResult preserves raw variable fields that generated models omit.
type variableSearchQueryResult struct {
	Items *[]variableSearchResult `json:"items"`
	Page  *variableSearchPage     `json:"page"`
}

type variableSearchPage struct {
	TotalItems        *int64  `json:"totalItems"`
	HasMoreTotalItems *bool   `json:"hasMoreTotalItems"`
	EndCursor         *string `json:"endCursor"`
}

// variableSearchResult mirrors the variable search payload fields needed for display and confirmation.
type variableSearchResult struct {
	Name               *string `json:"name"`
	Value              *string `json:"value"`
	VariableKey        *string `json:"variableKey"`
	ProcessInstanceKey *string `json:"processInstanceKey"`
	ScopeKey           *string `json:"scopeKey"`
	TenantId           *string `json:"tenantId"`
	IsTruncated        *bool   `json:"isTruncated,omitempty"`
	Truncated          *bool   `json:"truncated,omitempty"`
}

// fromVariableSearchResult maps a raw variable search result to the shared domain model.
func fromVariableSearchResult(r variableSearchResult) d.ProcessInstanceVariable {
	return d.ProcessInstanceVariable{
		Name:               *r.Name,
		Value:              *r.Value,
		VariableKey:        toolx.Deref(r.VariableKey, ""),
		ProcessInstanceKey: *r.ProcessInstanceKey,
		ScopeKey:           *r.ScopeKey,
		TenantId:           toolx.Deref(r.TenantId, ""),
		APITruncated:       variableAPITruncated(r),
	}
}

// decodeSearchVariablesResponse preserves raw metadata before filtering. Strict local reads
// reject contradictory selected-scope evidence without tightening legacy root reads.
func decodeSearchVariablesResponse(body []byte, request d.ProcessInstanceVariablePageRequest, query d.ScopeVariableQuery, strictScope bool) (d.ProcessInstanceVariablePage, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return d.ProcessInstanceVariablePage{}, d.ErrMalformedResponse
	}
	var result variableSearchQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: %v", d.ErrMalformedResponse, err)
	}
	if result.Items == nil || result.Page == nil {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable response is missing items or page", d.ErrMalformedResponse)
	}
	if (result.Page.TotalItems == nil) != (result.Page.HasMoreTotalItems == nil) {
		return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable response has incomplete total metadata", d.ErrMalformedResponse)
	}
	page := d.ProcessInstanceVariablePage{Request: request, RawItemCount: int32(len(*result.Items)), EndCursor: toolx.Deref(result.Page.EndCursor, "")}
	if result.Page.TotalItems != nil {
		kind := d.ProcessInstanceVariableReportedTotalKindExact
		if *result.Page.HasMoreTotalItems {
			kind = d.ProcessInstanceVariableReportedTotalKindLowerBound
		}
		page.ReportedTotal = &d.ProcessInstanceVariableReportedTotal{Count: *result.Page.TotalItems, Kind: kind}
		if err := page.ReportedTotal.Validate(); err != nil {
			return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: invalid variable total: %v", d.ErrMalformedResponse, err)
		}
	}
	page.Items = make([]d.ProcessInstanceVariable, 0, len(*result.Items))
	for i, item := range *result.Items {
		if item.Name == nil || item.Value == nil || item.ProcessInstanceKey == nil || item.ScopeKey == nil ||
			(!strictScope && (item.VariableKey == nil || item.TenantId == nil)) {
			return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable item %d is missing required fields", d.ErrMalformedResponse, i)
		}
		if strictScope {
			if strings.TrimSpace(*item.ProcessInstanceKey) == "" || strings.TrimSpace(*item.ScopeKey) == "" {
				return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable item %d has blank identity", d.ErrMalformedResponse, i)
			}
			if *item.ScopeKey != query.ScopeKey {
				continue
			}
			if *item.ProcessInstanceKey != query.ProcessInstanceKey {
				return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable item %d has inconsistent scope owner", d.ErrMalformedResponse, i)
			}
			expected, observed := toolx.Deref(query.ExpectedTenantId, ""), toolx.Deref(item.TenantId, "")
			if expected != "" && observed != "" && expected != observed {
				return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable item %d has inconsistent scope tenant", d.ErrMalformedResponse, i)
			}
		}
		if *item.ProcessInstanceKey == query.ProcessInstanceKey && *item.ScopeKey == query.ScopeKey {
			page.Items = append(page.Items, fromVariableSearchResult(item))
		}
	}

	return page, nil
}

// variableAPITruncated accepts both observed truncation field names used by Camunda responses.
func variableAPITruncated(r variableSearchResult) bool {
	if r.IsTruncated != nil {
		return *r.IsTruncated
	}
	return toolx.Deref(r.Truncated, false)
}

// newProcessInstanceKeyEqFilterPtr builds an equality filter for a process-instance key.
func newProcessInstanceKeyEqFilterPtr(v string) (*camundav89.ProcessInstanceKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.ProcessInstanceKeyFilterProperty
	if err := f.FromProcessInstanceKeyFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

// newScopeKeyEqFilterPtr builds an equality filter for a variable scope key.
func newScopeKeyEqFilterPtr(v string) (*camundav89.ScopeKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.ScopeKeyFilterProperty
	if err := f.FromScopeKeyFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}
