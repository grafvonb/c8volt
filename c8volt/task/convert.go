// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"slices"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
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

func copyUserTaskVariableValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return copyUserTaskVariableMap(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = copyUserTaskVariableValue(value[i])
		}
		return out
	default:
		return value
	}
}

func copyUserTaskVariableMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for name, value := range values {
		out[name] = copyUserTaskVariableValue(value)
	}
	return out
}

func fromDomainUserTaskVariablePlannedValue(x d.UserTaskVariablePlannedValue) UserTaskVariablePlannedValue {
	return UserTaskVariablePlannedValue{
		Name: x.Name, ScopeKey: x.ScopeKey, Inherited: x.Inherited,
		Value: copyUserTaskVariableValue(x.Value), APITruncated: x.APITruncated,
	}
}

func toDomainUserTaskVariablePlannedValue(x UserTaskVariablePlannedValue) d.UserTaskVariablePlannedValue {
	return d.UserTaskVariablePlannedValue{
		Name: x.Name, ScopeKey: x.ScopeKey, Inherited: x.Inherited,
		Value: copyUserTaskVariableValue(x.Value), APITruncated: x.APITruncated,
	}
}

func fromDomainUserTaskVariablePlannedChange(x d.UserTaskVariablePlannedChange) UserTaskVariablePlannedChange {
	return UserTaskVariablePlannedChange{
		Name: x.Name, ScopeKey: x.ScopeKey, Inherited: x.Inherited,
		Before: copyUserTaskVariableValue(x.Before), After: copyUserTaskVariableValue(x.After), APITruncated: x.APITruncated,
	}
}

func toDomainUserTaskVariablePlannedChange(x UserTaskVariablePlannedChange) d.UserTaskVariablePlannedChange {
	return d.UserTaskVariablePlannedChange{
		Name: x.Name, ScopeKey: x.ScopeKey, Inherited: x.Inherited,
		Before: copyUserTaskVariableValue(x.Before), After: copyUserTaskVariableValue(x.After), APITruncated: x.APITruncated,
	}
}

func fromDomainScopeVariableUpdateAssociation(x d.ScopeVariableUpdateAssociation) ScopeVariableUpdateAssociation {
	return ScopeVariableUpdateAssociation{UserTaskKey: x.UserTaskKey, Names: slices.Clone(x.Names)}
}

func toDomainScopeVariableUpdateAssociation(x ScopeVariableUpdateAssociation) d.ScopeVariableUpdateAssociation {
	return d.ScopeVariableUpdateAssociation{UserTaskKey: x.UserTaskKey, Names: slices.Clone(x.Names)}
}

func fromDomainScopeVariableUpdateTarget(x d.ScopeVariableUpdateTarget) ScopeVariableUpdateTarget {
	return ScopeVariableUpdateTarget{
		ScopeKey: x.ScopeKey, TenantId: x.TenantId, Variables: copyUserTaskVariableMap(x.Variables),
		Associations: toolx.MapSlice(x.Associations, fromDomainScopeVariableUpdateAssociation),
	}
}

func toDomainScopeVariableUpdateTarget(x ScopeVariableUpdateTarget) d.ScopeVariableUpdateTarget {
	return d.ScopeVariableUpdateTarget{
		ScopeKey: x.ScopeKey, TenantId: x.TenantId, Variables: copyUserTaskVariableMap(x.Variables),
		Associations: toolx.MapSlice(x.Associations, toDomainScopeVariableUpdateAssociation),
	}
}

func fromDomainUserTaskVariablePlan(x d.UserTaskVariablePlan) UserTaskVariablePlan {
	return UserTaskVariablePlan{
		UserTaskKey: x.UserTaskKey, ElementInstanceKey: x.ElementInstanceKey, TenantId: x.TenantId,
		Additions:          toolx.MapSlice(x.Additions, fromDomainUserTaskVariablePlannedValue),
		Changes:            toolx.MapSlice(x.Changes, fromDomainUserTaskVariablePlannedChange),
		UnchangedRequested: toolx.MapSlice(x.UnchangedRequested, fromDomainUserTaskVariablePlannedValue),
		Untouched:          toolx.MapSlice(x.Untouched, fromDomainUserTaskVariablePlannedValue),
		TargetScopeKeys:    slices.Clone(x.TargetScopeKeys),
	}
}

func toDomainUserTaskVariablePlan(x UserTaskVariablePlan) d.UserTaskVariablePlan {
	return d.UserTaskVariablePlan{
		UserTaskKey: x.UserTaskKey, ElementInstanceKey: x.ElementInstanceKey, TenantId: x.TenantId,
		Additions:          toolx.MapSlice(x.Additions, toDomainUserTaskVariablePlannedValue),
		Changes:            toolx.MapSlice(x.Changes, toDomainUserTaskVariablePlannedChange),
		UnchangedRequested: toolx.MapSlice(x.UnchangedRequested, toDomainUserTaskVariablePlannedValue),
		Untouched:          toolx.MapSlice(x.Untouched, toDomainUserTaskVariablePlannedValue),
		TargetScopeKeys:    slices.Clone(x.TargetScopeKeys),
	}
}

