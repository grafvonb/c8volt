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

// renderUpdateUserTaskCompletion keeps machine results on stdout and routes human
// summaries through the configured INFO logger. The runner owns warnings, errors and exits.
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
		if report.RequestedCount == 1 {
			summary = fmt.Sprintf("dry run: update user-task %s: completion: submit", report.RequestedKeys[0])
		} else {
			summary = fmt.Sprintf("dry run: update user-task completion: %d requested", report.RequestedCount)
		}
		if report.VariablesProvided {
			summary += "; variables: submit"
		}
		summary += "; no requests submitted"
	} else if report.RequestedCount == 1 && report.Counts.Accepted == 1 {
		summary = fmt.Sprintf("updated user-task %s: completion confirmed", report.RequestedKeys[0])
	} else {
		summary = fmt.Sprintf("update user-task: %d requested; %s", report.RequestedCount, formatUserTaskCompletionCounts(report.Counts))
	}
	renderHumanLine(cmd, "%s", summary)
	if flagVerbose {
		for _, item := range report.Items {
			renderHumanLine(cmd, "%s", formatUserTaskCompletionItem(item, report.VariablesProvided))
		}
	}
	return nil
}

// formatUserTaskCompletionCounts omits zero counters from human summaries only.
func formatUserTaskCompletionCounts(counts userTaskCompletionCounts) string {
	parts := []string{}
	for _, entry := range []struct {
		count int
		label string
	}{
		{counts.Accepted, "completed"}, {counts.Failed, "failed"},
		{counts.TimedOut, "timed out"}, {counts.Canceled, "canceled"},
		{counts.NotAttempted, "not attempted"},
	} {
		if entry.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", entry.count, entry.label))
		}
	}
	return strings.Join(parts, ", ")
}

// formatUserTaskCompletionConflict explains the observed state before its consequence.
func formatUserTaskCompletionConflict(count int, variablesProvided bool) string {
	message := "task is already completed; completion request not sent"
	if count > 1 {
		message = fmt.Sprintf("%d tasks are already completed; their completion requests were not sent", count)
	}
	if variablesProvided {
		message += "; supplied variables were not applied"
	}
	return message
}

// formatUserTaskCompletionItem allows only known status/submission labels into
// verbose results. Error strings and unknown enum text remain machine evidence.
func formatUserTaskCompletionItem(item task.UserTaskCompletionResult, variablesProvided bool) string {
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
	if item.Status == task.UserTaskCompletionStatusAccepted && item.Submission == task.UserTaskCompletionSubmissionAccepted {
		line = fmt.Sprintf("updated user-task %s: completion confirmed", item.Key)
	}
	if item.Status == task.UserTaskCompletionStatusFailed && item.Submission == task.UserTaskCompletionSubmissionNotSubmitted {
		line += "; " + formatUserTaskCompletionConflict(1, variablesProvided)
	}
	if label := http.StatusText(item.StatusCode); label != "" {
		line += fmt.Sprintf("; HTTP %d %s", item.StatusCode, label)
	}
	return line
}

// formatUserTaskCompletionFailure returns one bounded diagnostic for the runner
// to emit through its configured logger. No raw item or joined error is formatted.
func formatUserTaskCompletionFailure(report userTaskCompletionReport, operationErr error) string {
	parts := []string{}
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
		return "user-task completion: " + strings.ReplaceAll(string(ferrors.Classify(operationErr)), "_", " ")
	}
	other := 0
	for i, reason := range reasons {
		if i < 3 {
			if reason == "already completed" {
				parts = append(parts, formatUserTaskCompletionConflict(counts[reason], report.VariablesProvided))
			} else {
				parts = append(parts, fmt.Sprintf("%s: %d", reason, counts[reason]))
			}
		} else {
			other += counts[reason]
		}
	}
	if other > 0 {
		parts = append(parts, fmt.Sprintf("other failures: %d", other))
	}
	return "user-task completion: " + strings.Join(parts, "; ")
}

// userTaskCompletionFailureReason never promotes a last HTTP rejection into
// certainty when an earlier attempt left acceptance unknown.
func userTaskCompletionFailureReason(item task.UserTaskCompletionResult) string {
	if item.Status == task.UserTaskCompletionStatusFailed && item.Submission == task.UserTaskCompletionSubmissionNotSubmitted {
		return "already completed"
	}
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

// renderUserTaskCompletionPlan emits pre-confirmation information through the configured logger.
// The supplied counts come from preparation; rendering never reads task state.
func renderUserTaskCompletionPlan(cmd *cobra.Command, input userTaskCompletionInput, plan task.UserTaskCompletionPlan) {
	if pickMode() != RenderModeOneLine || flagQuiet || automationModeEnabled(cmd) {
		return
	}
	renderAttachedTenantContext(cmd)
	line := ""
	if len(input.Keys) == 1 && plan.ReadyCount == 1 {
		line = fmt.Sprintf("plan: update user-task %s: completion: submit", input.Keys[0])
	} else {
		line = fmt.Sprintf("plan: update user-task completion: %d requested", len(input.Keys))
		if plan.ReadyCount > 0 {
			line += fmt.Sprintf("; %d eligible", plan.ReadyCount)
		}
		if plan.CompletedCount > 0 {
			line += fmt.Sprintf("; %d already completed", plan.CompletedCount)
		}
	}
	if plan.ReadyCount == 0 {
		line += "; fail-fast: no completion requests will be sent"
	} else if input.Variables != nil {
		line += "; variables: submit"
	}
	renderHumanLine(cmd, "%s", line)
}

// formatUserTaskCompletionAssigneeWarning keeps singular and bulk warnings actionable.
func formatUserTaskCompletionAssigneeWarning(requested, unassigned int) string {
	if requested == 1 {
		return "user-task completion: task has no assignee"
	}
	if unassigned == 1 {
		return "user-task completion: 1 eligible task has no assignee"
	}
	return fmt.Sprintf("user-task completion: %d eligible tasks have no assignee", unassigned)
}

func formatUserTaskCompletionLookupWarning(count int) string {
	noun := "user tasks"
	if count == 1 {
		noun = "user task"
	}
	return fmt.Sprintf("user-task completion: could not check state or assignee for %d %s; backend validation will determine eligibility", count, noun)
}
