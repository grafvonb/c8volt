// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import "time"

// TotalCertainty classifies whether a progress count is exact, approximate, or unavailable.
type TotalCertainty string

const (
	// TotalCertaintyExact means the count describes the complete selected scope.
	TotalCertaintyExact TotalCertainty = "exact"
	// TotalCertaintyLowerBound means at least the displayed count exists, but more may exist.
	TotalCertaintyLowerBound TotalCertainty = "lower_bound"
	// TotalCertaintyEstimated means the count is a c8volt-derived approximation.
	TotalCertaintyEstimated TotalCertainty = "estimated"
	// TotalCertaintyUnknown means no trustworthy count is available.
	TotalCertaintyUnknown TotalCertainty = "unknown"
)

// PageCountKind classifies whether a page count is exact, estimated, or unavailable.
type PageCountKind string

const (
	// PageCountKindExact means the total page count is known.
	PageCountKindExact PageCountKind = "exact"
	// PageCountKindEstimated means the total page count is approximate.
	PageCountKindEstimated PageCountKind = "estimated"
	// PageCountKindUnknown means the total page count is unavailable.
	PageCountKindUnknown PageCountKind = "unknown"
)

// OverflowState normalizes backend continuation state for progress renderers.
type OverflowState string

const (
	// OverflowStateNoMore means the backend indicated traversal is complete.
	OverflowStateNoMore OverflowState = "no_more"
	// OverflowStateHasMore means the backend indicated another page is available.
	OverflowStateHasMore OverflowState = "has_more"
	// OverflowStateIndeterminate means the backend could not prove whether more pages exist.
	OverflowStateIndeterminate OverflowState = "indeterminate"
	// OverflowStateUnknown means no continuation metadata was available.
	OverflowStateUnknown OverflowState = "unknown"
)

// ProgressEventKind identifies the kind of structured progress fact emitted by a service workflow.
type ProgressEventKind string

const (
	// ProgressEventKindPreflight carries the best available scope summary before expensive work.
	ProgressEventKindPreflight ProgressEventKind = "preflight"
	// ProgressEventKindPage carries paged discovery progress.
	ProgressEventKindPage ProgressEventKind = "page"
	// ProgressEventKindFrozenScope carries exact counters for a frozen work set.
	ProgressEventKindFrozenScope ProgressEventKind = "frozen_scope"
	// ProgressEventKindETA carries timing samples used for approximate ETA rendering.
	ProgressEventKindETA ProgressEventKind = "eta"
)

// ProgressMode identifies an output context for progress-channel gating.
type ProgressMode string

const (
	// ProgressModeHuman is the default interactive command output mode.
	ProgressModeHuman ProgressMode = "human"
	// ProgressModeVerbose is human output with durable progress detail enabled.
	ProgressModeVerbose ProgressMode = "verbose"
	// ProgressModeDebug is diagnostic output where low-level traces may already exist.
	ProgressModeDebug ProgressMode = "debug"
	// ProgressModeJSON is the JSON result contract.
	ProgressModeJSON ProgressMode = "json"
	// ProgressModeKeysOnly is the one-key-per-line result contract.
	ProgressModeKeysOnly ProgressMode = "keys-only"
	// ProgressModeQuiet suppresses non-error progress chatter.
	ProgressModeQuiet ProgressMode = "quiet"
	// ProgressModeAutomation is deterministic unattended execution.
	ProgressModeAutomation ProgressMode = "automation"
)

// ProgressChannel records where progress may be emitted for a command mode.
type ProgressChannel struct {
	Mode                    ProgressMode `json:"mode,omitempty"`
	TransientAllowed        bool         `json:"transientAllowed,omitempty"`
	DurableAllowed          bool         `json:"durableAllowed,omitempty"`
	StdoutAllowed           bool         `json:"stdoutAllowed,omitempty"`
	StderrAllowed           bool         `json:"stderrAllowed,omitempty"`
	StructuredReportAllowed bool         `json:"structuredReportAllowed,omitempty"`
}

// ConsequenceSummary describes follow-on work in operator-facing structured parts.
type ConsequenceSummary struct {
	ResourceSummary  string `json:"resourceSummary,omitempty"`
	WorkSummary      string `json:"workSummary,omitempty"`
	RiskSummary      string `json:"riskSummary,omitempty"`
	ConfirmationText string `json:"confirmationText,omitempty"`
}

// PreflightScope summarizes apparent command scope before deeper work starts.
type PreflightScope struct {
	Phase                string             `json:"phase,omitempty"`
	Command              string             `json:"command,omitempty"`
	CoreResource         string             `json:"coreResource,omitempty"`
	SelectorSummary      string             `json:"selectorSummary,omitempty"`
	Total                *int64             `json:"total,omitempty"`
	TotalKind            TotalCertainty     `json:"totalKind,omitempty"`
	PageSize             int32              `json:"pageSize,omitempty"`
	PageCount            *int64             `json:"pageCount,omitempty"`
	PageCountKind        PageCountKind      `json:"pageCountKind,omitempty"`
	ConsequenceSummary   ConsequenceSummary `json:"consequenceSummary,omitempty"`
	RequiresConfirmation bool               `json:"requiresConfirmation,omitempty"`
	ExpensivePreflight   bool               `json:"expensivePreflight,omitempty"`
}

// PageProgress reports one discovery page without owning rendering language.
type PageProgress struct {
	Phase            string        `json:"phase,omitempty"`
	CurrentPage      int           `json:"currentPage,omitempty"`
	PageCount        *int64        `json:"pageCount,omitempty"`
	PageCountKind    PageCountKind `json:"pageCountKind,omitempty"`
	PageSize         int32         `json:"pageSize,omitempty"`
	CurrentPageCount int           `json:"currentPageCount,omitempty"`
	Seen             int           `json:"seen,omitempty"`
	Selected         int           `json:"selected,omitempty"`
	OverflowState    OverflowState `json:"overflowState,omitempty"`
	LimitReached     bool          `json:"limitReached,omitempty"`
}

// FrozenScopeProgress reports exact progress across an immutable work set.
type FrozenScopeProgress struct {
	Phase        string         `json:"phase,omitempty"`
	CoreResource string         `json:"coreResource,omitempty"`
	Done         int            `json:"done,omitempty"`
	Total        int            `json:"total,omitempty"`
	Elapsed      time.Duration  `json:"elapsed,omitempty"`
	Rate         *float64       `json:"rate,omitempty"`
	ETA          *time.Duration `json:"eta,omitempty"`
	Errors       int            `json:"errors,omitempty"`
}

// ETASampleWindow carries enough timing data for command formatters to decide whether ETA is useful.
type ETASampleWindow struct {
	Phase             string         `json:"phase,omitempty"`
	StartedAt         time.Time      `json:"startedAt,omitempty"`
	CompletedSamples  int            `json:"completedSamples,omitempty"`
	Total             int            `json:"total,omitempty"`
	Elapsed           time.Duration  `json:"elapsed,omitempty"`
	MinimumSamplesMet bool           `json:"minimumSamplesMet,omitempty"`
	Rate              *float64       `json:"rate,omitempty"`
	Remaining         *time.Duration `json:"remaining,omitempty"`
}

// ProgressEvent is a typed envelope for service progress callbacks.
type ProgressEvent struct {
	Kind        ProgressEventKind    `json:"kind,omitempty"`
	Preflight   *PreflightScope      `json:"preflight,omitempty"`
	Page        *PageProgress        `json:"page,omitempty"`
	FrozenScope *FrozenScopeProgress `json:"frozenScope,omitempty"`
	ETA         *ETASampleWindow     `json:"eta,omitempty"`
}
