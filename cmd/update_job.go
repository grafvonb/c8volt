// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/consts"
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

// updateJobCmd retains scalar updates and routes selector completion to the frozen batch lifecycle.
var updateJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Update a job or complete selected jobs",
	Long: `Update a Camunda job by key on Camunda 8.8 or newer.

Supports retry and timeout updates and worker outcomes. Explicit --key uses backend authorization without tenant filtering.

For bulk --complete, omit --key and supply one or more AND predicates: --state, --type, --pi-key, --element-instance-key, --element-id, --worker, --retries, --kind, or --listener-event-type. Explicit --retries 0 is an exact selector in this mode; keyed --complete still rejects --retries. Other update modes require one key and reject selectors.

Selector discovery uses the configured tenant; --tenant overrides it, and empty --tenant or --all-tenants searches all accessible tenants. --batch-size controls each page; --limit caps unique selected jobs (0 means uncapped). Controls alone do not select jobs.

The complete selection is frozen before one confirmation. Incomplete discovery submits nothing. Empty results succeed without confirmation or mutation. --workers bounds concurrency; --fail-fast stops new work after an unsatisfied result. New sequential listener jobs require another invocation.

Completing a listener job acknowledges its work without executing its business logic. Accepted completion skips independent confirmation and does not verify process termination. Bulk requests are not automatically replayed; ambiguous outcomes use bounded observation, with at most one read under --no-wait. No retries, incidents or processes are repaired implicitly.

Bulk dry-run previews show count, filters and tenant scope; --verbose lists candidate context. JSON retains all preview candidates or ordered execution outcomes, including partial failures. Quiet suppresses prose while preserving requested JSON. A later selector invocation discovers a fresh selection.

c8volt plans the update and asks for confirmation before material interactive mutations. Retry updates are verified by reading the job; timeout updates and worker outcomes return after acceptance without waiting for confirmation.

Use --dry-run to inspect the plan without mutation. With --json, mutations require --dry-run, --auto-confirm, or --automation; --json cannot be combined with --verbose.`,
	Example: `  ./c8volt update job --key <job-key> --retries 3 --dry-run
  ./c8volt --tenant tenant-a update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --key <job-key> --retries 3 --auto-confirm
  ./c8volt update job --key <job-key> --timeout 5m --auto-confirm
  ./c8volt update job --key <job-key> --fail --retries 0 --message "worker unavailable" --dry-run
  ./c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message "card declined" --dry-run
  ./c8volt update job --key <job-key> --complete --vars '{"approved":true}' --dry-run
  ./c8volt --json update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --kind TASK_LISTENER --listener-event-type CANCELING --state CREATED --type removeTaskData --complete --dry-run --verbose
  ./c8volt update job --type removeTaskData --retries 0 --limit 1000 --batch-size 100 --complete --auto-confirm --json`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		request, batchRequest, err := parseUpdateJobCommandRequest(cmd)
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

// init registers update flags and advertises alternative completion selection.
func init() {
	updateCmd.AddCommand(updateJobCmd)

	fs := updateJobCmd.Flags()
	fs.StringVarP(&flagUpdateJobKey, "key", "k", "", "job key to update")
	fs.Int32Var(&flagUpdateJobRetries, "retries", 0, "retry count to set, remaining retries for --fail, or exact selector with keyless --complete (including 0)")
	fs.StringVar(&flagUpdateJobTimeoutRaw, "timeout", "", "timeout duration to submit for the job, for example 60s, 5m, or 1h")
	fs.BoolVar(&flagUpdateJobFail, "fail", false, "report a technical job failure")
	fs.StringVar(&flagUpdateJobRetryBackoffRaw, "retry-backoff", "", "duration before a failed job becomes retryable, for example 60s, 5m, or 1h")
	fs.StringVar(&flagUpdateJobMessage, "message", "", "operator message for worker outcome modes")
	fs.StringVar(&flagUpdateJobBPMNError, "throw-bpmn-error", "", "BPMN error code to throw for the job")
	fs.BoolVar(&flagUpdateJobComplete, "complete", false, "complete the job through the worker outcome API")
	fs.StringVar(&flagUpdateJobVariables, "vars", "", "JSON object with variables for BPMN error or completion outcomes")
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
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent completion workers (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new completions after the first unsatisfied outcome")

	useInvalidInputFlagErrors(updateJobCmd)
	setCommandMutation(updateJobCmd, CommandMutationStateChanging)
	setContractSupport(updateJobCmd, ContractSupportFull)
	setAutomationSupport(updateJobCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
	setOutputModes(updateJobCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}
