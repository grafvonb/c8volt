// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// NewProcessInstanceCancellationConfirmationFailure records the available
// state evidence without changing the supplied cancellation-wait error text.
func NewProcessInstanceCancellationConfirmationFailure(operation, root string, scope []string, timeout time.Duration, err error) *d.ProcessInstanceMutationFailure {
	states, reason := processInstanceConfirmationFailureDetails(err)
	return &d.ProcessInstanceMutationFailure{
		Operation:     operation,
		Phase:         "cancellation confirmation",
		FailureReason: reason,
		RootKey:       root,
		Scope:         append([]string(nil), scope...),
		Timeout:       timeout,
		LastStates:    states,
		Err:           err,
	}
}

// EnrichProcessInstanceDeleteCancellationFailure adds delete-stage facts when
// a cancel-before-delete workflow fails, preserving the original cause chain.
// It copies the annotation without mutating the source; scope and state facts
// are shared read-only. Unannotated errors receive only the delete-cancel prefix.
func EnrichProcessInstanceDeleteCancellationFailure(err error, conflictKey string) error {
	cause := fmt.Errorf("delete cancel: %w", err)
	var failure *d.ProcessInstanceMutationFailure
	if !errors.As(err, &failure) {
		return cause
	}
	enriched := *failure
	enriched.Operation = "delete"
	enriched.DeleteConflictKey = conflictKey
	enriched.Err = cause
	return &enriched
}

// processInstanceConfirmationFailureDetails collects known last states and the
// highest-priority stop reason across waits in one pass. Lookup and attempt-limit
// failures outrank sibling timeouts or cancellations; absent observations are omitted.
func processInstanceConfirmationFailureDetails(err error) (map[string]d.State, string) {
	states := make(map[string]d.State)
	waits := d.ProcessInstanceWaitFailures(err)
	reason := cancellationStopReason(err, "failed")
	priority := map[string]int{"was canceled": 0, "timed out": 1, "exhausted polling attempts": 2, "failed during lookup": 3, "failed": 4}
	for i, wait := range waits {
		if wait.Key != "" && wait.LastState != "" && wait.LastState != d.StateUnknown {
			states[wait.Key] = wait.LastState
		}
		next := cancellationStopReason(wait.Err, wait.Reason)
		if i == 0 || priority[next] > priority[reason] {
			reason = next
		}
	}
	return states, reason
}

// cancellationStopReason identifies deadline or context cancellation first, then
// uses the supplied wait reason. Missing reasons fall back to a generic failure;
// a lookup error alone does not establish that the confirmation budget expired.
func cancellationStopReason(err error, fallback string) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	case errors.Is(err, context.Canceled):
		return "was canceled"
	case fallback != "":
		return fallback
	default:
		return "failed"
	}
}

// NewProcessInstanceCancellationDiscoveryFailure retains accepted submission
// when discovering the confirmation scope fails. No scope is inferred.
func NewProcessInstanceCancellationDiscoveryFailure(root string, err error) *d.ProcessInstanceMutationFailure {
	failure := NewProcessInstanceCancellationConfirmationFailure("cancel", root, nil, 0, err)
	failure.Phase = "cancellation scope discovery"
	return failure
}
