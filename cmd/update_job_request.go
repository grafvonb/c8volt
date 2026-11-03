// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	retriesChanged := cmd.Flags().Changed("set-retries")
	timeoutChanged := cmd.Flags().Changed("set-timeout")
	if !retriesChanged && !timeoutChanged {
		return job.UpdateRequest{}, invalidFlagValuef("update job requires --set-retries, --set-timeout, or both")
	}
	request := job.UpdateRequest{
		Key:         flagUpdateJobKey,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	if retriesChanged {
		if flagUpdateJobSetRetries < 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --set-retries: %d, expected non-negative integer", flagUpdateJobSetRetries)
		}
		retries := flagUpdateJobSetRetries
		request.Retries = &retries
		request.ConfirmRetries = !flagNoWait
	}
	if timeoutChanged {
		timeout, err := time.ParseDuration(flagUpdateJobTimeoutRaw)
		if err != nil || timeout <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --set-timeout: %q, expected positive duration such as 60s, 5m, or 1h", flagUpdateJobTimeoutRaw)
		}
		timeoutMillis := timeout.Milliseconds()
		if timeoutMillis <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --set-timeout: %q, duration must be at least 1ms", flagUpdateJobTimeoutRaw)
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

// parseUpdateJobCommandRequest chooses the input contract before deduplication.
// A scalar key keeps its established request and payload shape.
func parseUpdateJobCommandRequest(cmd *cobra.Command, args ...string) (job.UpdateRequest, *job.CompletionRequest, error) {
	if err := validateUpdateJobFlagIntent(cmd, args); err != nil {
		return job.UpdateRequest{}, nil, err
	}
	if cmd.Flags().Changed("set-bid") {
		if !flagUpdateJobComplete || cmd.Flags().Changed("fail") || cmd.Flags().Changed("throw-bpmn-error") {
			return job.UpdateRequest{}, nil, invalidFlagValuef("--set-bid requires --complete")
		}
		if err := job.ValidateCompletionBusinessID(&flagUpdateJobSetBID); err != nil {
			return job.UpdateRequest{}, nil, invalidFlagValuef("invalid --set-bid: %v", err)
		}
	}
	if err := validateOptionalDashArg(args); err != nil {
		return job.UpdateRequest{}, nil, err
	}
	// GetStringArray round-trips through CSV and loses a single empty value.
	// Read the stored slice so an explicit --key "" remains invalid.
	keyInputs := cmd.Flags().Lookup("key").Value.(pflag.SliceValue).GetSlice()
	stdin := len(args) == 1
	collection := stdin || len(keyInputs) > 1 || (len(keyInputs) == 1 && strings.Contains(keyInputs[0], ","))
	if collection && !flagUpdateJobComplete {
		return job.UpdateRequest{}, nil, invalidFlagValuef("key collections and stdin require --complete")
	}
	flagUpdateJobKey = ""
	if len(keyInputs) == 1 && !collection {
		flagUpdateJobKey = keyInputs[0]
	}
	selectorFlags := []string{"state", "type", "pi-key", "element-instance-key", "element-id", "worker", "kind", "listener-event-type", "batch-size", "limit"}
	hasSelection := false
	for _, name := range selectorFlags {
		hasSelection = hasSelection || cmd.Flags().Changed(name)
	}
	keyed := stdin || cmd.Flags().Changed("key")
	if hasSelection && (keyed || !flagUpdateJobComplete) {
		return job.UpdateRequest{}, nil, mutuallyExclusiveFlagsf("job search selectors and controls require --complete without explicit keys or stdin")
	}
	if collection {
		request, err := parseUpdateJobBatchRequest(cmd, &job.CompletionSelection{Mode: job.CompletionSelectionExplicit, InputForm: job.CompletionInputCollection})
		if err == nil {
			request.Selection, err = parseUpdateJobExplicitSelection(cmd, keyInputs, stdin)
		}
		return job.UpdateRequest{}, &request, err
	}
	if !keyed && flagUpdateJobComplete {
		request, err := parseUpdateJobBatchRequest(cmd, nil)
		return job.UpdateRequest{}, &request, err
	}
	for _, name := range []string{"workers", "fail-fast", "no-worker-limit"} {
		if cmd.Flags().Changed(name) {
			return job.UpdateRequest{}, nil, invalidFlagValuef("--%s requires batch completion", name)
		}
	}
	request, err := parseUpdateJobRequest(cmd)
	return request, nil, err
}

// validateUpdateJobArgs rejects ambiguous retained flag meanings before remote
// bootstrap; the diagnostic wrapper permits only these static messages to survive.
func validateUpdateJobArgs(cmd *cobra.Command, args []string) error {
	if err := validateOptionalDashArg(args); err != nil {
		return err
	}
	if err := validateUpdateJobFlagIntent(cmd, args); err != nil {
		return &migrationSemanticError{err}
	}
	return nil
}

// validateUpdateJobFlagIntent separates predicates, submitted values and HTTP
// controls without reading stdin, decoding payloads or constructing services.
func validateUpdateJobFlagIntent(cmd *cobra.Command, args []string) error {
	fs := cmd.Flags()
	if fs.Changed("retries") {
		if !flagUpdateJobComplete || fs.Changed("fail") || fs.Changed("throw-bpmn-error") {
			return invalidFlagValuef("--retries is a selector only; use --set-retries for job retry updates or technical failure")
		}
		if fs.Changed("key") || len(args) > 0 {
			return mutuallyExclusiveFlagsf("keyed --complete cannot be combined with --retries")
		}
	}
	if flagUpdateJobComplete && fs.Changed("set-retries") {
		return mutuallyExclusiveFlagsf("--complete cannot be combined with --set-retries")
	}
	if fs.Changed("timeout") && !fs.Changed("set-timeout") && !fs.Changed("set-retries") && !fs.Changed("complete") && !fs.Changed("fail") && !fs.Changed("throw-bpmn-error") {
		return invalidFlagValuef("--timeout controls HTTP execution only; use --set-timeout to update the job timeout")
	}
	return nil
}
