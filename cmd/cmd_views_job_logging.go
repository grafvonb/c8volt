// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/job"
)

// formatJobCompletionFailure uses bounded report facts, never raw error text.
// HTTP diagnostics remain owned by the redacted DEBUG transport; JSON retains
// the original error and per-job evidence independently of this human summary.
func formatJobCompletionFailure(report job.CompletionBatchReport, err error) string {
	class := strings.ReplaceAll(string(ferrors.Classify(err)), "_", " ")
	switch report.Status {
	case job.CompletionBatchAborted:
		return "job completion: aborted by user; no changes applied"
	case job.CompletionBatchPreparationFailed:
		return fmt.Sprintf("job completion: preparation failed (%s); no changes applied", class)
	}

	prefix := "job completion: "
	if report.Status == job.CompletionBatchInterrupted {
		prefix += "interrupted; "
	}
	message := fmt.Sprintf("%s%d unsatisfied job(s)", prefix, report.Counts.Unsatisfied)
	if report.Counts.NotStarted > 0 {
		message += fmt.Sprintf("; %d not started", report.Counts.NotStarted)
	}
	counts := map[string]int{}
	reasons := []string{}
	for _, outcome := range report.Outcomes {
		if outcome.Disposition != job.CompletionDispositionUnsatisfied {
			continue
		}
		reason := jobCompletionFailureReason(outcome)
		if counts[reason] == 0 {
			reasons = append(reasons, reason)
		}
		counts[reason]++
	}
	if len(reasons) == 0 {
		return message + "; " + class
	}
	// Preserve first-seen report order and bound the summary even for large,
	// heterogeneous batches. Verbose and JSON retain individual outcome facts.
	other := 0
	for i, reason := range reasons {
		if i < 3 {
			message += fmt.Sprintf("; %s: %d", reason, counts[reason])
		} else {
			other += counts[reason]
		}
	}
	if other > 0 {
		message += fmt.Sprintf("; other failures: %d", other)
	}
	return message
}

// jobCompletionFailureReason translates only known outcome codes and HTTP status
// codes; backend diagnostics and unknown reason strings cannot enter human logs.
func jobCompletionFailureReason(outcome job.CompletionOutcome) string {
	switch outcome.Reason {
	case job.CompletionOutcomeRejected:
		if status := outcome.Submission.HTTPStatus; status != nil && *status >= 400 && *status <= 599 {
			if label := http.StatusText(*status); label != "" {
				return fmt.Sprintf("%s (HTTP %d)", strings.ToLower(label), *status)
			}
		}
		return "completion rejected"
	case job.CompletionOutcomeUncertain:
		return "completion unconfirmed"
	case job.CompletionOutcomeScopeChanged:
		return "selection changed"
	case job.CompletionOutcomeOtherTerminal:
		return "already in another terminal state"
	case job.CompletionOutcomeUnsupportedState:
		return "unsupported job state"
	case job.CompletionOutcomeMissing:
		return "job no longer found"
	case job.CompletionOutcomeUnreadable:
		return "job could not be read"
	case job.CompletionOutcomePayloadIncompatible:
		return "incompatible completion payload"
	case job.CompletionOutcomeInterrupted:
		return "interrupted"
	default:
		return "completion unresolved"
	}
}
