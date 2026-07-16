// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import d "github.com/grafvonb/c8volt/internal/domain"

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
	}
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
