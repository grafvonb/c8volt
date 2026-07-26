// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/typex"
)

// SlowProcessAnalysisSelectionMode identifies how process instances are selected for analysis.
type SlowProcessAnalysisSelectionMode string

const (
	// SlowProcessAnalysisSelectionModeExplicitKeys records flag/stdin process-instance keys.
	SlowProcessAnalysisSelectionModeExplicitKeys SlowProcessAnalysisSelectionMode = "explicit_keys"
	// SlowProcessAnalysisSelectionModeProcessDefinitionSearch records process-definition scoped discovery.
	SlowProcessAnalysisSelectionModeProcessDefinitionSearch SlowProcessAnalysisSelectionMode = "process_definition_search"
)

// SlowProcessAnalysisTimelineEntryKind identifies a calculated timeline row.
type SlowProcessAnalysisTimelineEntryKind string

const (
	// SlowProcessAnalysisTimelineEntryKindElement represents a runtime element timing row.
	SlowProcessAnalysisTimelineEntryKindElement SlowProcessAnalysisTimelineEntryKind = "element"
	// SlowProcessAnalysisTimelineEntryKindTransition represents timing between adjacent runtime elements.
	SlowProcessAnalysisTimelineEntryKindTransition SlowProcessAnalysisTimelineEntryKind = "transition"
)

// SlowProcessAnalysisRequest captures the normalized input for one slow analysis run.
type SlowProcessAnalysisRequest struct {
	CommandName               string                                          `json:"commandName,omitempty"`
	SelectionMode             SlowProcessAnalysisSelectionMode                `json:"selectionMode,omitempty"`
	InputKeys                 typex.Keys                                      `json:"inputKeys,omitempty"`
	ProcessDefinitionSelector SlowProcessAnalysisProcessDefinitionSelector    `json:"processDefinitionSelector,omitempty"`
	ProcessInstanceFilters    SlowProcessAnalysisProcessInstanceSearchFilters `json:"processInstanceFilters,omitempty"`
	DetailFilters             SlowProcessAnalysisDetailFilters                `json:"detailFilters,omitempty"`
	RootDurationLonger        time.Duration                                   `json:"rootDurationLonger,omitempty"`
	BatchSize                 int32                                           `json:"batchSize,omitempty"`
	Limit                     int32                                           `json:"limit,omitempty"`
	CapturedNow               time.Time                                       `json:"capturedNow,omitempty"`
	OutputMode                string                                          `json:"outputMode,omitempty"`
	WithListeners             bool                                            `json:"withListeners,omitempty"`
}

// SlowProcessAnalysisProcessDefinitionSelector identifies the process-definition cohort for search mode.
type SlowProcessAnalysisProcessDefinitionSelector struct {
	BpmnProcessID        string `json:"bpmnProcessId,omitempty"`
	ProcessDefinitionKey string `json:"processDefinitionKey,omitempty"`
}

// SlowProcessAnalysisProcessInstanceSearchFilters carries discovery-only process-instance filters.
type SlowProcessAnalysisProcessInstanceSearchFilters struct {
	State           State  `json:"state,omitempty"`
	StartDateAfter  string `json:"startDateAfter,omitempty"`
	StartDateBefore string `json:"startDateBefore,omitempty"`
	EndDateAfter    string `json:"endDateAfter,omitempty"`
	EndDateBefore   string `json:"endDateBefore,omitempty"`
	NoIncidentsOnly bool   `json:"noIncidentsOnly,omitempty"`
}

// SlowProcessAnalysisDetailFilters carries post-calculation timeline visibility filters.
type SlowProcessAnalysisDetailFilters struct {
	ElementID     string        `json:"elementId,omitempty"`
	Type          string        `json:"type,omitempty"`
	ElementState  string        `json:"elementState,omitempty"`
	DurationAfter time.Duration `json:"durationAfter,omitempty"`
}

