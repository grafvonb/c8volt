// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// userTaskAssignmentCounts accounts for every selected key, including work that
// never started. Zero counts remain present in the machine contract.
type userTaskAssignmentCounts struct {
	Preview      int `json:"preview"`
	Accepted     int `json:"accepted"`
	Failed       int `json:"failed"`
	TimedOut     int `json:"timedOut"`
	Canceled     int `json:"canceled"`
	NotAttempted int `json:"notAttempted"`
}

// userTaskAssignmentReport keeps frozen assignment intent and ordered native
// response evidence in the stable machine contract.
type userTaskAssignmentReport struct {
	// Operation is assign or unassign.
	Operation string `json:"operation"`
	// Phase is preview or execution.
	Phase string `json:"phase"`
	// RequestedKeys is the frozen unique selection, always an initialized array.
	RequestedKeys []string `json:"requestedKeys"`
	// RequestedCount is the length of RequestedKeys.
	RequestedCount int `json:"requestedCount"`
	// Assignee and AllowOverride are absent for unassignment; false is explicit for assignment.
	Assignee      *string `json:"assignee,omitempty"`
	AllowOverride *bool   `json:"allowOverride,omitempty"`
	// Items is an initialized array in requested order.
	Items []task.UserTaskAssignmentResult `json:"items"`
	// Counts correspond exactly to Items and sum to RequestedCount for complete reports.
	Counts userTaskAssignmentCounts `json:"counts"`
}

// newUserTaskAssignmentPreview derives intent only from validated frozen local
// input. It neither calls the facade nor claims submission or backend eligibility.
func newUserTaskAssignmentPreview(input userTaskAssignmentInput) userTaskAssignmentReport {
	items := make([]task.UserTaskAssignmentResult, len(input.Keys))
	for i, key := range input.Keys {
		items[i] = task.UserTaskAssignmentResult{Key: key, Status: task.UserTaskAssignmentStatusPreview, Submission: task.UserTaskAssignmentSubmissionNotSubmitted}
	}
	report := newUserTaskAssignmentReport(input, task.UserTaskAssignmentResults{Items: items})
	report.Phase = "preview"
	return report
}

// newUserTaskAssignmentReport copies the service's complete ordered evidence.
// Local validation failures use the existing error envelope without a report;
// missing outcomes must never be fabricated here as successful or unattempted work.
func newUserTaskAssignmentReport(input userTaskAssignmentInput, results task.UserTaskAssignmentResults) userTaskAssignmentReport {
	report := userTaskAssignmentReport{
		Operation: input.Operation, Phase: "execution",
		RequestedKeys: append([]string{}, input.Keys...), RequestedCount: len(input.Keys),
		Items: append([]task.UserTaskAssignmentResult{}, results.Items...),
	}
	if input.Operation == "assign" {
		assignee, override := input.Assignee, input.AllowOverride
		report.Assignee, report.AllowOverride = &assignee, &override
	}
	for _, item := range report.Items {
		switch item.Status {
		case task.UserTaskAssignmentStatusPreview:
			report.Counts.Preview++
		case task.UserTaskAssignmentStatusAccepted:
			report.Counts.Accepted++
		case task.UserTaskAssignmentStatusFailed:
			report.Counts.Failed++
		case task.UserTaskAssignmentStatusTimedOut:
			report.Counts.TimedOut++
		case task.UserTaskAssignmentStatusCanceled:
			report.Counts.Canceled++
		case task.UserTaskAssignmentStatusNotAttempted:
			report.Counts.NotAttempted++
		}
	}
	return report
}

// renderUpdateUserTaskAssignment keeps machine results on stdout and routes human
// summaries through the configured INFO logger. The runner owns warnings, errors and exits.
func renderUpdateUserTaskAssignment(cmd *cobra.Command, report userTaskAssignmentReport, operationErr error) error {
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
			if (report.Phase == "preview" && item.Status == task.UserTaskAssignmentStatusPreview) ||
				(report.Phase == "execution" && item.Status == task.UserTaskAssignmentStatusAccepted) {
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
	// Never fall through renderHumanLine's legacy stdout fallback.
	if _, err := logging.FromContext(cmd.Context()); err != nil {
		return err
	}
	label := assignmentOperationLabel(report.Operation)
	var summary string
	if report.Phase == "preview" {
		summary = "dry run: " + formatUserTaskAssignmentIntent(report) + "; no requests submitted"
	} else if report.RequestedCount == 1 && report.Counts.Accepted == 1 {
		summary = fmt.Sprintf("updated user-task %s: %s confirmed", report.RequestedKeys[0], label)
	} else {
		summary = fmt.Sprintf("update user-task %s: %d requested; %s", label, report.RequestedCount, formatUserTaskAssignmentCounts(report.Counts))
	}
	renderHumanLine(cmd, "%s", summary)
	// Bulk previews expose the entire frozen scope even without verbose output.
	if report.Phase == "preview" && report.RequestedCount > 1 {
		for _, key := range report.RequestedKeys {
			renderHumanLine(cmd, "selected user-task %s", key)
		}
	}
	if flagVerbose {
		for _, item := range report.Items {
			renderHumanLine(cmd, "%s", formatUserTaskAssignmentItem(report.Operation, item))
		}
	}
	return nil
}

// assignmentOperationLabel uses only the two locally validated action labels.
func assignmentOperationLabel(operation string) string {
	if operation == "unassign" {
		return "unassignment"
	}
	return "assignment"
}

// formatUserTaskAssignmentIntent safely quotes opaque identity bytes for plans
// and previews. It describes local intent only, without claiming eligibility.
func formatUserTaskAssignmentIntent(report userTaskAssignmentReport) string {
	label := assignmentOperationLabel(report.Operation)
	line := fmt.Sprintf("update user-task %s: %d requested", label, report.RequestedCount)
	if report.RequestedCount == 1 {
		line = fmt.Sprintf("update user-task %s: %s", report.RequestedKeys[0], label)
	}
	if report.Operation == "assign" && report.Assignee != nil && report.AllowOverride != nil {
		permission := "not allowed"
		if *report.AllowOverride {
			permission = "allowed"
		}
		line += fmt.Sprintf(" to %q (reassignment: %s)", *report.Assignee, permission)
	}
	return line
}

// formatUserTaskAssignmentCounts omits zero counters from human summaries only.
func formatUserTaskAssignmentCounts(counts userTaskAssignmentCounts) string {
	parts := []string{}
	for _, entry := range []struct {
		count int
		label string
	}{
		{counts.Preview, "preview"}, {counts.Accepted, "confirmed"}, {counts.Failed, "failed"},
		{counts.TimedOut, "timed out"}, {counts.Canceled, "canceled"},
		{counts.NotAttempted, "not attempted"},
	} {
		if entry.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", entry.count, entry.label))
		}
	}
	return strings.Join(parts, ", ")
}

