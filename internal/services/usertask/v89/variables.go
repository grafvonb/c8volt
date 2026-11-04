// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// rawEffectiveVariablePage preserves required-field presence and response fields omitted by generated models.
type rawEffectiveVariablePage struct {
	Items *[]rawEffectiveVariable    `json:"items"`
	Page  *rawEffectiveVariableTotal `json:"page"`
}

// rawEffectiveVariable preserves valid empty strings while distinguishing them from absent required fields.
type rawEffectiveVariable struct {
	Name               *string `json:"name"`
	Value              *string `json:"value"`
	VariableKey        *string `json:"variableKey"`
	ProcessInstanceKey *string `json:"processInstanceKey"`
	ScopeKey           *string `json:"scopeKey"`
	TenantID           *string `json:"tenantId"`
	IsTruncated        *bool   `json:"isTruncated"`
	Truncated          *bool   `json:"truncated"`
}

// rawEffectiveVariableTotal preserves required total and capped-total presence plus optional continuation evidence.
type rawEffectiveVariableTotal struct {
	TotalItems        *int64  `json:"totalItems"`
	HasMoreTotalItems *bool   `json:"hasMoreTotalItems"`
	EndCursor         *string `json:"endCursor"`
}

// SearchUserTaskEffectiveVariablesPage reads one native offset page without applying discovery tenant filters.
func (s *Service) SearchUserTaskEffectiveVariablesPage(ctx context.Context, key string, pageReq d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error) {
	if err := pageReq.Validate(); err != nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("validate user-task effective-variable page: %w", err)
	}
	_ = services.ApplyCallOptions(opts)
	truncateValues := false
	order := camundav89.ASC
	sort := []camundav89.UserTaskVariableSearchQuerySortRequest{{
		Field: camundav89.UserTaskVariableSearchQuerySortRequestFieldName,
		Order: &order,
	}}
	body := camundav89.SearchUserTaskEffectiveVariablesJSONRequestBody{
		Page: &camundav89.OffsetPagination{From: &pageReq.From, Limit: &pageReq.Size},
		Sort: &sort,
	}
	resp, err := s.cc.SearchUserTaskEffectiveVariablesWithResponse(
		ctx,
		camundav89.UserTaskKey(key),
		&camundav89.SearchUserTaskEffectiveVariablesParams{TruncateValues: &truncateValues},
		body,
	)
	if err != nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("search effective variables for user task %s: %w", key, err)
	}
	if _, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200); err != nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("search effective variables for user task %s: %w", key, err)
	}
	page, err := decodeEffectiveVariablePage(resp.Body, pageReq)
	if err != nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("search effective variables for user task %s: %w", key, err)
	}
	return page, nil
}

// decodeEffectiveVariablePage validates the raw response before exposing stable domain paging facts.
func decodeEffectiveVariablePage(body []byte, pageReq d.UserTaskVariablePageRequest) (d.UserTaskVariablePage, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return d.UserTaskVariablePage{}, fmt.Errorf("%w: effective-variable response body is empty", d.ErrMalformedResponse)
	}
	var raw rawEffectiveVariablePage
	if err := json.Unmarshal(body, &raw); err != nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("%w: decode effective-variable response: %v", d.ErrMalformedResponse, err)
	}
	if raw.Items == nil || raw.Page == nil || raw.Page.TotalItems == nil || raw.Page.HasMoreTotalItems == nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("%w: effective-variable response is missing items or required page metadata", d.ErrMalformedResponse)
	}
	kind := d.UserTaskReportedTotalKindExact
	if *raw.Page.HasMoreTotalItems {
		kind = d.UserTaskReportedTotalKindLowerBound
	}
	reportedTotal := d.UserTaskReportedTotal{Count: *raw.Page.TotalItems, Kind: kind}
	if err := reportedTotal.Validate(); err != nil {
		return d.UserTaskVariablePage{}, fmt.Errorf("%w: invalid effective-variable total: %v", d.ErrMalformedResponse, err)
	}
	items := make([]d.ProcessInstanceVariable, len(*raw.Items))
	for i, item := range *raw.Items {
		if item.Name == nil || item.Value == nil || item.VariableKey == nil || item.ProcessInstanceKey == nil || item.ScopeKey == nil || item.TenantID == nil {
			return d.UserTaskVariablePage{}, fmt.Errorf("%w: effective-variable item %d is missing required fields", d.ErrMalformedResponse, i)
		}
		items[i] = d.ProcessInstanceVariable{
			Name:               *item.Name,
			Value:              *item.Value,
			VariableKey:        *item.VariableKey,
			ProcessInstanceKey: *item.ProcessInstanceKey,
			ScopeKey:           *item.ScopeKey,
			TenantId:           *item.TenantID,
			APITruncated:       effectiveVariableAPITruncated(item),
		}
	}
	hasContinuation := raw.Page.EndCursor != nil && *raw.Page.EndCursor != ""
	return d.UserTaskVariablePage{
		Items:                   items,
		Request:                 pageReq,
		RawItemCount:            int32(len(items)),
		ReportedTotal:           reportedTotal,
		HasContinuationEvidence: hasContinuation,
	}, nil
}

// effectiveVariableAPITruncated gives the newer field precedence while accepting the legacy alias.
func effectiveVariableAPITruncated(item rawEffectiveVariable) bool {
	if item.IsTruncated != nil {
		return *item.IsTruncated
	}
	return toolx.Deref(item.Truncated, false)
}
