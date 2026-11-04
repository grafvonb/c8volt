// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	flagUpdateJobKey             string
	flagUpdateJobRetries         int32
	flagUpdateJobSetRetries      int32
	flagUpdateJobTimeoutRaw      string
	flagUpdateJobFail            bool
	flagUpdateJobRetryBackoffRaw string
	flagUpdateJobMessage         string
	flagUpdateJobBPMNError       string
	// flagUpdateJobThrowBPMNError selects the action; token validation rejects attached values.
	flagUpdateJobThrowBPMNError bool
	flagUpdateJobComplete       bool
	flagUpdateJobVariables      string
	flagUpdateJobSetBID         string
)

// updateJobCmd retains scalar updates and routes collection completion to the frozen batch lifecycle.
var updateJobCmd = &cobra.Command{
	Use:   "job [-]",
	Short: "Update a job or complete selected jobs",
	Long: `Update job retries or timeout, or submit worker outcomes.

Use --set-retries for retries and --set-timeout for the job timeout.
Inherited --timeout controls HTTP requests only. Use --complete to complete a
job, --fail with --set-retries to fail it, or --throw-bpmn-error with
--set-error-code to report a BPMN error. Add --set-error-message for context.

Completion accepts explicit keys or search filters. --retries is an exact
completion filter, including zero; it never writes retries. The selection is
frozen before confirmation. Completing a listener acknowledges its work without
executing its business logic. Other actions require one key.

Use --dry-run to review the plan. Retry updates are verified; timeout updates
and worker outcomes return after acceptance. JSON retains partial failures.`,
	Example: `  c8volt update job --key <job-key> --set-retries 3 --dry-run
  c8volt update job --key <job-key> --set-timeout 5m
  c8volt update job --key <job-key> --complete --set-vars '{"approved":true}'
  c8volt update job --key <job-key> --fail --set-retries 0 --set-error-message "worker unavailable"
  c8volt update job --key <job-key> --throw-bpmn-error --set-error-code PAYMENT_DECLINED`,
	Args: validateUpdateJobArgs,
	Run: func(cmd *cobra.Command, args []string) {
		request, batchRequest, err := parseUpdateJobCommandRequest(cmd, args...)
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
		if batchRequest != nil {
			err := runUpdateJobBatch(cmd, cli, cfg, *batchRequest)
			if err != nil {
				exitAfterRenderedResult(cfg.App.NoErrCodes, err)
			}
			return
		}
		if request.WorkerOutcome != nil && request.WorkerOutcome.SetBusinessID != nil && cfg.App.CamundaVersion != toolx.V810 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w: business-ID assignment requires Camunda 8.10", ferrors.ErrUnsupported))
		}
		plan, err := planUpdateJob(cmd.Context(), cli, request)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan job update: %w", err), jobWorkerOutcomeErrorMessage(request, err, true)...)
		}
		attachUpdateJobExplicitTenantContext(cmd, cfg, plan)
		request.UpdatePlan = &plan
		if err := validateUpdateJobPlanPreconditions(plan, request); err != nil {
			if request.WorkerOutcome != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err, formatCompletedJobGuard(*request.WorkerOutcome))
			}
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
				if request.WorkerOutcome.SetBusinessID != nil {
					exitAfterRenderedResult(cfg.App.NoErrCodes, err)
				}
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err, jobWorkerOutcomeErrorMessage(request, err, false)...)
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

// init registers update flags and advertises alternative completion selection.
func init() {
	updateCmd.AddCommand(updateJobCmd)

	fs := updateJobCmd.Flags()
	fs.StringArrayP("key", "k", nil, "job key to update; for --complete, repeat or use comma-separated keys and stdin '-'")
	fs.Int32Var(&flagUpdateJobRetries, "retries", 0, "exact retry count selector with keyless --complete (including 0)")
	fs.Int32Var(&flagUpdateJobSetRetries, "set-retries", 0, "retry count to set or remaining retries for --fail (including 0)")
	fs.StringVar(&flagUpdateJobTimeoutRaw, "set-timeout", "", "timeout duration to submit for the job, for example 60s, 5m, or 1h")
	fs.BoolVar(&flagUpdateJobFail, "fail", false, "report a technical job failure")
	fs.StringVar(&flagUpdateJobRetryBackoffRaw, "set-retry-backoff", "", "duration before a failed job becomes retryable, for example 60s, 5m, or 1h")
	fs.StringVar(&flagUpdateJobMessage, "set-error-message", "", "operator message for worker outcome modes")
	fs.BoolVar(&flagUpdateJobThrowBPMNError, "throw-bpmn-error", false, "throw a BPMN error; requires --set-error-code")
	fs.StringVar(&flagUpdateJobBPMNError, "set-error-code", "", "BPMN error code to submit with --throw-bpmn-error")
	fs.BoolVar(&flagUpdateJobComplete, "complete", false, "complete the job through the worker outcome API")
	fs.StringVar(&flagUpdateJobSetBID, "set-bid", "", "assign the first business ID atomically with --complete (only c8.10+)")
	fs.StringVar(&flagUpdateJobVariables, "set-vars", "", "JSON object with variables for BPMN error or completion outcomes")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview job updates without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip retry confirmation; bulk completion permits one reconciliation read without polling")

	fs.StringP("state", "s", "", "job state selector for --complete; case-insensitive")
	fs.String("type", "", "job type selector for --complete")
	fs.String("pi-key", "", "process instance key selector for --complete")
	fs.String("element-instance-key", "", "element instance key selector for --complete")
	fs.String("element-id", "", "BPMN element ID selector for --complete")
	fs.String("worker", "", "worker name selector for --complete")
	fs.String("kind", "", "job kind selector for --complete; case-insensitive")
	fs.String("listener-event-type", "", "listener event selector for --complete; case-insensitive")
	fs.Int32P("batch-size", "n", consts.MaxPISearchSize, "jobs per discovery page for selector completion")
	fs.Int32P("limit", "l", 0, "maximum unique selector completion targets; 0 means uncapped")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent completion workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new completions after the first unsatisfied outcome")

	useInvalidInputFlagErrors(updateJobCmd)
	setFlagMigrations(updateJobCmd, map[string]flagMigration{
		"vars":             {Replacement: "--set-vars", TakesValue: true},
		"message":          {Replacement: "--set-error-message", TakesValue: true},
		"retry-backoff":    {Replacement: "--set-retry-backoff", TakesValue: true},
		"throw-bpmn-error": {Replacement: "--throw-bpmn-error --set-error-code CODE", Valueless: true},
	})
	setCommandMutation(updateJobCmd, CommandMutationStateChanging)
	setContractSupport(updateJobCmd, ContractSupportFull)
	setAutomationSupport(updateJobCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
	setOutputModes(updateJobCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}
