// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// Element represents one runtime BPMN element execution instance.
type Element struct {
	ElementInstanceKey     string
	ElementId              string
	ElementName            string
	Type                   string
	State                  string
	StartDate              string
	EndDate                string
	ProcessInstanceKey     string
	RootProcessInstanceKey string
	ProcessDefinitionId    string
	ProcessDefinitionKey   string
	TenantId               string
	HasIncident            bool
	IncidentKey            string
}

// ElementSearchQuery carries the version-neutral filters and bounds for runtime element search.
type ElementSearchQuery struct {
	Key                  string
	ProcessInstanceKey   string
	ElementId            string
	State                string
	Type                 string
	ProcessDefinitionKey string
	BpmnProcessId        string
	BatchSize            int32
	Limit                int32
}

// HasKey reports whether the query requests direct lookup by element instance key.
func (q ElementSearchQuery) HasKey() bool {
	return q.Key != ""
}

// HasSearchFilters reports whether the query carries any non-key search selector.
func (q ElementSearchQuery) HasSearchFilters() bool {
	return q.ProcessInstanceKey != "" ||
		q.ElementId != "" ||
		q.State != "" ||
		q.Type != "" ||
		q.ProcessDefinitionKey != "" ||
		q.BpmnProcessId != ""
}

// HasSearchControls reports whether the query carries paging or bounding controls.
func (q ElementSearchQuery) HasSearchControls() bool {
	return q.BatchSize > 0 || q.Limit > 0
}

// ElementSearchResult contains a bounded collected element search result.
type ElementSearchResult struct {
	Items []Element
	Total int32
}

// ElementReportedTotalKind describes the backend total count semantics.
type ElementReportedTotalKind string

const (
	// ElementReportedTotalKindExact indicates the backend reported a complete count.
	ElementReportedTotalKindExact ElementReportedTotalKind = "exact"
	// ElementReportedTotalKindLowerBound indicates the backend reported a lower-bound count.
	ElementReportedTotalKindLowerBound ElementReportedTotalKind = "lower_bound"
)

// ElementReportedTotal carries a backend-reported element count and its semantics.
type ElementReportedTotal struct {
	Count int64
	Kind  ElementReportedTotalKind
}

// ElementPageRequest describes one version-neutral element search page request.
type ElementPageRequest struct {
	From int32
	Size int32
}

// ElementSearchPage contains one page of element search results plus continuation metadata.
type ElementSearchPage struct {
	Items         []Element
	Request       ElementPageRequest
	OverflowState ProcessInstanceOverflowState
	ReportedTotal *ElementReportedTotal
}
