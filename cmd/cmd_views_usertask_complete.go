// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
)

// userTaskCompletionCounts accounts for every selected key, including work that
// never started. Zero counts remain present in the machine contract.
type userTaskCompletionCounts struct {
	Preview      int `json:"preview"`
	Accepted     int `json:"accepted"`
	Failed       int `json:"failed"`
	TimedOut     int `json:"timedOut"`
	Canceled     int `json:"canceled"`
	NotAttempted int `json:"notAttempted"`
}

// userTaskCompletionReport separates local intent from execution evidence and
// exposes payload presence/count without echoing variable names or values.
type userTaskCompletionReport struct {
	// Operation is always complete.
	Operation string `json:"operation"`
	// Phase is preview or execution.
	Phase string `json:"phase"`
	// RequestedKeys is the frozen unique selection, always an initialized array.
	RequestedKeys []string `json:"requestedKeys"`
	// RequestedCount is the length of RequestedKeys.
	RequestedCount int `json:"requestedCount"`
	// VariablesProvided distinguishes omitted source from {}.
	VariablesProvided bool `json:"variablesProvided"`
	// VariableCount counts top-level supplied variable names; no values are echoed.
	VariableCount int `json:"variableCount"`
	// Items is an initialized array in requested order.
	Items []task.UserTaskCompletionResult `json:"items"`
	// Counts correspond exactly to Items and sum to RequestedCount for complete reports.
	Counts userTaskCompletionCounts `json:"counts"`
}

// newUserTaskCompletionPreview derives intent only from validated frozen local
// input. It neither calls the facade nor claims submission or backend eligibility.
func newUserTaskCompletionPreview(input userTaskCompletionInput) userTaskCompletionReport {
	items := make([]task.UserTaskCompletionResult, len(input.Keys))
	for i, key := range input.Keys {
		items[i] = task.UserTaskCompletionResult{Key: key, Status: task.UserTaskCompletionStatusPreview, Submission: task.UserTaskCompletionSubmissionNotSubmitted}
	}
	report := newUserTaskCompletionReport(input, task.UserTaskCompletionResults{Items: items})
	report.Phase = "preview"
	return report
}

// newUserTaskCompletionReport copies the service's complete ordered evidence.
// Local validation failures use the existing error envelope without a report;
// missing outcomes must never be fabricated here as successful or unattempted work.
func newUserTaskCompletionReport(input userTaskCompletionInput, results task.UserTaskCompletionResults) userTaskCompletionReport {
	report := userTaskCompletionReport{
		Operation: "complete", Phase: "execution",
		RequestedKeys: append([]string{}, input.Keys...), RequestedCount: len(input.Keys),
		VariablesProvided: input.Variables != nil, VariableCount: len(input.Variables),
		Items: append([]task.UserTaskCompletionResult{}, results.Items...),
	}
	for _, item := range report.Items {
		switch item.Status {
		case task.UserTaskCompletionStatusPreview:
			report.Counts.Preview++
		case task.UserTaskCompletionStatusAccepted:
			report.Counts.Accepted++
		case task.UserTaskCompletionStatusFailed:
			report.Counts.Failed++
		case task.UserTaskCompletionStatusTimedOut:
			report.Counts.TimedOut++
		case task.UserTaskCompletionStatusCanceled:
			report.Counts.Canceled++
		case task.UserTaskCompletionStatusNotAttempted:
			report.Counts.NotAttempted++
		}
	}
	return report
}

