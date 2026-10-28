// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// decodeAssignmentEnvelope rejects duplicate or trailing output after a result.
func decodeAssignmentEnvelope(t *testing.T, output string) ResultEnvelope[userTaskAssignmentReport] {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	var envelope ResultEnvelope[userTaskAssignmentReport]
	require.NoError(t, decoder.Decode(&envelope))
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	return envelope
}

// TestUserTaskAssignmentReportPreview proves frozen intent, explicit false/true
// override, omission for unassignment and required zero counters without backend work.
func TestUserTaskAssignmentReportPreview(t *testing.T) {
	for _, selector := range []string{"--assign=  opaque\n\x1bidentity  ", "--unassign"} {
		for _, override := range []bool{false, true} {
			if selector == "--unassign" && override {
				continue
			}
			t.Run(fmt.Sprint(selector, override), func(t *testing.T) {
				argv := []string{selector, "-k", completionKeyB + "," + completionKeyA, "--key", completionKeyB, "--dry-run"}
				if override {
					argv = append(argv, "--allow-override")
				}
				cmd := newAssignmentInputCommand(t, argv...)
				cmd.SetIn(strings.NewReader(completionKeyA + "\n"))
				input, err := parseUpdateUserTaskAssignmentInput(cmd, []string{"-"})
				require.NoError(t, err)
				report := newUserTaskAssignmentPreview(input)
				input.Keys[0], input.Assignee, input.AllowOverride = "changed", "changed", !override
				require.Equal(t, []string{completionKeyB, completionKeyA}, report.RequestedKeys)
				require.Equal(t, 2, report.RequestedCount)
				require.Equal(t, userTaskAssignmentCounts{Preview: 2}, report.Counts)
				for i, item := range report.Items {
					require.Equal(t, report.RequestedKeys[i], item.Key)
					require.Equal(t, task.UserTaskAssignmentStatusPreview, item.Status)
					require.Equal(t, task.UserTaskAssignmentSubmissionNotSubmitted, item.Submission)
					require.Zero(t, item.StatusCode)
					require.Empty(t, item.Error)
				}
				raw, err := json.Marshal(report)
				require.NoError(t, err)
				var fields map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(raw, &fields))
				require.NotContains(t, fields, "variablesProvided")
				require.NotContains(t, fields, "variableCount")
				if selector == "--unassign" {
					require.NotContains(t, fields, "assignee")
					require.NotContains(t, fields, "allowOverride")
				} else {
					require.Equal(t, "  opaque\n\x1bidentity  ", *report.Assignee)
					require.Equal(t, override, *report.AllowOverride)
					require.Equal(t, fmt.Sprint(override), string(fields["allowOverride"]))
				}
				require.JSONEq(t, `{"preview":2,"accepted":0,"failed":0,"timedOut":0,"canceled":0,"notAttempted":0}`, string(fields["counts"]))
				require.JSONEq(t, `[{"key":"2251799813685250","status":"preview","submission":"not_submitted"},{"key":"2251799813685249","status":"preview","submission":"not_submitted"}]`, string(fields["items"]))
			})
		}
	}
	report := newUserTaskAssignmentReport(userTaskAssignmentInput{Operation: "unassign"}, task.UserTaskAssignmentResults{})
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.JSONEq(t, `{"operation":"unassign","phase":"execution","requestedKeys":[],"requestedCount":0,"items":[],"counts":{"preview":0,"accepted":0,"failed":0,"timedOut":0,"canceled":0,"notAttempted":0}}`, string(raw))
}

