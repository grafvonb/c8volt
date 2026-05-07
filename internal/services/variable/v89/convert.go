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

type variableSearchQueryResult struct {
	Items []variableSearchResult             `json:"items"`
	Page  camundav89.SearchQueryPageResponse `json:"page"`
}

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

func variableAPITruncated(r variableSearchResult) bool {
	if r.IsTruncated != nil {
		return *r.IsTruncated
	}
	return toolx.Deref(r.Truncated, false)
}

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