// renderUpdateUserTaskCompletion writes selected-format results only. The runner
// owns configured diagnostic logging and exits with the original operation error.
func renderUpdateUserTaskCompletion(cmd *cobra.Command, report userTaskCompletionReport, operationErr error) error {
	if pickMode() == RenderModeJSON {
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = report
			return renderResultEnvelope(cmd, envelope)
		}
		if report.Phase == "preview" {
			return renderSucceededResult(cmd, report)
		}
		return renderAcceptedResult(cmd, report)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, item := range report.Items {
			if (report.Phase == "preview" && item.Status == task.UserTaskCompletionStatusPreview) ||
				(report.Phase == "execution" && item.Status == task.UserTaskCompletionStatusAccepted) {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if flagQuiet {
		return nil
	}
	var summary string
	if report.Phase == "preview" {
		summary = fmt.Sprintf("dry run: complete user-task: %d user task(s), %d supplied variable(s); no requests submitted", report.RequestedCount, report.VariableCount)
	} else {
		summary = fmt.Sprintf("complete user-task: %d requested; %s", report.RequestedCount, formatUserTaskCompletionCounts(report.Counts))
		if report.Counts.Accepted > 0 {
			summary += "; requests accepted without verification; confirmation skipped"
		} else {
			summary += "; no acceptance established; confirmation skipped"
		}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), summary); err != nil {
		return err
	}
	if flagVerbose {
		for _, item := range report.Items {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), formatUserTaskCompletionItem(item)); err != nil {
				return err
			}
		}
	}
	return nil
}

// formatUserTaskCompletionCounts is shared by results and diagnostics so the
// same accepted, interrupted and unscheduled evidence remains visible in both.
func formatUserTaskCompletionCounts(counts userTaskCompletionCounts) string {
	return fmt.Sprintf("%d accepted, %d failed, %d timed out, %d canceled, %d not attempted", counts.Accepted, counts.Failed, counts.TimedOut, counts.Canceled, counts.NotAttempted)
}

// formatUserTaskCompletionItem allows only known status/submission labels into
// verbose results. Error strings and unknown enum text remain machine evidence.
func formatUserTaskCompletionItem(item task.UserTaskCompletionResult) string {
	status := "unknown"
	switch item.Status {
	case task.UserTaskCompletionStatusPreview, task.UserTaskCompletionStatusAccepted, task.UserTaskCompletionStatusFailed, task.UserTaskCompletionStatusTimedOut, task.UserTaskCompletionStatusCanceled, task.UserTaskCompletionStatusNotAttempted:
		status = string(item.Status)
	}
	submission := "unknown"
	switch item.Submission {
	case task.UserTaskCompletionSubmissionNotSubmitted, task.UserTaskCompletionSubmissionAccepted, task.UserTaskCompletionSubmissionRejected, task.UserTaskCompletionSubmissionUnknown:
		submission = string(item.Submission)
	}
	line := fmt.Sprintf("%s: %s; submission: %s", item.Key, status, submission)
	if label := http.StatusText(item.StatusCode); label != "" {
		line += fmt.Sprintf("; HTTP %d %s", item.StatusCode, label)
	}
	return line
}

// formatUserTaskCompletionFailure returns one bounded diagnostic for the runner
// to emit through its configured logger. No raw item or joined error is formatted.
func formatUserTaskCompletionFailure(report userTaskCompletionReport, operationErr error) string {
	message := "user-task completion: " + formatUserTaskCompletionCounts(report.Counts)
	counts := map[string]int{}
	reasons := []string{}
	for _, item := range report.Items {
		switch item.Status {
		case task.UserTaskCompletionStatusAccepted, task.UserTaskCompletionStatusPreview, task.UserTaskCompletionStatusNotAttempted:
			continue
		}
		reason := userTaskCompletionFailureReason(item)
		if counts[reason] == 0 {
			reasons = append(reasons, reason)
		}
		counts[reason]++
	}
	if len(reasons) == 0 {
		return message + "; " + strings.ReplaceAll(string(ferrors.Classify(operationErr)), "_", " ")
	}
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

// userTaskCompletionFailureReason never promotes a last HTTP rejection into
// certainty when an earlier attempt left acceptance unknown.
func userTaskCompletionFailureReason(item task.UserTaskCompletionResult) string {
	switch item.Status {
	case task.UserTaskCompletionStatusTimedOut:
		return "timed out"
	case task.UserTaskCompletionStatusCanceled:
		return "canceled"
	}
	switch item.Submission {
	case task.UserTaskCompletionSubmissionUnknown:
		return "acceptance unknown"
	case task.UserTaskCompletionSubmissionRejected:
		if item.StatusCode >= 400 && item.StatusCode <= 599 {
			if label := http.StatusText(item.StatusCode); label != "" {
				return fmt.Sprintf("%s (HTTP %d)", strings.ToLower(label), item.StatusCode)
			}
		}
		return "completion rejected"
	default:
		return "completion failed"
	}
}
