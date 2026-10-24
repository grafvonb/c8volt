// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

const completionRawError = "POST https://distinctive.invalid/v2/user-tasks/secret-key body=SECRET-BODY"

// decodeCompletionEnvelope requires exactly one document, including on failures.
func decodeCompletionEnvelope(t *testing.T, output string) ResultEnvelope[userTaskCompletionReport] {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	var envelope ResultEnvelope[userTaskCompletionReport]
	require.NoError(t, decoder.Decode(&envelope))
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	return envelope
}

// TestUpdateUserTaskCompletionViewParsedPreview connects real local parsing to
// every preview surface; no client or backend dependency is constructed.
func TestUpdateUserTaskCompletionViewParsedPreview(t *testing.T) {
	for _, source := range []string{"omitted", "{}", `{"private":"SECRET-VALUE"}`} {
		for _, mode := range []string{"human", "verbose", "json", "keys", "quiet", "quiet-json", "quiet-keys", "json-keys"} {
			t.Run(source+"/"+mode, func(t *testing.T) {
				argv := []string{"-k", completionKeyB + "," + completionKeyA, "--dry-run"}
				if source != "omitted" {
					argv = append(argv, "--vars", source)
				}
				cmd := newCompletionInputCommand(t, argv...)
				cmd.SetIn(strings.NewReader(completionKeyA + "\n"))
				request, err := parseUpdateUserTaskCompletionInput(cmd, []string{"-"})
				require.NoError(t, err)
				report := newUserTaskCompletionPreview(request)
				request.Keys[0] = "changed"
				request.Variables = map[string]any{"changed": true}
				require.Equal(t, []string{completionKeyB, completionKeyA}, report.RequestedKeys)
				require.Equal(t, 2, report.RequestedCount)
				require.Equal(t, source != "omitted", report.VariablesProvided)
				wantVariables := 0
				if strings.Contains(source, "private") {
					wantVariables = 1
				}
				require.Equal(t, wantVariables, report.VariableCount)
				require.Equal(t, userTaskCompletionCounts{Preview: 2}, report.Counts)
				for i, item := range report.Items {
					require.Equal(t, report.RequestedKeys[i], item.Key)
					require.Equal(t, task.UserTaskCompletionStatusPreview, item.Status)
					require.Equal(t, task.UserTaskCompletionSubmissionNotSubmitted, item.Submission)
				}
				flagViewAsJson = strings.Contains(mode, "json")
				flagViewKeysOnly = strings.Contains(mode, "keys")
				flagQuiet = strings.Contains(mode, "quiet")
				flagVerbose = mode == "verbose"
				var out, diagnostics bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&diagnostics)
				require.NoError(t, renderUpdateUserTaskCompletion(cmd, report, nil))
				require.Empty(t, diagnostics.String())
				require.NotContains(t, out.String(), "SECRET-VALUE")
				if flagViewAsJson {
					envelope := decodeCompletionEnvelope(t, out.String())
					require.Equal(t, OutcomeSucceeded, envelope.Outcome)
					require.Equal(t, report, envelope.Payload)
				} else if flagViewKeysOnly {
					require.Equal(t, completionKeyB+"\n"+completionKeyA+"\n", out.String())
				} else if flagQuiet {
					require.Empty(t, out.String())
				} else {
					want := fmt.Sprintf("dry run: complete user-task: 2 user task(s), %d supplied variable(s); no requests submitted\n", wantVariables)
					if flagVerbose {
						want += completionKeyB + ": preview; submission: not_submitted\n" + completionKeyA + ": preview; submission: not_submitted\n"
					}
					require.Equal(t, want, out.String())
					require.NotContains(t, out.String(), "accepted")
				}
			})
		}
	}
}

