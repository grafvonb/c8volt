// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"slices"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
)

// fromDomainProcessDefinition preserves supplied lifecycle values without inference or normalization.
func fromDomainProcessDefinition(x d.ProcessDefinition) ProcessDefinition {
	return ProcessDefinition{
		State:             x.State,
		BpmnProcessId:     x.BpmnProcessId,
		Key:               x.Key,
		Name:              x.Name,
		TenantId:          x.TenantId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
		Statistics:        toolx.MapPtr(x.Statistics, fromProcessDefinitionStatistics),
	}
}

// fromProcessDefinitionStatistics maps public counts without exposing internal cleanup evidence.
func fromProcessDefinitionStatistics(r d.ProcessDefinitionStatistics) ProcessDefinitionStatistics {
	return ProcessDefinitionStatistics{
		Active:                  r.Active,
		Suspended:               r.Suspended,
		SuspendedCountSupported: r.SuspendedCountSupported,
		Canceled:                r.Canceled,
		Completed:               r.Completed,
		Incidents:               r.Incidents,
		IncidentCountSupported:  r.IncidentCountSupported,
	}
}

func fromDomainProcessDefinitions(xs []d.ProcessDefinition) ProcessDefinitions {
	items := toolx.MapSlice(xs, fromDomainProcessDefinition)
	return ProcessDefinitions{
		Total: int32(len(items)),
		Items: items,
	}
}

func toDomainProcessDefinitionPageRequest(x ProcessDefinitionPageRequest) d.ProcessDefinitionPageRequest {
	return d.ProcessDefinitionPageRequest{
		From:  x.From,
		Size:  x.Size,
		After: x.After,
	}
}

func fromDomainProcessDefinitionPageRequest(x d.ProcessDefinitionPageRequest) ProcessDefinitionPageRequest {
	return ProcessDefinitionPageRequest{
		From:  x.From,
		Size:  x.Size,
		After: x.After,
	}
}

func fromDomainProcessDefinitionReportedTotal(x d.ProcessDefinitionReportedTotal) ProcessDefinitionReportedTotal {
	return ProcessDefinitionReportedTotal{
		Count: x.Count,
		Kind:  ProcessDefinitionReportedTotalKind(x.Kind),
	}
}

func fromDomainProcessDefinitionPage(x d.ProcessDefinitionPage) ProcessDefinitionPage {
	return ProcessDefinitionPage{
		Request:       fromDomainProcessDefinitionPageRequest(x.Request),
		OverflowState: ProcessInstanceOverflowState(x.OverflowState),
		ReportedTotal: toolx.MapPtr(x.ReportedTotal, fromDomainProcessDefinitionReportedTotal),
		EndCursor:     x.EndCursor,
		Items:         toolx.MapSlice(x.Items, fromDomainProcessDefinition),
	}
}

func toDomainProcessDefinitionSearchRequest(x ProcessDefinitionSearchRequest) d.ProcessDefinitionSearchRequest {
	return d.ProcessDefinitionSearchRequest{
		Filter: toDomainProcessDefinitionFilter(x.Filter),
		Page:   toDomainProcessDefinitionPageRequest(x.Page),
		Limit:  x.Limit,
		Latest: x.Latest,
	}
}

func toDomainProcessDefinitionSearchPageVisitor(visitor ProcessDefinitionSearchPageVisitor) d.ProcessDefinitionSearchPageVisitor {
	if visitor == nil {
		return nil
	}
	return func(step d.ProcessDefinitionSearchPageStep) (d.ProcessDefinitionSearchPageAction, error) {
		action, err := visitor(fromDomainProcessDefinitionSearchPageStep(step))
		return d.ProcessDefinitionSearchPageAction(action), err
	}
}

func fromDomainProcessDefinitionSearchPageStep(x d.ProcessDefinitionSearchPageStep) ProcessDefinitionSearchPageStep {
	return ProcessDefinitionSearchPageStep{
		Page:            fromDomainProcessDefinitionPage(x.Page),
		CumulativeCount: x.CumulativeCount,
		LimitReached:    x.LimitReached,
	}
}

func fromDomainProcessDefinitionSearchPagesResult(x d.ProcessDefinitionSearchPagesResult) ProcessDefinitionSearchPagesResult {
	return ProcessDefinitionSearchPagesResult{
		Items: toolx.MapSlice(x.Items, fromDomainProcessDefinition),
		Limit: x.Limit,
		Pages: x.Pages,
	}
}

// toDomainProcessDefinitionWatchSnapshotRequest maps public watch snapshot
// selection into the service-owned domain model.
func toDomainProcessDefinitionWatchSnapshotRequest(x ProcessDefinitionWatchSnapshotRequest) d.ProcessDefinitionWatchSnapshotRequest {
	return d.ProcessDefinitionWatchSnapshotRequest{
		Key:                    x.Key,
		Filter:                 toDomainProcessDefinitionFilter(x.Filter),
		Page:                   toDomainProcessDefinitionPageRequest(x.Page),
		Latest:                 x.Latest,
		WatchAllWhenUnselected: x.WatchAllWhenUnselected,
	}
}

// fromDomainProcessDefinitionWatchSnapshot maps service-collected watch
// snapshots across the public facade boundary.
func fromDomainProcessDefinitionWatchSnapshot(x d.ProcessDefinitionWatchSnapshot) ProcessDefinitionWatchSnapshot {
	return ProcessDefinitionWatchSnapshot{
		Items:         toolx.MapSlice(x.Items, fromDomainProcessDefinition),
		Total:         x.Total,
		Pages:         x.Pages,
		ReportedTotal: toolx.MapPtr(x.ReportedTotal, fromDomainProcessDefinitionReportedTotal),
		Empty:         x.Empty,
	}
}

// fromDomainProcessInstance preserves current metadata and snapshots nested variable storage at the public boundary.
func fromDomainProcessInstance(x d.ProcessInstance) ProcessInstance {
	return ProcessInstance{
		BusinessId:               toolx.CopyPtr(x.BusinessId),
		BpmnProcessId:            x.BpmnProcessId,
		EndDate:                  x.EndDate,
		Incident:                 x.Incident,
		Key:                      x.Key,
		ParentElementInstanceKey: x.ParentElementInstanceKey,
		ParentKey:                x.ParentKey,
		ProcessDefinitionKey:     x.ProcessDefinitionKey,
		RootProcessInstanceKey:   x.RootProcessInstanceKey,
		ProcessVersion:           x.ProcessVersion,
		ProcessVersionTag:        x.ProcessVersionTag,
		StartDate:                x.StartDate,
		SuspendedDate:            x.SuspendedDate,
		State:                    State(x.State),
		TenantId:                 x.TenantId,
		Variables:                copyProcessInstancePlanVariableMap(x.Variables),
	}
}

// fromDomainProcessInstanceCreation copies optional business identity without inferring an absent value.
func fromDomainProcessInstanceCreation(x d.ProcessInstanceCreation) ProcessInstance {
	return ProcessInstance{
		BusinessId:           toolx.CopyPtr(x.BusinessId),
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		ProcessVersion:       x.ProcessDefinitionVersion,
		State:                State(x.State),
		Variables:            toolx.CopyMap(x.Variables),
		TenantId:             x.TenantId,
		StartDate:            x.StartDate,
	}
}

func fromDomainProcessInstances(xs []d.ProcessInstance) ProcessInstances {
	items := toolx.MapSlice(xs, fromDomainProcessInstance)
	return ProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}
}

func fromDomainProcessInstanceIncidentDetail(x d.ProcessInstanceIncidentDetail) ProcessInstanceIncidentDetail {
	return ProcessInstanceIncidentDetail{
		IncidentKey:            x.IncidentKey,
		CreationTime:           x.CreationTime,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		TenantId:               x.TenantId,
		State:                  x.State,
		ErrorType:              x.ErrorType,
		ErrorMessage:           x.ErrorMessage,
		ElementId:              x.ElementId,
		ElementInstanceKey:     x.ElementInstanceKey,
		JobKey:                 x.JobKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
	}
}

