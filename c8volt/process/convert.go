// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDomainProcessDefinition(x d.ProcessDefinition) ProcessDefinition {
	return ProcessDefinition{
		BpmnProcessId:     x.BpmnProcessId,
		Key:               x.Key,
		Name:              x.Name,
		TenantId:          x.TenantId,
		ProcessVersion:    x.ProcessVersion,
		ProcessVersionTag: x.ProcessVersionTag,
		Statistics:        toolx.MapPtr(x.Statistics, fromProcessDefinitionStatistics),
	}
}

func fromProcessDefinitionStatistics(r d.ProcessDefinitionStatistics) ProcessDefinitionStatistics {
	return ProcessDefinitionStatistics{
		Active:                 r.Active,
		Canceled:               r.Canceled,
		Completed:              r.Completed,
		Incidents:              r.Incidents,
		IncidentCountSupported: r.IncidentCountSupported,
	}
}

func fromDomainProcessDefinitions(xs []d.ProcessDefinition) ProcessDefinitions {
	items := toolx.MapSlice(xs, fromDomainProcessDefinition)
	return ProcessDefinitions{
		Total: int32(len(items)),
		Items: items,
	}
}

func fromDomainProcessInstance(x d.ProcessInstance) ProcessInstance {
	return ProcessInstance{
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
		State:                    State(x.State),
		TenantId:                 x.TenantId,
		Variables:                toolx.CopyMap(x.Variables),
	}
}

func fromDomainProcessInstanceCreation(x d.ProcessInstanceCreation) ProcessInstance {
	return ProcessInstance{
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

func fromDomainRuntimeListenerJob(x d.RuntimeListenerJob) RuntimeListenerJob {
	return RuntimeListenerJob{
		JobKey:             x.JobKey,
		Kind:               x.Kind,
		ListenerEventType:  x.ListenerEventType,
		Type:               x.Type,
		State:              x.State,
		Retries:            x.Retries,
		Worker:             x.Worker,
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
		Variables:          toolx.CopyMap(x.Variables),
	}
}

func fromDomainProcessInstanceVariableUpdateResults(x d.ProcessInstanceVariableUpdateResults) ProcessInstanceVariableUpdateResults {
	return ProcessInstanceVariableUpdateResults{
		Items: toolx.MapSlice(x.Items, fromDomainProcessInstanceVariableUpdateResult),
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

func fromDomainDryRunPIKeyExpansion(x d.DryRunPIKeyExpansion) DryRunPIKeyExpansion {
	return DryRunPIKeyExpansion{
		Roots:                      append([]string(nil), x.Roots...),
		Collected:                  append([]string(nil), x.Collected...),
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
		Items: toolx.MapSlice(x.Items, fromDomainProcessInstance),
		Limit: x.Limit,
		Pages: x.Pages,
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

func toDomainProcessInstance(x ProcessInstance) d.ProcessInstance {
	return d.ProcessInstance{
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
		State:                    d.State(x.State),
		TenantId:                 x.TenantId,
		Variables:                toolx.CopyMap(x.Variables),
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

func toDomainProcessInstanceFilter(x ProcessInstanceFilter) d.ProcessInstanceFilter {
	return d.ProcessInstanceFilter{
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
		HasParent:            x.HasParent,
		HasIncident:          x.HasIncident,
		VariableFilters:      toDomainProcessInstanceVariableFilterSet(x.VariableFilters),
	}
}

func fromDomainProcessInstanceFilter(x d.ProcessInstanceFilter) ProcessInstanceFilter {
	return ProcessInstanceFilter{
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
		HasParent:            x.HasParent,
		HasIncident:          x.HasIncident,
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

func toProcessInstanceData(x ProcessInstanceData) d.ProcessInstanceData {
	return d.ProcessInstanceData{
		BpmnProcessId:               x.BpmnProcessId,
		ProcessDefinitionSpecificId: x.ProcessDefinitionSpecificId,
		ProcessDefinitionVersion:    x.ProcessDefinitionVersion,
		Variables:                   toolx.CopyMap(x.Variables),
		TenantId:                    x.TenantId,
	}
}