// TestUserTaskAssignmentReportEvidence copies all enums and raw machine evidence;
// absent service outcomes stay absent rather than being invented by the view.
func TestUserTaskAssignmentReportEvidence(t *testing.T) {
	statuses := []task.UserTaskAssignmentStatus{task.UserTaskAssignmentStatusPreview, task.UserTaskAssignmentStatusAccepted, task.UserTaskAssignmentStatusFailed, task.UserTaskAssignmentStatusTimedOut, task.UserTaskAssignmentStatusCanceled, task.UserTaskAssignmentStatusNotAttempted}
	submissions := []task.UserTaskAssignmentSubmission{task.UserTaskAssignmentSubmissionNotSubmitted, task.UserTaskAssignmentSubmissionAccepted, task.UserTaskAssignmentSubmissionRejected, task.UserTaskAssignmentSubmissionUnknown, task.UserTaskAssignmentSubmissionUnknown, task.UserTaskAssignmentSubmissionNotSubmitted}
	input := userTaskAssignmentInput{Operation: "assign", Keys: make(typex.Keys, len(statuses)), Assignee: "alice"}
	results := task.UserTaskAssignmentResults{Items: make([]task.UserTaskAssignmentResult, len(statuses))}
	for i, status := range statuses {
		key := fmt.Sprintf("22517998136852%02d", 49+i)
		input.Keys[i] = key
		results.Items[i] = task.UserTaskAssignmentResult{Key: key, Status: status, Submission: submissions[i], Error: completionRawError, StatusCode: 403}
	}
	report := newUserTaskAssignmentReport(input, results)
	require.Equal(t, results.Items, report.Items)
	results.Items[0].Key = "changed"
	input.Keys[0] = "changed"
	require.Equal(t, completionKeyA, report.Items[0].Key)
	require.Equal(t, completionKeyA, report.RequestedKeys[0])
	require.Equal(t, userTaskAssignmentCounts{Preview: 1, Accepted: 1, Failed: 1, TimedOut: 1, Canceled: 1, NotAttempted: 1}, report.Counts)
	partial := newUserTaskAssignmentReport(input, task.UserTaskAssignmentResults{Items: results.Items[:1]})
	require.Len(t, partial.Items, 1)
	require.Equal(t, 6, partial.RequestedCount)
	require.Equal(t, userTaskAssignmentCounts{Preview: 1}, partial.Counts)
}

// TestUserTaskAssignmentViewModes verifies machine precedence and one classified
// envelope with partial evidence, safe human records and exact selected keys.
func TestUserTaskAssignmentViewModes(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		input := userTaskAssignmentInput{Operation: operation, Keys: typex.Keys{completionKeyB, completionKeyA}, Assignee: "alice"}
		accepted := task.UserTaskAssignmentResult{Key: completionKeyB, Status: task.UserTaskAssignmentStatusAccepted, Submission: task.UserTaskAssignmentSubmissionAccepted, StatusCode: 204}
		failed := task.UserTaskAssignmentResult{Key: completionKeyA, Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionUnknown, StatusCode: 403, Error: completionRawError}
		operationErr := ferrors.WrapClass(ferrors.ErrConflict, errors.New(completionRawError))
		for _, scenario := range []string{"preview", "success", "partial", "failed"} {
			report := newUserTaskAssignmentPreview(input)
			var resultErr error
			if scenario != "preview" {
				items := []task.UserTaskAssignmentResult{accepted, accepted}
				items[1].Key = completionKeyA
				if scenario == "partial" || scenario == "failed" {
					items[1] = failed
					resultErr = operationErr
				}
				if scenario == "failed" {
					items[0] = failed
					items[0].Key = completionKeyB
				}
				report = newUserTaskAssignmentReport(input, task.UserTaskAssignmentResults{Items: items})
			}
			for _, mode := range []string{"human", "verbose", "json", "keys", "quiet", "quiet-json", "quiet-keys", "json-keys"} {
				t.Run(operation+"/"+scenario+"/"+mode, func(t *testing.T) {
					cmd := newAssignmentInputCommand(t)
					parent := &cobra.Command{Use: "update"}
					parent.AddCommand(cmd)
					flagViewAsJson = strings.Contains(mode, "json")
					flagViewKeysOnly = strings.Contains(mode, "keys")
					flagQuiet = strings.Contains(mode, "quiet")
					flagVerbose = mode == "verbose"
					var out, stderr, diagnostics bytes.Buffer
					cmd.SetOut(&out)
					cmd.SetErr(&stderr)
					cmd.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{Level: "info", Format: "json", Writer: &diagnostics})))
					require.NoError(t, renderUpdateUserTaskAssignment(cmd, report, resultErr))
					require.Empty(t, stderr.String())
					if flagViewAsJson || flagViewKeysOnly || flagQuiet {
						require.Empty(t, diagnostics.String())
					}
					if flagViewAsJson {
						envelope := decodeAssignmentEnvelope(t, out.String())
						expected := OutcomeAccepted
						if scenario == "preview" {
							expected = OutcomeSucceeded
						}
						if resultErr != nil {
							expected = OutcomeFailed
							require.Equal(t, string(ferrors.Classify(resultErr)), envelope.Class)
							require.Contains(t, envelope.Detail.Message, completionRawError)
						}
						require.Equal(t, expected, envelope.Outcome)
						require.Equal(t, "update user-task", envelope.Command)
						require.Equal(t, report, envelope.Payload)
					} else if flagViewKeysOnly {
						want := completionKeyB + "\n" + completionKeyA + "\n"
						if scenario == "partial" {
							want = completionKeyB + "\n"
						}
						if scenario == "failed" {
							want = ""
						}
						require.Equal(t, want, out.String())
					} else {
						require.Empty(t, out.String())
						if !flagQuiet {
							records := bulkCompletionDiagnosticRecords(t, diagnostics.String(), "json")
							require.NotEmpty(t, records)
							for _, record := range records {
								require.Equal(t, "INFO", record[0])
							}
							require.NotContains(t, diagnostics.String(), completionRawError)
							require.NotContains(t, diagnostics.String(), "SECRET-BODY")
							require.NotContains(t, diagnostics.String(), "already completed")
						}
					}
				})
			}
		}
	}
}

