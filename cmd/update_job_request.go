// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
)

// parseUpdateJobRequest maps update job flags into the facade request while preserving local validation order.
func parseUpdateJobRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if strings.TrimSpace(flagUpdateJobKey) == "" {
		return job.UpdateRequest{}, invalidFlagValuef("job update requires a non-empty --key")
	}
	if cmd.Flags().Changed("throw-bpmn-error") {
		return parseUpdateJobBPMNErrorRequest(cmd)
	}
	if cmd.Flags().Changed("fail") {
		return parseUpdateJobTechnicalFailureRequest(cmd)
	}
	if cmd.Flags().Changed("complete") {
		return parseUpdateJobCompletionRequest(cmd)
	}
	if workerFlags := changedUpdateJobWorkerOutcomeFlags(cmd); len(workerFlags) > 0 {
		return job.UpdateRequest{}, invalidFlagValuef("job worker outcome flags are reserved for the BPMN error and completion implementations: %s", strings.Join(workerFlags, ", "))
	}
	retriesChanged := cmd.Flags().Changed("retries")
	timeoutChanged := cmd.Flags().Changed("timeout")
	if !retriesChanged && !timeoutChanged {
		return job.UpdateRequest{}, invalidFlagValuef("update job requires --retries, --timeout, or both")
	}
	request := job.UpdateRequest{
		Key:         flagUpdateJobKey,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	if retriesChanged {
		if flagUpdateJobRetries < 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retries: %d, expected non-negative integer", flagUpdateJobRetries)
		}
		retries := flagUpdateJobRetries
		request.Retries = &retries
		request.ConfirmRetries = !flagNoWait
	}
	if timeoutChanged {
		timeout, err := time.ParseDuration(flagUpdateJobTimeoutRaw)
		if err != nil || timeout <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --timeout: %q, expected positive duration such as 60s, 5m, or 1h", flagUpdateJobTimeoutRaw)
		}
		timeoutMillis := timeout.Milliseconds()
		if timeoutMillis <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --timeout: %q, duration must be at least 1ms", flagUpdateJobTimeoutRaw)
		}
		request.Timeout = &timeout
		request.TimeoutRaw = flagUpdateJobTimeoutRaw
		request.TimeoutMillis = &timeoutMillis
	}
	return request, nil
}

// validateUpdateJobJSONGuardrails keeps machine output from prompting or mixing with verbose diagnostics.
func validateUpdateJobJSONGuardrails(cmd *cobra.Command) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update job")
	}
	if flagDryRun || pickMode() != RenderModeJSON || flagCmdAutoConfirm || flagCmdAutomation || updateJobAutomationEnabled(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json update job requires --dry-run, --auto-confirm, or --automation")
}

// updateJobAutomationEnabled reads command-context automation state without forcing callers to own nil checks.
func updateJobAutomationEnabled(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	return automationModeEnabled(cmd)
}
