// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

// Element represents one runtime BPMN element execution instance.
type Element struct {
	ElementInstanceKey     string `json:"elementInstanceKey,omitempty"`
	ElementId              string `json:"elementId,omitempty"`
	ElementName            string `json:"elementName,omitempty"`
	Type                   string `json:"type,omitempty"`
	State                  string `json:"state,omitempty"`
	StartDate              string `json:"startDate,omitempty"`
	EndDate                string `json:"endDate,omitempty"`
	ProcessInstanceKey     string `json:"processInstanceKey,omitempty"`
	RootProcessInstanceKey string `json:"rootProcessInstanceKey,omitempty"`
	ProcessDefinitionId    string `json:"processDefinitionId,omitempty"`
	ProcessDefinitionKey   string `json:"processDefinitionKey,omitempty"`
	TenantId               string `json:"tenantId,omitempty"`
	HasIncident            bool   `json:"hasIncident"`
	IncidentKey            string `json:"incidentKey,omitempty"`
}

// SearchRequest carries the public filters and bounds for runtime element search.
type SearchRequest struct {
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

// HasKey reports whether the request asks for direct lookup by element instance key.
func (r SearchRequest) HasKey() bool {
	return r.Key != ""
}

// HasSearchFilters reports whether the request carries any non-key search selector.
func (r SearchRequest) HasSearchFilters() bool {
	return r.ProcessInstanceKey != "" ||
		r.ElementId != "" ||
		r.State != "" ||
		r.Type != "" ||
		r.ProcessDefinitionKey != "" ||
		r.BpmnProcessId != ""
}

// HasSearchControls reports whether the request carries paging or bounding controls.
func (r SearchRequest) HasSearchControls() bool {
	return r.BatchSize > 0 || r.Limit > 0
}

// SearchResult contains a bounded collected element search result.
type SearchResult struct {
	Total int32     `json:"total"`
	Items []Element `json:"items"`
}

// PageRequest describes one element search page request.
type PageRequest struct {
	From int32 `json:"from,omitempty"`
	Size int32 `json:"size,omitempty"`
}

// ReportedTotalKind describes the backend total count semantics.
type ReportedTotalKind string

const (
	// ReportedTotalKindExact indicates the backend reported a complete count.
	ReportedTotalKindExact ReportedTotalKind = "exact"
	// ReportedTotalKindLowerBound indicates the backend reported a lower-bound count.
	ReportedTotalKindLowerBound ReportedTotalKind = "lower_bound"
)

// ReportedTotal carries a backend-reported element count and its semantics.
type ReportedTotal struct {
	Count int64             `json:"count,omitempty"`
	Kind  ReportedTotalKind `json:"kind,omitempty"`
}

// OverflowState describes whether more element search pages are available.
type OverflowState string

const (
	// OverflowStateNoMore means the current page exhausted available results.
	OverflowStateNoMore OverflowState = "no_more"
	// OverflowStateHasMore means another page may contain more results.
	OverflowStateHasMore OverflowState = "has_more"
)

// Page contains one page of element search results plus continuation metadata.
type Page struct {
	Items         []Element      `json:"items"`
	Request       PageRequest    `json:"request,omitempty"`
	OverflowState OverflowState  `json:"overflowState,omitempty"`
	ReportedTotal *ReportedTotal `json:"reportedTotal,omitempty"`
}
