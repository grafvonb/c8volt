// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/stretchr/testify/require"
)

// TestUpdateJobCommand_RejectsJSONVerboseBeforeLookupOrMutation verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestUpdateJobCommand_RejectsJSONVerboseBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagVerbose = true
	flagDryRun = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json cannot be combined with --verbose for update job")
}

// TestParseUpdateJobRequestParsesTechnicalFailure verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestParsesTechnicalFailure(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
	require.NoError(t, updateJobCmd.Flags().Set("retries", "2"))
	require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "5m"))
	t.Cleanup(func() {
		require.NoError(t, updateJobCmd.Flags().Set("fail", "false"))
		require.NoError(t, updateJobCmd.Flags().Set("retries", "0"))
		require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", ""))
	})

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobFail = true
	flagUpdateJobRetries = 2
	flagUpdateJobRetryBackoffRaw = "5m"
	flagUpdateJobMessage = "worker unavailable"

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.WorkerOutcome)
	require.Equal(t, job.WorkerOutcomeTechnicalFailure, request.WorkerOutcome.Mode)
	require.Equal(t, int32(2), *request.WorkerOutcome.Retries)
	require.Equal(t, int64(300000), *request.WorkerOutcome.RetryBackoffMillis)
	require.Equal(t, "worker unavailable", request.WorkerOutcome.Message)
	require.Nil(t, request.Retries)
	require.Nil(t, request.TimeoutMillis)
}

// TestParseUpdateJobRequestParsesBPMNError verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestParsesBPMNError(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
	require.NoError(t, updateJobCmd.Flags().Set("vars", `{"approved":false}`))
	t.Cleanup(func() {
		require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", ""))
		require.NoError(t, updateJobCmd.Flags().Set("vars", ""))
	})

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobBPMNError = "PAYMENT_DECLINED"
	flagUpdateJobMessage = "card declined"
	flagUpdateJobVariables = `{"approved":false}`

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.WorkerOutcome)
	require.Equal(t, job.WorkerOutcomeBPMNError, request.WorkerOutcome.Mode)
	require.Equal(t, "PAYMENT_DECLINED", request.WorkerOutcome.ErrorCode)
	require.Equal(t, "card declined", request.WorkerOutcome.Message)
	require.Equal(t, map[string]any{"approved": false}, request.WorkerOutcome.Variables)
	require.Nil(t, request.Retries)
	require.Nil(t, request.TimeoutMillis)
}

// TestParseUpdateJobRequestParsesCompletion verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestParsesCompletion(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
	require.NoError(t, updateJobCmd.Flags().Set("vars", `{"approved":true}`))
	t.Cleanup(func() {
		require.NoError(t, updateJobCmd.Flags().Set("complete", "false"))
		require.NoError(t, updateJobCmd.Flags().Set("vars", ""))
	})

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobComplete = true
	flagUpdateJobVariables = `{"approved":true}`

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.WorkerOutcome)
	require.Equal(t, job.WorkerOutcomeCompletion, request.WorkerOutcome.Mode)
	require.Equal(t, map[string]any{"approved": true}, request.WorkerOutcome.Variables)
	require.Nil(t, request.Retries)
	require.Nil(t, request.TimeoutMillis)
}

// TestParseUpdateJobRequestRejectsTechnicalFailureValidationErrors verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestRejectsTechnicalFailureValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		set     func(t *testing.T)
		message string
	}{
		{
			name: "missing retries",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
			},
			message: "technical job failure requires --retries",
		},
		{
			name: "negative retries",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "-1"))
				flagUpdateJobRetries = -1
			},
			message: "invalid value for --retries",
		},
		{
			name: "invalid retry backoff",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "0s"))
				flagUpdateJobRetries = 1
				flagUpdateJobRetryBackoffRaw = "0s"
			},
			message: "invalid value for --retry-backoff",
		},
		{
			name: "timeout conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
				flagUpdateJobRetries = 1
			},
			message: "--fail cannot be combined with --timeout",
		},
		{
			name: "bpmn conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				flagUpdateJobRetries = 1
			},
			message: "--throw-bpmn-error cannot be combined with --fail",
		},
		{
			name: "complete conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				flagUpdateJobRetries = 1
			},
			message: "--fail cannot be combined with --complete",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			resetCommandTreeFlags(Root())
			flagUpdateJobKey = "2251799813711967"
			tt.set(t)

			_, err := parseUpdateJobRequest(updateJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.message)
		})
	}
}

