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
	Long: `Update a Camunda job by key on Camunda 8.8 or newer.

Use the bare --throw-bpmn-error action with required --set-error-code; attached action values and legacy positional codes are rejected. Use --set-error-message for outcome messages and --set-retry-backoff for technical failure backoff.

Supports retry and timeout updates and worker outcomes. Use --set-retries for submitted retries and --set-timeout for the job timeout. In v5, inherited --timeout controls HTTP execution only; it never supplies a job mutation. Both timeouts may be supplied independently. Bare --retries is only a keyless completion selector, not a mutation alias. Explicit --key uses backend authorization without tenant filtering.

For bulk --complete, repeat --key, supply comma-separated keys, or use one positional '-' to read newline-separated keys. Flags precede stdin; blank lines are ignored, invalid or empty key tokens fail, and all input is validated before mutation. Duplicate keys are completed once. Empty stdin succeeds without mutation. Explicit keys retain backend authorization across tenants and show actual tenant context. One scalar key preserves the existing single-job result; every collection keeps the bulk result even after deduplication.

For selector --complete, omit --key and supply one or more AND predicates: --state, --type, --pi-key, --element-instance-key, --element-id, --worker, --retries, --kind, or --listener-event-type. Explicit --retries 0 is an exact selector in this mode; keyed --complete still rejects --retries. Other update modes require one scalar key and reject collections and selectors.

Selector discovery uses the configured tenant; --tenant overrides it, and empty --tenant or --all-tenants searches all accessible tenants. --batch-size controls each page; --limit caps unique selected jobs (0 means uncapped). Controls alone do not select jobs. Explicit keys cannot combine with predicates or search controls.

The complete selection is frozen before one confirmation. Incomplete discovery submits nothing. Empty results succeed without confirmation or mutation. --workers bounds concurrency; --fail-fast stops new work after an unsatisfied result. New sequential listener jobs require another invocation.

Completing a listener job acknowledges its work without executing its business logic. Accepted completion skips independent confirmation and does not verify process termination. Bulk requests are not automatically replayed; ambiguous outcomes use bounded observation, with at most one read under --no-wait. No retries, incidents or processes are repaired implicitly.

Bulk dry-run previews show count, filters or explicit selection and tenant scope; --verbose lists candidate context. JSON retains all preview candidates or ordered execution outcomes, including partial failures. Quiet suppresses prose while preserving requested JSON. A later selector invocation discovers a fresh selection. Camunda 8.10 update plans and frozen completion candidates retain each observed job's own business ID, including in verbose rows and partial results. Use --set-bid with --complete on Camunda 8.10 to atomically assign the first business ID and complete each job, with variables in the same request. --set-bid is payload, not a selector. Assignment requires an active root instance without a business ID and uniqueness disabled; even an identical existing ID conflicts. Direct process assignment and atomic completion are alternative first assignments.

Assignment preserves the exact value, including spaces and commas; empty/blank values, invalid UTF-8 and values over 256 UTF-16 units are rejected. Preview cannot reserve eligibility. BID-bearing requests are never replayed or retried without the BID. Accepted means acknowledged, not independently confirmed; --no-wait does not establish assignment from a later COMPLETED job. Lost responses remain unknown with the original failure. Observed job/task/listener snapshots remain independent of requested identity.

Bulk previews, summaries and verbose rows use the configured logger at INFO level. Bulk failures produce one compact logged error after the result summary, grouping repeated reasons. Quiet retains that error. Use --verbose for per-job outcomes and --debug or configured debug logging for HTTP diagnostics. JSON preserves the original failure evidence and partial results in one envelope.

c8volt plans the update and asks for confirmation before material interactive mutations. Retry updates are verified by reading the job; timeout updates and worker outcomes return after acceptance without waiting for confirmation.

For a single job, --complete, --fail and --throw-bpmn-error reject an already completed job before confirmation or submission, including during dry-run. Other states remain subject to backend validation. Human failures use compact logged reasons; JSON preserves backend error evidence and exit codes. Use --debug for HTTP diagnostics.

Use --dry-run to inspect the plan without mutation. With --json, mutations require --dry-run, --auto-confirm, or --automation; --json cannot be combined with --verbose.`,
	Example: `  ./c8volt update job --key <job-key> --set-retries 3 --dry-run
  ./c8volt --tenant tenant-a update job --key <job-key> --set-retries 3 --dry-run
  ./c8volt update job --key <job-key> --set-retries 3 --auto-confirm
  ./c8volt update job --key <job-key> --set-timeout 5m --auto-confirm
  ./c8volt update job --key <job-key> --fail --set-retries 0 --set-error-message "worker unavailable" --dry-run
  ./c8volt update job --key <job-key> --throw-bpmn-error --set-error-code PAYMENT_DECLINED --set-error-message "card declined" --dry-run
  ./c8volt update job --key <job-key> --complete --set-vars '{"approved":true}' --dry-run
  ./c8volt update job --key <job-key> --complete --set-bid ORDER-123 --set-vars '{"approved":true}' --dry-run
  ./c8volt --json update job --key <job-key> --set-retries 3 --dry-run
  ./c8volt update job --kind TASK_LISTENER --listener-event-type CANCELING --state CREATED --type removeTaskData --complete --dry-run --verbose
  ./c8volt update job --type removeTaskData --retries 0 --limit 1000 --batch-size 100 --complete --auto-confirm --json
  ./c8volt update job --key 2251799870796844,2251799870796848 --key 2251799870796849 --complete --dry-run
  ./c8volt update job - --complete --auto-confirm --json < reviewed-job-keys.txt`,
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
	fs.BoolVar(&flagUpdateJobThrowBPMNError, "throw-bpmn-error", false, "throw a BPMN error using --set-error-code (bare action only)")
	fs.StringVar(&flagUpdateJobBPMNError, "set-error-code", "", "BPMN error code to submit with --throw-bpmn-error")
	fs.BoolVar(&flagUpdateJobComplete, "complete", false, "complete the job through the worker outcome API")
	fs.StringVar(&flagUpdateJobSetBID, "set-bid", "", "assign the first business ID atomically with --complete (Camunda 8.10)")
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
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent completion workers (default: min(count, 2*GOMAXPROCS, 32))")
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