func fromDomainProcessInstanceIncidentDetails(xs []d.ProcessInstanceIncidentDetail) []ProcessInstanceIncidentDetail {
	return toolx.MapSlice(xs, fromDomainProcessInstanceIncidentDetail)
}

func fromDomainProcessInstanceVariable(x d.ProcessInstanceVariable) ProcessInstanceVariable {
	return ProcessInstanceVariable{
		Name:               x.Name,
		Value:              x.Value,
		VariableKey:        x.VariableKey,
		ProcessInstanceKey: x.ProcessInstanceKey,
		ScopeKey:           x.ScopeKey,
		TenantId:           x.TenantId,
		APITruncated:       x.APITruncated,
	}
}

func fromDomainProcessInstanceVariables(xs []d.ProcessInstanceVariable) []ProcessInstanceVariable {
	return toolx.MapSlice(xs, fromDomainProcessInstanceVariable)
}

// fromDomainIncidentEnrichedProcessInstance maps one service-enriched process instance into the public facade model.
func fromDomainIncidentEnrichedProcessInstance(x d.IncidentEnrichedProcessInstance) IncidentEnrichedProcessInstance {
	return IncidentEnrichedProcessInstance{
		Item:      fromDomainProcessInstance(x.Item),
		Incidents: fromDomainProcessInstanceIncidentDetails(x.Incidents),
	}
}

// fromDomainIncidentEnrichedProcessInstances maps service-enriched process instances into the public facade model.
func fromDomainIncidentEnrichedProcessInstances(x d.IncidentEnrichedProcessInstances) IncidentEnrichedProcessInstances {
	return IncidentEnrichedProcessInstances{
		Total: x.Total,
		Items: toolx.MapSlice(x.Items, fromDomainIncidentEnrichedProcessInstance),
	}
}

// fromDomainVariableEnrichedProcessInstance maps one service-enriched process instance and its variables into the public facade model.
func fromDomainVariableEnrichedProcessInstance(x d.VariableEnrichedProcessInstance) VariableEnrichedProcessInstance {
	return VariableEnrichedProcessInstance{
		Item:      fromDomainProcessInstance(x.Item),
		Variables: fromDomainProcessInstanceVariables(x.Variables),
	}
}

// fromDomainVariableEnrichedProcessInstances maps service-enriched variables into the public facade model.
func fromDomainVariableEnrichedProcessInstances(x d.VariableEnrichedProcessInstances) VariableEnrichedProcessInstances {
	return VariableEnrichedProcessInstances{
		Total: x.Total,
		Items: toolx.MapSlice(x.Items, fromDomainVariableEnrichedProcessInstance),
	}
}

// fromDomainProcessInstanceElement maps one attached runtime element into the public process facade model.
func fromDomainProcessInstanceElement(x d.Element) ProcessInstanceElement {
	return ProcessInstanceElement{
		ElementInstanceKey:     x.ElementInstanceKey,
		ElementId:              x.ElementId,
		ElementName:            x.ElementName,
		Type:                   x.Type,
		State:                  x.State,
		StartDate:              x.StartDate,
		EndDate:                x.EndDate,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		TenantId:               x.TenantId,
		HasIncident:            x.HasIncident,
		IncidentKey:            x.IncidentKey,
		Listeners:              fromDomainRuntimeListenerJobsPtr(x.Listeners),
	}
}

// fromDomainRuntimeListenerJob copies optional listener identity across the facade boundary.
func fromDomainRuntimeListenerJob(x d.RuntimeListenerJob) RuntimeListenerJob {
	return RuntimeListenerJob{
		BusinessId:         toolx.CopyPtr(x.BusinessId),
		JobKey:             x.JobKey,
		Kind:               x.Kind,
		ListenerEventType:  x.ListenerEventType,
		Type:               x.Type,
		State:              x.State,
		Retries:            x.Retries,
		Worker:             x.Worker,
		CreationTime:       x.CreationTime,
		EndTime:            x.EndTime,
		Deadline:           x.Deadline,
		ProcessInstanceKey: x.ProcessInstanceKey,
		ElementInstanceKey: x.ElementInstanceKey,
		ElementId:          x.ElementId,
		TenantId:           x.TenantId,
		ErrorCode:          x.ErrorCode,
		ErrorMessage:       x.ErrorMessage,
	}
}

func fromDomainRuntimeListenerJobsPtr(xs *[]d.RuntimeListenerJob) *[]RuntimeListenerJob {
	if xs == nil {
		return nil
	}
	out := toolx.MapSlice(*xs, fromDomainRuntimeListenerJob)
	return &out
}

// fromDomainProcessInstanceElements copies attached runtime element rows across the facade boundary.
func fromDomainProcessInstanceElements(xs []d.Element) []ProcessInstanceElement {
	return toolx.MapSlice(xs, fromDomainProcessInstanceElement)
}

// fromDomainElementEnrichedProcessInstance maps one element-enriched process instance into the public facade model.
func fromDomainElementEnrichedProcessInstance(x d.ElementEnrichedProcessInstance) ElementEnrichedProcessInstance {
	return ElementEnrichedProcessInstance{
		Item:     fromDomainProcessInstance(x.Item),
		Elements: fromDomainProcessInstanceElements(x.Elements),
	}
}

// fromDomainElementEnrichedProcessInstances maps service-enriched elements into the public facade model.
func fromDomainElementEnrichedProcessInstances(x d.ElementEnrichedProcessInstances) ElementEnrichedProcessInstances {
	return ElementEnrichedProcessInstances{
		Total: x.Total,
		Items: toolx.MapSlice(x.Items, fromDomainElementEnrichedProcessInstance),
	}
}

// fromDomainIncidentEnrichedTraversalItem maps one service-enriched traversal item into the public facade model.
func fromDomainIncidentEnrichedTraversalItem(x d.IncidentEnrichedTraversalItem) IncidentEnrichedTraversalItem {
	return IncidentEnrichedTraversalItem{
		Item:      fromDomainProcessInstance(x.Item),
		Incidents: fromDomainProcessInstanceIncidentDetails(x.Incidents),
	}
}

// fromDomainIncidentEnrichedTraversalResult maps service-enriched traversal output into the public facade model.
func fromDomainIncidentEnrichedTraversalResult(x d.IncidentEnrichedTraversalResult) IncidentEnrichedTraversalResult {
	return IncidentEnrichedTraversalResult{
		Mode:             TraversalMode(x.Mode),
		Outcome:          TraversalOutcome(x.Outcome),
		StartKey:         x.StartKey,
		RootKey:          x.RootKey,
		Keys:             append([]string(nil), x.Keys...),
		Edges:            x.Edges,
		Items:            toolx.MapSlice(x.Items, fromDomainIncidentEnrichedTraversalItem),
		MissingAncestors: toolx.MapSlice(x.MissingAncestors, fromDomainMissingAncestor),
		Warning:          x.Warning,
	}
}

// fromDomainMissingAncestor maps one domain missing-ancestor marker into the public facade model.
func fromDomainMissingAncestor(item d.MissingAncestor) MissingAncestor {
	return MissingAncestor{Key: item.Key, StartKey: item.StartKey}
}

func fromDomainProcessInstanceVariableUpdateResult(x d.ProcessInstanceVariableUpdateResult) ProcessInstanceVariableUpdateResult {
	return ProcessInstanceVariableUpdateResult{
		Key:                x.Key,
		Status:             ProcessInstanceVariableUpdateStatus(x.Status),
		MutationAccepted:   x.MutationAccepted,
		ConfirmationStatus: x.ConfirmationStatus,
		StatusCode:         x.StatusCode,
		Message:            x.Message,
		Error:              x.Error,
		Variables:          copyProcessInstancePlanVariableMap(x.Variables),
	}
}

