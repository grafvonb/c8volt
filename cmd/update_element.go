// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// Element update flags are local to explicit selection and payload validation.
var (
	flagUpdateElementKeys     []string
	flagUpdateElementVars     string
	flagUpdateElementVarsFile string
)

// updateElementCmd exposes exact-local updates; backend planning and execution belong to services.
var updateElementCmd = &cobra.Command{
	Use:     "element",
	Aliases: []string{"ei"},
	Short:   "Update variables at explicit element scopes",
	Long: `Update variables owned directly by selected runtime elements.

Provide keys and one JSON object through --set-vars or --set-vars-file.
The plan changes only element-local values; parent and sibling scopes are excluded.
Use --dry-run to review additions and changes. Unchanged plans need no writes.
By default, wait until requested values are visible.`,
	Example: `  c8volt update element --key <element-instance-key> --set-vars '{"approved":true}' --dry-run
  c8volt update element --key <element-instance-key> --set-vars-file vars.json`,
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
		keys, variables, err := parseElementVariableUpdateInput(cmd, args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
		runUpdateElementVariables(cmd, cli, log, cfg, keys, variables)
	},
}

// parseElementVariableUpdateInput completes local validation before any planning request.
func parseElementVariableUpdateInput(cmd *cobra.Command, args []string) (typex.Keys, map[string]any, error) {
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return nil, nil, invalidFlagValuef("--workers must be positive integer")
	}
	if pickMode() == RenderModeJSON && flagVerbose {
		return nil, nil, mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update element")
	}
	if pickMode() == RenderModeJSON && !flagDryRun && !shouldImplicitlyConfirm(cmd) {
		return nil, nil, missingDependentFlagsf("--json update element requires --dry-run, --auto-confirm, or --automation")
	}
	variables, err := parseUpdateVariablesFromFlags(cmd, flagUpdateElementVars, flagUpdateElementVarsFile)
	if err != nil {
		return nil, nil, err
	}
	stdinKeys, err := readKeysIfDash(args)
	if err != nil {
		return nil, nil, err
	}
	keys := append(typex.Keys{}, flagUpdateElementKeys...)
	keys = append(keys, stdinKeys...)
	if len(keys) == 0 {
		return nil, nil, localPreconditionError(fmt.Errorf("no element keys provided or found to update"))
	}
	if ok, _, index := validateKeys(keys); !ok {
		return nil, nil, invalidFlagValuef("element key at index %d must contain 16 digits", index)
	}
	for i := range keys {
		keys[i] = strings.TrimSpace(keys[i])
	}
	return keys.Unique(), variables, nil
}

// init registers mutation policy and the existing worker/wait controls.
func init() {
	updateCmd.AddCommand(updateElementCmd)
	setFlagMigrations(updateElementCmd, map[string]flagMigration{
		"vars":      {Replacement: "--set-vars", TakesValue: true},
		"vars-file": {Replacement: "--set-vars-file", TakesValue: true},
	})
	fs := updateElementCmd.Flags()
	fs.StringSliceVarP(&flagUpdateElementKeys, "key", "k", nil, "runtime element key(s); repeat, comma-separate or combine with stdin '-'")
	fs.StringVar(&flagUpdateElementVars, "set-vars", "", "JSON object with variables to set locally on each element")
	fs.StringVar(&flagUpdateElementVarsFile, "set-vars-file", "", "path to JSON object file with variables to set locally")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview exact local changes without mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after accepted writes without local variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new updates after the first error")
	useInvalidInputFlagErrors(updateElementCmd)
	setCommandMutation(updateElementCmd, CommandMutationStateChanging)
	setContractSupport(updateElementCmd, ContractSupportFull)
	setAutomationSupport(updateElementCmd, AutomationSupportFull, "supports frozen local previews, confirmed updates and explicit no-wait acceptance")
	setOutputModes(updateElementCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true},
	)
}