// TestParseUpdateJobRequestRejectsBPMNErrorValidationErrors verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestRejectsBPMNErrorValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		set     func(t *testing.T)
		message string
	}{
		{
			name: "empty code",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", ""))
			},
			message: "BPMN error requires a non-empty --throw-bpmn-error",
		},
		{
			name: "fail conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
			},
			message: "--throw-bpmn-error cannot be combined with --fail",
		},
		{
			name: "complete conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
			},
			message: "--throw-bpmn-error cannot be combined with --complete",
		},
		{
			name: "retries conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
			},
			message: "--throw-bpmn-error cannot be combined with --retries",
		},
		{
			name: "timeout conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
			},
			message: "--throw-bpmn-error cannot be combined with --timeout",
		},
		{
			name: "retry backoff conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "5m"))
			},
			message: "--throw-bpmn-error cannot be combined with --retry-backoff",
		},
		{
			name: "invalid variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", "{"))
				flagUpdateJobVariables = "{"
			},
			message: "--vars must be a valid JSON object",
		},
		{
			name: "non-object variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", `["bad"]`))
				flagUpdateJobVariables = `["bad"]`
			},
			message: "--vars must be a JSON object",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			resetCommandTreeFlags(Root())
			flagUpdateJobKey = "2251799813711967"
			flagUpdateJobBPMNError = "PAYMENT_DECLINED"
			tt.set(t)

			_, err := parseUpdateJobRequest(updateJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.message)
		})
	}
}

// TestParseUpdateJobRequestRejectsCompletionValidationErrors verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestRejectsCompletionValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		set     func(t *testing.T)
		message string
	}{
		{
			name: "fail conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
			},
			message: "--fail cannot be combined with --complete",
		},
		{
			name: "bpmn conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
			},
			message: "--throw-bpmn-error cannot be combined with --complete",
		},
		{
			name: "retries conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
			},
			message: "--complete cannot be combined with --retries",
		},
		{
			name: "timeout conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
			},
			message: "--complete cannot be combined with --timeout",
		},
		{
			name: "retry backoff conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "5m"))
			},
			message: "--complete cannot be combined with --retry-backoff",
		},
		{
			name: "invalid variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", "{"))
				flagUpdateJobVariables = "{"
			},
			message: "--vars must be a valid JSON object",
		},
		{
			name: "non-object variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", `["bad"]`))
				flagUpdateJobVariables = `["bad"]`
			},
			message: "--vars must be a JSON object",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			resetCommandTreeFlags(Root())
			flagUpdateJobKey = "2251799813711967"
			tt.set(t)

			_, err := parseUpdateJobRequest(updateJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.message)
		})
	}
}

// TestParseUpdateJobRequestParsesTimeoutMillis verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestParsesTimeoutMillis(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
	t.Cleanup(func() { require.NoError(t, updateJobCmd.Flags().Set("timeout", "")) })

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobTimeoutRaw = "5m"

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.TimeoutMillis)
	require.Equal(t, int64(300000), *request.TimeoutMillis)
	require.False(t, request.ConfirmRetries)
}

// TestParseUpdateJobRequestPreservesRetryUpdateMode protects the legacy retry path while worker outcome flags are added.
func TestParseUpdateJobRequestPreservesRetryUpdateMode(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("retries", "3"))
	t.Cleanup(func() { require.NoError(t, updateJobCmd.Flags().Set("retries", "0")) })

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobRetries = 3
	flagNoWait = false

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.Equal(t, "2251799813711967", request.Key)
	require.NotNil(t, request.Retries)
	require.Equal(t, int32(3), *request.Retries)
	require.True(t, request.ConfirmRetries)
	require.Nil(t, request.TimeoutMillis)
}

// TestParseUpdateJobRequestPreservesTimeoutUpdateMode protects timeout conversion without enabling retry confirmation.
func TestParseUpdateJobRequestPreservesTimeoutUpdateMode(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
	t.Cleanup(func() { require.NoError(t, updateJobCmd.Flags().Set("timeout", "")) })

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobTimeoutRaw = "5m"

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.Equal(t, "2251799813711967", request.Key)
	require.Nil(t, request.Retries)
	require.NotNil(t, request.TimeoutMillis)
	require.Equal(t, int64(300000), *request.TimeoutMillis)
	require.False(t, request.ConfirmRetries)
}

// TestUpdateJobCommand_RejectsJSONMutationWithoutAutoConfirmOrAutomationBeforeLookupOrMutation verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestUpdateJobCommand_RejectsJSONMutationWithoutAutoConfirmOrAutomationBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json update job requires --dry-run, --auto-confirm, or --automation")
}