// toDomainProcessInstanceVariablePlannedValue copies one public classified value.
func toDomainProcessInstanceVariablePlannedValue(x ProcessInstanceVariablePlannedValue) d.ProcessInstanceVariablePlannedValue {
	return d.ProcessInstanceVariablePlannedValue{Name: x.Name, Value: copyProcessInstancePlanVariableValue(x.Value), APITruncated: x.APITruncated}
}

// toDomainProcessInstanceVariablePlannedChange copies one public classified change.
func toDomainProcessInstanceVariablePlannedChange(x ProcessInstanceVariablePlannedChange) d.ProcessInstanceVariablePlannedChange {
	return d.ProcessInstanceVariablePlannedChange{
		Name: x.Name, Before: copyProcessInstancePlanVariableValue(x.Before),
		After: copyProcessInstancePlanVariableValue(x.After), APITruncated: x.APITruncated,
	}
}

// toDomainProcessInstanceVariablePlan copies one public root plan.
func toDomainProcessInstanceVariablePlan(x ProcessInstanceVariablePlan) d.ProcessInstanceVariablePlan {
	return d.ProcessInstanceVariablePlan{
		BusinessId:         toolx.CopyPtr(x.BusinessId),
		ProcessInstanceKey: x.ProcessInstanceKey, ScopeKey: x.ScopeKey, TenantId: x.TenantId,
		Additions:          toolx.MapSlice(x.Additions, toDomainProcessInstanceVariablePlannedValue),
		Changes:            toolx.MapSlice(x.Changes, toDomainProcessInstanceVariablePlannedChange),
		UnchangedRequested: toolx.MapSlice(x.UnchangedRequested, toDomainProcessInstanceVariablePlannedValue),
		Untouched:          toolx.MapSlice(x.Untouched, toDomainProcessInstanceVariablePlannedValue),
		MutationVariables:  copyProcessInstancePlanVariableMap(x.MutationVariables),
		VerificationValues: copyProcessInstancePlanVariableMap(x.VerificationValues),
	}
}

// toDomainProcessInstanceVariableTenantContext copies public tenant evidence.
func toDomainProcessInstanceVariableTenantContext(x options.TenantContext) d.TenantContext {
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

// toDomainProcessInstanceVariableUpdatePlan deeply copies an executable public plan.
func toDomainProcessInstanceVariableUpdatePlan(x ProcessInstanceVariableUpdatePlan) d.ProcessInstanceVariableUpdatePlan {
	return d.ProcessInstanceVariableUpdatePlan{
		Selection: toolx.MapPtr(x.Selection, toDomainProcessInstanceSelection), SelectionTenant: x.SelectionTenant,
		RequestedKeys: slices.Clone(x.RequestedKeys), RequestedValues: copyProcessInstancePlanVariableMap(x.RequestedValues),
		Instances: toolx.MapSlice(x.Instances, toDomainProcessInstanceVariablePlan), RequestedCount: x.RequestedCount,
		UpdateCount: x.UpdateCount, VariableAddCount: x.VariableAddCount, VariableChangeCount: x.VariableChangeCount,
		VariableUnchangedCount: x.VariableUnchangedCount, VariableUntouchedCount: x.VariableUntouchedCount,
		MutationSubmitted: x.MutationSubmitted, TenantContext: toDomainProcessInstanceVariableTenantContext(x.TenantContext),
	}
}

// copyProcessInstancePlanVariableValue isolates mutable JSON storage while
// retaining the established representation of ordinary interface collections.
func copyProcessInstancePlanVariableValue(value any) any {
	switch value := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return copyProcessInstancePlanVariableMap(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = copyProcessInstancePlanVariableValue(value[i])
		}
		return out
	case json.RawMessage:
		return append(json.RawMessage(nil), value...)
	default:
		return snapshotTypedProcessInstancePlanJSONValue(value)
	}
}

// snapshotTypedProcessInstancePlanJSONValue round-trips typed JSON collections
// into the same concrete type to sever their backing storage at the facade.
func snapshotTypedProcessInstancePlanJSONValue(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	target := reflect.New(reflect.TypeOf(value))
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target.Interface()); err != nil {
		return value
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return value
	}
	return target.Elem().Interface()
}

// copyProcessInstancePlanVariableMap copies a variable map without sharing
// nested object or array values across the facade boundary.
func copyProcessInstancePlanVariableMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for name, value := range values {
		out[name] = copyProcessInstancePlanVariableValue(value)
	}
	return out
}

// fromDomainProcessInstanceVariablePlannedValue maps one classified value and
// copies its nested JSON data.
func fromDomainProcessInstanceVariablePlannedValue(x d.ProcessInstanceVariablePlannedValue) ProcessInstanceVariablePlannedValue {
	return ProcessInstanceVariablePlannedValue{Name: x.Name, Value: copyProcessInstancePlanVariableValue(x.Value), APITruncated: x.APITruncated}
}

// fromDomainProcessInstanceVariablePlannedChange maps one classified change
// and copies both observed and requested values.
func fromDomainProcessInstanceVariablePlannedChange(x d.ProcessInstanceVariablePlannedChange) ProcessInstanceVariablePlannedChange {
	return ProcessInstanceVariablePlannedChange{
		Name: x.Name, Before: copyProcessInstancePlanVariableValue(x.Before),
		After: copyProcessInstancePlanVariableValue(x.After), APITruncated: x.APITruncated,
	}
}

// fromDomainProcessInstanceVariablePlan maps one root plan without sharing its
// classification or payload values.
func fromDomainProcessInstanceVariablePlan(x d.ProcessInstanceVariablePlan) ProcessInstanceVariablePlan {
	return ProcessInstanceVariablePlan{
		BusinessId:         toolx.CopyPtr(x.BusinessId),
		ProcessInstanceKey: x.ProcessInstanceKey, ScopeKey: x.ScopeKey, TenantId: x.TenantId,
		Additions:          toolx.MapSlice(x.Additions, fromDomainProcessInstanceVariablePlannedValue),
		Changes:            toolx.MapSlice(x.Changes, fromDomainProcessInstanceVariablePlannedChange),
		UnchangedRequested: toolx.MapSlice(x.UnchangedRequested, fromDomainProcessInstanceVariablePlannedValue),
		Untouched:          toolx.MapSlice(x.Untouched, fromDomainProcessInstanceVariablePlannedValue),
		MutationVariables:  copyProcessInstancePlanVariableMap(x.MutationVariables),
		VerificationValues: copyProcessInstancePlanVariableMap(x.VerificationValues),
	}
}

