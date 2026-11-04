// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// resumptionCounts keeps acceptance independent from overlapping confirmed/failed outcomes.
func resumptionCounts(results process.ResumptionResults) (accepted, confirmed, failed, skipped int) {
	for _, item := range results.Items {
		if item.Submission == process.ResumptionSubmissionAccepted {
			accepted++
		}
		switch item.Status {
		case process.ResumptionStatusConfirmed:
			confirmed++
		case process.ResumptionStatusNotAttempted:
			skipped++
		case process.ResumptionStatusFailed, process.ResumptionStatusTimedOut, process.ResumptionStatusInterrupted:
			failed++
		}
	}
	return
}

// renderProcessInstanceResumption logs human summaries and writes machine results to stdout. Errors precede no-wait,
// and neither previews nor empty collections can claim accepted submission.
func renderProcessInstanceResumption(cmd *cobra.Command, results process.ResumptionResults, preview bool, operationErr error) error {
	if results.Items == nil {
		results.Items = []process.ResumptionResult{}
	}
	accepted, confirmed, failed, skipped := resumptionCounts(results)
	switch pickMode() {
	case RenderModeJSON:
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = results
			return renderResultEnvelope(cmd, envelope)
		}
		if !preview && flagNoWait && accepted > 0 {
			return renderAcceptedResult(cmd, results)
		}
		return renderSucceededResult(cmd, results)
	case RenderModeKeysOnly:
		for _, item := range results.Items {
			if preview || item.Submission == process.ResumptionSubmissionAccepted {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
					return err
				}
			}
		}
	default:
		if flagQuiet {
			return nil
		}
		if preview {
			renderHumanLine(cmd, "dry run: resume process-instance; selected: %d; eligibility not checked; no changes applied", len(results.Items))
		} else {
			renderHumanLine(cmd, "%s", formatProcessInstanceResumptionSummary(len(results.Items), accepted, confirmed, failed, skipped))
		}
		if flagVerbose {
			for _, item := range results.Items {
				renderHumanLine(cmd, "%s", formatProcessInstanceResumptionOutcome(item))
			}
		}
	}
	return nil
}

// formatProcessInstanceResumptionSummary distinguishes submission from observation.
// Accepted-but-unconfirmed work may also be failed; these are not disjoint buckets.
func formatProcessInstanceResumptionSummary(selected, accepted, confirmed, failed, skipped int) string {
	if selected == 0 {
		return "resumption: no process instances selected; no changes applied"
	}
	line := fmt.Sprintf("resumption: resumed %d/%d process instance(s)", confirmed, selected)
	if flagNoWait {
		line = fmt.Sprintf("resumption: submitted %d/%d process instance(s); confirmation not requested", accepted, selected)
	} else if accepted > confirmed {
		line += fmt.Sprintf("; accepted but unconfirmed %d", accepted-confirmed)
	}
	if failed > 0 {
		line += fmt.Sprintf("; failed %d", failed)
	}
	if skipped > 0 {
		line += fmt.Sprintf("; not attempted %d", skipped)
	}
	return line
}

// formatProcessInstanceResumptionOutcome allowlists lifecycle facts; machine errors and arbitrary
// backend text never become human detail. Dates remain available in the JSON evidence.
func formatProcessInstanceResumptionOutcome(item process.ResumptionResult) string {
	submission, confirmation, state := "unknown", "unknown", "not observed"
	switch item.Submission {
	case process.ResumptionSubmissionNotSubmitted, process.ResumptionSubmissionAccepted, process.ResumptionSubmissionRejected, process.ResumptionSubmissionUnknown:
		submission = strings.ReplaceAll(string(item.Submission), "_", " ")
	}
	switch item.Confirmation {
	case process.ResumptionConfirmationNotRequested, process.ResumptionConfirmationUnconfirmed, process.ResumptionConfirmationConfirmed:
		confirmation = strings.ReplaceAll(string(item.Confirmation), "_", " ")
	}
	if item.HasObservation {
		state = "UNKNOWN"
		switch item.LastObservedState {
		case process.StateActive, process.StateSuspended, process.StateCompleted, process.StateCanceled, process.StateTerminated, process.StateAbsent:
			state = string(item.LastObservedState)
		}
	}
	return fmt.Sprintf("%s: submission: %s; confirmation: %s; last state: %s", item.Key, submission, confirmation, state)
}

// formatProcessInstanceResumptionFailure groups bounded facts in one configured ERROR record.
// It never formats raw operation errors or per-key machine error evidence.
func formatProcessInstanceResumptionFailure(results process.ResumptionResults, operationErr error) string {
	counts := map[string]int{}
	reasons := []string{}
	for _, item := range results.Items {
		if item.Status == process.ResumptionStatusConfirmed || item.Status == process.ResumptionStatusAccepted || item.Status == process.ResumptionStatusPreview {
			continue
		}
		reason := "request not sent"
		switch {
		case item.Status == process.ResumptionStatusNotAttempted:
			reason = "not attempted"
		case item.Status == process.ResumptionStatusTimedOut:
			reason = "timed out"
		case item.Status == process.ResumptionStatusInterrupted:
			reason = "interrupted"
		case item.Submission == process.ResumptionSubmissionRejected:
			reason = "rejected"
		case item.Submission == process.ResumptionSubmissionUnknown:
			reason = "acceptance unknown"
		case item.Submission == process.ResumptionSubmissionAccepted:
			reason = "confirmation failed"
			if item.HasObservation {
				switch item.LastObservedState {
				case process.StateCanceled, process.StateTerminated:
					reason = "cancellation observed"
				}
			}
		}
		if counts[reason] == 0 {
			reasons = append(reasons, reason)
		}
		counts[reason]++
	}
	if len(reasons) == 0 {
		return "resume process instances: operation failed: 1; " + boundedCommandFailureReason(operationErr)
	}
	parts := []string{}
	other := 0
	for i, reason := range reasons {
		if i < 3 {
			parts = append(parts, fmt.Sprintf("%s: %d", reason, counts[reason]))
		} else {
			other += counts[reason]
		}
	}
	if other > 0 {
		parts = append(parts, fmt.Sprintf("other failures: %d", other))
	}
	return "resume process instances: " + strings.Join(parts, "; ")
}
