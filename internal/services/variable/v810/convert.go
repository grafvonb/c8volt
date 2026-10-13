// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"bytes"
	"encoding/json"
	"fmt"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
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
		VariableKey:        *r.VariableKey,
		ProcessInstanceKey: *r.ProcessInstanceKey,
		ScopeKey:           *r.ScopeKey,
		TenantId:           *r.TenantId,
		APITruncated:       variableAPITruncated(r),
	}
}

// decodeSearchVariablesResponse reads raw JSON because the generated v8.10 model drops value and truncation fields.
func decodeSearchVariablesResponse(body []byte, request d.ProcessInstanceVariablePageRequest, key string) (d.ProcessInstanceVariablePage, error) {
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
		if item.Name == nil || item.Value == nil || item.VariableKey == nil || item.ProcessInstanceKey == nil || item.ScopeKey == nil || item.TenantId == nil {
			return d.ProcessInstanceVariablePage{}, fmt.Errorf("%w: variable item %d is missing required fields", d.ErrMalformedResponse, i)
		}
		if *item.ProcessInstanceKey == key && *item.ScopeKey == key {
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
func newProcessInstanceKeyEqFilterPtr(v string) (*camundav810.ProcessInstanceKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.ProcessInstanceKeyFilterProperty
	if err := f.FromProcessInstanceKeyFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

// newScopeKeyEqFilterPtr builds an equality filter for a variable scope key.
func newScopeKeyEqFilterPtr(v string) (*camundav810.ScopeKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.ScopeKeyFilterProperty
	if err := f.FromScopeKeyFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}