// fromDomainProcessInstanceVariableTenantContext maps frozen tenant evidence
// into the public options model with independent slices.
func fromDomainProcessInstanceVariableTenantContext(x d.TenantContext) options.TenantContext {
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

// fromDomainProcessInstanceVariableUpdatePlan maps the complete frozen plan
// while preserving stable order, counts, and tenant evidence.
func fromDomainProcessInstanceVariableUpdatePlan(x d.ProcessInstanceVariableUpdatePlan) ProcessInstanceVariableUpdatePlan {
	return ProcessInstanceVariableUpdatePlan{
		Selection: toolx.MapPtr(x.Selection, fromDomainProcessInstanceSelection), SelectionTenant: x.SelectionTenant,
		RequestedKeys: slices.Clone(x.RequestedKeys), RequestedValues: copyProcessInstancePlanVariableMap(x.RequestedValues),
		Instances: toolx.MapSlice(x.Instances, fromDomainProcessInstanceVariablePlan), RequestedCount: x.RequestedCount,
		UpdateCount: x.UpdateCount, VariableAddCount: x.VariableAddCount, VariableChangeCount: x.VariableChangeCount,
		VariableUnchangedCount: x.VariableUnchangedCount, VariableUntouchedCount: x.VariableUntouchedCount,
		MutationSubmitted: x.MutationSubmitted, TenantContext: fromDomainProcessInstanceVariableTenantContext(x.TenantContext),
	}
}

// fromDomainProcessInstanceVariableUpdateResults copies selection alongside partial execution evidence.
func fromDomainProcessInstanceVariableUpdateResults(x d.ProcessInstanceVariableUpdateResults) ProcessInstanceVariableUpdateResults {
	return ProcessInstanceVariableUpdateResults{
		Selection: toolx.MapPtr(x.Selection, fromDomainProcessInstanceSelection),
		Items:     toolx.MapSlice(x.Items, fromDomainProcessInstanceVariableUpdateResult),
	}
}

func fromDomainProcessInstanceVariableUpdateResponse(x d.ProcessInstanceVariableUpdateResponse, variables map[string]any) ProcessInstanceVariableUpdateResult {
	status := ProcessInstanceVariableUpdateStatusSubmitted
	if !x.Ok {
		status = ProcessInstanceVariableUpdateStatusMutationFailed
	}
	return ProcessInstanceVariableUpdateResult{
		Key:              x.Key,
		Status:           status,
		MutationAccepted: x.Ok,
		StatusCode:       x.StatusCode,
		Message:          x.Status,
		Variables:        toolx.CopyMap(variables),
	}
}

func toDomainProcessInstanceVariableUpdateRequest(x ProcessInstanceVariableUpdateRequest) d.ProcessInstanceVariableUpdateRequest {
	return d.ProcessInstanceVariableUpdateRequest{
		Key:       x.Key,
		Variables: toolx.CopyMap(x.Variables),
	}
}

func fromDomainReporter(x d.Reporter) Reporter {
	return Reporter{
		Key:        x.Key,
		Ok:         x.Ok,
		StatusCode: x.StatusCode,
		Status:     x.Status,
	}
}

func fromDomainCancelReports(xs []d.Reporter) CancelReports {
	return CancelReports{Items: toolx.MapSlice(xs, func(x d.Reporter) CancelReport { return fromDomainReporter(x) })}
}

func fromDomainDeleteReports(xs []d.Reporter) DeleteReports {
	return DeleteReports{Items: toolx.MapSlice(xs, func(x d.Reporter) DeleteReport { return fromDomainReporter(x) })}
}

// fromDomainTenantEvidence copies service-resolved tenant evidence without
// adding operation semantics, which command code attaches later.
func fromDomainTenantEvidence(x d.TenantEvidence) TenantEvidence {
	return TenantEvidence{
		ResolvedTenantIDs:  append([]string(nil), x.ResolvedTenantIDs...),
		UnknownTargetCount: x.UnknownTargetCount,
		TargetCount:        x.TargetCount,
		Targets: toolx.MapSlice(x.Targets, func(target d.TenantEvidenceTarget) TenantEvidenceTarget {
			return TenantEvidenceTarget{Key: target.Key, TenantID: target.TenantID}
		}),
	}
}

func fromDomainDryRunPIKeyExpansion(x d.DryRunPIKeyExpansion) DryRunPIKeyExpansion {
	return DryRunPIKeyExpansion{
		Roots:                      append([]string(nil), x.Roots...),
		Collected:                  append([]string(nil), x.Collected...),
		TenantEvidence:             fromDomainTenantEvidence(x.TenantEvidence),
		SelectedFinalState:         toolx.MapSlice(x.SelectedFinalState, fromDomainProcessInstance),
		RequiresCancelBeforeDelete: toolx.MapSlice(x.RequiresCancelBeforeDelete, fromDomainProcessInstance),
		MissingAncestors: toolx.MapSlice(x.MissingAncestors, func(item d.MissingAncestor) MissingAncestor {
			return MissingAncestor{Key: item.Key, StartKey: item.StartKey}
		}),
		Warning: x.Warning,
		Outcome: TraversalOutcome(x.Outcome),
	}
}

func fromDomainProcessInstancePage(x d.ProcessInstancePage) ProcessInstancePage {
	return ProcessInstancePage{
		Request: ProcessInstancePageRequest{
			From:  x.Request.From,
			Size:  x.Request.Size,
			After: x.Request.After,
		},
		OverflowState: ProcessInstanceOverflowState(x.OverflowState),
		ReportedTotal: toolx.MapPtr(x.ReportedTotal, fromDomainProcessInstanceReportedTotal),
		EndCursor:     x.EndCursor,
		Items:         toolx.MapSlice(x.Items, fromDomainProcessInstance),
	}
}

// fromDomainProcessInstanceSearchPagesResult maps service-owned traversal results into the public facade model.
func fromDomainProcessInstanceSearchPagesResult(x d.ProcessInstanceSearchPagesResult) ProcessInstanceSearchPagesResult {
	return ProcessInstanceSearchPagesResult{
		Completion: ProcessInstanceSearchCompletion(x.Completion),
		Items:      toolx.MapSlice(x.Items, fromDomainProcessInstance),
		Limit:      x.Limit,
		Pages:      x.Pages,
	}
}

// fromDomainProcessInstanceSearchPageStep maps one service page callback into the public facade model.
func fromDomainProcessInstanceSearchPageStep(x d.ProcessInstanceSearchPageStep) ProcessInstanceSearchPageStep {
	return ProcessInstanceSearchPageStep{
		Page:            fromDomainProcessInstancePage(x.Page),
		CumulativeCount: x.CumulativeCount,
		LimitReached:    x.LimitReached,
	}
}

// fromDomainProcessInstanceMutationPlanPagesResult maps service-owned mutation
// planning traversal results into the public facade model.
func fromDomainProcessInstanceMutationPlanPagesResult(x d.ProcessInstanceMutationPlanPagesResult) ProcessInstanceMutationPlanPagesResult {
	return ProcessInstanceMutationPlanPagesResult{
		Plans:            toolx.MapSlice(x.Plans, fromDomainProcessInstanceMutationPlanStep),
		TenantEvidence:   fromDomainTenantEvidence(x.TenantEvidence),
		Limit:            x.Limit,
		Pages:            x.Pages,
		RequestedCount:   x.RequestedCount,
		CumulativeImpact: x.CumulativeImpact,
		Stopped:          x.Stopped,
	}
}

// fromDomainProcessInstanceMutationPlanStep maps one service page-level
// mutation plan into the public facade model.
func fromDomainProcessInstanceMutationPlanStep(x d.ProcessInstanceMutationPlanStep) ProcessInstanceMutationPlanStep {
	return ProcessInstanceMutationPlanStep{
		Page:             fromDomainProcessInstancePage(x.Page),
		RequestedKeys:    append([]string(nil), x.RequestedKeys...),
		Plan:             fromDomainDryRunPIKeyExpansion(x.Plan),
		CumulativeCount:  x.CumulativeCount,
		CumulativeImpact: x.CumulativeImpact,
		LimitReached:     x.LimitReached,
	}
}

// fromDomainProcessInstanceSearchTotalStep maps service total diagnostics into the public facade model.
func fromDomainProcessInstanceSearchTotalStep(x d.ProcessInstanceSearchTotalStep) ProcessInstanceSearchTotalStep {
	return ProcessInstanceSearchTotalStep{
		Page:             fromDomainProcessInstancePage(x.Page),
		FilteredCount:    x.FilteredCount,
		TotalBefore:      x.TotalBefore,
		TotalAfter:       x.TotalAfter,
		CountingByPaging: x.CountingByPaging,
		ExactTotalUsed:   x.ExactTotalUsed,
	}
}

func fromDomainOrphanDiscovery(x pisvc.OrphanDiscovery) OrphanDiscovery {
	return OrphanDiscovery{
		Filter: fromDomainProcessInstanceFilter(x.Filter),
		Items:  toolx.MapSlice(x.Items, fromDomainProcessInstance),
		Keys:   append([]string(nil), x.Keys...),
	}
}

func fromDomainOrphanDiscoveryProgress(x pisvc.OrphanDiscoveryProgress) OrphanDiscoveryProgress {
	return OrphanDiscoveryProgress{
		Page:                  x.Page,
		Phase:                 x.Phase,
		CurrentPageCandidates: x.CurrentPageCandidates,
		CurrentPageOrphans:    x.CurrentPageOrphans,
		CandidatesChecked:     x.CandidatesChecked,
		OrphansFound:          x.OrphansFound,
		Limit:                 x.Limit,
		OverflowState:         ProcessInstanceOverflowState(x.OverflowState),
	}
}

func fromDomainProcessInstanceReportedTotal(x d.ProcessInstanceReportedTotal) ProcessInstanceReportedTotal {
	return ProcessInstanceReportedTotal{
		Count: x.Count,
		Kind:  ProcessInstanceReportedTotalKind(x.Kind),
	}
}

func fromDomainProcessInstanceMap(xs map[string]d.ProcessInstance) map[string]ProcessInstance {
	return toolx.MapMap(xs, fromDomainProcessInstance)
}

// toDomainProcessInstanceMap maps public process-instance maps into domain values for service workflows.
func toDomainProcessInstanceMap(xs map[string]ProcessInstance) map[string]d.ProcessInstance {
	return toolx.MapMap(xs, toDomainProcessInstance)
}

// toServiceTraversalResult maps public traversal output back into service-layer traversal input for enrichment.
func toServiceTraversalResult(in TraversalResult) pitraversal.Result {
	return pitraversal.Result{
		Mode:             pitraversal.Mode(in.Mode),
		StartKey:         in.StartKey,
		RootKey:          in.RootKey,
		Keys:             append([]string(nil), in.Keys...),
		Edges:            in.Edges,
		Chain:            toDomainProcessInstanceMap(in.Chain),
		MissingAncestors: toServiceMissingAncestors(in.MissingAncestors),
		Warning:          in.Warning,
		Outcome:          pitraversal.Outcome(in.Outcome),
	}
}

// toServiceMissingAncestors maps public missing-ancestor markers into traversal package values.
func toServiceMissingAncestors(in []MissingAncestor) []pitraversal.MissingAncestor {
	if len(in) == 0 {
		return nil
	}
	out := make([]pitraversal.MissingAncestor, len(in))
	for i, item := range in {
		out[i] = pitraversal.MissingAncestor{
			Key:      item.Key,
			StartKey: item.StartKey,
		}
	}
	return out
}

func fromDomainProcessInstanceExpectationResponse(x d.ProcessInstanceExpectationResponse) ProcessInstanceExpectationReport {
	return ProcessInstanceExpectationReport{
		Key:      x.Key,
		Ok:       x.Ok,
		State:    State(x.State),
		Incident: fromDomainIncidentExpectation(x.Incident),
		Status:   x.Status,
	}
}

func fromDomainProcessInstanceExpectationResponses(xs d.ProcessInstanceExpectationResponses) ProcessInstanceExpectationReports {
	return ProcessInstanceExpectationReports{
		Items: toolx.MapSlice(xs.Items, fromDomainProcessInstanceExpectationResponse),
	}
}

func fromDomainIncidentExpectation(x *bool) *IncidentExpectation {
	if x == nil {
		return nil
	}
	out := IncidentExpectation(*x)
	return &out
}

// toDomainProcessInstance preserves metadata and snapshots caller-owned observations for service plans.
func toDomainProcessInstance(x ProcessInstance) d.ProcessInstance {
	return d.ProcessInstance{
		BusinessId:               toolx.CopyPtr(x.BusinessId),
		BpmnProcessId:            x.BpmnProcessId,
		EndDate:                  x.EndDate,
		Incident:                 x.Incident,
		Key:                      x.Key,
		ParentElementInstanceKey: x.ParentElementInstanceKey,
		ParentKey:                x.ParentKey,
		ProcessDefinitionKey:     x.ProcessDefinitionKey,
		RootProcessInstanceKey:   x.RootProcessInstanceKey,
		ProcessVersion:           x.ProcessVersion,
		ProcessVersionTag:        x.ProcessVersionTag,
		StartDate:                x.StartDate,
		SuspendedDate:            x.SuspendedDate,
		State:                    d.State(x.State),
		TenantId:                 x.TenantId,
		Variables:                copyProcessInstancePlanVariableMap(x.Variables),
	}
}

func toDomainProcessInstanceExpectationRequest(x ProcessInstanceExpectationRequest) d.ProcessInstanceExpectationRequest {
	return d.ProcessInstanceExpectationRequest{
		States:   toolx.MapSlice(x.States, func(s State) d.State { return d.State(s) }),
		Incident: toDomainIncidentExpectation(x.Incident),
	}
}

func toDomainIncidentExpectation(x *IncidentExpectation) *bool {
	if x == nil {
		return nil
	}
	out := x.Bool()
	return &out
}

func toDomainProcessInstanceIncidentDetail(x ProcessInstanceIncidentDetail) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
		IncidentKey:            x.IncidentKey,
		CreationTime:           x.CreationTime,
		ProcessInstanceKey:     x.ProcessInstanceKey,
		TenantId:               x.TenantId,
		State:                  x.State,
		ErrorType:              x.ErrorType,
		ErrorMessage:           x.ErrorMessage,
		ElementId:              x.ElementId,
		ElementInstanceKey:     x.ElementInstanceKey,
		JobKey:                 x.JobKey,
		RootProcessInstanceKey: x.RootProcessInstanceKey,
		ProcessDefinitionKey:   x.ProcessDefinitionKey,
		ProcessDefinitionId:    x.ProcessDefinitionId,
	}
}

