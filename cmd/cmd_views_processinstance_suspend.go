// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// suspensionCounts keeps acceptance independent from overlapping confirmed/failed outcomes.
func suspensionCounts(results process.SuspensionResults) (accepted, confirmed, failed, skipped int) {
	for _, item := range results.Items {
		if item.Submission == process.SuspensionSubmissionAccepted {
			accepted++
		}
		switch item.Status {
		case process.SuspensionStatusConfirmed:
			confirmed++
		case process.SuspensionStatusNotAttempted:
			skipped++
		case process.SuspensionStatusFailed, process.SuspensionStatusTimedOut, process.SuspensionStatusInterrupted:
			failed++
		}
	}
	return
}

// renderProcessInstanceSuspension writes selected-format results only. Errors precede no-wait,
// and neither previews nor empty collections can claim accepted submission.
func renderProcessInstanceSuspension(cmd *cobra.Command, results process.SuspensionResults, preview bool, operationErr error) error {
	if results.Items == nil {
		results.Items = []process.SuspensionResult{}
	}
	accepted, confirmed, failed, skipped := suspensionCounts(results)
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
			if preview || item.Submission == process.SuspensionSubmissionAccepted {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
					return err
				}
			}
		}
	default:
		if flagQuiet {
			return nil
		}
		var err error
		if preview {
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "dry run: suspend process instances: %d selected; no changes applied; eligibility not checked\n", len(results.Items))
		} else {
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "suspend process instances: %d selected; accepted: %d, confirmed: %d, failed: %d, skipped: %d\n", len(results.Items), accepted, confirmed, failed, skipped)
		}
		if err != nil {
			return err
		}
		if flagVerbose {
			for _, item := range results.Items {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), formatProcessInstanceSuspensionOutcome(item)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// formatProcessInstanceSuspensionOutcome allowlists lifecycle facts; machine errors and arbitrary
// backend text never become human detail. Dates remain available in the JSON evidence.
func formatProcessInstanceSuspensionOutcome(item process.SuspensionResult) string {
	submission, confirmation, state := "unknown", "unknown", "not observed"
	switch item.Submission {
	case process.SuspensionSubmissionNotSubmitted, process.SuspensionSubmissionAccepted, process.SuspensionSubmissionRejected, process.SuspensionSubmissionUnknown:
		submission = string(item.Submission)
	}
	switch item.Confirmation {
	case process.SuspensionConfirmationNotRequested, process.SuspensionConfirmationUnconfirmed, process.SuspensionConfirmationConfirmed:
		confirmation = string(item.Confirmation)
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

// formatProcessInstanceSuspensionFailure groups bounded facts in one configured ERROR record.
// It never formats raw operation errors or per-key machine error evidence.
func formatProcessInstanceSuspensionFailure(results process.SuspensionResults, operationErr error) string {
	counts := map[string]int{}
	reasons := []string{}
	for _, item := range results.Items {
		if item.Status == process.SuspensionStatusConfirmed || item.Status == process.SuspensionStatusAccepted || item.Status == process.SuspensionStatusPreview {
			continue
		}
		reason := "request not sent"
		switch {
		case item.Status == process.SuspensionStatusNotAttempted:
			reason = "not attempted"
		case item.Status == process.SuspensionStatusTimedOut:
			reason = "timed out"
		case item.Status == process.SuspensionStatusInterrupted:
			reason = "interrupted"
		case item.Submission == process.SuspensionSubmissionRejected:
			reason = "rejected"
		case item.Submission == process.SuspensionSubmissionUnknown:
			reason = "acceptance unknown"
		case item.Submission == process.SuspensionSubmissionAccepted:
			reason = "confirmation failed"
			if item.HasObservation {
				switch item.LastObservedState {
				case process.StateCompleted, process.StateCanceled, process.StateTerminated:
					reason = "terminal observed"
				}
			}
		}
		if counts[reason] == 0 {
			reasons = append(reasons, reason)
		}
		counts[reason]++
	}
	if len(reasons) == 0 {
		return "suspend process instances: operation failed: 1; " + boundedCommandFailureReason(operationErr)
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
	return "suspend process instances: " + strings.Join(parts, "; ")
}
