// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagUpdatePISetBID   string
	flagUpdatePIKeys     []string
	flagUpdatePIVars     string
	flagUpdatePIVarsFile string
)

// updateProcessInstanceCmd updates variables or assigns native identity to exact instances.
var updateProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Update process-instance variables or assign a business ID",
	Long: `Update root-scope variables on selected process instances.

Select by key or exact --bid (only c8.10+), then supply one JSON object through
--set-vars or --set-vars-file. The plan changes only root variables and waits
until requested values are visible. Unchanged plans need no writes.

Alternatively, --set-bid assigns the first native business ID to an active root
instance with uniqueness disabled (only c8.10+). Use explicit keys; assignment
cannot be combined with variable updates, and reassignment conflicts.
Use --dry-run to review the plan.`,
	Example: `  c8volt update process-instance --key <process-instance-key> --set-vars '{"customerTier":"gold"}' --dry-run
  c8volt update process-instance --key <process-instance-key> --set-vars-file vars.json
  c8volt update process-instance --bid ORDER-123 --set-vars '{"approved":true}' --dry-run`,
	Aliases: []string{"pi"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if !cmd.Flags().Changed("bid") && (cmd.Flags().Changed("limit") || cmd.Flags().Changed("batch-size")) {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("--limit and --batch-size require --bid"))
			return
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		if cmd.Flags().Changed("set-bid") {
			runUpdateProcessInstanceAssignment(cmd, cli, log, cfg, args)
			return
		}
		variables, err := parseUpdateProcessInstanceVariablesFromFlags(cmd, flagUpdatePIVars, flagUpdatePIVarsFile)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("bid") {
			runUpdateProcessInstanceBusinessID(cmd, cli, log, cfg, args, variables)
			return
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagUpdatePIKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to update")))
		}
		if err := validateUpdateProcessInstanceJSONConfirmation(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		prepared, err := planUpdateProcessInstanceVariables(cmd.Context(), cmd, cli, keys, variables)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan process-instance variable update: %w", err))
		}
		preview := prepared.preview
		if flagDryRun {
			if err := renderUpdateProcessInstanceVariablePreview(cmd, preview); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update dry-run result: %w", err))
			}
			return
		}
		if !preview.HasPlannedChanges() {
			if err := renderUpdateProcessInstanceVariablePlan(cmd, preview); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update plan: %w", err))
			}
			return
		}
		if !shouldImplicitlyConfirm(cmd) {
			if err := renderUpdateProcessInstanceVariablePlan(cmd, preview); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update plan: %w", err))
			}
			requestedUpdates := preview.VariableAddCount + preview.VariableChangeCount
			prompt := fmt.Sprintf("You are about to update %d requested variable value(s) on %d process instance(s). Do you want to proceed?", requestedUpdates, preview.UpdateCount)
			if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		}
		results, executeErr := cli.ExecuteProcessInstanceVariableUpdates(cmd.Context(), prepared.plan, flagWorkers, collectExplicitPIAdminInputOptions()...)
		if executeErr != nil {
			if len(results.Items) > 0 {
				if err := renderUpdateProcessInstanceVariableFailure(cmd, results, executeErr); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render partial update result: %w", err))
				}
				exitAfterRenderedResult(cfg.App.NoErrCodes, executeErr)
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("update process-instance variables: %w", executeErr))
		}
		if err := renderUpdateProcessInstanceVariableResults(cmd, results); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update result: %w", err))
		}
	},
}

// init registers assignment and variable-update flags with the shared mutation contract.
func init() {
	updateCmd.AddCommand(updateProcessInstanceCmd)

	setFlagMigrations(updateProcessInstanceCmd, map[string]flagMigration{
		"vars":      {Replacement: "--set-vars", TakesValue: true},
		"vars-file": {Replacement: "--set-vars-file", TakesValue: true},
	})
	fs := updateProcessInstanceCmd.Flags()
	registerBusinessIDActionFlags(updateProcessInstanceCmd)
	fs.StringVar(&flagUpdatePISetBID, "set-bid", "", "assign an exact native business ID to explicit process keys (only c8.10+)")
	fs.StringSliceVarP(&flagUpdatePIKeys, "key", "k", nil, "process instance key(s) to update; repeat or combine with stdin '-'")
	fs.StringVar(&flagUpdatePIVars, "set-vars", "", "JSON object with variables to set on each process instance")
	fs.StringVar(&flagUpdatePIVarsFile, "set-vars-file", "", "path to JSON object file with variables to set on each process instance")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview variable updates or business-ID assignment without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new updates after the first error")

	useInvalidInputFlagErrors(updateProcessInstanceCmd)
	setCommandMutation(updateProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(updateProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(updateProcessInstanceCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
}