// TestUpdateUserTaskCompletionViewExecution preserves all typed outcomes,
// acceptance-only envelopes and raw machine evidence without human disclosure.
func TestUpdateUserTaskCompletionViewExecution(t *testing.T) {
	statuses := []task.UserTaskCompletionStatus{task.UserTaskCompletionStatusAccepted, task.UserTaskCompletionStatusFailed, task.UserTaskCompletionStatusTimedOut, task.UserTaskCompletionStatusCanceled, task.UserTaskCompletionStatusNotAttempted}
	submissions := []task.UserTaskCompletionSubmission{task.UserTaskCompletionSubmissionAccepted, task.UserTaskCompletionSubmissionRejected, task.UserTaskCompletionSubmissionUnknown, task.UserTaskCompletionSubmissionUnknown, task.UserTaskCompletionSubmissionNotSubmitted}
	input := userTaskCompletionInput{Keys: make(typex.Keys, len(statuses))}
	results := task.UserTaskCompletionResults{Items: make([]task.UserTaskCompletionResult, len(statuses))}
	for i, status := range statuses {
		key := fmt.Sprintf("22517998136852%02d", 49+i)
		input.Keys[i] = key
		results.Items[i] = task.UserTaskCompletionResult{Key: key, Status: status, Submission: submissions[i], Error: completionRawError}
	}
	results.Items[0].Error = ""
	results.Items[0].StatusCode = 204
	results.Items[1].StatusCode = 403
	report := newUserTaskCompletionReport(input, results)
	results.Items[0].Key = "changed"
	require.Equal(t, completionKeyA, report.Items[0].Key)
	require.Equal(t, userTaskCompletionCounts{Accepted: 1, Failed: 1, TimedOut: 1, Canceled: 1, NotAttempted: 1}, report.Counts)
	require.Equal(t, 5, report.RequestedCount)
	operationErr := ferrors.WrapClass(ferrors.ErrConflict, errors.New(completionRawError))
	for _, mode := range []string{"human", "verbose", "json", "keys", "quiet", "quiet-json", "quiet-keys", "json-keys"} {
		t.Run(mode, func(t *testing.T) {
			cmd := newCompletionInputCommand(t)
			flagViewAsJson = strings.Contains(mode, "json")
			flagViewKeysOnly = strings.Contains(mode, "keys")
			flagQuiet = strings.Contains(mode, "quiet")
			flagVerbose = mode == "verbose"
			var out, diagnostics bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&diagnostics)
			require.NoError(t, renderUpdateUserTaskCompletion(cmd, report, operationErr))
			require.Empty(t, diagnostics.String(), "the runner owns logger emission")
			if flagViewAsJson {
				envelope := decodeCompletionEnvelope(t, out.String())
				require.Equal(t, OutcomeFailed, envelope.Outcome)
				require.Equal(t, string(ferrors.Classify(operationErr)), envelope.Class)
				require.Contains(t, envelope.Detail.Message, completionRawError)
				require.Equal(t, report, envelope.Payload)
			} else {
				require.NotContains(t, out.String(), completionRawError)
				require.NotContains(t, out.String(), "SECRET-BODY")
				if flagViewKeysOnly {
					require.Equal(t, completionKeyA+"\n", out.String())
				} else if flagQuiet {
					require.Empty(t, out.String())
				} else {
					want := "complete user-task: 5 requested; 1 accepted, 1 failed, 1 timed out, 1 canceled, 1 not attempted; backend confirmed completion of accepted tasks; no additional state polling\n"
					if flagVerbose {
						want += completionKeyA + ": accepted; submission: accepted; HTTP 204 No Content\n"
						want += completionKeyB + ": failed; submission: rejected; HTTP 403 Forbidden\n"
						want += "2251799813685251: timed_out; submission: unknown\n2251799813685252: canceled; submission: unknown\n2251799813685253: not_attempted; submission: not_submitted\n"
					}
					require.Equal(t, want, out.String())
				}
			}
		})
	}
}

// TestUpdateUserTaskCompletionViewAcceptedAndEmptyKeys protects explicit accepted
// rendering without no-wait, and zero-byte output when no submission was accepted.
func TestUpdateUserTaskCompletionViewAcceptedAndEmptyKeys(t *testing.T) {
	cmd := newCompletionInputCommand(t, "--json", "--auto-confirm")
	input := userTaskCompletionInput{Keys: typex.Keys{completionKeyA}}
	accepted := task.UserTaskCompletionResults{Items: []task.UserTaskCompletionResult{{Key: completionKeyA, Status: task.UserTaskCompletionStatusAccepted, Submission: task.UserTaskCompletionSubmissionAccepted}}}
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, renderUpdateUserTaskCompletion(cmd, newUserTaskCompletionReport(input, accepted), nil))
	require.Equal(t, OutcomeAccepted, decodeCompletionEnvelope(t, out.String()).Outcome)
	require.False(t, flagNoWait)
	accepted.Items[0] = task.UserTaskCompletionResult{Key: completionKeyA, Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionUnknown, Error: completionRawError}
	flagViewAsJson = false
	flagViewKeysOnly = true
	out.Reset()
	require.NoError(t, renderUpdateUserTaskCompletion(cmd, newUserTaskCompletionReport(input, accepted), errors.New(completionRawError)))
	require.Zero(t, out.Len())
}

