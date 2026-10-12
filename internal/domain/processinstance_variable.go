// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

// ProcessInstanceVariablePageRequest describes one root-variable search page.
// Cursor and offset positions are mutually exclusive.
type ProcessInstanceVariablePageRequest struct {
	From  int32
	Size  int32
	After string
}

const processInstanceVariablePageSizeLimit int32 = 1000

// Validate rejects requests that cannot advance safely.
func (r ProcessInstanceVariablePageRequest) Validate() error {
	if r.From < 0 {
		return errors.New("process-instance variable page request offset cannot be negative")
	}
	if r.Size <= 0 {
		return errors.New("process-instance variable page request size must be positive")
	}
	if r.Size > processInstanceVariablePageSizeLimit {
		return fmt.Errorf("process-instance variable page request size cannot exceed %d", processInstanceVariablePageSizeLimit)
	}
	if r.From != 0 && r.After != "" {
		return errors.New("process-instance variable page request cannot combine offset and cursor positions")
	}
	return nil
}

// ProcessInstanceVariableReportedTotalKind identifies exact and capped totals.
type ProcessInstanceVariableReportedTotalKind string

const (
	ProcessInstanceVariableReportedTotalKindExact      ProcessInstanceVariableReportedTotalKind = "exact"
	ProcessInstanceVariableReportedTotalKindLowerBound ProcessInstanceVariableReportedTotalKind = "lower_bound"
)

// Validate rejects total semantics that traversal cannot interpret.
func (k ProcessInstanceVariableReportedTotalKind) Validate() error {
	switch k {
	case ProcessInstanceVariableReportedTotalKindExact, ProcessInstanceVariableReportedTotalKindLowerBound:
		return nil
	default:
		return fmt.Errorf("invalid process-instance variable reported total kind %q", k)
	}
}

// ProcessInstanceVariableReportedTotal preserves whether a backend count is exact.
type ProcessInstanceVariableReportedTotal struct {
	Count int64
	Kind  ProcessInstanceVariableReportedTotalKind
}

// Validate rejects malformed backend total metadata.
func (t ProcessInstanceVariableReportedTotal) Validate() error {
	if t.Count < 0 {
		return errors.New("process-instance variable reported total cannot be negative")
	}
	return t.Kind.Validate()
}

// ProcessInstanceVariablePage carries one normalized backend page. RawItemCount
// is measured before the adapter's defensive root-scope filtering.
type ProcessInstanceVariablePage struct {
	Items         []ProcessInstanceVariable
	Request       ProcessInstanceVariablePageRequest
	RawItemCount  int32
	ReportedTotal *ProcessInstanceVariableReportedTotal
	EndCursor     string
}

// ProcessInstanceVariablePlannedValue describes one requested or observed
// root variable whose classification needs only one value.
type ProcessInstanceVariablePlannedValue struct {
	Name         string
	Value        any
	APITruncated bool
}

// ProcessInstanceVariablePlannedChange preserves the observed and requested
// values for one root variable that requires mutation.
type ProcessInstanceVariablePlannedChange struct {
	Name         string
	Before       any
	After        any
	APITruncated bool
}

// ProcessInstanceVariablePlan is the complete root-only classification and
// frozen payload for one selected process instance.
type ProcessInstanceVariablePlan struct {
	ProcessInstanceKey string
	ScopeKey           string
	TenantId           string
	Additions          []ProcessInstanceVariablePlannedValue
	Changes            []ProcessInstanceVariablePlannedChange
	UnchangedRequested []ProcessInstanceVariablePlannedValue
	Untouched          []ProcessInstanceVariablePlannedValue
	MutationVariables  map[string]any
	VerificationValues map[string]any
}

// ProcessInstanceVariableUpdatePlan is the immutable-by-convention result of
// successful collection-wide discovery and classification.
type ProcessInstanceVariableUpdatePlan struct {
	RequestedKeys          []string
	RequestedValues        map[string]any
	Instances              []ProcessInstanceVariablePlan
	RequestedCount         int
	UpdateCount            int
	VariableAddCount       int
	VariableChangeCount    int
	VariableUnchangedCount int
	VariableUntouchedCount int
	MutationSubmitted      bool
	TenantContext          TenantContext
}

type ProcessInstanceVariableUpdateStatus string

const (
	ProcessInstanceVariableUpdateStatusSubmitted          ProcessInstanceVariableUpdateStatus = "submitted"
	ProcessInstanceVariableUpdateStatusConfirmed          ProcessInstanceVariableUpdateStatus = "confirmed"
	ProcessInstanceVariableUpdateStatusMutationFailed     ProcessInstanceVariableUpdateStatus = "mutation_failed"
	ProcessInstanceVariableUpdateStatusConfirmationFailed ProcessInstanceVariableUpdateStatus = "confirmation_failed"
)

type ProcessInstanceVariableUpdateResult struct {
	Key                string
	Status             ProcessInstanceVariableUpdateStatus
	MutationAccepted   bool
	ConfirmationStatus string
	StatusCode         int
	Message            string
	Error              string
	Variables          map[string]any
}

type ProcessInstanceVariableUpdateResults struct {
	Items []ProcessInstanceVariableUpdateResult
}