// TestUpdateJobCommand_RejectsJSONNoWaitWithoutAutoConfirmOrAutomationBeforeLookupOrMutation verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestUpdateJobCommand_RejectsJSONNoWaitWithoutAutoConfirmOrAutomationBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagNoWait = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json update job requires --dry-run, --auto-confirm, or --automation")
}

// TestUpdateJobCommand_AllowsJSONDryRunWithoutAutoConfirm verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestUpdateJobCommand_AllowsJSONDryRunWithoutAutoConfirm(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagDryRun = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.NoError(t, err)
}

// TestParseUpdateJobRequestRequiresUpdateFlag verifies the update job request parsing and guardrail behavior covered by this scenario.
func TestParseUpdateJobRequestRequiresUpdateFlag(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())

	flagUpdateJobKey = "2251799813711967"

	_, err := parseUpdateJobRequest(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "update job requires --retries, --timeout, or both")
}

// TestUpdateJobBulkCompletionRequest validates selector routing without sharing get-job flag storage.
func TestUpdateJobBulkCompletionRequest(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		valid bool
		batch bool
	}{
		{"all predicates", []string{"--complete", "--state", "created", "--type", "cleanup", "--pi-key", "2251799813711000", "--element-instance-key", "2251799813711001", "--element-id", "task", "--worker", "worker", "--retries", "0", "--kind", "task_listener", "--listener-event-type", "canceling", "--limit", "0"}, true, true},
		{"retries alone", []string{"--complete", "--retries", "0"}, true, true},
		{"scalar", []string{"--complete", "--key", "123"}, true, false},
		{"unqualified", []string{"--complete"}, false, false},
		{"controls only", []string{"--complete", "--batch-size", "10", "--workers", "1"}, false, false},
		{"blank predicate", []string{"--complete", "--type", " "}, false, false},
		{"key plus selector", []string{"--complete", "--key", "123", "--state", "created"}, false, false},
		{"key plus retries", []string{"--complete", "--key", "123", "--retries", "0"}, false, false},
		{"key plus controls", []string{"--complete", "--key", "123", "--limit", "1"}, false, false},
		{"other mode", []string{"--fail", "--key", "123", "--retries", "0", "--state", "created"}, false, false},
		{"timeout mode", []string{"--timeout", "1m", "--type", "cleanup"}, false, false},
		{"disabled completion", []string{"--complete=false", "--type", "cleanup"}, false, false},
		{"negative retries", []string{"--complete", "--retries", "-1"}, false, false},
		{"negative limit", []string{"--complete", "--type", "cleanup", "--limit", "-1"}, false, false},
		{"invalid state", []string{"--complete", "--state", "bogus"}, false, false},
		{"invalid kind", []string{"--complete", "--kind", "bogus"}, false, false},
		{"invalid event", []string{"--complete", "--listener-event-type", "bogus"}, false, false},
		{"invalid key predicate", []string{"--complete", "--pi-key", "-1"}, false, false},
		{"invalid workers", []string{"--complete", "--type", "cleanup", "--workers", "0"}, false, false},
		{"invalid size", []string{"--complete", "--type", "cleanup", "--batch-size", "0"}, false, false},
		{"completion timeout", []string{"--complete", "--type", "cleanup", "--timeout", "1m"}, false, false},
		{"completion message", []string{"--complete", "--type", "cleanup", "--message", "oops"}, false, false},
		{"completion failure", []string{"--complete", "--type", "cleanup", "--fail"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetCommandTreeFlags(Root())
			resetUpdateJobFlagState()
			t.Cleanup(func() { resetCommandTreeFlags(Root()); resetUpdateJobFlagState() })
			require.NoError(t, updateJobCmd.ParseFlags(tc.args))
			_, batch, err := parseUpdateJobCommandRequest(updateJobCmd)
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.batch, batch != nil)
			if tc.batch {
				require.Equal(t, job.CompletionSelectionSearch, batch.Selection.Mode)
				require.NotNil(t, batch.Selection.Filter.Retries)
				require.Zero(t, *batch.Selection.Filter.Retries)
			}
			if tc.name == "all predicates" {
				require.Equal(t, "CREATED", batch.Selection.Filter.State)
				require.Equal(t, "TASK_LISTENER", batch.Selection.Filter.Kind)
				require.Equal(t, "CANCELING", batch.Selection.Filter.ListenerEventType)
				require.Equal(t, "2251799813711000", batch.Selection.Filter.ProcessInstanceKey)
			}
		})
	}
}
