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
	Long: `Update local variables on selected runtime element instances on Camunda 8.8 or newer.

Provide repeated or comma-separated --key values, or newline-separated keys from stdin with '-'. Supply exactly one non-null JSON object through --vars or --vars-file. Keys are deduplicated in first-input order. Explicit keys bypass the discovery tenant filter while retaining backend authorization.

The command reads every selected element and its complete local variables before approval. It freezes additions and changes at element:<key>, then writes only those values with local scope. Parent, sibling, descendant and called-process variables are not changed. An inherited same-name value does not prevent a local addition. Truncated values cannot prove equality. A null property assigns null; it does not delete a variable.

Use --dry-run to preview without prompts or writes. An empty object or an all-unchanged plan completes without confirmation, mutation or read-back. Normal execution confirms all requested values locally. --no-wait skips only read-back and reports accepted submissions, not confirmed values. Partial failures preserve acceptance and uncertainty with a nonzero exit; there is no rollback or resubmission after confirmation failure.

JSON requires --dry-run, --auto-confirm or --automation and cannot be combined with --verbose. Keys-only previews list changed destinations; executed results list accepted destinations even if confirmation failed. Quiet preserves explicit machine results and required errors.`,
	Example: `  ./c8volt update element --key <element-key> --vars '{"approved":true}' --dry-run
  ./c8volt update ei --key <key-a>,<key-b> --key <key-a> --vars-file ./vars.json --auto-confirm
  printf '%s\n' "$ELEMENT_KEY_A" "$ELEMENT_KEY_B" | ./c8volt update element - --vars '{"status":null}' --auto-confirm
  ./c8volt --json update element --key <element-key> --vars '{"approved":true}' --auto-confirm --no-wait`,
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
	fs := updateElementCmd.Flags()
	fs.StringSliceVarP(&flagUpdateElementKeys, "key", "k", nil, "runtime element key(s); repeat, comma-separate or combine with stdin '-'")
	fs.StringVar(&flagUpdateElementVars, "vars", "", "JSON object with variables to set locally on each element")
	fs.StringVar(&flagUpdateElementVarsFile, "vars-file", "", "path to JSON object file with variables to set locally")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview exact local changes without mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after accepted writes without local variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers (default: min(count, 2*GOMAXPROCS, 32))")
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
