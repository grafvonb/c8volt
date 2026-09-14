// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/spf13/cobra"
)

// formatProcessInstanceMutationCommandFailures keeps the final human error
// compact; per-tree warnings carry the available scope and observation detail.
func formatProcessInstanceMutationCommandFailures(failures []*domain.ProcessInstanceMutationFailure) string {
	if len(failures) == 0 {
		return "process-instance mutation failed"
	}
	operation := strings.TrimSpace(failures[0].Operation)
	if operation == "" {
		operation = "process-instance mutation"
	} else {
		operation += " process instances"
	}
	phase := strings.TrimSpace(failures[0].Phase)
	if phase == "" {
		phase = "confirmation"
	}
	reason := failures[0].FailureReason
	if reason == "" {
		reason = "failed"
	}
	for _, failure := range failures[1:] {
		if failure.Phase != phase {
			phase = "cancellation follow-up"
		}
		if failure.FailureReason != reason {
			reason = "failed"
		}
	}
	message := fmt.Sprintf("%s: %s %s", operation, phase, reason)
	roots := processInstanceMutationFailureRoots(failures)
	switch len(roots) {
	case 1:
		message += " for root " + roots[0] + " (1 tree)"
	case 0:
		message += fmt.Sprintf(" for %d tree(s)", len(failures))
	default:
		message += fmt.Sprintf(" for roots %s (%d trees)", strings.Join(roots, ","), len(roots))
	}
	submitted := len(failures)
	if submitted == 1 {
		message += "; cancellation submitted, outcome unconfirmed"
	} else if submitted > 1 {
		message += "; cancellations submitted, outcomes unconfirmed"
	}
	return message
}

// processInstanceMutationFailureRoots returns unique known roots in stable key order.
func processInstanceMutationFailureRoots(failures []*domain.ProcessInstanceMutationFailure) []string {
	seen := make(map[string]struct{}, len(failures))
	for _, failure := range failures {
		if failure == nil {
			continue
		}
		root := strings.TrimSpace(failure.RootKey)
		if root != "" {
			seen[root] = struct{}{}
		}
	}
	roots := make([]string, 0, len(seen))
	for root := range seen {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots
}

// handleProcessInstanceMutationError owns cancellation diagnostics; the shared
// boundary still owns JSON envelopes, class selection and process exit.
func handleProcessInstanceMutationError(cmd *cobra.Command, log *slog.Logger, noErrCodes bool, err error) {
	failures := domain.ProcessInstanceMutationFailures(err)
	if len(failures) == 0 {
		handleCommandError(cmd, log, noErrCodes, err)
		return
	}
	log.Debug("process-instance mutation failure", "error", ferrors.Normalize(err).Error())
	handleCommandError(cmd, log, noErrCodes, err, formatProcessInstanceMutationCommandFailures(failures))
}
