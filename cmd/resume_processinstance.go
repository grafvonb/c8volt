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
	Short:   "Resume suspended process instances by key, business ID or subtree",
	Long: `Resume suspended process instances (only c8.10+).

Select exact keys or one exact --bid. Add --tree to include suspended descendants;
ancestors and siblings are excluded. The target set is fixed before confirmation.
By default, wait until ACTIVE or immediate COMPLETED is observed.

Use --dry-run to preview. An exact-key preview makes no requests; tree and BID
previews discover scope. Resumption releases buffered work and does not guarantee
incident-free execution. Partial failures do not roll back accepted changes.`,
	Example: `  c8volt resume process-instance --key <process-instance-key> --dry-run
  c8volt resume process-instance --key <process-instance-key> --tree --dry-run
  c8volt resume process-instance --key <process-instance-key> --tree --auto-confirm`,
	Args: func(cmd *cobra.Command, args []string) error { return validateOptionalDashArg(args) },
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
			return
		}
		if !cmd.Flags().Changed("bid") && (cmd.Flags().Changed("limit") || cmd.Flags().Changed("batch-size")) {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("--limit and --batch-size require --bid"))
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
		if cmd.Flags().Changed("bid") {
			runResumeProcessInstanceBusinessID(cmd, cli, log, cfg)
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
	if cmd.Flags().Changed("bid") {
		return nil, validateBusinessIDActionInput(cmd, args)
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
	registerBusinessIDActionFlags(resumeProcessInstanceCmd)
	fs.StringSliceVarP(&flagResumePIKeys, "key", "k", nil, "process instance key(s); repeat, comma-separate or combine with stdin '-'")
	fs.BoolVar(&flagResumePITree, "tree", false, "include selected instances and descendants; resume only observed SUSPENDED members")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview exact keys locally or discover eligible tree targets; no mutations or approval")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after accepted resumption without confirmation reads")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
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
