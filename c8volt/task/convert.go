// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromDomainUserTask maps a native domain task into an independently owned public value.
func fromDomainUserTask(x d.UserTask) UserTask {
	return UserTask{
		Key:                      x.Key,
		State:                    x.State,
		Name:                     x.Name,
		ElementId:                x.ElementId,
		ElementInstanceKey:       x.ElementInstanceKey,
		Assignee:                 x.Assignee,
		CandidateUsers:           append([]string(nil), x.CandidateUsers...),
		CandidateGroups:          append([]string(nil), x.CandidateGroups...),
		ProcessInstanceKey:       x.ProcessInstanceKey,
		ProcessDefinitionKey:     x.ProcessDefinitionKey,
		ProcessDefinitionId:      x.ProcessDefinitionId,
		ProcessDefinitionVersion: x.ProcessDefinitionVersion,
		TenantId:                 x.TenantId,
	}
}

// toDomainUserTask maps a selected public task into an independently owned
// service value without interpreting or replacing its metadata.
func toDomainUserTask(x UserTask) d.UserTask {
	return d.UserTask{
		Key:                      x.Key,
		State:                    x.State,
		Name:                     x.Name,
		ElementId:                x.ElementId,
		ElementInstanceKey:       x.ElementInstanceKey,
		Assignee:                 x.Assignee,
		CandidateUsers:           append([]string(nil), x.CandidateUsers...),
		CandidateGroups:          append([]string(nil), x.CandidateGroups...),
		ProcessInstanceKey:       x.ProcessInstanceKey,
		ProcessDefinitionKey:     x.ProcessDefinitionKey,
		ProcessDefinitionId:      x.ProcessDefinitionId,
		ProcessDefinitionVersion: x.ProcessDefinitionVersion,
		TenantId:                 x.TenantId,
	}
}

// fromDomainUserTasks maps a task slice and always initializes the public collection.
func fromDomainUserTasks(xs []d.UserTask) UserTasks {
	items := toolx.MapSlice(xs, fromDomainUserTask)
	if items == nil {
		items = []UserTask{}
	}
	return UserTasks{Total: int64(len(items)), Items: items}
}

// fromDomainUserTaskVariable maps every received variable field without
// applying process-root filtering or presentation shortening.
func fromDomainUserTaskVariable(x d.ProcessInstanceVariable) UserTaskVariable {
	return UserTaskVariable{
		Name:               x.Name,
		Value:              x.Value,
		VariableKey:        x.VariableKey,
		ProcessInstanceKey: x.ProcessInstanceKey,
		ScopeKey:           x.ScopeKey,
		TenantId:           x.TenantId,
		APITruncated:       x.APITruncated,
	}
}

// fromDomainVariableEnrichedUserTask maps one unchanged selected task and
// ensures its effective-variable collection remains initialized when empty.
func fromDomainVariableEnrichedUserTask(x d.VariableEnrichedUserTask) VariableEnrichedUserTask {
	variables := toolx.MapSlice(x.Variables, fromDomainUserTaskVariable)
	if variables == nil {
		variables = []UserTaskVariable{}
	}
	return VariableEnrichedUserTask{
		Item:      fromDomainUserTask(x.Item),
		Variables: variables,
	}
}

// fromDomainVariableEnrichedUserTasks maps the service result while preserving
// its returned-item total and initialized empty collection contract.
func fromDomainVariableEnrichedUserTasks(x d.VariableEnrichedUserTasks) VariableEnrichedUserTasks {
	items := toolx.MapSlice(x.Items, fromDomainVariableEnrichedUserTask)
	if items == nil {
		items = []VariableEnrichedUserTask{}
	}
	return VariableEnrichedUserTasks{Total: x.Total, Items: items}
}

// toDomainSearchRequest maps public selectors and bounds without adding tenant scope.
func toDomainSearchRequest(x SearchRequest) d.UserTaskSearchQuery {
	return d.UserTaskSearchQuery{
		ProcessInstanceKey:   x.ProcessInstanceKey,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		BpmnProcessId:        x.BpmnProcessId,
		ElementId:            x.ElementId,
		State:                x.State,
		Assignee:             x.Assignee,
		CandidateUser:        x.CandidateUser,
		CandidateGroup:       x.CandidateGroup,
		BatchSize:            x.BatchSize,
		Limit:                x.Limit,
	}
}

// fromDomainPageRequest maps page facts for observation without transferring ownership.
func fromDomainPageRequest(x d.UserTaskPageRequest) PageRequest {
	return PageRequest{From: x.From, Size: x.Size, After: x.After}
}

// fromDomainReportedTotal maps normalized backend total metadata.
func fromDomainReportedTotal(x d.UserTaskReportedTotal) ReportedTotal {
	return ReportedTotal{Count: x.Count, Kind: ReportedTotalKind(x.Kind)}
}

// fromDomainSearchPage maps a selected page and copies all task-owned slices.
func fromDomainSearchPage(x d.UserTaskSearchPage) SearchPage {
	items := toolx.MapSlice(x.Items, fromDomainUserTask)
	if items == nil {
		items = []UserTask{}
	}
	return SearchPage{
		Items:             items,
		Request:           fromDomainPageRequest(x.Request),
		RawItemCount:      x.RawItemCount,
		EndCursor:         x.EndCursor,
		ReportedTotal:     toolx.MapPtr(x.ReportedTotal, fromDomainReportedTotal),
		ContinuationState: ContinuationState(x.ContinuationState),
	}
}

// fromDomainSearchPageStep maps service-owned progress into visitor-facing facts.
func fromDomainSearchPageStep(x d.UserTaskSearchPageStep) SearchPageStep {
	return SearchPageStep{
		Page:            fromDomainSearchPage(x.Page),
		CumulativeCount: x.CumulativeCount,
		LimitReached:    x.LimitReached,
	}
}

// toDomainSearchPageVisitor mechanically adapts public visitor actions and errors.
func toDomainSearchPageVisitor(visitor SearchPageVisitor) d.UserTaskSearchPageVisitor {
	if visitor == nil {
		return nil
	}
	return func(step d.UserTaskSearchPageStep) (d.UserTaskSearchPageAction, error) {
		action, err := visitor(fromDomainSearchPageStep(step))
		return d.UserTaskSearchPageAction(action), err
	}
}

// fromDomainSearchPagesResult maps traversal output and initializes its item collection.
func fromDomainSearchPagesResult(x d.UserTaskSearchPagesResult) SearchPagesResult {
	items := toolx.MapSlice(x.Items, fromDomainUserTask)
	if items == nil {
		items = []UserTask{}
	}
	return SearchPagesResult{
		Items:      items,
		Total:      x.Total,
		Pages:      x.Pages,
		Completion: SearchCompletionDisposition(x.Completion),
	}
}
