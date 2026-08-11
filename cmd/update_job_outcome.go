// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
)

// executeUpdateJobWorkerOutcome submits a worker outcome mutation and renders the accepted result.
func executeUpdateJobWorkerOutcome(cmd *cobra.Command, cli c8volt.API, request job.UpdateRequest, plan job.UpdatePlan) error {
	request.WorkerOutcome.OutcomePlan = &plan
	result, err := cli.SubmitJobWorkerOutcome(cmd.Context(), *request.WorkerOutcome, collectOptions()...)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	if err := jobWorkerOutcomeResultView(cmd, result); err != nil {
		return fmt.Errorf("render job update result: %w", err)
	}
	return nil
}

// parseUpdateJobBPMNErrorRequest validates BPMN error flags and builds the worker outcome request.
func parseUpdateJobBPMNErrorRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if cmd.Flags().Changed("fail") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --fail")
	}
	if cmd.Flags().Changed("complete") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --complete")
	}
	if cmd.Flags().Changed("retries") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --retries")
	}
	if cmd.Flags().Changed("timeout") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --timeout")
	}
	if cmd.Flags().Changed("retry-backoff") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --retry-backoff")
	}
	errorCode := strings.TrimSpace(flagUpdateJobBPMNError)
	if errorCode == "" {
		return job.UpdateRequest{}, invalidFlagValuef("BPMN error requires a non-empty --throw-bpmn-error")
	}
	var variables map[string]any
	if cmd.Flags().Changed("vars") {
		parsed, err := parseUpdateJobVariables(flagUpdateJobVariables)
		if err != nil {
			return job.UpdateRequest{}, err
		}
		variables = parsed
	}
	outcome := job.WorkerOutcomeRequest{
		Key:         flagUpdateJobKey,
		Mode:        job.WorkerOutcomeBPMNError,
		Message:     flagUpdateJobMessage,
		Variables:   variables,
		ErrorCode:   errorCode,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	return job.UpdateRequest{
		Key:           flagUpdateJobKey,
		NoWait:        flagNoWait,
		AutoConfirm:   flagCmdAutoConfirm,
		Automation:    updateJobAutomationEnabled(cmd),
		DryRun:        flagDryRun,
		WorkerOutcome: &outcome,
	}, nil
}

// parseUpdateJobCompletionRequest validates completion flags and builds the worker outcome request.
func parseUpdateJobCompletionRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if cmd.Flags().Changed("fail") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --fail")
	}
	if cmd.Flags().Changed("throw-bpmn-error") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --throw-bpmn-error")
	}
	if cmd.Flags().Changed("retries") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --retries")
	}
	if cmd.Flags().Changed("timeout") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --timeout")
	}
	if cmd.Flags().Changed("retry-backoff") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --retry-backoff")
	}
	if cmd.Flags().Changed("message") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --message")
	}
	var variables map[string]any
	if cmd.Flags().Changed("vars") {
		parsed, err := parseUpdateJobVariables(flagUpdateJobVariables)
		if err != nil {
			return job.UpdateRequest{}, err
		}
		variables = parsed
	}
	outcome := job.WorkerOutcomeRequest{
		Key:         flagUpdateJobKey,
		Mode:        job.WorkerOutcomeCompletion,
		Variables:   variables,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	return job.UpdateRequest{
		Key:           flagUpdateJobKey,
		NoWait:        flagNoWait,
		AutoConfirm:   flagCmdAutoConfirm,
		Automation:    updateJobAutomationEnabled(cmd),
		DryRun:        flagDryRun,
		WorkerOutcome: &outcome,
	}, nil
}

// parseUpdateJobTechnicalFailureRequest validates technical-failure flags and builds the worker outcome request.
func parseUpdateJobTechnicalFailureRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if cmd.Flags().Changed("throw-bpmn-error") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--fail cannot be combined with --throw-bpmn-error")
	}
	if cmd.Flags().Changed("complete") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--fail cannot be combined with --complete")
	}
	if cmd.Flags().Changed("timeout") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--fail cannot be combined with --timeout")
	}
	if cmd.Flags().Changed("vars") {
		return job.UpdateRequest{}, invalidFlagValuef("--vars is reserved for BPMN error and completion implementations")
	}
	if !cmd.Flags().Changed("retries") {
		return job.UpdateRequest{}, invalidFlagValuef("technical job failure requires --retries")
	}
	if flagUpdateJobRetries < 0 {
		return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retries: %d, expected non-negative integer", flagUpdateJobRetries)
	}
	retries := flagUpdateJobRetries
	outcome := job.WorkerOutcomeRequest{
		Key:             flagUpdateJobKey,
		Mode:            job.WorkerOutcomeTechnicalFailure,
		Message:         flagUpdateJobMessage,
		Retries:         &retries,
		RetryBackoffRaw: flagUpdateJobRetryBackoffRaw,
		NoWait:          flagNoWait,
		AutoConfirm:     flagCmdAutoConfirm,
		Automation:      updateJobAutomationEnabled(cmd),
		DryRun:          flagDryRun,
	}
	if cmd.Flags().Changed("retry-backoff") {
		retryBackoff, err := time.ParseDuration(flagUpdateJobRetryBackoffRaw)
		if err != nil || retryBackoff <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retry-backoff: %q, expected positive duration such as 60s, 5m, or 1h", flagUpdateJobRetryBackoffRaw)
		}
		retryBackoffMillis := retryBackoff.Milliseconds()
		if retryBackoffMillis <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retry-backoff: %q, duration must be at least 1ms", flagUpdateJobRetryBackoffRaw)
		}
		outcome.RetryBackoff = &retryBackoff
		outcome.RetryBackoffMillis = &retryBackoffMillis
	}
	return job.UpdateRequest{
		Key:           flagUpdateJobKey,
		NoWait:        flagNoWait,
		AutoConfirm:   flagCmdAutoConfirm,
		Automation:    updateJobAutomationEnabled(cmd),
		DryRun:        flagDryRun,
		WorkerOutcome: &outcome,
	}, nil
}

// parseUpdateJobVariables accepts only JSON object variables for worker outcome payloads.
func parseUpdateJobVariables(raw string) (map[string]any, error) {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, invalidFlagValuef("--vars must be a valid JSON object: %v", err)
	}
	variables, ok := decoded.(map[string]any)
	if !ok || variables == nil {
		return nil, invalidFlagValuef("--vars must be a JSON object")
	}
	return variables, nil
}

// changedUpdateJobWorkerOutcomeFlags lists outcome-only flags that are invalid for retry or timeout updates.
func changedUpdateJobWorkerOutcomeFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	names := []string{
		"fail",
		"retry-backoff",
		"message",
		"throw-bpmn-error",
		"complete",
		"vars",
	}
	changed := make([]string, 0, len(names))
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			changed = append(changed, "--"+name)
		}
	}
	return changed
}
