// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"bytes"
	"encoding/json"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// variableSearchQueryResult preserves raw variable fields that generated models omit.
type variableSearchQueryResult struct {
	Items []variableSearchResult             `json:"items"`
	Page  camundav89.SearchQueryPageResponse `json:"page"`
}

// variableSearchResult mirrors the variable search payload fields needed for display and confirmation.
type variableSearchResult struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	VariableKey        string `json:"variableKey"`
	ProcessInstanceKey string `json:"processInstanceKey"`
	ScopeKey           string `json:"scopeKey"`
	TenantId           string `json:"tenantId"`
	IsTruncated        *bool  `json:"isTruncated,omitempty"`
	Truncated          *bool  `json:"truncated,omitempty"`
}

// fromVariableSearchResult maps a raw variable search result to the shared domain model.
func fromVariableSearchResult(r variableSearchResult) d.ProcessInstanceVariable {
	return d.ProcessInstanceVariable{
		Name:               r.Name,
		Value:              r.Value,
		VariableKey:        r.VariableKey,
		ProcessInstanceKey: r.ProcessInstanceKey,
		ScopeKey:           r.ScopeKey,
		TenantId:           r.TenantId,
		APITruncated:       variableAPITruncated(r),
	}
}

// decodeSearchVariablesResponse reads raw JSON because the generated v8.9 model drops value and truncation fields.
func decodeSearchVariablesResponse(body []byte, page *camundav89.VariableSearchQueryResult) (variableSearchQueryResult, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return variableSearchQueryResult{}, d.ErrMalformedResponse
	}
	var result variableSearchQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return variableSearchQueryResult{}, err
	}
	if page != nil {
		result.Page = page.Page
	}
	return result, nil
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
