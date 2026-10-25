// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromDomainElement maps an internal runtime element into the public facade type.
func fromDomainElement(result d.Element) Element {
	return Element{
		ElementInstanceKey:     result.ElementInstanceKey,
		ElementId:              result.ElementId,
		ElementName:            result.ElementName,
		Type:                   result.Type,
		State:                  result.State,
		StartDate:              result.StartDate,
		EndDate:                result.EndDate,
		ProcessInstanceKey:     result.ProcessInstanceKey,
		RootProcessInstanceKey: result.RootProcessInstanceKey,
		ProcessDefinitionId:    result.ProcessDefinitionId,
		ProcessDefinitionKey:   result.ProcessDefinitionKey,
		TenantId:               result.TenantId,
		HasIncident:            result.HasIncident,
		IncidentKey:            result.IncidentKey,
		Listeners:              fromDomainRuntimeListenerJobsPtr(result.Listeners),
		Variables:              fromDomainVariablesPtr(result.Variables),
	}
}

// fromDomainRuntimeListenerJob preserves optional listener lifecycle facts at the facade boundary.
func fromDomainRuntimeListenerJob(result d.RuntimeListenerJob) RuntimeListenerJob {
	return RuntimeListenerJob{
		JobKey:             result.JobKey,
		Kind:               result.Kind,
		ListenerEventType:  result.ListenerEventType,
		Type:               result.Type,
		State:              result.State,
		Retries:            result.Retries,
		Worker:             result.Worker,
		CreationTime:       result.CreationTime,
		EndTime:            result.EndTime,
		Deadline:           result.Deadline,
		ProcessInstanceKey: result.ProcessInstanceKey,
		ElementInstanceKey: result.ElementInstanceKey,
		ElementId:          result.ElementId,
		TenantId:           result.TenantId,
		ErrorCode:          result.ErrorCode,
		ErrorMessage:       result.ErrorMessage,
	}
}

func fromDomainRuntimeListenerJobsPtr(items *[]d.RuntimeListenerJob) *[]RuntimeListenerJob {
	if items == nil {
		return nil
	}
	out := toolx.MapSlice(*items, fromDomainRuntimeListenerJob)
	return &out
}

// toDomainSearchRequest maps public search filters into the internal service query.
func toDomainSearchRequest(request SearchRequest) d.ElementSearchQuery {
	return d.ElementSearchQuery{
		Key:                  request.Key,
		ProcessInstanceKey:   request.ProcessInstanceKey,
		ElementId:            request.ElementId,
		State:                request.State,
		Type:                 request.Type,
		ProcessDefinitionKey: request.ProcessDefinitionKey,
		BpmnProcessId:        request.BpmnProcessId,
		BatchSize:            request.BatchSize,
		Limit:                request.Limit,
	}
}

// fromDomainSearchResult maps a collected internal search result into the public result.
func fromDomainSearchResult(result d.ElementSearchResult) SearchResult {
	return SearchResult{
		Total: result.Total,
		Items: mapDomainElements(result.Items),
	}
}

// fromDomainSearchPagesResult maps service-owned traversal results into the
// public paged result shape.
func fromDomainSearchPagesResult(result d.ElementSearchPagesResult) SearchPagesResult {
	return SearchPagesResult{
		Items: mapDomainElements(result.Items),
		Total: result.Total,
		Pages: result.Pages,
	}
}

// toDomainSearchPageVisitor maps public page callbacks without letting facade
// callers depend on internal domain types.
func toDomainSearchPageVisitor(visitor SearchPageVisitor) d.ElementSearchPageVisitor {
	if visitor == nil {
		return nil
	}
	return func(step d.ElementSearchPageStep) (d.ElementSearchPageAction, error) {
		action, err := visitor(SearchPageStep{
			Page:            fromDomainPage(step.Page),
			CumulativeCount: step.CumulativeCount,
			LimitReached:    step.LimitReached,
		})
		return d.ElementSearchPageAction(action), err
	}
}

// toDomainPageRequest maps public pagination controls into the internal page request.
func toDomainPageRequest(request PageRequest) d.ElementPageRequest {
	return d.ElementPageRequest{
		From: request.From,
		Size: request.Size,
	}
}

// fromDomainPage maps an internal search page into the public page type.
func fromDomainPage(result d.ElementSearchPage) Page {
	return Page{
		Items: mapDomainElements(result.Items),
		Request: PageRequest{
			From: result.Request.From,
			Size: result.Request.Size,
		},
		OverflowState: fromDomainOverflowState(result.OverflowState),
		ReportedTotal: fromDomainReportedTotal(result.ReportedTotal),
	}
}

// fromDomainReportedTotal maps optional backend total metadata across the facade boundary.
func fromDomainReportedTotal(total *d.ElementReportedTotal) *ReportedTotal {
	if total == nil {
		return nil
	}
	return &ReportedTotal{
		Count: total.Count,
		Kind:  ReportedTotalKind(total.Kind),
	}
}

// fromDomainOverflowState keeps unsupported or indeterminate states conservative for public callers.
func fromDomainOverflowState(value d.ProcessInstanceOverflowState) OverflowState {
	switch value {
	case d.ProcessInstanceOverflowStateHasMore:
		return OverflowStateHasMore
	default:
		return OverflowStateNoMore
	}
}

// mapDomainElements converts element slices without sharing backing arrays with callers.
func mapDomainElements(items []d.Element) []Element {
	out := make([]Element, 0, len(items))
	for _, item := range items {
		out = append(out, fromDomainElement(item))
	}
	return out
}

