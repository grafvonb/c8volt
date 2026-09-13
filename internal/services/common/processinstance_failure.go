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
	return &d.ProcessInstanceMutationFailure{
		Operation:             operation,
		Phase:                 "cancellation confirmation",
		FailureReason:         processInstanceConfirmationFailureReason(err),
		RootKey:               root,
		Scope:                 append([]string(nil), scope...),
		Timeout:               timeout,
		LastStates:            d.ProcessInstanceWaitObservations(err),
		CancellationSubmitted: true,
		Err:                   err,
	}
}

// EnrichProcessInstanceDeleteCancellationFailure adds delete-stage facts when
// a cancel-before-delete workflow fails, preserving the original cause chain.
func EnrichProcessInstanceDeleteCancellationFailure(err error, conflictKey string) error {
	cause := fmt.Errorf("delete cancel: %w", err)
	var failure *d.ProcessInstanceMutationFailure
	if !errors.As(err, &failure) {
		return cause
	}
	enriched := d.CloneProcessInstanceMutationFailure(failure)
	enriched.Operation = "delete"
	enriched.DeleteConflictKey = conflictKey
	enriched.ResumedDeletionReached = false
	enriched.Err = cause
	return enriched
}

func processInstanceConfirmationFailureReason(err error) string {
	// Inspect each wait independently: a sibling's later timeout or fail-fast
	// cancellation must not hide the lookup failure that stopped another worker.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		reason := "was canceled"
		priority := map[string]int{"was canceled": 0, "timed out": 1, "exhausted polling attempts": 2, "failed during lookup": 3, "failed": 4}
		for _, child := range joined.Unwrap() {
			childReason := processInstanceConfirmationFailureReason(child)
			if priority[childReason] > priority[reason] {
				reason = childReason
			}
		}
		return reason
	}
	if wait, ok := err.(*d.ProcessInstanceWaitFailure); ok {
		switch {
		case errors.Is(wait, context.DeadlineExceeded):
			return "timed out"
		case errors.Is(wait, context.Canceled):
			return "was canceled"
		case wait.Reason != "":
			return wait.Reason
		}
	}
	if cause := errors.Unwrap(err); cause != nil {
		return processInstanceConfirmationFailureReason(cause)
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
		return "timed out"
	case errors.Is(err, context.Canceled):
		return "was canceled"
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
