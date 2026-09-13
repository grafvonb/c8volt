// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
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

func (e *ProcessInstanceWaitFailure) Error() string { return e.Err.Error() }
func (e *ProcessInstanceWaitFailure) Unwrap() error { return e.Err }

// ProcessInstanceMutationFailure records only the operational facts needed to
// explain an unconfirmed cancel-before-delete workflow.
type ProcessInstanceMutationFailure struct {
	Operation              string
	Phase                  string
	FailureReason          string
	RootKey                string
	Scope                  []string
	Timeout                time.Duration
	LastStates             map[string]State
	DeleteConflictKey      string
	CancellationSubmitted  bool
	ResumedDeletionReached bool
	Err                    error
}

func (e *ProcessInstanceMutationFailure) Error() string { return e.Err.Error() }
func (e *ProcessInstanceMutationFailure) Unwrap() error { return e.Err }

// ProcessInstanceWaitObservations extracts available last states from single
// or joined wait failures without manufacturing observations for unseen keys.
func ProcessInstanceWaitObservations(err error) map[string]State {
	observations := make(map[string]State)
	visitErrors(err, func(candidate error) {
		var failure *ProcessInstanceWaitFailure
		if errors.As(candidate, &failure) && failure.Key != "" && failure.LastState != "" && failure.LastState != StateUnknown {
			observations[failure.Key] = failure.LastState
		}
	})
	return observations
}

func visitErrors(err error, visit func(error)) {
	if err == nil {
		return
	}
	visit(err)
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			visitErrors(child, visit)
		}
		return
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		visitErrors(wrapped, visit)
	}
}

// CloneProcessInstanceMutationFailure makes slice/map facts safe to enrich at
// the next workflow boundary while retaining the same underlying cause.
func CloneProcessInstanceMutationFailure(in *ProcessInstanceMutationFailure) *ProcessInstanceMutationFailure {
	out := *in
	out.Scope = slices.Clone(in.Scope)
	out.LastStates = make(map[string]State, len(in.LastStates))
	for key, state := range in.LastStates {
		out.LastStates[key] = state
	}
	return &out
}