// toDomainEnrichment maps the independent enrichment choices without adding policy.
func toDomainEnrichment(e Enrichment) d.ElementEnrichment {
	return d.ElementEnrichment{Variables: e.Variables, Listeners: e.Listeners}
}

// fromDomainVariable preserves raw serialized values and optional observation evidence.
func fromDomainVariable(v d.ProcessInstanceVariable) Variable {
	return Variable{Name: v.Name, Value: v.Value, VariableKey: v.VariableKey, ProcessInstanceKey: v.ProcessInstanceKey, ScopeKey: v.ScopeKey, TenantId: v.TenantId, APITruncated: v.APITruncated}
}

// fromDomainVariablesPtr distinguishes omitted enrichment from a successful empty read.
func fromDomainVariablesPtr(values *[]d.ProcessInstanceVariable) *[]Variable {
	if values == nil {
		return nil
	}
	out := toolx.MapSlice(*values, fromDomainVariable)
	if out == nil {
		out = []Variable{}
	}
	return &out
}

// copyElementPlanVariableValue isolates mutable JSON storage while
// retaining the established representation of ordinary interface collections.
func copyElementPlanVariableValue(value any) any {
	switch value := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return copyElementPlanVariableMap(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = copyElementPlanVariableValue(value[i])
		}
		return out
	case json.RawMessage:
		return append(json.RawMessage(nil), value...)
	default:
		return snapshotTypedElementPlanJSONValue(value)
	}
}

// snapshotTypedElementPlanJSONValue retains concrete typed JSON values when they round-trip faithfully.
// Marshal-only types fall back to their decoded JSON value so custom storage
// cannot alias the caller or silently change the approved serialized payload.
func snapshotTypedElementPlanJSONValue(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		// Invalid input is rejected before copying.
		return value
	}
	target := reflect.New(reflect.TypeOf(value))
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target.Interface()); err == nil {
		snapshot := target.Elem().Interface()
		if encoded, err := json.Marshal(snapshot); err == nil && bytes.Equal(raw, encoded) {
			return snapshot
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var snapshot any
	_ = decoder.Decode(&snapshot) // json.Marshal above guarantees one valid JSON value.
	return snapshot
}

// copyElementPlanVariableMap copies a variable map without sharing
// nested object or array values across the facade boundary.
func copyElementPlanVariableMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for name, value := range values {
		out[name] = copyElementPlanVariableValue(value)
	}
	return out
}

// fromDomainElementVariablePlannedValue maps one classified value and
// copies its nested JSON data.
func fromDomainElementVariablePlannedValue(x d.ElementVariablePlannedValue) ElementVariablePlannedValue {
	return ElementVariablePlannedValue{Name: x.Name, Value: copyElementPlanVariableValue(x.Value), APITruncated: x.APITruncated}
}

// fromDomainElementVariablePlannedChange maps one classified change
// and copies both observed and requested values.
func fromDomainElementVariablePlannedChange(x d.ElementVariablePlannedChange) ElementVariablePlannedChange {
	return ElementVariablePlannedChange{
		Name: x.Name, Before: copyElementPlanVariableValue(x.Before),
		After: copyElementPlanVariableValue(x.After), APITruncated: x.APITruncated,
	}
}

// fromDomainElementVariablePlan maps one local plan without sharing its
// classification or payload values.
func fromDomainElementVariablePlan(x d.ElementVariablePlan) ElementVariablePlan {
	return ElementVariablePlan{
		ElementKey: x.ElementKey, ProcessInstanceKey: x.ProcessInstanceKey, ScopeKey: x.ScopeKey, TenantId: x.TenantId,
		Additions:          toolx.MapSlice(x.Additions, fromDomainElementVariablePlannedValue),
		Changes:            toolx.MapSlice(x.Changes, fromDomainElementVariablePlannedChange),
		UnchangedRequested: toolx.MapSlice(x.UnchangedRequested, fromDomainElementVariablePlannedValue),
		Untouched:          toolx.MapSlice(x.Untouched, fromDomainElementVariablePlannedValue),
		MutationVariables:  copyElementPlanVariableMap(x.MutationVariables),
		VerificationValues: copyElementPlanVariableMap(x.VerificationValues),
	}
}

// fromDomainElementVariableTenantContext maps frozen tenant evidence
// into the public options model with independent slices.
func fromDomainElementVariableTenantContext(x d.TenantContext) options.TenantContext {
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

// fromDomainElementVariableUpdatePlan maps the complete frozen plan
// while preserving stable order, counts, and tenant evidence.
func fromDomainElementVariableUpdatePlan(x d.ElementVariableUpdatePlan) ElementVariableUpdatePlan {
	return ElementVariableUpdatePlan{
		RequestedKeys: slices.Clone(x.RequestedKeys), RequestedValues: copyElementPlanVariableMap(x.RequestedValues),
		Elements: toolx.MapSlice(x.Elements, fromDomainElementVariablePlan), RequestedCount: x.RequestedCount,
		UpdateCount: x.UpdateCount, VariableAddCount: x.VariableAddCount, VariableChangeCount: x.VariableChangeCount,
		VariableUnchangedCount: x.VariableUnchangedCount, VariableUntouchedCount: x.VariableUntouchedCount,
		MutationSubmitted: x.MutationSubmitted, TenantContext: fromDomainElementVariableTenantContext(x.TenantContext),
	}
}
