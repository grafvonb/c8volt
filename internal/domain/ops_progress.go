// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "time"

// OpsTotalCertainty classifies whether a progress count is exact, approximate, or unavailable.
type OpsTotalCertainty string

const (
	// OpsTotalCertaintyExact means the count describes the complete selected scope.
	OpsTotalCertaintyExact OpsTotalCertainty = "exact"
	// OpsTotalCertaintyLowerBound means at least the displayed count exists, but more may exist.
	OpsTotalCertaintyLowerBound OpsTotalCertainty = "lower_bound"
	// OpsTotalCertaintyEstimated means the count is a c8volt-derived approximation.
	OpsTotalCertaintyEstimated OpsTotalCertainty = "estimated"
	// OpsTotalCertaintyUnknown means no trustworthy count is available.
	OpsTotalCertaintyUnknown OpsTotalCertainty = "unknown"
)

// OpsPageCountKind classifies whether a page count is exact, estimated, or unavailable.
type OpsPageCountKind string

const (
	// OpsPageCountKindExact means the total page count is known.
	OpsPageCountKindExact OpsPageCountKind = "exact"
	// OpsPageCountKindEstimated means the total page count is approximate.
	OpsPageCountKindEstimated OpsPageCountKind = "estimated"
	// OpsPageCountKindUnknown means the total page count is unavailable.
	OpsPageCountKindUnknown OpsPageCountKind = "unknown"
)

// OpsOverflowState normalizes backend continuation state for progress renderers.
type OpsOverflowState string

const (
	// OpsOverflowStateNoMore means the backend indicated traversal is complete.
	OpsOverflowStateNoMore OpsOverflowState = "no_more"
	// OpsOverflowStateHasMore means the backend indicated another page is available.
	OpsOverflowStateHasMore OpsOverflowState = "has_more"
	// OpsOverflowStateIndeterminate means the backend could not prove whether more pages exist.
	OpsOverflowStateIndeterminate OpsOverflowState = "indeterminate"
	// OpsOverflowStateUnknown means no continuation metadata was available.
	OpsOverflowStateUnknown OpsOverflowState = "unknown"
)

// OpsProgressEventKind identifies the kind of structured progress fact emitted by a service workflow.
type OpsProgressEventKind string

const (
	// OpsProgressEventKindPreflight carries the best available scope summary before expensive work.
	OpsProgressEventKindPreflight OpsProgressEventKind = "preflight"
	// OpsProgressEventKindPage carries paged discovery progress.
	OpsProgressEventKindPage OpsProgressEventKind = "page"
	// OpsProgressEventKindFrozenScope carries exact counters for a frozen work set.
	OpsProgressEventKindFrozenScope OpsProgressEventKind = "frozen_scope"
	// OpsProgressEventKindETA carries timing samples used for approximate ETA rendering.
	OpsProgressEventKindETA OpsProgressEventKind = "eta"
)

// OpsProgressMode identifies an output context for progress-channel gating.
type OpsProgressMode string

const (
	// OpsProgressModeHuman is the default interactive command output mode.
	OpsProgressModeHuman OpsProgressMode = "human"
	// OpsProgressModeVerbose is human output with durable progress detail enabled.
	OpsProgressModeVerbose OpsProgressMode = "verbose"
	// OpsProgressModeDebug is diagnostic output where low-level traces may already exist.
	OpsProgressModeDebug OpsProgressMode = "debug"
	// OpsProgressModeJSON is the JSON result contract.
	OpsProgressModeJSON OpsProgressMode = "json"
	// OpsProgressModeKeysOnly is the one-key-per-line result contract.
	OpsProgressModeKeysOnly OpsProgressMode = "keys-only"
	// OpsProgressModeQuiet suppresses non-error progress chatter.
	OpsProgressModeQuiet OpsProgressMode = "quiet"
	// OpsProgressModeAutomation is deterministic unattended execution.
	OpsProgressModeAutomation OpsProgressMode = "automation"
)

