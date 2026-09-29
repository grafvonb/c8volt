// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// UserTaskVariablePlannedValue describes one requested or existing variable
// whose plan category does not require both a before and after value.
type UserTaskVariablePlannedValue struct {
	Name         string
	ScopeKey     string
	Inherited    bool
	Value        any
	APITruncated bool
}

// UserTaskVariablePlannedChange preserves the observed and requested values at
// the scope selected during planning.
type UserTaskVariablePlannedChange struct {
	Name         string
	ScopeKey     string
	Inherited    bool
	Before       any
	After        any
	APITruncated bool
}

// ScopeVariableUpdateAssociation records which names on one selected task
// depend on a unique scope write.
type ScopeVariableUpdateAssociation struct {
	UserTaskKey string
	Names       []string
}

// ScopeVariableUpdateTarget is one frozen local write grouped by scope. Its
// Variables map is nonempty for every executable target.
type ScopeVariableUpdateTarget struct {
	ScopeKey     string
	TenantId     string
	Variables    map[string]any
	Associations []ScopeVariableUpdateAssociation
}

// UserTaskVariablePlan classifies the complete effective-variable view for one
// selected user task and references its frozen mutation targets by scope.
type UserTaskVariablePlan struct {
	UserTaskKey        string
	ElementInstanceKey string
	TenantId           string
	Additions          []UserTaskVariablePlannedValue
	Changes            []UserTaskVariablePlannedChange
	UnchangedRequested []UserTaskVariablePlannedValue
	Untouched          []UserTaskVariablePlannedValue
	TargetScopeKeys    []string
}

// UserTaskVariableUpdatePlan is the complete, immutable-by-convention result
// of successful task and effective-variable discovery.
type UserTaskVariableUpdatePlan struct {
	RequestedKeys          []string
	UserTasks              []UserTaskVariablePlan
	Targets                []ScopeVariableUpdateTarget
	RequestedCount         int
	UpdateCount            int
	VariableAddCount       int
	VariableChangeCount    int
	VariableUnchangedCount int
	VariableUntouchedCount int
	MutationSubmitted      bool
	TenantContext          TenantContext
}

// ScopeVariableUpdateStatus identifies whether a frozen target was submitted,
// failed during mutation, or was not started.
type ScopeVariableUpdateStatus string

const (
	// ScopeVariableUpdateStatusSubmitted records an accepted scope write.
	ScopeVariableUpdateStatusSubmitted ScopeVariableUpdateStatus = "submitted"
	// ScopeVariableUpdateStatusMutationFailed records a failed scope write.
	ScopeVariableUpdateStatusMutationFailed ScopeVariableUpdateStatus = "mutation_failed"
	// ScopeVariableUpdateStatusSkipped records a scope write that was not started.
	ScopeVariableUpdateStatusSkipped ScopeVariableUpdateStatus = "skipped"
)

// ScopeVariableUpdateOutcome retains submission facts for one frozen target.
type ScopeVariableUpdateOutcome struct {
	ScopeKey   string
	Names      []string
	Status     ScopeVariableUpdateStatus
	Accepted   bool
	StatusCode int
	Message    string
	Error      string
}

// UserTaskVariableUpdateStatus names the task-level terminal or unfinished
// states used by execution and confirmation.
type UserTaskVariableUpdateStatus string

const (
	// UserTaskVariableUpdateStatusConfirmed means all requested values were observed at their planned scopes.
	UserTaskVariableUpdateStatusConfirmed UserTaskVariableUpdateStatus = "confirmed"
	// UserTaskVariableUpdateStatusSubmitted means all writes were accepted and confirmation was skipped.
	UserTaskVariableUpdateStatusSubmitted UserTaskVariableUpdateStatus = "submitted"
	// UserTaskVariableUpdateStatusMutationFailed means at least one required write failed.
	UserTaskVariableUpdateStatusMutationFailed UserTaskVariableUpdateStatus = "mutation_failed"
	// UserTaskVariableUpdateStatusConfirmationFailed means accepted writes could not be confirmed.
	UserTaskVariableUpdateStatusConfirmationFailed UserTaskVariableUpdateStatus = "confirmation_failed"
	// UserTaskVariableUpdateStatusUnchanged means planning found no required write for the task.
	UserTaskVariableUpdateStatusUnchanged UserTaskVariableUpdateStatus = "unchanged"
	// UserTaskVariableUpdateStatusSkipped means required work was not started.
	UserTaskVariableUpdateStatusSkipped UserTaskVariableUpdateStatus = "skipped"
)

// UserTaskVariableUpdateResult retains task-level status plus every dependent
// scope outcome so partial and shared writes remain truthful.
type UserTaskVariableUpdateResult struct {
	Key                string
	Status             UserTaskVariableUpdateStatus
	MutationAccepted   bool
	ConfirmationStatus string
	Message            string
	Error              string
	Variables          map[string]any
	Scopes             []ScopeVariableUpdateOutcome
}

// UserTaskVariableUpdateResults preserves unique requested task order.
type UserTaskVariableUpdateResults struct {
	Items []UserTaskVariableUpdateResult
}

// ScopeVariableUpdateResponse records whether a scope-local variable write was accepted.
// Confirmation belongs to the user-task update workflow and is intentionally absent here.
type ScopeVariableUpdateResponse struct {
	ScopeKey   string
	Accepted   bool
	StatusCode int
	Status     string
}