func fromDomainTenantContext(x d.TenantContext) options.TenantContext {
	return options.TenantContext{
		Mode: options.TenantContextMode(x.Mode), Filter: options.TenantContextFilter(x.Filter),
		ConfiguredTenantID: x.ConfiguredTenantID, TargetTenantID: x.TargetTenantID,
		ResolvedTenantIDs: slices.Clone(x.ResolvedTenantIDs), UnknownTargetCount: x.UnknownTargetCount,
		CrossTenant: x.CrossTenant,
		Warnings: toolx.MapSlice(x.Warnings, func(w d.TenantContextWarning) options.TenantContextWarning {
			return options.TenantContextWarning{Code: options.TenantContextWarningCode(w.Code), Message: w.Message}
		}),
	}
}

func toDomainTenantContext(x options.TenantContext) d.TenantContext {
	return d.TenantContext{
		Mode: d.TenantContextMode(x.Mode), Filter: d.TenantContextFilter(x.Filter),
		ConfiguredTenantID: x.ConfiguredTenantID, TargetTenantID: x.TargetTenantID,
		ResolvedTenantIDs: slices.Clone(x.ResolvedTenantIDs), UnknownTargetCount: x.UnknownTargetCount,
		CrossTenant: x.CrossTenant,
		Warnings: toolx.MapSlice(x.Warnings, func(w options.TenantContextWarning) d.TenantContextWarning {
			return d.TenantContextWarning{Code: d.TenantContextWarningCode(w.Code), Message: w.Message}
		}),
	}
}

func fromDomainUserTaskVariableUpdatePlan(x d.UserTaskVariableUpdatePlan) UserTaskVariableUpdatePlan {
	return UserTaskVariableUpdatePlan{
		RequestedKeys: slices.Clone(x.RequestedKeys), UserTasks: toolx.MapSlice(x.UserTasks, fromDomainUserTaskVariablePlan),
		Targets: toolx.MapSlice(x.Targets, fromDomainScopeVariableUpdateTarget), RequestedCount: x.RequestedCount,
		UpdateCount: x.UpdateCount, VariableAddCount: x.VariableAddCount, VariableChangeCount: x.VariableChangeCount,
		VariableUnchangedCount: x.VariableUnchangedCount, VariableUntouchedCount: x.VariableUntouchedCount,
		MutationSubmitted: x.MutationSubmitted, TenantContext: fromDomainTenantContext(x.TenantContext),
	}
}

func toDomainUserTaskVariableUpdatePlan(x UserTaskVariableUpdatePlan) d.UserTaskVariableUpdatePlan {
	return d.UserTaskVariableUpdatePlan{
		RequestedKeys: slices.Clone(x.RequestedKeys), UserTasks: toolx.MapSlice(x.UserTasks, toDomainUserTaskVariablePlan),
		Targets: toolx.MapSlice(x.Targets, toDomainScopeVariableUpdateTarget), RequestedCount: x.RequestedCount,
		UpdateCount: x.UpdateCount, VariableAddCount: x.VariableAddCount, VariableChangeCount: x.VariableChangeCount,
		VariableUnchangedCount: x.VariableUnchangedCount, VariableUntouchedCount: x.VariableUntouchedCount,
		MutationSubmitted: x.MutationSubmitted, TenantContext: toDomainTenantContext(x.TenantContext),
	}
}

func fromDomainScopeVariableUpdateOutcome(x d.ScopeVariableUpdateOutcome) ScopeVariableUpdateOutcome {
	return ScopeVariableUpdateOutcome{
		ScopeKey: x.ScopeKey, Names: slices.Clone(x.Names), Status: ScopeVariableUpdateStatus(x.Status),
		Accepted: x.Accepted, StatusCode: x.StatusCode, Message: x.Message, Error: x.Error,
	}
}

func fromDomainUserTaskVariableUpdateResult(x d.UserTaskVariableUpdateResult) UserTaskVariableUpdateResult {
	return UserTaskVariableUpdateResult{
		Key: x.Key, Status: UserTaskVariableUpdateStatus(x.Status), MutationAccepted: x.MutationAccepted,
		ConfirmationStatus: x.ConfirmationStatus, Message: x.Message, Error: x.Error,
		Variables: copyUserTaskVariableMap(x.Variables), Scopes: toolx.MapSlice(x.Scopes, fromDomainScopeVariableUpdateOutcome),
	}
}

func fromDomainUserTaskVariableUpdateResults(x d.UserTaskVariableUpdateResults) UserTaskVariableUpdateResults {
	return UserTaskVariableUpdateResults{Items: toolx.MapSlice(x.Items, fromDomainUserTaskVariableUpdateResult)}
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

// toDomainVariableFilterClause maps one predicate without retaining its
// caller-owned optional existence pointer.
func toDomainVariableFilterClause(x VariableFilterClause) d.ProcessInstanceVariableFilterClause {
	return d.ProcessInstanceVariableFilterClause{
		Name:     x.Name,
		Operator: d.ProcessInstanceVariableFilterOperator(x.Operator),
		Value:    x.Value,
		Exists:   toolx.CopyPtr(x.Exists),
		Source:   x.Source,
	}
}

// toDomainVariableFilterSet allocates an independently owned ordered clause
// collection for the internal user-task query.
func toDomainVariableFilterSet(x VariableFilterSet) d.ProcessInstanceVariableFilterSet {
	return d.ProcessInstanceVariableFilterSet{
		Clauses: toolx.MapSlice(x.Clauses, toDomainVariableFilterClause),
	}
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
		VariableFilters:      toDomainVariableFilterSet(x.VariableFilters),
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