// OpsProgressChannel records where progress may be emitted for a command mode.
type OpsProgressChannel struct {
	Mode                    OpsProgressMode `json:"mode,omitempty"`
	TransientAllowed        bool            `json:"transientAllowed,omitempty"`
	DurableAllowed          bool            `json:"durableAllowed,omitempty"`
	StdoutAllowed           bool            `json:"stdoutAllowed,omitempty"`
	StderrAllowed           bool            `json:"stderrAllowed,omitempty"`
	StructuredReportAllowed bool            `json:"structuredReportAllowed,omitempty"`
}

// OpsConsequenceSummary describes follow-on work in operator-facing structured parts.
type OpsConsequenceSummary struct {
	ResourceSummary  string `json:"resourceSummary,omitempty"`
	WorkSummary      string `json:"workSummary,omitempty"`
	RiskSummary      string `json:"riskSummary,omitempty"`
	ConfirmationText string `json:"confirmationText,omitempty"`
}

// OpsPreflightScope summarizes apparent command scope before deeper work starts.
type OpsPreflightScope struct {
	Phase                string                `json:"phase,omitempty"`
	Command              string                `json:"command,omitempty"`
	CoreResource         string                `json:"coreResource,omitempty"`
	SelectorSummary      string                `json:"selectorSummary,omitempty"`
	Total                *int64                `json:"total,omitempty"`
	TotalKind            OpsTotalCertainty     `json:"totalKind,omitempty"`
	PageSize             int32                 `json:"pageSize,omitempty"`
	PageCount            *int64                `json:"pageCount,omitempty"`
	PageCountKind        OpsPageCountKind      `json:"pageCountKind,omitempty"`
	ConsequenceSummary   OpsConsequenceSummary `json:"consequenceSummary,omitempty"`
	RequiresConfirmation bool                  `json:"requiresConfirmation,omitempty"`
	ExpensivePreflight   bool                  `json:"expensivePreflight,omitempty"`
}

// OpsPageProgress reports one discovery page without owning rendering language.
type OpsPageProgress struct {
	Phase            string           `json:"phase,omitempty"`
	CurrentPage      int              `json:"currentPage,omitempty"`
	PageCount        *int64           `json:"pageCount,omitempty"`
	PageCountKind    OpsPageCountKind `json:"pageCountKind,omitempty"`
	PageSize         int32            `json:"pageSize,omitempty"`
	CurrentPageCount int              `json:"currentPageCount,omitempty"`
	Seen             int              `json:"seen,omitempty"`
	Selected         int              `json:"selected,omitempty"`
	OverflowState    OpsOverflowState `json:"overflowState,omitempty"`
	LimitReached     bool             `json:"limitReached,omitempty"`
}

// OpsFrozenScopeProgress reports exact progress across an immutable work set.
type OpsFrozenScopeProgress struct {
	Phase        string         `json:"phase,omitempty"`
	CoreResource string         `json:"coreResource,omitempty"`
	Done         int            `json:"done,omitempty"`
	Total        int            `json:"total,omitempty"`
	Elapsed      time.Duration  `json:"elapsed,omitempty"`
	Rate         *float64       `json:"rate,omitempty"`
	ETA          *time.Duration `json:"eta,omitempty"`
	Errors       int            `json:"errors,omitempty"`
}

// OpsETASampleWindow carries enough timing data for command formatters to decide whether ETA is useful.
type OpsETASampleWindow struct {
	Phase             string         `json:"phase,omitempty"`
	StartedAt         time.Time      `json:"startedAt,omitempty"`
	CompletedSamples  int            `json:"completedSamples,omitempty"`
	Total             int            `json:"total,omitempty"`
	Elapsed           time.Duration  `json:"elapsed,omitempty"`
	MinimumSamplesMet bool           `json:"minimumSamplesMet,omitempty"`
	Rate              *float64       `json:"rate,omitempty"`
	Remaining         *time.Duration `json:"remaining,omitempty"`
}

// OpsProgressEvent is a typed envelope for service progress callbacks.
type OpsProgressEvent struct {
	Kind        OpsProgressEventKind    `json:"kind,omitempty"`
	Preflight   *OpsPreflightScope      `json:"preflight,omitempty"`
	Page        *OpsPageProgress        `json:"page,omitempty"`
	FrozenScope *OpsFrozenScopeProgress `json:"frozenScope,omitempty"`
	ETA         *OpsETASampleWindow     `json:"eta,omitempty"`
}