// TestUserTaskAssignmentViewHumanRouting checks exact safe identity wording and
// logger-only INFO output, including configured format, level and quiet suppression.
func TestUserTaskAssignmentViewHumanRouting(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, level := range []string{"info", "warn"} {
				for _, quiet := range []bool{false, true} {
					for _, preview := range []bool{false, true} {
						t.Run(fmt.Sprint(operation, format, level, quiet, preview), func(t *testing.T) {
							cmd := newAssignmentInputCommand(t)
							flagQuiet = quiet
							var out, stderr, diagnostics bytes.Buffer
							cmd.SetOut(&out)
							cmd.SetErr(&stderr)
							cmd.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{Level: level, Format: format, Writer: &diagnostics})))
							input := userTaskAssignmentInput{Operation: operation, Keys: typex.Keys{completionKeyA}, Assignee: "  alice\n\x1b\t  ", AllowOverride: true}
							report := newUserTaskAssignmentReport(input, task.UserTaskAssignmentResults{Items: []task.UserTaskAssignmentResult{{Key: completionKeyA, Status: task.UserTaskAssignmentStatusAccepted, Submission: task.UserTaskAssignmentSubmissionAccepted, StatusCode: 204}}})
							want := "updated user-task " + completionKeyA + ": " + assignmentOperationLabel(operation) + " confirmed\n"
							if preview {
								report = newUserTaskAssignmentPreview(input)
								want = "dry run: update user-task " + completionKeyA + ": " + assignmentOperationLabel(operation)
								if operation == "assign" {
									want += ` to "  alice\n\x1b\t  " (reassignment: allowed)`
								}
								want += "; no requests submitted\n"
							}
							require.NoError(t, renderUpdateUserTaskAssignment(cmd, report, nil))
							require.Empty(t, out.String())
							require.Empty(t, stderr.String())
							if quiet || level == "warn" {
								require.Empty(t, diagnostics.String())
							} else {
								require.Equal(t, completionInfoRecords(want), bulkCompletionDiagnosticRecords(t, diagnostics.String(), format))
							}
						})
					}
				}
			}
		}
	}
}

// TestUserTaskAssignmentViewFailureReasons bounds repeated known reasons and
// keeps unknown acceptance visible even with a last rejection or canceled request.
func TestUserTaskAssignmentViewFailureReasons(t *testing.T) {
	items := []task.UserTaskAssignmentResult{
		{Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionRejected, StatusCode: 403},
		{Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionRejected, StatusCode: 403},
		{Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionUnknown, StatusCode: 403},
		{Status: task.UserTaskAssignmentStatusTimedOut, Submission: task.UserTaskAssignmentSubmissionUnknown},
		{Status: task.UserTaskAssignmentStatusCanceled, Submission: task.UserTaskAssignmentSubmissionUnknown},
		{Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionRejected, StatusCode: 409},
		{Status: task.UserTaskAssignmentStatusNotAttempted, Submission: task.UserTaskAssignmentSubmissionNotSubmitted},
	}
	for i := range items {
		items[i].Error = completionRawError
		items[i].Key = completionKeyA
	}
	for _, operation := range []string{"assign", "unassign"} {
		report := newUserTaskAssignmentReport(userTaskAssignmentInput{Operation: operation}, task.UserTaskAssignmentResults{Items: items})
		require.Equal(t, "user-task "+assignmentOperationLabel(operation)+": forbidden (HTTP 403): 2; acceptance unknown: 1; timed out (acceptance unknown): 1; other failures: 2", formatUserTaskAssignmentFailure(report, errors.New(completionRawError)))
	}
	for _, tc := range []struct {
		item task.UserTaskAssignmentResult
		want string
	}{
		{task.UserTaskAssignmentResult{Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionNotSubmitted}, "request not submitted"},
		{task.UserTaskAssignmentResult{Status: task.UserTaskAssignmentStatusFailed, Submission: task.UserTaskAssignmentSubmissionRejected, StatusCode: 499}, "request rejected"},
		{task.UserTaskAssignmentResult{Status: task.UserTaskAssignmentStatusTimedOut, Submission: task.UserTaskAssignmentSubmissionRejected}, "timed out"},
		{task.UserTaskAssignmentResult{Status: task.UserTaskAssignmentStatusCanceled, Submission: task.UserTaskAssignmentSubmissionUnknown}, "canceled (acceptance unknown)"},
		{task.UserTaskAssignmentResult{Status: task.UserTaskAssignmentStatus(completionRawError), Submission: task.UserTaskAssignmentSubmission(completionRawError)}, "request failed"},
	} {
		require.Equal(t, tc.want, userTaskAssignmentFailureReason(tc.item))
	}
	hostile := task.UserTaskAssignmentResult{Key: completionKeyA, Status: task.UserTaskAssignmentStatus(completionRawError), Submission: task.UserTaskAssignmentSubmission(completionRawError), Error: completionRawError, StatusCode: 499}
	require.Equal(t, completionKeyA+": unknown; submission: unknown", formatUserTaskAssignmentItem("assign", hostile))
	require.Equal(t, "user-task assignment: conflict", formatUserTaskAssignmentFailure(userTaskAssignmentReport{Operation: "assign"}, ferrors.WrapClass(ferrors.ErrConflict, errors.New(completionRawError))))
}