func toDomainProcessDefinitionFilter(x ProcessDefinitionFilter) d.ProcessDefinitionFilter {
	return d.ProcessDefinitionFilter{
		Key:               x.Key,
		BpmnProcessId:     x.BpmnProcessId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
	}
}

// toDomainProcessInstanceFilter copies optional selection criteria across the facade boundary.
func toDomainProcessInstanceFilter(x ProcessInstanceFilter) d.ProcessInstanceFilter {
	return d.ProcessInstanceFilter{
		BusinessId:           toolx.CopyPtr(x.BusinessId),
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessVersion:       x.ProcessVersion,
		ProcessVersionTag:    x.ProcessVersionTag,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		StartDateAfter:       x.StartDateAfter,
		StartDateBefore:      x.StartDateBefore,
		EndDateAfter:         x.EndDateAfter,
		EndDateBefore:        x.EndDateBefore,
		State:                d.State(x.State),
		ParentKey:            x.ParentKey,
		HasParent:            toolx.CopyPtr(x.HasParent),
		HasIncident:          toolx.CopyPtr(x.HasIncident),
		VariableFilters:      toDomainProcessInstanceVariableFilterSet(x.VariableFilters),
	}
}

// fromDomainProcessInstanceFilter copies optional selection criteria across the facade boundary.
func fromDomainProcessInstanceFilter(x d.ProcessInstanceFilter) ProcessInstanceFilter {
	return ProcessInstanceFilter{
		BusinessId:           toolx.CopyPtr(x.BusinessId),
		Key:                  x.Key,
		BpmnProcessId:        x.BpmnProcessId,
		ProcessVersion:       x.ProcessVersion,
		ProcessVersionTag:    x.ProcessVersionTag,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		StartDateAfter:       x.StartDateAfter,
		StartDateBefore:      x.StartDateBefore,
		EndDateAfter:         x.EndDateAfter,
		EndDateBefore:        x.EndDateBefore,
		State:                State(x.State),
		ParentKey:            x.ParentKey,
		HasParent:            toolx.CopyPtr(x.HasParent),
		HasIncident:          toolx.CopyPtr(x.HasIncident),
		VariableFilters:      fromDomainProcessInstanceVariableFilterSet(x.VariableFilters),
	}
}

// toDomainProcessInstanceVariableFilterSet copies facade variable clauses into the service-facing domain filter.
func toDomainProcessInstanceVariableFilterSet(x ProcessInstanceVariableFilterSet) d.ProcessInstanceVariableFilterSet {
	return d.ProcessInstanceVariableFilterSet{
		Clauses: toolx.MapSlice(x.Clauses, toDomainProcessInstanceVariableFilterClause),
	}
}

