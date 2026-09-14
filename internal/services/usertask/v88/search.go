// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// SearchUserTasksPage executes one backend-filtered v8.8 user-task search request.
func (s *Service) SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, pageReq d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error) {
	if err := pageReq.Validate(); err != nil {
		return d.UserTaskSearchPage{}, fmt.Errorf("validate user-task search page: %w", err)
	}
	callCfg := services.ApplyCallOptions(opts)
	tenantID := common.EffectiveTenant(s.cfg)
	if callCfg.IgnoreTenant {
		tenantID = ""
	}
	body, err := userTaskSearchRequest(query, pageReq, tenantID)
	if err != nil {
		return d.UserTaskSearchPage{}, fmt.Errorf("build user-task search request: %w", err)
	}
	resp, err := s.cc.SearchUserTasksWithResponse(ctx, body)
	if err != nil {
		return d.UserTaskSearchPage{}, fmt.Errorf("search user tasks: %w", err)
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.UserTaskSearchPage{}, fmt.Errorf("search user tasks: %w", err)
	}
	return userTaskSearchPage(*payload, pageReq)
}

// userTaskSearchRequest maps stable filters to the scalar selector shapes used by v8.8.
func userTaskSearchRequest(query d.UserTaskSearchQuery, pageReq d.UserTaskPageRequest, tenantID string) (camundav88.SearchUserTasksJSONRequestBody, error) {
	assignee, err := common.NewStringEqFilterPtr(query.Assignee)
	if err != nil {
		return camundav88.SearchUserTasksJSONRequestBody{}, err
	}
	candidateUser, err := common.NewStringEqFilterPtr(query.CandidateUser)
	if err != nil {
		return camundav88.SearchUserTasksJSONRequestBody{}, err
	}
	candidateGroup, err := common.NewStringEqFilterPtr(query.CandidateGroup)
	if err != nil {
		return camundav88.SearchUserTasksJSONRequestBody{}, err
	}
	tenant, err := common.NewStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav88.SearchUserTasksJSONRequestBody{}, err
	}
	state, err := newUserTaskStateEqFilterPtr(query.State)
	if err != nil {
		return camundav88.SearchUserTasksJSONRequestBody{}, err
	}
	filter := &camundav88.UserTaskFilter{
		ProcessInstanceKey:   stringPtrAs[camundav88.ProcessInstanceKey](query.ProcessInstanceKey),
		ProcessDefinitionKey: stringPtrAs[camundav88.ProcessDefinitionKey](query.ProcessDefinitionKey),
		ProcessDefinitionId:  stringPtrAs[camundav88.ProcessDefinitionId](query.BpmnProcessId),
		ElementId:            stringPtrAs[camundav88.ElementId](query.ElementId),
		State:                state,
		Assignee:             assignee,
		CandidateUser:        candidateUser,
		CandidateGroup:       candidateGroup,
		TenantId:             tenant,
	}
	page := newUserTaskSearchPageRequest(pageReq)
	return camundav88.SearchUserTasksJSONRequestBody{Filter: filter, Page: &page}, nil
}

// stringPtrAs omits unset scalar filters while preserving generated string aliases.
func stringPtrAs[T ~string](value string) *T {
	if value == "" {
		return nil
	}
	converted := T(value)
	return &converted
}

// newUserTaskStateEqFilterPtr builds the generated equality union for a supplied task state.
func newUserTaskStateEqFilterPtr(value string) (*camundav88.UserTaskStateFilterProperty, error) {
	if value == "" {
		return nil, nil
	}
	filter := new(camundav88.UserTaskStateFilterProperty)
	if err := filter.FromUserTaskStateFilterProperty0(camundav88.UserTaskStateEnum(value)); err != nil {
		return nil, err
	}
	return filter, nil
}

// newUserTaskSearchPageRequest selects exactly one generated pagination union variant.
func newUserTaskSearchPageRequest(pageReq d.UserTaskPageRequest) camundav88.SearchQueryPageRequest {
	page := camundav88.SearchQueryPageRequest{}
	switch {
	case pageReq.After != "":
		_ = page.FromCursorForwardPagination(camundav88.CursorForwardPagination{After: camundav88.EndCursor(pageReq.After), Limit: &pageReq.Size})
	case pageReq.From != 0:
		_ = page.FromOffsetPagination(camundav88.OffsetPagination{From: &pageReq.From, Limit: &pageReq.Size})
	default:
		_ = page.FromLimitPagination(camundav88.LimitPagination{Limit: &pageReq.Size})
	}
	return page
}

// userTaskSearchPage maps backend items and normalizes total and continuation metadata.
func userTaskSearchPage(payload camundav88.UserTaskSearchQueryResult, pageReq d.UserTaskPageRequest) (d.UserTaskSearchPage, error) {
	items := make([]d.UserTask, len(payload.Items))
	for i, raw := range payload.Items {
		items[i] = fromUserTaskResult(raw)
		if err := validateSearchUserTask(items[i]); err != nil {
			return d.UserTaskSearchPage{}, err
		}
	}
	endCursor := ""
	if payload.Page.EndCursor != nil {
		endCursor = string(*payload.Page.EndCursor)
	}
	totalKind := d.UserTaskReportedTotalKindExact
	if payload.Page.HasMoreTotalItems {
		totalKind = d.UserTaskReportedTotalKindLowerBound
	}
	rawCount := int32(len(payload.Items))
	return d.UserTaskSearchPage{
		Items:         items,
		Request:       pageReq,
		RawItemCount:  rawCount,
		EndCursor:     endCursor,
		ReportedTotal: &d.UserTaskReportedTotal{Count: payload.Page.TotalItems, Kind: totalKind},
		ContinuationState: userTaskContinuationState(
			pageReq, rawCount, endCursor, payload.Page.TotalItems, payload.Page.HasMoreTotalItems,
		),
	}, nil
}

// validateSearchUserTask rejects successful search items missing stable identities.
func validateSearchUserTask(task d.UserTask) error {
	if task.Key == "" || task.State == "" || task.ProcessInstanceKey == "" {
		return fmt.Errorf("%w: user-task search returned an item missing key, state, or process instance key", d.ErrMalformedResponse)
	}
	return nil
}

// userTaskContinuationState preserves cursor evidence while distinguishing capped totals from exhaustion.
func userTaskContinuationState(pageReq d.UserTaskPageRequest, rawCount int32, endCursor string, total int64, capped bool) d.UserTaskContinuationState {
	if capped {
		if pageReq.After == "" && int64(pageReq.From)+int64(rawCount) < total {
			return d.UserTaskContinuationStateHasMore
		}
		if endCursor != "" && endCursor != pageReq.After {
			return d.UserTaskContinuationStateHasMore
		}
		return d.UserTaskContinuationStateIndeterminate
	}
	if pageReq.After == "" && int64(pageReq.From)+int64(rawCount) < total {
		return d.UserTaskContinuationStateHasMore
	}
	return d.UserTaskContinuationStateNoMore
}