// SlowProcessAnalysisProcessInstance is the root result row for one selected process instance.
type SlowProcessAnalysisProcessInstance struct {
	Key                    string                             `json:"key,omitempty"`
	TenantID               string                             `json:"tenantId,omitempty"`
	BpmnProcessID          string                             `json:"bpmnProcessId,omitempty"`
	ProcessDefinitionKey   string                             `json:"processDefinitionKey,omitempty"`
	ProcessVersion         int32                              `json:"processVersion,omitempty"`
	State                  State                              `json:"state,omitempty"`
	StartDate              string                             `json:"startDate,omitempty"`
	EndDate                string                             `json:"endDate,omitempty"`
	ParentKey              string                             `json:"parentKey,omitempty"`
	RootProcessInstanceKey string                             `json:"rootProcessInstanceKey,omitempty"`
	Incident               bool                               `json:"incident,omitempty"`
	Duration               string                             `json:"duration,omitempty"`
	DurationMillis         int64                              `json:"durationMillis,omitempty"`
	DurationAvailable      bool                               `json:"durationAvailable"`
	RelativePercentile     int                                `json:"relativePercentile,omitempty"`
	ComparisonSampleCount  int                                `json:"comparisonSampleCount,omitempty"`
	RelativeBar            string                             `json:"relativeBar,omitempty"`
	Timeline               []SlowProcessAnalysisTimelineEntry `json:"timeline,omitempty"`
}

// SlowProcessAnalysisTimelineEntry is a unified calculated element or transition row.
type SlowProcessAnalysisTimelineEntry struct {
	Kind                   SlowProcessAnalysisTimelineEntryKind `json:"kind,omitempty"`
	ElementInstanceKey     string                               `json:"elementInstanceKey,omitempty"`
	ElementID              string                               `json:"elementId,omitempty"`
	Type                   string                               `json:"type,omitempty"`
	State                  string                               `json:"state,omitempty"`
	StartDate              string                               `json:"startDate,omitempty"`
	EndDate                string                               `json:"endDate,omitempty"`
	HasIncident            bool                                 `json:"hasIncident,omitempty"`
	IncidentKey            string                               `json:"incidentKey,omitempty"`
	FromElementInstanceKey string                               `json:"fromElementInstanceKey,omitempty"`
	FromElementID          string                               `json:"fromElementId,omitempty"`
	FromElementType        string                               `json:"fromElementType,omitempty"`
	FromEndDate            string                               `json:"fromEndDate,omitempty"`
	ToElementInstanceKey   string                               `json:"toElementInstanceKey,omitempty"`
	ToElementID            string                               `json:"toElementId,omitempty"`
	ToElementType          string                               `json:"toElementType,omitempty"`
	ToStartDate            string                               `json:"toStartDate,omitempty"`
	Duration               string                               `json:"duration,omitempty"`
	DurationMillis         int64                                `json:"durationMillis,omitempty"`
	DurationAvailable      bool                                 `json:"durationAvailable"`
	RelativePercentile     int                                  `json:"relativePercentile,omitempty"`
	ComparisonSampleCount  int                                  `json:"comparisonSampleCount,omitempty"`
	RelativeBar            string                               `json:"relativeBar,omitempty"`
	ProcessDurationShare   int                                  `json:"processDurationShare,omitempty"`
	Listeners              *[]RuntimeListenerJob                `json:"listeners,omitempty"`
}

// SlowProcessAnalysisResult carries the complete render-independent analysis payload.
type SlowProcessAnalysisResult struct {
	Request               SlowProcessAnalysisRequest           `json:"request,omitempty"`
	DiscoveredScopeStatus DiscoveryScopeStatus                 `json:"discoveredScopeStatus,omitempty"`
	CapturedAt            time.Time                            `json:"capturedAt,omitempty"`
	Items                 []SlowProcessAnalysisProcessInstance `json:"items"`
	Count                 int                                  `json:"count"`
	Empty                 bool                                 `json:"empty"`
	Warnings              []string                             `json:"warnings,omitempty"`
}
