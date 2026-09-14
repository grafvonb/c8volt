// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"time"
)

// ProcessInstanceWaitFailure retains the last successful observation when a
// state wait ends without confirmation. Error text and the original cause are
// deliberately unchanged for callers that consume the existing error contract.
type ProcessInstanceWaitFailure struct {
	Reason    string
	Key       string
	LastState State
	Attempts  int
	Elapsed   time.Duration
	Err       error
}

// Error preserves the underlying error text without adding annotation details.
func (e *ProcessInstanceWaitFailure) Error() string { return e.Err.Error() }

// Unwrap retains the original cause for errors.Is and errors.As inspection.
func (e *ProcessInstanceWaitFailure) Unwrap() error { return e.Err }

// ProcessInstanceMutationFailure annotates failed follow-up after accepted
// cancellation. Its presence means submission succeeded but confirmation did
// not; in a delete workflow, resumed deletion has therefore not been reached.
type ProcessInstanceMutationFailure struct {
	Operation         string
	Phase             string
	FailureReason     string
	RootKey           string
	Scope             []string
	Timeout           time.Duration
	LastStates        map[string]State
	DeleteConflictKey string
	Err               error
}

// Error preserves the underlying error text without adding annotation details.
func (e *ProcessInstanceMutationFailure) Error() string { return e.Err.Error() }

// Unwrap retains the original cause for errors.Is and errors.As inspection.
func (e *ProcessInstanceMutationFailure) Unwrap() error { return e.Err }

// ProcessInstanceWaitFailures extracts one annotation per stopped wait.
func ProcessInstanceWaitFailures(err error) []*ProcessInstanceWaitFailure {
	var failures []*ProcessInstanceWaitFailure
	visitErrors(err, func(candidate error) bool {
		if failure, ok := candidate.(*ProcessInstanceWaitFailure); ok {
			failures = append(failures, failure)
			return false
		}
		return true
	})
	return failures
}

// ProcessInstanceMutationFailures returns the outer annotation for each tree;
// an enriched inner annotation describes the same failure and is skipped.
func ProcessInstanceMutationFailures(err error) []*ProcessInstanceMutationFailure {
	var failures []*ProcessInstanceMutationFailure
	visitErrors(err, func(candidate error) bool {
		if failure, ok := candidate.(*ProcessInstanceMutationFailure); ok {
			failures = append(failures, failure)
			return false
		}
		return true
	})
	return failures
}

// visitErrors visits a non-nil error before its descendants. Returning false from
// descend prunes that branch without skipping siblings. Cause takes precedence
// over Unwrap; joined children are visited in order, followed recursively by their
// descendants. Callers must supply an acyclic error tree.
func visitErrors(err error, descend func(error) bool) {
	if err == nil || !descend(err) {
		return
	}
	if cause, ok := err.(interface{ Cause() error }); ok {
		// Facade normalization keeps the selected class in Unwrap and the
		// inspectable cancellation error here, without copying its facts.
		visitErrors(cause.Cause(), descend)
	} else if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			visitErrors(child, descend)
		}
	} else {
		visitErrors(errors.Unwrap(err), descend)
	}
}