// toDomainProcessInstanceVariableFilterClause maps one normalized facade clause without interpreting its value.
func toDomainProcessInstanceVariableFilterClause(x ProcessInstanceVariableFilterClause) d.ProcessInstanceVariableFilterClause {
	return d.ProcessInstanceVariableFilterClause{
		Name:     x.Name,
		Operator: d.ProcessInstanceVariableFilterOperator(x.Operator),
		Value:    x.Value,
		Exists:   toolx.CopyPtr(x.Exists),
		Source:   x.Source,
	}
}

// fromDomainProcessInstanceVariableFilterSet copies domain variable clauses back to the public model.
func fromDomainProcessInstanceVariableFilterSet(x d.ProcessInstanceVariableFilterSet) ProcessInstanceVariableFilterSet {
	return ProcessInstanceVariableFilterSet{
		Clauses: toolx.MapSlice(x.Clauses, fromDomainProcessInstanceVariableFilterClause),
	}
}

// fromDomainProcessInstanceVariableFilterClause maps one service-facing clause back to the facade type.
func fromDomainProcessInstanceVariableFilterClause(x d.ProcessInstanceVariableFilterClause) ProcessInstanceVariableFilterClause {
	return ProcessInstanceVariableFilterClause{
		Name:     x.Name,
		Operator: ProcessInstanceVariableFilterOperator(x.Operator),
		Value:    x.Value,
		Exists:   toolx.CopyPtr(x.Exists),
		Source:   x.Source,
	}
}

func toDomainProcessInstancePageRequest(x ProcessInstancePageRequest) d.ProcessInstancePageRequest {
	return d.ProcessInstancePageRequest{
		From:  x.From,
		Size:  x.Size,
		After: x.After,
	}
}

// toDomainProcessInstanceSearchRequest copies process search mechanics into the service-facing contract.
func toDomainProcessInstanceSearchRequest(x ProcessInstanceSearchRequest) d.ProcessInstanceSearchRequest {
	return d.ProcessInstanceSearchRequest{
		Filter:               toDomainProcessInstanceFilter(x.Filter),
		Page:                 toDomainProcessInstancePageRequest(x.Page),
		Limit:                x.Limit,
		LocalFilters:         toDomainProcessInstanceSearchLocalFilters(x.LocalFilters),
		DirectIncidentIndex:  x.DirectIncidentIndex,
		DirectIncidentFilter: toDomainProcessInstanceIncidentSearchFilter(x.DirectIncidentFilter),
		ReportedTotalAllowed: x.ReportedTotalAllowed,
	}
}

// toDomainProcessInstanceMutationPlanRequest maps facade search-selected
// mutation planning controls into the service contract.
func toDomainProcessInstanceMutationPlanRequest(x ProcessInstanceMutationPlanRequest) d.ProcessInstanceMutationPlanRequest {
	return d.ProcessInstanceMutationPlanRequest{
		SearchRequest: toDomainProcessInstanceSearchRequest(x.SearchRequest),
		Workers:       x.Workers,
	}
}

// toDomainProcessInstanceSearchLocalFilters maps compatibility-filter toggles without applying CLI policy.
func toDomainProcessInstanceSearchLocalFilters(x ProcessInstanceSearchLocalFilters) d.ProcessInstanceSearchLocalFilters {
	return d.ProcessInstanceSearchLocalFilters{
		ChildrenOnly:         x.ChildrenOnly,
		RootsOnly:            x.RootsOnly,
		OrphanChildrenOnly:   x.OrphanChildrenOnly,
		IncidentsOnly:        x.IncidentsOnly,
		DirectIncidentsOnly:  x.DirectIncidentsOnly,
		NoIncidentsOnly:      x.NoIncidentsOnly,
		IncidentState:        x.IncidentState,
		IncidentErrorType:    x.IncidentErrorType,
		IncidentErrorMessage: x.IncidentErrorMessage,
	}
}

// toDomainProcessInstanceIncidentSearchFilter maps the direct incident-index filter.
func toDomainProcessInstanceIncidentSearchFilter(x ProcessInstanceIncidentSearchFilter) d.IncidentFilter {
	return d.IncidentFilter{
		State:                x.State,
		ErrorType:            x.ErrorType,
		ErrorMessage:         x.ErrorMessage,
		ProcessDefinitionKey: x.ProcessDefinitionKey,
		ProcessDefinitionId:  x.ProcessDefinitionId,
	}
}

// toDomainProcessInstanceMutationPlanVisitor maps page-level planning visitor
// decisions across the facade boundary.
func toDomainProcessInstanceMutationPlanVisitor(visitor ProcessInstanceMutationPlanVisitor) d.ProcessInstanceMutationPlanVisitor {
	if visitor == nil {
		return nil
	}
	return func(step d.ProcessInstanceMutationPlanStep) (d.ProcessInstanceSearchPageAction, error) {
		action, err := visitor(fromDomainProcessInstanceMutationPlanStep(step))
		return d.ProcessInstanceSearchPageAction(action), err
	}
}

// toDomainProcessInstanceSearchPageVisitor maps page visitor decisions across the facade boundary.
func toDomainProcessInstanceSearchPageVisitor(visitor ProcessInstanceSearchPageVisitor) d.ProcessInstanceSearchPageVisitor {
	if visitor == nil {
		return nil
	}
	return func(step d.ProcessInstanceSearchPageStep) (d.ProcessInstanceSearchPageAction, error) {
		action, err := visitor(fromDomainProcessInstanceSearchPageStep(step))
		return d.ProcessInstanceSearchPageAction(action), err
	}
}

// toDomainProcessInstanceSearchTotalVisitor maps total fallback diagnostics across the facade boundary.
func toDomainProcessInstanceSearchTotalVisitor(visitor ProcessInstanceSearchTotalVisitor) d.ProcessInstanceSearchTotalVisitor {
	if visitor == nil {
		return nil
	}
	return func(step d.ProcessInstanceSearchTotalStep) error {
		return visitor(fromDomainProcessInstanceSearchTotalStep(step))
	}
}

// toProcessInstanceData copies optional business identity without inferring an absent value.
func toProcessInstanceData(x ProcessInstanceData) d.ProcessInstanceData {
	return d.ProcessInstanceData{
		BusinessId:                  toolx.CopyPtr(x.BusinessId),
		BpmnProcessId:               x.BpmnProcessId,
		ProcessDefinitionSpecificId: x.ProcessDefinitionSpecificId,
		ProcessDefinitionVersion:    x.ProcessDefinitionVersion,
		Variables:                   toolx.CopyMap(x.Variables),
		TenantId:                    x.TenantId,
	}
}

// fromDomainSuspensionResult mechanically preserves submission and observation evidence.
func fromDomainSuspensionResult(r d.ProcessInstanceSuspensionResult) SuspensionResult {
	return SuspensionResult{
		Key: r.Key, Status: SuspensionStatus(r.Status),
		Submission: SuspensionSubmission(r.Submission), Confirmation: SuspensionConfirmation(r.Confirmation),
		StatusCode: r.StatusCode, HasObservation: r.HasObservation,
		LastObservedState: State(r.LastObservedState), SuspendedDate: r.SuspendedDate, Error: r.Error,
	}
}

// fromDomainSuspensionResults copies items and preserves an empty JSON array.
func fromDomainSuspensionResults(r d.ProcessInstanceSuspensionResults) SuspensionResults {
	out := SuspensionResults{Selection: toolx.MapPtr(r.Selection, fromDomainProcessInstanceSelection), Items: make([]SuspensionResult, len(r.Items))}
	for i, item := range r.Items {
		out.Items[i] = fromDomainSuspensionResult(item)
	}
	return out
}