// TestUserTaskAssignmentViewWriteFailure returns result-write failures untouched;
// rendering never invokes another envelope or writes a diagnostic directly.
func TestUserTaskAssignmentViewWriteFailure(t *testing.T) {
	for _, mode := range []string{"json", "keys", "human"} {
		cmd := newAssignmentInputCommand(t)
		flagViewAsJson = mode == "json"
		flagViewKeysOnly = mode == "keys"
		cmd.SetOut(completionFailWriter{})
		var stderr, diagnostics bytes.Buffer
		cmd.SetErr(&stderr)
		cmd.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{Level: "info", Writer: &diagnostics})))
		err := renderUpdateUserTaskAssignment(cmd, newUserTaskAssignmentPreview(userTaskAssignmentInput{Operation: "unassign", Keys: typex.Keys{completionKeyA}}), nil)
		if mode == "human" {
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics.String())
		} else {
			require.ErrorIs(t, err, io.ErrClosedPipe)
			require.Empty(t, diagnostics.String())
		}
		require.Empty(t, stderr.String())
	}
}

// TestUserTaskAssignmentViewRequiresLogger prevents the shared human renderer's
// legacy stdout fallback when a future caller forgets command logger setup.
func TestUserTaskAssignmentViewRequiresLogger(t *testing.T) {
	cmd := newAssignmentInputCommand(t)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	report := newUserTaskAssignmentPreview(userTaskAssignmentInput{Operation: "unassign", Keys: typex.Keys{completionKeyA}})
	require.Error(t, renderUpdateUserTaskAssignment(cmd, report, nil))
	require.Empty(t, out.String())
	require.Empty(t, stderr.String())
}

// TestUserTaskAssignmentViewBulkPreview lists each frozen key without verbose
// and quotes control characters while preserving explicit protection wording.
func TestUserTaskAssignmentViewBulkPreview(t *testing.T) {
	cmd := newAssignmentInputCommand(t, "--assign=alice", "--key", completionKeyB+","+completionKeyA, "--dry-run")
	input, err := parseUpdateUserTaskAssignmentInput(cmd, nil)
	require.NoError(t, err)
	var out, stderr, diagnostics bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{Level: "info", Format: "json", Writer: &diagnostics})))
	require.NoError(t, renderUpdateUserTaskAssignment(cmd, newUserTaskAssignmentPreview(input), nil))
	require.Empty(t, out.String())
	require.Empty(t, stderr.String())
	require.Equal(t, completionInfoRecords("dry run: update user-task assignment: 2 requested to \"alice\" (reassignment: not allowed); no requests submitted\nselected user-task "+completionKeyB+"\nselected user-task "+completionKeyA+"\n"), bulkCompletionDiagnosticRecords(t, diagnostics.String(), "json"))
}

// TestUserTaskAssignmentViewValidationEnvelope keeps local failure payload-free
// using the existing error contract, with one document and no result fabrication.
func TestUserTaskAssignmentViewValidationEnvelope(t *testing.T) {
	cmd := newAssignmentInputCommand(t, "--assign=", "--key", completionKeyA, "--json", "--auto-confirm")
	_, err := parseUpdateUserTaskAssignmentInput(cmd, nil)
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, renderResultEnvelope(cmd, resultEnvelopeForError(cmd, err)))
	decoder := json.NewDecoder(&out)
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.Equal(t, "invalid", envelope["outcome"])
	require.NotContains(t, envelope, "payload")
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
}
