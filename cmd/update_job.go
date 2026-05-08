// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
)

var (
	flagUpdateJobKey        string
	flagUpdateJobRetries    int32
	flagUpdateJobTimeoutRaw string
)

var updateJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Update a job by key",
	Long: "Update a Camunda job by key.\n\n" +
		"The command supports retries and timeout updates for Camunda 8.8 and 8.9. It builds a pre-mutation plan, supports --dry-run previews, asks for confirmation before material interactive mutations, and can return after acceptance with --no-wait. Camunda 8.7 returns an unsupported-version error before mutation.",
	Example: `  ./c8volt update job --key 2251799813711967 --retries 3
  ./c8volt update job --key 2251799813711967 --timeout 5m
  ./c8volt update job --key 2251799813711967 --retries 3 --timeout 5m
  ./c8volt update job --key 2251799813711967 --retries 3 --dry-run
  ./c8volt update job --key 2251799813711967 --retries 3 --auto-confirm
  ./c8volt update job --key 2251799813711967 --retries 3 --no-wait
  ./c8volt --json update job --key 2251799813711967 --retries 3 --dry-run
  ./c8volt --json update job --key 2251799813711967 --retries 3 --auto-confirm`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		request, err := parseUpdateJobRequest(cmd)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		if err := validateUpdateJobJSONGuardrails(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		result, err := cli.UpdateJob(cmd.Context(), request, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("update job: %w", err))
		}
		if err := jobUpdateResultView(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update result: %w", err))
		}
	},
}

func init() {
	updateCmd.AddCommand(updateJobCmd)

	fs := updateJobCmd.Flags()
	fs.StringVar(&flagUpdateJobKey, "key", "", "job key to update")
	fs.Int32Var(&flagUpdateJobRetries, "retries", 0, "retry count to set on the job")
	fs.StringVar(&flagUpdateJobTimeoutRaw, "timeout", "", "timeout duration to submit for the job, for example 60s, 5m, or 1h")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview job updates without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without retry confirmation")

	useInvalidInputFlagErrors(updateJobCmd)
	setCommandMutation(updateJobCmd, CommandMutationStateChanging)
	setContractSupport(updateJobCmd, ContractSupportFull)
	setAutomationSupport(updateJobCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results with --no-wait")
	setFlagContractRequired(updateJobCmd, "key")
}

func parseUpdateJobRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if strings.TrimSpace(flagUpdateJobKey) == "" {
		return job.UpdateRequest{}, invalidFlagValuef("job update requires a non-empty --key")
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
		Automation:  automationModeEnabled(cmd),
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

func validateUpdateJobJSONGuardrails(cmd *cobra.Command) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update job")
	}
	if flagDryRun || pickMode() != RenderModeJSON || shouldImplicitlyConfirm(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json update job requires --dry-run, --auto-confirm, or --automation")
}