// fromDomainProcessInstanceSuspensionPlan copies all slices, retaining non-nil
// successful arrays and the zero plan on failure without interpreting observations.
func fromDomainProcessInstanceSuspensionPlan(plan d.ProcessInstanceSuspensionPlan) ProcessInstanceSuspensionPlan {
	return ProcessInstanceSuspensionPlan{
		Selection: toolx.MapPtr(plan.Selection, fromDomainProcessInstanceSelection), SelectionTenant: plan.SelectionTenant, Exact: plan.Exact,
		RequestedKeys: slices.Clone(plan.RequestedKeys),
		Instances:     toolx.MapSlice(plan.Instances, fromDomainProcessInstanceSuspensionPlanItem),
		TargetKeys:    slices.Clone(plan.TargetKeys),
	}
}

// fromDomainProcessInstanceSuspensionPlanItem maps planning facts, never execution evidence.
func fromDomainProcessInstanceSuspensionPlanItem(item d.ProcessInstanceSuspensionPlanItem) ProcessInstanceSuspensionPlanItem {
	return ProcessInstanceSuspensionPlanItem{BusinessId: toolx.CopyPtr(item.BusinessId), Key: item.Key, ObservedState: State(item.ObservedState), Classification: ProcessInstanceSuspensionPlanClassification(item.Classification)}
}

// fromDomainResumptionResult mechanically preserves submission and observation evidence.
func fromDomainResumptionResult(r d.ProcessInstanceResumptionResult) ResumptionResult {
	return ResumptionResult{
		Key: r.Key, Status: ResumptionStatus(r.Status),
		Submission: ResumptionSubmission(r.Submission), Confirmation: ResumptionConfirmation(r.Confirmation),
		StatusCode: r.StatusCode, HasObservation: r.HasObservation,
		LastObservedState: State(r.LastObservedState), SuspendedDate: r.SuspendedDate, Error: r.Error,
	}
}

// fromDomainResumptionResults copies items and preserves an empty JSON array.
func fromDomainResumptionResults(r d.ProcessInstanceResumptionResults) ResumptionResults {
	out := ResumptionResults{Selection: toolx.MapPtr(r.Selection, fromDomainProcessInstanceSelection), Items: make([]ResumptionResult, len(r.Items))}
	for i, item := range r.Items {
		out.Items[i] = fromDomainResumptionResult(item)
	}
	return out
}

// fromDomainProcessInstanceResumptionPlan copies all slices, retaining non-nil
// successful arrays and the zero plan on failure without interpreting observations.
func fromDomainProcessInstanceResumptionPlan(plan d.ProcessInstanceResumptionPlan) ProcessInstanceResumptionPlan {
	return ProcessInstanceResumptionPlan{
		Selection: toolx.MapPtr(plan.Selection, fromDomainProcessInstanceSelection), SelectionTenant: plan.SelectionTenant, Exact: plan.Exact,
		RequestedKeys: slices.Clone(plan.RequestedKeys),
		Instances:     toolx.MapSlice(plan.Instances, fromDomainProcessInstanceResumptionPlanItem),
		TargetKeys:    slices.Clone(plan.TargetKeys),
	}
}

// fromDomainProcessInstanceResumptionPlanItem maps planning facts, never execution evidence.
func fromDomainProcessInstanceResumptionPlanItem(item d.ProcessInstanceResumptionPlanItem) ProcessInstanceResumptionPlanItem {
	return ProcessInstanceResumptionPlanItem{BusinessId: toolx.CopyPtr(item.BusinessId), Key: item.Key, ObservedState: State(item.ObservedState), Classification: ProcessInstanceResumptionPlanClassification(item.Classification)}
}

// fromDomainProcessInstanceSelection copies provenance and observations without inferring readiness.
func fromDomainProcessInstanceSelection(x d.ProcessInstanceSelection) ProcessInstanceSelection {
	return ProcessInstanceSelection{
		Request:        ProcessInstanceSelectionRequest{SearchRequest: fromDomainProcessInstanceSearchRequest(x.Request.SearchRequest), Cardinality: ProcessInstanceSelectionCardinality(x.Request.Cardinality)},
		Origin:         x.Origin,
		Items:          toolx.MapSlice(x.Items, fromDomainProcessInstance),
		Keys:           append([]string(nil), x.Keys...),
		Completion:     ProcessInstanceSearchCompletion(x.Completion),
		Pages:          x.Pages,
		TenantEvidence: fromDomainTenantEvidence(x.TenantEvidence),
		IgnoreTenant:   x.IgnoreTenant,
	}
}

// fromDomainProcessInstanceSearchRequest retains every service search control in selection evidence.
func fromDomainProcessInstanceSearchRequest(x d.ProcessInstanceSearchRequest) ProcessInstanceSearchRequest {
	return ProcessInstanceSearchRequest{
		Filter:               fromDomainProcessInstanceFilter(x.Filter),
		Page:                 ProcessInstancePageRequest(x.Page),
		Limit:                x.Limit,
		LocalFilters:         ProcessInstanceSearchLocalFilters(x.LocalFilters),
		DirectIncidentIndex:  x.DirectIncidentIndex,
		ReportedTotalAllowed: x.ReportedTotalAllowed,
		DirectIncidentFilter: ProcessInstanceIncidentSearchFilter{State: x.DirectIncidentFilter.State, ErrorType: x.DirectIncidentFilter.ErrorType, ErrorMessage: x.DirectIncidentFilter.ErrorMessage, ProcessDefinitionKey: x.DirectIncidentFilter.ProcessDefinitionKey, ProcessDefinitionId: x.DirectIncidentFilter.ProcessDefinitionId},
	}
}

// toDomainSelectionRequest copies the original search criteria and cardinality.
func toDomainSelectionRequest(x ProcessInstanceSelectionRequest) d.ProcessInstanceSelectionRequest {
	return d.ProcessInstanceSelectionRequest{SearchRequest: toDomainProcessInstanceSearchRequest(x.SearchRequest), Cardinality: d.ProcessInstanceSelectionCardinality(x.Cardinality)}
}

// toDomainSelectedMutationRequest maps the reviewed operation and selection controls.
func toDomainSelectedMutationRequest(x ProcessInstanceSelectedMutationRequest) d.ProcessInstanceSelectedMutationRequest {
	return d.ProcessInstanceSelectedMutationRequest{Operation: x.Operation, Selection: toDomainSelectionRequest(x.Selection), Workers: x.Workers}
}

// toDomainSelectedMutationPlan isolates nested membership, observations and criteria at the facade boundary.
func toDomainSelectedMutationPlan(x ProcessInstanceSelectedMutationPlan) d.ProcessInstanceSelectedMutationPlan {
	return d.ProcessInstanceSelectedMutationPlan{
		Request:   toDomainSelectedMutationRequest(x.Request),
		Selection: toDomainProcessInstanceSelection(x.Selection),
		Expansion: toDomainDryRunPIKeyExpansion(x.Expansion),
		Tenant:    x.Tenant,
		Families: toolx.MapSlice(x.Families, func(f ProcessInstanceMutationFamily) d.ProcessInstanceMutationFamily {
			return d.ProcessInstanceMutationFamily{RootKey: f.RootKey, Keys: append([]string(nil), f.Keys...), Items: toolx.MapSlice(f.Items, toDomainProcessInstance)}
		})}
}

// fromDomainSelectionRequest copies the original search criteria and cardinality.
func fromDomainSelectionRequest(x d.ProcessInstanceSelectionRequest) ProcessInstanceSelectionRequest {
	return ProcessInstanceSelectionRequest{SearchRequest: fromDomainProcessInstanceSearchRequest(x.SearchRequest), Cardinality: ProcessInstanceSelectionCardinality(x.Cardinality)}
}

// fromDomainSelectedMutationRequest maps the reviewed operation and selection controls.
func fromDomainSelectedMutationRequest(x d.ProcessInstanceSelectedMutationRequest) ProcessInstanceSelectedMutationRequest {
	return ProcessInstanceSelectedMutationRequest{Operation: x.Operation, Selection: fromDomainSelectionRequest(x.Selection), Workers: x.Workers}
}

