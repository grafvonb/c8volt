// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
)

// formatProcessInstanceMutationCommandFailure keeps the final human error
// compact; per-tree warnings carry the available scope and observation detail.
func formatProcessInstanceMutationCommandFailure(failure *ferrors.ProcessInstanceMutationFailure) string {
	operation := strings.TrimSpace(failure.Operation)
	if operation == "" {
		operation = "process-instance mutation"
	} else {
		operation += " process instances"
	}
	message := fmt.Sprintf("%s: %s timed out", operation, failure.Phase)
	if failure.RootKey != "" {
		message += " for root " + failure.RootKey + " (1 tree)"
	}
	if failure.CancellationSubmitted {
		message += "; cancellation submitted, outcome unconfirmed"
	}
	return message
}
