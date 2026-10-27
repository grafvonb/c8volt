// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// flagResumePIKeys holds explicit selection; services normalize the frozen scope again.
var flagResumePIKeys []string

// flagResumePITree enables explicit descendant planning before approval.
var flagResumePITree bool

// resumeProcessInstanceCmd exposes one collection workflow for single and multiple keys.
var resumeProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Aliases: []string{"pi"},
	Short:   "Resume suspended process instances by key or subtree",
	Long: `Resume SUSPENDED process instances on Camunda 8.10 by explicit key or selected subtree.

Provide repeated or comma-separated --key values, or newline-separated keys from stdin with '-'. Keys are deduplicated in first-input order. Without --tree, each selected parent or child is independent: relatives are neither discovered nor resumed. Explicit keys use backend authorization without discovery tenant filtering.

When authorization is enabled, resumption requires SUSPEND_PROCESS_INSTANCE on PROCESS_DEFINITION. Default confirmation also requires permission to read the instance. Exact-key execution has no mandatory preflight read; only the submission response establishes acceptance. Active, terminal, terminating, missing or unauthorized targets remain failures when rejected.

Default execution waits for observed ACTIVE or immediate COMPLETED after acceptance within the configured backoff timeout (two minutes by default, adjustable with --backoff-timeout). --timeout bounds each HTTP request. --no-wait returns accepted submission without confirmation reads. Partial failures retain acceptance and observation evidence with a nonzero exit. Without --tree, --dry-run previews exact keys without reads, writes or confirmation prompts; eligibility is not checked, including with --no-wait.

Add --tree to include each selected instance and its descendants, merging overlapping subtrees without escalating to ancestors or siblings. Discovery traverses active and terminal parents before selecting only observed SUSPENDED members; independently suspended descendants remain eligible. All other states are excluded. Tree execution and tree dry-run require read/search permissions. Tree dry-run reports selected, discovered, eligible and excluded counts without mutation. Failed or incomplete planning stops before approval or submission. A complete plan with no suspended targets succeeds without approval, activity or mutation, including with --no-wait.

Tree execution asks once for the expanded eligible count and uses that frozen scope without rediscovery. Discovery retains the limit of up to 1,000 direct children per parent lookup; coverage beyond that limit is not guaranteed. The scope is not atomic: state races can cause authoritative rejection, and descendants created after discovery may be omitted. There is no automatic rollback or parent-before-child execution guarantee. --no-wait skips confirmation reads, not tree discovery.

Resumption releases buffered work. Failures while processing buffered work can leave the instance suspended. Observed CANCELED or TERMINATED is a cancellation failure, never confirmed resumption. Persistent suspension, absence, unknown state or failed reads remain unconfirmed within the wait bound; missing suspendedDate alone is not proof. Resumption does not guarantee incident-free execution, process completion, or immediate subscription readiness. Multiple-key execution is not atomic and partial failure has no automatic rollback.

Use --auto-confirm or --automation for unattended execution. JSON requires --dry-run, --auto-confirm or --automation and cannot be combined with --verbose. Keys-only previews list selected exact keys or eligible tree targets; execution keys indicate acceptance, not confirmed resumption, even when confirmation fails. Use JSON and the exit status for confirmation evidence. Use --verbose for per-instance submission, confirmation, and last observed state. Quiet preserves explicit machine results and required errors. Use --debug for redacted HTTP diagnostics independently of --verbose.`,
	Example: `  ./c8volt resume process-instance --key <process-instance-key>
  ./c8volt resume pi --key <key-a>,<key-b> --key <key-a> --dry-run --json
  ./c8volt resume pi --key <process-instance-key> --auto-confirm --no-wait --json
  ./c8volt resume pi --key <process-instance-key> --tree --dry-run
  ./c8volt resume pi --key <parent-key>,<child-key> --tree --auto-confirm --json
  printf '%s\n' "$INSTANCE_A" "$INSTANCE_B" | ./c8volt resume pi --auto-confirm -`,
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
		keys, err := parseResumeProcessInstanceInput(cmd, args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
		if flagResumePITree {
			runResumeProcessInstanceTree(cmd, cli, log, cfg, keys)
			return
		}
		runResumeProcessInstances(cmd, cli, log, cfg, keys)
	},
}

// parseResumeProcessInstanceInput validates the entire explicit selection before any service work.
func parseResumeProcessInstanceInput(cmd *cobra.Command, args []string) (typex.Keys, error) {
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return nil, invalidFlagValuef("--workers must be positive integer")
	}
	if pickMode() == RenderModeJSON && flagVerbose {
		return nil, mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for resume process-instance")
	}
	if pickMode() == RenderModeJSON && !flagDryRun && !shouldImplicitlyConfirm(cmd) {
		return nil, missingDependentFlagsf("--json resume process-instance requires --dry-run, --auto-confirm, or --automation")
	}
	stdinKeys, err := readKeysIfDash(args)
	if err != nil {
		return nil, err
	}
	keys := append(typex.Keys{}, flagResumePIKeys...)
	keys = append(keys, stdinKeys...)
	if len(keys) == 0 {
		return nil, localPreconditionError(fmt.Errorf("no process instance keys provided or found to resume"))
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
	resumeCmd.AddCommand(resumeProcessInstanceCmd)
	fs := resumeProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagResumePIKeys, "key", "k", nil, "process instance key(s); repeat, comma-separate or combine with stdin '-'")
	fs.BoolVar(&flagResumePITree, "tree", false, "include selected instances and descendants; resume only observed SUSPENDED members")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview exact keys locally or discover eligible tree targets; no mutations or approval")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after accepted resumption without confirmation reads")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")
	useInvalidInputFlagErrors(resumeProcessInstanceCmd)
	setCommandMutation(resumeProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(resumeProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(resumeProcessInstanceCmd, AutomationSupportFull, "supports exact-key and tree previews, confirmed resumption and explicit no-wait acceptance")
	setOutputModes(resumeProcessInstanceCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true},
	)
}
