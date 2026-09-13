// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
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
