// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"

	"github.com/grafvonb/c8volt/toolx"
)

type ProcessDefinition struct {
	BpmnProcessId     string                       `json:"bpmnProcessId,omitempty"`
	Key               string                       `json:"key,omitempty"`
	Name              string                       `json:"name,omitempty"`
	TenantId          string                       `json:"tenantId,omitempty"`
	ProcessVersion    int32                        `json:"processVersion,omitempty"`
	ProcessVersionTag string                       `json:"versionTag,omitempty"`
	Statistics        *ProcessDefinitionStatistics `json:"statistics,omitempty"`
}

type ProcessDefinitionStatistics struct {
	Active                 int64 `json:"active,omitempty"`
	Canceled               int64 `json:"canceled,omitempty"`
	Completed              int64 `json:"completed,omitempty"`
	Incidents              int64 `json:"incidents,omitempty"`
	IncidentCountSupported bool  `json:"incidentCountSupported,omitempty"`
}

type ProcessDefinitionFilter struct {
	BpmnProcessId     string `json:"bpmnProcessId,omitempty"`
	Key               string `json:"key,omitempty"`
	TenantId          string `json:"tenantId,omitempty"`
	ProcessVersion    int32  `json:"processVersion,omitempty"`
	ProcessVersionTag string `json:"processVersionTag,omitempty"`
	IsLatestVersion   bool   `json:"isLatestVersion,omitempty"`
}

func (f ProcessDefinitionFilter) String() string {
	parts := make([]string, 0, 6)
	parts = toolx.AppendQuotedField(parts, "bpmnProcessId", f.BpmnProcessId)
	parts = toolx.AppendQuotedField(parts, "key", f.Key)
	parts = toolx.AppendQuotedField(parts, "tenantId", f.TenantId)
	parts = toolx.AppendInt32Field(parts, "processVersion", f.ProcessVersion)
	parts = toolx.AppendQuotedField(parts, "processVersionTag", f.ProcessVersionTag)
	parts = toolx.AppendTrueBoolField(parts, "isLatestVersion", f.IsLatestVersion)
	return toolx.FormatActiveFields(parts)
}

type ProcessDefinitionStatisticsFilter struct {
	TenantId string `json:"tenantId,omitempty"`
}

func (f ProcessDefinitionStatisticsFilter) String() string {
	parts := make([]string, 0, 1)
	parts = toolx.AppendQuotedField(parts, "tenantId", f.TenantId)
	return toolx.FormatActiveFields(parts)
}

// ProcessDefinitionPageRequest captures one backend process-definition search page request.
type ProcessDefinitionPageRequest struct {
	From  int32
	Size  int32
	After string
}

// ProcessDefinitionReportedTotalKind records whether the backend total is exact or capped.
type ProcessDefinitionReportedTotalKind string

const (
	// ProcessDefinitionReportedTotalKindExact means the backend total is the full matching count.
	ProcessDefinitionReportedTotalKindExact ProcessDefinitionReportedTotalKind = "exact"
	// ProcessDefinitionReportedTotalKindLowerBound means the backend total is capped below the true count.
	ProcessDefinitionReportedTotalKindLowerBound ProcessDefinitionReportedTotalKind = "lower_bound"
)

// ProcessDefinitionReportedTotal carries the backend-reported process-definition total for a page.
type ProcessDefinitionReportedTotal struct {
	Count int64
	Kind  ProcessDefinitionReportedTotalKind
}

// ProcessDefinitionPage is one process-definition search page plus continuation metadata.
type ProcessDefinitionPage struct {
	Items         []ProcessDefinition
	Request       ProcessDefinitionPageRequest
	OverflowState ProcessInstanceOverflowState
	ReportedTotal *ProcessDefinitionReportedTotal
	EndCursor     string
}

// ProcessDefinitionSearchPageAction tells service-owned page traversal whether
// the caller needs more process-definition pages after observing the current page.
type ProcessDefinitionSearchPageAction string

const (
	// ProcessDefinitionSearchPageActionContinue keeps collecting the next available page.
	ProcessDefinitionSearchPageActionContinue ProcessDefinitionSearchPageAction = "continue"
	// ProcessDefinitionSearchPageActionStop stops traversal after the current page.
	ProcessDefinitionSearchPageActionStop ProcessDefinitionSearchPageAction = "stop"
)

// ProcessDefinitionSearchRequest contains process-definition search mechanics
// while callers retain CLI rendering, prompts, and mode selection.
type ProcessDefinitionSearchRequest struct {
	Filter ProcessDefinitionFilter
	Page   ProcessDefinitionPageRequest
	Limit  int32
}

// ProcessDefinitionSearchPageStep carries one selected process-definition page
// plus traversal state while keeping page advancement and limit trimming below
// command ownership.
type ProcessDefinitionSearchPageStep struct {
	Page            ProcessDefinitionPage
	CumulativeCount int32
	LimitReached    bool
}

// ProcessDefinitionSearchPageVisitor observes selected pages during service-owned traversal.
type ProcessDefinitionSearchPageVisitor func(ProcessDefinitionSearchPageStep) (ProcessDefinitionSearchPageAction, error)

// ProcessDefinitionSearchPagesResult captures a full or caller-stopped process-definition discovery.
type ProcessDefinitionSearchPagesResult struct {
	Items []ProcessDefinition
	Limit int32
	Pages int32
}

func SortByVersionDesc(pds []ProcessDefinition) {
	slices.SortFunc(pds, func(a, b ProcessDefinition) int {
		switch {
		case a.ProcessVersion > b.ProcessVersion:
			return -1 // a before b
		case a.ProcessVersion < b.ProcessVersion:
			return 1 // b before a
		default:
			return 0
		}
	})
}

func SortByBpmnProcessIdAscThenByVersionDesc(pds []ProcessDefinition) {
	slices.SortFunc(pds, func(a, b ProcessDefinition) int {
		if a.BpmnProcessId < b.BpmnProcessId {
			return -1
		}
		if a.BpmnProcessId > b.BpmnProcessId {
			return 1
		}
		switch {
		case a.ProcessVersion > b.ProcessVersion:
			return -1
		case a.ProcessVersion < b.ProcessVersion:
			return 1
		default:
			return 0
		}
	})
}
