// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// flagSuspendPIKeys holds explicit selection; services normalize the frozen scope again.
var flagSuspendPIKeys []string

// suspendProcessInstanceCmd exposes one collection workflow for single and multiple keys.
var suspendProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Aliases: []string{"pi"},
	Short:   "Suspend explicitly selected active process instances",
	Long: `Suspend ACTIVE process instances on Camunda 8.10 by explicit key.

Provide repeated or comma-separated --key values, or newline-separated keys from stdin with '-'. Keys are deduplicated in first-input order. Each selected parent or child is independent: relatives are neither discovered nor suspended. Explicit keys use backend authorization without discovery tenant filtering.

When authorization is enabled, suspension requires SUSPEND_PROCESS_INSTANCE on PROCESS_DEFINITION. Default confirmation also requires permission to read the instance. There is no mandatory preflight read; only the submission response establishes acceptance. Already-suspended, terminal, terminating, missing or unauthorized targets remain failures when rejected.

Default execution waits for observed SUSPENDED within the configured backoff timeout (two minutes by default, adjustable with --backoff-timeout). --timeout bounds each HTTP request. --no-wait returns accepted submission without confirmation reads. Partial failures retain acceptance and observation evidence with a nonzero exit. --dry-run previews exact keys without reads, writes or confirmation prompts; eligibility is not checked, including with --no-wait.

Suspension pauses engine execution while retaining runtime state and resources; it does not cancel the instance. It does not interrupt already-running worker code or undo external effects. Job completion/failure attempts are rejected during suspension, and affected jobs may execute again after resumption. Backpressure can delay or reject suspension; this is not an instantaneous emergency stop.

Use --auto-confirm or --automation for unattended execution. JSON requires --dry-run, --auto-confirm or --automation and cannot be combined with --verbose. Keys-only previews list selected keys; execution keys indicate acceptance, not confirmed suspension, even when confirmation fails. Use JSON and the exit status for confirmation evidence. Use --verbose for per-instance submission, confirmation, and last observed state. Quiet preserves explicit machine results and required errors. Use --debug for redacted HTTP diagnostics independently of --verbose.`,
	Example: `  ./c8volt suspend process-instance --key <process-instance-key>
  ./c8volt suspend pi --key <key-a>,<key-b> --key <key-a> --dry-run --json
  ./c8volt suspend pi --key <process-instance-key> --auto-confirm --no-wait --json
  printf '%s\n' "$INSTANCE_A" "$INSTANCE_B" | ./c8volt suspend pi --auto-confirm -`,
	Args: func(cmd *cobra.Command, args []string) error { return validateOptionalDashArg(args) },
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
			return
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
		keys, err := parseSuspendProcessInstanceInput(cmd, args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
		runSuspendProcessInstances(cmd, cli, log, cfg, keys)
	},
}

// parseSuspendProcessInstanceInput validates the entire explicit selection before any service work.
func parseSuspendProcessInstanceInput(cmd *cobra.Command, args []string) (typex.Keys, error) {
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return nil, invalidFlagValuef("--workers must be positive integer")
	}
	if pickMode() == RenderModeJSON && flagVerbose {
		return nil, mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for suspend process-instance")
	}
	if pickMode() == RenderModeJSON && !flagDryRun && !shouldImplicitlyConfirm(cmd) {
		return nil, missingDependentFlagsf("--json suspend process-instance requires --dry-run, --auto-confirm, or --automation")
	}
	stdinKeys, err := readKeysIfDash(args)
	if err != nil {
		return nil, err
	}
	keys := append(typex.Keys{}, flagSuspendPIKeys...)
	keys = append(keys, stdinKeys...)
	if len(keys) == 0 {
		return nil, localPreconditionError(fmt.Errorf("no process instance keys provided or found to suspend"))
	}
	if ok, _, index := validateKeys(keys); !ok {
		return nil, invalidFlagValuef("process instance key at index %d must contain 16 digits", index)
	}
	for i := range keys {
		keys[i] = strings.TrimSpace(keys[i])
	}
	return keys.Unique(), nil
}

// init declares the supported mutation, automation and result contracts.
func init() {
	suspendCmd.AddCommand(suspendProcessInstanceCmd)
	fs := suspendProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagSuspendPIKeys, "key", "k", nil, "process instance key(s); repeat, comma-separate or combine with stdin '-'")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview exact keys without reads or mutations; eligibility is not checked")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after accepted suspension without confirmation reads")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")
	useInvalidInputFlagErrors(suspendProcessInstanceCmd)
	setCommandMutation(suspendProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(suspendProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(suspendProcessInstanceCmd, AutomationSupportFull, "supports exact-key previews, confirmed suspension and explicit no-wait acceptance")
	setOutputModes(suspendProcessInstanceCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true},
	)
}
