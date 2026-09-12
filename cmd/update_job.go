// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagUpdateJobKey             string
	flagUpdateJobRetries         int32
	flagUpdateJobTimeoutRaw      string
	flagUpdateJobFail            bool
	flagUpdateJobRetryBackoffRaw string
	flagUpdateJobMessage         string
	flagUpdateJobBPMNError       string
	flagUpdateJobComplete        bool
	flagUpdateJobVariables       string
)

var updateJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Update a job by key",
	Long: `Update a Camunda job by key on Camunda 8.8 or newer.

Supports retry and timeout updates and worker outcomes. Explicit --key uses backend authorization without tenant filtering.

c8volt plans the update and asks for confirmation before material interactive mutations. Retry updates are verified by reading the job; timeout updates and worker outcomes return after acceptance without waiting for confirmation.

Use --dry-run to inspect the plan without mutation. With --json, mutations require --dry-run, --auto-confirm, or --automation; --json cannot be combined with --verbose.`,
	Example: `  ./c8volt update job --key <job-key> --retries 3 --dry-run
  ./c8volt --tenant tenant-a update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --key <job-key> --retries 3 --auto-confirm
  ./c8volt update job --key <job-key> --timeout 5m --auto-confirm
  ./c8volt update job --key <job-key> --fail --retries 0 --message "worker unavailable" --dry-run
  ./c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message "card declined" --dry-run
  ./c8volt update job --key <job-key> --complete --vars '{"approved":true}' --dry-run
  ./c8volt --json update job --key <job-key> --retries 3 --dry-run`,
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
		plan, err := planUpdateJob(cmd.Context(), cli, request)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan job update: %w", err))
		}
		attachUpdateJobExplicitTenantContext(cmd, cfg, plan)
		request.UpdatePlan = &plan
		if err := validateUpdateJobPlanPreconditions(plan, request); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if flagDryRun {
			if err := jobUpdatePlanView(cmd, plan, "dry run"); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update dry-run result: %w", err))
			}
			return
		}
		if !plan.HasMaterialChange() {
			if err := jobUpdatePlanView(cmd, plan, "plan"); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update plan: %w", err))
			}
			return
		}
		if !shouldImplicitlyConfirm(cmd) {
			if err := jobUpdatePlanView(cmd, plan, "plan"); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update plan: %w", err))
			}
			prompt := fmt.Sprintf("You are about to update job %s. Do you want to proceed?", request.Key)
			if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		} else {
			renderAttachedTenantContext(cmd)
		}
		if request.WorkerOutcome != nil {
			if err := executeUpdateJobWorkerOutcome(cmd, cli, request, plan); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			return
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
	fs.StringVarP(&flagUpdateJobKey, "key", "k", "", "job key to update")
	fs.Int32Var(&flagUpdateJobRetries, "retries", 0, "retry count to set, or remaining retries for --fail")
	fs.StringVar(&flagUpdateJobTimeoutRaw, "timeout", "", "timeout duration to submit for the job, for example 60s, 5m, or 1h")
	fs.BoolVar(&flagUpdateJobFail, "fail", false, "report a technical job failure")
	fs.StringVar(&flagUpdateJobRetryBackoffRaw, "retry-backoff", "", "duration before a failed job becomes retryable, for example 60s, 5m, or 1h")
	fs.StringVar(&flagUpdateJobMessage, "message", "", "operator message for worker outcome modes")
	fs.StringVar(&flagUpdateJobBPMNError, "throw-bpmn-error", "", "BPMN error code to throw for the job")
	fs.BoolVar(&flagUpdateJobComplete, "complete", false, "complete the job through the worker outcome API")
	fs.StringVar(&flagUpdateJobVariables, "vars", "", "JSON object with variables for BPMN error or completion outcomes")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview job updates without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without retry confirmation")

	useInvalidInputFlagErrors(updateJobCmd)
	setCommandMutation(updateJobCmd, CommandMutationStateChanging)
	setContractSupport(updateJobCmd, ContractSupportFull)
	setAutomationSupport(updateJobCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
	setOutputModes(updateJobCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
	setFlagContractRequired(updateJobCmd, "key")
}