// TestUpdateUserTaskCompletionViewFailureReasons bounds repeated causes using
// only allowlisted facts, even when backend strings or enum values are hostile.
func TestUpdateUserTaskCompletionViewFailureReasons(t *testing.T) {
	items := []task.UserTaskCompletionResult{
		{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionRejected, StatusCode: 403},
		{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionRejected, StatusCode: 403},
		{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionUnknown, StatusCode: 403},
		{Status: task.UserTaskCompletionStatusTimedOut, Submission: task.UserTaskCompletionSubmissionUnknown},
		{Status: task.UserTaskCompletionStatusCanceled, Submission: task.UserTaskCompletionSubmissionUnknown},
		{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionRejected, StatusCode: 409},
		{Status: task.UserTaskCompletionStatusNotAttempted, Submission: task.UserTaskCompletionSubmissionNotSubmitted},
	}
	input := userTaskCompletionInput{Keys: make(typex.Keys, len(items))}
	for i := range items {
		items[i].Key = completionKeyA
		items[i].Error = completionRawError
		input.Keys[i] = completionKeyA
	}
	report := newUserTaskCompletionReport(input, task.UserTaskCompletionResults{Items: items})
	require.Equal(t, "user-task completion: 0 accepted, 4 failed, 1 timed out, 1 canceled, 1 not attempted; forbidden (HTTP 403): 2; acceptance unknown: 1; timed out: 1; other failures: 2", formatUserTaskCompletionFailure(report, errors.New(completionRawError)))
	for _, tc := range []struct {
		item task.UserTaskCompletionResult
		want string
	}{
		{task.UserTaskCompletionResult{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionRejected, StatusCode: 499}, "completion rejected"},
		{task.UserTaskCompletionResult{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionRejected, StatusCode: 503}, "service unavailable (HTTP 503)"},
		{task.UserTaskCompletionResult{Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionUnknown, StatusCode: 504}, "acceptance unknown"},
		{task.UserTaskCompletionResult{Status: task.UserTaskCompletionStatusCanceled}, "canceled"},
		{task.UserTaskCompletionResult{Status: task.UserTaskCompletionStatus(completionRawError), Submission: task.UserTaskCompletionSubmission(completionRawError)}, "completion failed"},
	} {
		require.Equal(t, tc.want, userTaskCompletionFailureReason(tc.item))
	}
	require.NotContains(t, formatUserTaskCompletionFailure(userTaskCompletionReport{}, errors.New(completionRawError)), completionRawError)
}

// completionFailWriter makes destination loss observable to the future runner.
type completionFailWriter struct{}

// Write fails every result write without writing a second diagnostic surface.
func (completionFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestUpdateUserTaskCompletionViewWriteFailure requires all render modes to
// return destination errors instead of claiming a delivered result.
func TestUpdateUserTaskCompletionViewWriteFailure(t *testing.T) {
	for _, mode := range []string{"human", "keys", "json"} {
		t.Run(mode, func(t *testing.T) {
			cmd := newCompletionInputCommand(t)
			cmd.SetOut(completionFailWriter{})
			flagViewAsJson = mode == "json"
			flagViewKeysOnly = mode == "keys"
			report := newUserTaskCompletionPreview(userTaskCompletionInput{Keys: typex.Keys{completionKeyA}})
			require.ErrorIs(t, renderUpdateUserTaskCompletion(cmd, report, nil), io.ErrClosedPipe)
		})
	}
}

// TestUpdateUserTaskCompletionViewSchema preserves initialized arrays, all zero
// count fields, and optional evidence omission without changing variable schemas.
func TestUpdateUserTaskCompletionViewSchema(t *testing.T) {
	report := newUserTaskCompletionReport(userTaskCompletionInput{}, task.UserTaskCompletionResults{})
	encoded, err := json.Marshal(report)
	require.NoError(t, err)
	require.JSONEq(t, `{"operation":"complete","phase":"execution","requestedKeys":[],"requestedCount":0,"variablesProvided":false,"variableCount":0,"items":[],"counts":{"preview":0,"accepted":0,"failed":0,"timedOut":0,"canceled":0,"notAttempted":0}}`, string(encoded))
	preview := newUserTaskCompletionPreview(userTaskCompletionInput{Keys: typex.Keys{completionKeyA}, Variables: map[string]any{}})
	encoded, err = json.Marshal(preview.Items)
	require.NoError(t, err)
	require.JSONEq(t, `[{"key":"2251799813685249","status":"preview","submission":"not_submitted"}]`, string(encoded))
	require.True(t, preview.VariablesProvided)
	require.Zero(t, preview.VariableCount)
}

// TestUpdateUserTaskCompletionViewNoAcceptance keeps all-failure human results
// truthful and excludes hostile enum/error strings even from verbose facts.
func TestUpdateUserTaskCompletionViewNoAcceptance(t *testing.T) {
	cmd := newCompletionInputCommand(t)
	var out, diagnostics bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostics)
	report := newUserTaskCompletionReport(userTaskCompletionInput{Keys: typex.Keys{completionKeyA}}, task.UserTaskCompletionResults{Items: []task.UserTaskCompletionResult{{Key: completionKeyA, Status: task.UserTaskCompletionStatusFailed, Submission: task.UserTaskCompletionSubmissionUnknown, Error: completionRawError}}})
	require.NoError(t, renderUpdateUserTaskCompletion(cmd, report, errors.New(completionRawError)))
	require.Equal(t, "complete user-task: 1 requested; 0 accepted, 1 failed, 0 timed out, 0 canceled, 0 not attempted; no successful completion response received\n", out.String())
	require.Empty(t, diagnostics.String())
	require.Equal(t, completionKeyA+": unknown; submission: unknown", formatUserTaskCompletionItem(task.UserTaskCompletionResult{Key: completionKeyA, Status: task.UserTaskCompletionStatus(completionRawError), Submission: task.UserTaskCompletionSubmission(completionRawError), Error: completionRawError, StatusCode: 499}))
}