// formatUserTaskAssignmentItem allows only known status/submission labels into
// verbose results. Error strings and unknown enum text remain machine evidence.
func formatUserTaskAssignmentItem(operation string, item task.UserTaskAssignmentResult) string {
	status := "unknown"
	switch item.Status {
	case task.UserTaskAssignmentStatusPreview, task.UserTaskAssignmentStatusAccepted, task.UserTaskAssignmentStatusFailed, task.UserTaskAssignmentStatusTimedOut, task.UserTaskAssignmentStatusCanceled, task.UserTaskAssignmentStatusNotAttempted:
		status = string(item.Status)
	}
	submission := "unknown"
	switch item.Submission {
	case task.UserTaskAssignmentSubmissionNotSubmitted, task.UserTaskAssignmentSubmissionAccepted, task.UserTaskAssignmentSubmissionRejected, task.UserTaskAssignmentSubmissionUnknown:
		submission = string(item.Submission)
	}
	line := fmt.Sprintf("%s: %s; submission: %s", item.Key, status, submission)
	if item.Status == task.UserTaskAssignmentStatusAccepted && item.Submission == task.UserTaskAssignmentSubmissionAccepted {
		line = fmt.Sprintf("updated user-task %s: %s confirmed", item.Key, assignmentOperationLabel(operation))
	}
	if label := http.StatusText(item.StatusCode); label != "" {
		line += fmt.Sprintf("; HTTP %d %s", item.StatusCode, label)
	}
	return line
}

// formatUserTaskAssignmentFailure returns one bounded diagnostic for the runner
// to emit through its configured logger. No raw item or joined error is formatted.
func formatUserTaskAssignmentFailure(report userTaskAssignmentReport, operationErr error) string {
	parts := []string{}
	counts := map[string]int{}
	reasons := []string{}
	for _, item := range report.Items {
		switch item.Status {
		case task.UserTaskAssignmentStatusAccepted, task.UserTaskAssignmentStatusPreview, task.UserTaskAssignmentStatusNotAttempted:
			continue
		}
		reason := userTaskAssignmentFailureReason(item)
		if counts[reason] == 0 {
			reasons = append(reasons, reason)
		}
		counts[reason]++
	}
	if len(reasons) == 0 {
		return "user-task " + assignmentOperationLabel(report.Operation) + ": " + strings.ReplaceAll(string(ferrors.Classify(operationErr)), "_", " ")
	}
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
	return "user-task " + assignmentOperationLabel(report.Operation) + ": " + strings.Join(parts, "; ")
}

// userTaskAssignmentFailureReason never promotes a last HTTP rejection into
// certainty when an earlier attempt left acceptance unknown.
func userTaskAssignmentFailureReason(item task.UserTaskAssignmentResult) string {
	switch item.Status {
	case task.UserTaskAssignmentStatusTimedOut, task.UserTaskAssignmentStatusCanceled:
		reason := "timed out"
		if item.Status == task.UserTaskAssignmentStatusCanceled {
			reason = "canceled"
		}
		if item.Submission == task.UserTaskAssignmentSubmissionUnknown {
			reason += " (acceptance unknown)"
		}
		return reason
	}
	switch item.Submission {
	case task.UserTaskAssignmentSubmissionUnknown:
		return "acceptance unknown"
	case task.UserTaskAssignmentSubmissionRejected:
		if item.StatusCode >= 400 && item.StatusCode <= 599 {
			if label := http.StatusText(item.StatusCode); label != "" {
				return fmt.Sprintf("%s (HTTP %d)", strings.ToLower(label), item.StatusCode)
			}
		}
		return "request rejected"
	case task.UserTaskAssignmentSubmissionNotSubmitted:
		return "request not submitted"
	default:
		return "request failed"
	}
}

// renderUserTaskAssignmentPlan lists bulk keys before approval. The prompt already
// describes the intent, so only verbose mode repeats the plan and tenant context.
// Machine and quiet modes omit these informational records.
func renderUserTaskAssignmentPlan(cmd *cobra.Command, input userTaskAssignmentInput) {
	if flagQuiet || pickMode() == RenderModeJSON || pickMode() == RenderModeKeysOnly {
		return
	}
	if flagVerbose {
		renderAttachedTenantContext(cmd)
		renderHumanLine(cmd, "plan: %s", formatUserTaskAssignmentIntent(newUserTaskAssignmentPreview(input)))
	}
	if len(input.Keys) > 1 {
		for _, key := range input.Keys {
			renderHumanLine(cmd, "selected user-task %s", key)
		}
	}
}