// fromDomainSelectedMutationPlan isolates nested membership, observations and criteria at the facade boundary.
func fromDomainSelectedMutationPlan(x d.ProcessInstanceSelectedMutationPlan) ProcessInstanceSelectedMutationPlan {
	return ProcessInstanceSelectedMutationPlan{
		Request:   fromDomainSelectedMutationRequest(x.Request),
		Selection: fromDomainProcessInstanceSelection(x.Selection),
		Expansion: fromDomainDryRunPIKeyExpansion(x.Expansion),
		Tenant:    x.Tenant,
		Families: toolx.MapSlice(x.Families, func(f d.ProcessInstanceMutationFamily) ProcessInstanceMutationFamily {
			return ProcessInstanceMutationFamily{RootKey: f.RootKey, Keys: append([]string(nil), f.Keys...), Items: toolx.MapSlice(f.Items, fromDomainProcessInstance)}
		})}
}

// toDomainProcessInstanceSelection preserves caller evidence without inferring completion.
func toDomainProcessInstanceSelection(x ProcessInstanceSelection) d.ProcessInstanceSelection {
	return d.ProcessInstanceSelection{
		Request:        toDomainSelectionRequest(x.Request),
		Origin:         x.Origin,
		Items:          toolx.MapSlice(x.Items, toDomainProcessInstance),
		Keys:           append([]string(nil), x.Keys...),
		Completion:     d.ProcessInstanceSearchCompletion(x.Completion),
		Pages:          x.Pages,
		TenantEvidence: toDomainSelectionTenantEvidence(x.TenantEvidence),
		IgnoreTenant:   x.IgnoreTenant,
	}
}

// toDomainSelectionTenantEvidence copies target-level tenant evidence.
func toDomainSelectionTenantEvidence(x TenantEvidence) d.TenantEvidence {
	return d.TenantEvidence{ResolvedTenantIDs: append([]string(nil), x.ResolvedTenantIDs...), UnknownTargetCount: x.UnknownTargetCount, TargetCount: x.TargetCount, Targets: toolx.MapSlice(x.Targets, func(t TenantEvidenceTarget) d.TenantEvidenceTarget {
		return d.TenantEvidenceTarget{Key: t.Key, TenantID: t.TenantID}
	})}
}

// toDomainDryRunPIKeyExpansion copies frozen expansion without performing fresh discovery.
func toDomainDryRunPIKeyExpansion(x DryRunPIKeyExpansion) d.DryRunPIKeyExpansion {
	return d.DryRunPIKeyExpansion{
		Roots:                      append([]string(nil), x.Roots...),
		Collected:                  append([]string(nil), x.Collected...),
		TenantEvidence:             toDomainSelectionTenantEvidence(x.TenantEvidence),
		SelectedFinalState:         toolx.MapSlice(x.SelectedFinalState, toDomainProcessInstance),
		RequiresCancelBeforeDelete: toolx.MapSlice(x.RequiresCancelBeforeDelete, toDomainProcessInstance),
		Warning:                    x.Warning,
		Outcome:                    d.TraversalOutcome(x.Outcome),
		MissingAncestors:           toolx.MapSlice(x.MissingAncestors, func(m MissingAncestor) d.MissingAncestor { return d.MissingAncestor{Key: m.Key, StartKey: m.StartKey} }),
	}
}

// toDomainProcessInstanceSuspensionPlan copies reviewed selection and descendant classifications.
func toDomainProcessInstanceSuspensionPlan(plan ProcessInstanceSuspensionPlan) d.ProcessInstanceSuspensionPlan {
	return d.ProcessInstanceSuspensionPlan{
		Selection: toolx.MapPtr(plan.Selection, toDomainProcessInstanceSelection), SelectionTenant: plan.SelectionTenant, Exact: plan.Exact,
		RequestedKeys: slices.Clone(plan.RequestedKeys), TargetKeys: slices.Clone(plan.TargetKeys),
		Instances: toolx.MapSlice(plan.Instances, func(item ProcessInstanceSuspensionPlanItem) d.ProcessInstanceSuspensionPlanItem {
			return d.ProcessInstanceSuspensionPlanItem{BusinessId: toolx.CopyPtr(item.BusinessId), Key: item.Key, ObservedState: d.State(item.ObservedState), Classification: d.ProcessInstanceSuspensionPlanClassification(item.Classification)}
		}),
	}
}

// toDomainProcessInstanceResumptionPlan copies reviewed selection and descendant classifications.
func toDomainProcessInstanceResumptionPlan(plan ProcessInstanceResumptionPlan) d.ProcessInstanceResumptionPlan {
	return d.ProcessInstanceResumptionPlan{
		Selection: toolx.MapPtr(plan.Selection, toDomainProcessInstanceSelection), SelectionTenant: plan.SelectionTenant, Exact: plan.Exact,
		RequestedKeys: slices.Clone(plan.RequestedKeys), TargetKeys: slices.Clone(plan.TargetKeys),
		Instances: toolx.MapSlice(plan.Instances, func(item ProcessInstanceResumptionPlanItem) d.ProcessInstanceResumptionPlanItem {
			return d.ProcessInstanceResumptionPlanItem{BusinessId: toolx.CopyPtr(item.BusinessId), Key: item.Key, ObservedState: d.State(item.ObservedState), Classification: d.ProcessInstanceResumptionPlanClassification(item.Classification)}
		}),
	}
}

// fromDomainBusinessIDAssignmentPlan isolates preview storage from service and caller mutation.
func fromDomainBusinessIDAssignmentPlan(in d.BusinessIDAssignmentPlan) BusinessIDAssignmentPlan {
	out := BusinessIDAssignmentPlan{RequestedBusinessID: toolx.CopyPtr(in.RequestedBusinessID), RequestedKeys: append([]string{}, in.RequestedKeys...), Ready: in.Ready, Items: make([]BusinessIDAssignmentItem, len(in.Items))}
	for i, item := range in.Items {
		out.Items[i] = BusinessIDAssignmentItem{Key: item.Key, PreviouslyObservedBusinessID: toolx.CopyPtr(item.PreviouslyObservedBusinessID), State: State(item.State)}
	}
	return out
}

// toDomainBusinessIDAssignmentPlan snapshots all public mutable storage before delegation.
func toDomainBusinessIDAssignmentPlan(in BusinessIDAssignmentPlan) d.BusinessIDAssignmentPlan {
	out := d.BusinessIDAssignmentPlan{RequestedBusinessID: toolx.CopyPtr(in.RequestedBusinessID), RequestedKeys: append([]string{}, in.RequestedKeys...), Ready: in.Ready, Items: make([]d.BusinessIDAssignmentItem, len(in.Items))}
	for i, item := range in.Items {
		out.Items[i] = d.BusinessIDAssignmentItem{Key: item.Key, PreviouslyObservedBusinessID: toolx.CopyPtr(item.PreviouslyObservedBusinessID), State: d.State(item.State)}
	}
	return out
}

// fromDomainBusinessIDAssignmentResults preserves ordered evidence and copies optional facts.
func fromDomainBusinessIDAssignmentResults(in d.BusinessIDAssignmentResults) BusinessIDAssignmentResults {
	out := BusinessIDAssignmentResults{Items: make([]BusinessIDAssignmentResult, len(in.Items))}
	for i, item := range in.Items {
		out.Items[i] = BusinessIDAssignmentResult{Key: item.Key, RequestedBusinessID: toolx.CopyPtr(item.RequestedBusinessID), PreviouslyObservedBusinessID: toolx.CopyPtr(item.PreviouslyObservedBusinessID), Status: item.Status, Confirmation: item.Confirmation, ObservedBusinessID: toolx.CopyPtr(item.ObservedBusinessID), HasObservation: item.HasObservation, HTTPStatus: toolx.CopyPtr(item.HTTPStatus), Error: item.Error}
	}
	return out
}
